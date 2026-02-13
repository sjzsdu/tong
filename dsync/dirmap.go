package dsync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// DirMap 嵌套 map 类型别名
type DirMap = map[string]interface{}

// DirData 目录的核心数据结构
//
// 包含一个大 map（Data）以及元信息，用于区分哪些 key 是目录、哪些是文件、
// 哪些是 configFile 中的配置项。这样从 map 写回目录时能正确还原。
type DirData struct {
	Data       DirMap            // 大 map
	DirKeys    map[string]bool   // 顶层中哪些 key 来自子目录
	FileKeys   map[string]bool   // 顶层中哪些 key 来自独立文件（非 configFile）
	ConfigFile string            // 配置文件名（如 "opencode.json"）
}

// NewEmptyDirData 创建一个空的 DirData
func NewEmptyDirData(configFile string) *DirData {
	return &DirData{
		Data:       make(DirMap),
		DirKeys:    make(map[string]bool),
		FileKeys:   make(map[string]bool),
		ConfigFile: configFile,
	}
}

// BuildDirData 从目录构建 DirData
//
// dirPath: 目录路径
// configFile: JSON 配置文件名（如 "opencode.json"），其内容展开到顶层 map
func BuildDirData(dirPath string, configFile string) (*DirData, error) {
	dd := &DirData{
		Data:       make(DirMap),
		DirKeys:    make(map[string]bool),
		FileKeys:   make(map[string]bool),
		ConfigFile: configFile,
	}

	// 1. 读取 configFile，展开到顶层
	configPath := filepath.Join(dirPath, configFile)
	if data, err := os.ReadFile(configPath); err == nil {
		var configMap map[string]interface{}
		if err := json.Unmarshal(data, &configMap); err != nil {
			cleaned := cleanJSON(string(data))
			json.Unmarshal([]byte(cleaned), &configMap)
		}
		for k, v := range configMap {
			dd.Data[k] = v
		}
	}

	// 2. 遍历目录，填入子目录和其他文件
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("无法读取目录 %s: %w", dirPath, err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if name == configFile {
			continue
		}

		fullPath := filepath.Join(dirPath, name)

		if entry.IsDir() {
			subMap, err := buildSubMap(fullPath)
			if err != nil {
				return nil, err
			}
			dd.Data[name] = subMap
			dd.DirKeys[name] = true
		} else {
			content, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}
			dd.Data[name] = string(content)
			dd.FileKeys[name] = true
		}
	}

	return dd, nil
}

// buildSubMap 递归构建子目录的 map
func buildSubMap(dirPath string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("无法读取目录 %s: %w", dirPath, err)
	}

	for _, entry := range entries {
		name := entry.Name()
		fullPath := filepath.Join(dirPath, name)

		if entry.IsDir() {
			subMap, err := buildSubMap(fullPath)
			if err != nil {
				return nil, err
			}
			result[name] = subMap
		} else {
			content, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}
			result[name] = string(content)
		}
	}

	return result, nil
}

// navigate 智能导航：在嵌套结构中根据 parts 查找值
//
// 处理文件名含 "." 的情况：当单个 part 找不到时，尝试合并后续 parts 匹配
// 例如 parts=["agents","agent1","md"] 能匹配到 map 中的 "agent1.md"
//
// 同时支持无扩展名匹配：parts=["agents","agent1"] 能匹配 "agent1.md"
func navigate(data interface{}, parts []string) (interface{}, []string, bool) {
	if len(parts) == 0 {
		return data, nil, true
	}

	switch c := data.(type) {
	case map[string]interface{}:
		// 1. 精确匹配单个 part
		if val, ok := c[parts[0]]; ok {
			result, realParts, found := navigate(val, parts[1:])
			if found {
				return result, append([]string{parts[0]}, realParts...), true
			}
		}
		// 2. 尝试合并连续 parts 匹配含 "." 的 key（如 "agent1.md"）
		for i := 2; i <= len(parts); i++ {
			combined := strings.Join(parts[:i], ".")
			if val, ok := c[combined]; ok {
				result, realParts, found := navigate(val, parts[i:])
				if found {
					return result, append([]string{combined}, realParts...), true
				}
			}
		}
		// 3. 无扩展名匹配
		for k, v := range c {
			if stripExt(k) == parts[0] {
				result, realParts, found := navigate(v, parts[1:])
				if found {
					return result, append([]string{k}, realParts...), true
				}
			}
		}
		return nil, nil, false

	case []interface{}:
		idx, err := strconv.Atoi(parts[0])
		if err != nil || idx < 0 || idx >= len(c) {
			return nil, nil, false
		}
		result, realParts, found := navigate(c[idx], parts[1:])
		if found {
			return result, append([]string{parts[0]}, realParts...), true
		}
		return nil, nil, false

	default:
		return nil, nil, false
	}
}

// GetValue 根据 "." 分隔的 key 路径获取值
//
// 支持文件名含 ".": agents.agent1.md 能正确匹配
// 支持无扩展名: agents.agent1 能匹配 agent1.md
// 支持数组索引: plugin.0
func (dd *DirData) GetValue(keyPath string) (interface{}, bool) {
	parts := strings.Split(keyPath, ".")
	result, _, found := navigate(dd.Data, parts)
	return result, found
}

// ResolveSegments 将用户输入的 key 路径解析为真实的 key 段列表
//
// 例如 "agents.agent1" -> ["agents", "agent1.md"]
// 每一段都是 map 中实际存在的 key，不会被 "." 拆散
func (dd *DirData) ResolveSegments(keyPath string) ([]string, bool) {
	parts := strings.Split(keyPath, ".")
	_, realParts, found := navigate(dd.Data, parts)
	return realParts, found
}

// setBySegments 根据 key 段列表设置值（每段是完整的 map key，不会被拆分）
func (dd *DirData) setBySegments(segments []string, value interface{}) {
	if len(segments) == 0 {
		return
	}
	if len(segments) == 1 {
		dd.Data[segments[0]] = value
		return
	}

	// 逐层定位到父容器
	current := dd.Data
	for i := 0; i < len(segments)-1; i++ {
		seg := segments[i]

		next, ok := current[seg]
		if !ok {
			// 下一段是数字 -> 创建数组，否则创建 map
			if _, err := strconv.Atoi(segments[i+1]); err == nil {
				next = make([]interface{}, 0)
			} else {
				next = make(map[string]interface{})
			}
			current[seg] = next
		}

		switch c := next.(type) {
		case map[string]interface{}:
			current = c
		case []interface{}:
			// 数组操作：append
			lastSeg := segments[len(segments)-1]
			if _, err := strconv.Atoi(lastSeg); err == nil {
				c = append(c, value)
				current[seg] = c
				return
			}
			return
		default:
			return
		}
	}

	current[segments[len(segments)-1]] = value
}

// deleteBySegments 根据 key 段列表删除值
func (dd *DirData) deleteBySegments(segments []string) bool {
	if len(segments) == 0 {
		return false
	}

	if len(segments) == 1 {
		if _, ok := dd.Data[segments[0]]; ok {
			delete(dd.Data, segments[0])
			delete(dd.DirKeys, segments[0])
			delete(dd.FileKeys, segments[0])
			return true
		}
		return false
	}

	// navigate 找到倒数第二层
	parentSegs := segments[:len(segments)-1]
	parent, _, ok := navigate(dd.Data, parentSegs)
	if !ok {
		return false
	}

	lastKey := segments[len(segments)-1]
	switch c := parent.(type) {
	case map[string]interface{}:
		if _, ok := c[lastKey]; ok {
			delete(c, lastKey)
			return true
		}
	case []interface{}:
		idx, err := strconv.Atoi(lastKey)
		if err != nil || idx < 0 || idx >= len(c) {
			return false
		}
		newArr := append(c[:idx], c[idx+1:]...)
		// 回写到祖父层
		if len(parentSegs) >= 1 {
			grandSegs := parentSegs[:len(parentSegs)-1]
			if len(grandSegs) == 0 {
				dd.Data[parentSegs[0]] = newArr
			} else {
				grandParent, _, ok := navigate(dd.Data, grandSegs)
				if ok {
					if m, ok := grandParent.(map[string]interface{}); ok {
						m[parentSegs[len(parentSegs)-1]] = newArr
					}
				}
			}
		}
		return true
	}
	return false
}

// DeleteValue 根据 "." 分隔的 key 路径删除值
func (dd *DirData) DeleteValue(keyPath string) bool {
	segments, found := dd.ResolveSegments(keyPath)
	if !found {
		return false
	}
	return dd.deleteBySegments(segments)
}

// SyncFrom 从源 DirData 同步指定 key 到当前 DirData
//
// 支持 "." 分隔的 key 路径，如 "theme", "mcp.git-mcp-server", "agents"
// 支持无扩展名的 key，如 "agents.agent1" 会自动匹配 "agents.agent1.md"
func (dd *DirData) SyncFrom(source *DirData, keyPath string) error {
	// 在源中解析出真实的 key 段列表
	segments, found := source.ResolveSegments(keyPath)
	if !found {
		return fmt.Errorf("源中未找到 key: %s", keyPath)
	}

	// 用段列表获取值
	value, _, ok := navigate(source.Data, segments)
	if !ok {
		return fmt.Errorf("源中未找到 key: %s", keyPath)
	}

	// 用段列表设置到目标（每段是完整 key，不会被 "." 拆散）
	dd.setBySegments(segments, value)

	// 同步顶层 key 的元信息
	topKey := segments[0]
	if source.DirKeys[topKey] {
		dd.DirKeys[topKey] = true
	}
	if source.FileKeys[topKey] {
		dd.FileKeys[topKey] = true
	}

	return nil
}

// WriteTo 将 DirData 写回目标目录
//
// 根据 DirKeys/FileKeys 元信息：
//   - DirKeys 中的 key -> 写为子目录
//   - FileKeys 中的 key -> 写为独立文件
//   - 其余 key -> 写入 configFile
func (dd *DirData) WriteTo(targetPath string) error {
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return err
	}

	configData := make(map[string]interface{})

	for key, value := range dd.Data {
		if dd.DirKeys[key] {
			// 子目录
			if subMap, ok := value.(map[string]interface{}); ok {
				if err := writeSubDir(filepath.Join(targetPath, key), subMap); err != nil {
					return err
				}
			}
		} else if dd.FileKeys[key] {
			// 独立文件
			switch v := value.(type) {
			case string:
				if err := os.WriteFile(filepath.Join(targetPath, key), []byte(v), 0644); err != nil {
					return err
				}
			default:
				b, _ := json.MarshalIndent(v, "", "  ")
				if err := os.WriteFile(filepath.Join(targetPath, key), b, 0644); err != nil {
					return err
				}
			}
		} else {
			// 配置项 -> configFile
			configData[key] = value
		}
	}

	// 写入 configFile
	if len(configData) > 0 {
		b, err := json.MarshalIndent(configData, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(targetPath, dd.ConfigFile), b, 0644); err != nil {
			return err
		}
	}

	return nil
}

func writeSubDir(dirPath string, data map[string]interface{}) error {
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}
	for key, value := range data {
		switch v := value.(type) {
		case map[string]interface{}:
			if err := writeSubDir(filepath.Join(dirPath, key), v); err != nil {
				return err
			}
		case string:
			if err := os.WriteFile(filepath.Join(dirPath, key), []byte(v), 0644); err != nil {
				return err
			}
		default:
			b, _ := json.MarshalIndent(v, "", "  ")
			if err := os.WriteFile(filepath.Join(dirPath, key), b, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

// stripExt 去掉文件扩展名: "agent1.md" -> "agent1"
func stripExt(name string) string {
	if idx := strings.LastIndex(name, "."); idx > 0 {
		return name[:idx]
	}
	return name
}

// cleanJSON 清理非标准 JSON（去除尾逗号）
func cleanJSON(content string) string {
	re1 := regexp.MustCompile(`,\s*}`)
	content = re1.ReplaceAllString(content, "}")
	re2 := regexp.MustCompile(`,\s*\]`)
	content = re2.ReplaceAllString(content, "]")
	return content
}

// --- 展示 ---

// DisplayDirMap 展示全部（不限深度）
func DisplayDirMap(data DirMap) string {
	return DisplayDirMapWithDepth(data, 0)
}

// DisplayDirMapWithDepth 展示 DirMap，key 用 "." 连接完整路径
//
// maxDepth=0 显示全部，maxDepth=1 只显示顶层，maxDepth=2 显示两层，以此类推
func DisplayDirMapWithDepth(data DirMap, maxDepth int) string {
	if len(data) == 0 {
		return "(empty)\n"
	}
	var sb strings.Builder
	displayFlat(&sb, data, "", 1, maxDepth, 200)
	return sb.String()
}

func displayFlat(sb *strings.Builder, data map[string]interface{}, prefix string, depth int, maxDepth int, maxLen int) {
	keys := sortedKeys(data)
	for _, key := range keys {
		value := data[key]

		// 文件名（string 值）去掉扩展名显示
		displayKey := key
		if _, isStr := value.(string); isStr && strings.Contains(key, ".") {
			displayKey = stripExt(key)
		}

		fullKey := displayKey
		if prefix != "" {
			fullKey = prefix + "." + displayKey
		}

		displayValue(sb, fullKey, value, depth, maxDepth, maxLen)
	}
}

func displayValue(sb *strings.Builder, fullKey string, value interface{}, depth int, maxDepth int, maxLen int) {
	switch v := value.(type) {
	case map[string]interface{}:
		if maxDepth > 0 && depth >= maxDepth {
			childKeys := sortedKeys(v)
			if len(childKeys) == 0 {
				sb.WriteString(fmt.Sprintf("%s: (empty)\n", fullKey))
			} else {
				sb.WriteString(fmt.Sprintf("%s: {%s}\n", fullKey, strings.Join(childKeys, ", ")))
			}
		} else {
			displayFlat(sb, v, fullKey, depth+1, maxDepth, maxLen)
		}
	case []interface{}:
		// 数组始终作为整体显示，不展开
		b, _ := json.Marshal(v)
		display := string(b)
		if maxLen > 0 && len(display) > maxLen {
			display = display[:maxLen] + "..."
		}
		sb.WriteString(fmt.Sprintf("%s: %s\n", fullKey, display))
	case string:
		display := strings.TrimSpace(v)
		display = strings.ReplaceAll(display, "\n", " ")
		if maxLen > 0 && len(display) > maxLen {
			display = display[:maxLen] + "..."
		}
		sb.WriteString(fmt.Sprintf("%s: %s\n", fullKey, display))
	default:
		b, _ := json.Marshal(v)
		display := string(b)
		if maxLen > 0 && len(display) > maxLen {
			display = display[:maxLen] + "..."
		}
		sb.WriteString(fmt.Sprintf("%s: %s\n", fullKey, display))
	}
}

// ToJSON 转为格式化 JSON 字符串
func ToJSON(data DirMap) (string, error) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// sortedKeys map 类型排前面，然后按名称排序
func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		_, iIsMap := m[keys[i]].(map[string]interface{})
		_, jIsMap := m[keys[j]].(map[string]interface{})
		if iIsMap != jIsMap {
			return iIsMap
		}
		return keys[i] < keys[j]
	})
	return keys
}
