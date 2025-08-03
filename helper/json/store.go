package json

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/share"
)

// JSONStore 管理特定目录下的JSON文件
type JSONStore struct {
	// 基础目录，默认为用户家目录下的 .tong
	BaseDir string
	// 子目录，用于区分不同类型的JSON文件
	SubDir string
	// 完整目录路径 (BaseDir + SubDir)
	Path string
}

// NewJSONStore 创建一个新的JSONStore
// subDir 是可选的子目录名称，用于在baseDir下组织文件
func NewJSONStore(subDir string) (*JSONStore, error) {
	// 获取用户家目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("无法获取用户家目录: %w", err)
	}

	// 构建基础目录路径
	baseDir := filepath.Join(homeDir, share.PATH)

	// 构建完整目录路径
	path := baseDir
	if subDir != "" {
		path = filepath.Join(baseDir, subDir)
	}

	// 确保目录存在
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("创建目录失败 %s: %w", path, err)
	}

	return &JSONStore{
		BaseDir: baseDir,
		SubDir:  subDir,
		Path:    path,
	}, nil
}

// ensureJSONExtension 确保文件名有.json扩展名
func ensureJSONExtension(filename string) string {
	if !strings.HasSuffix(strings.ToLower(filename), ".json") {
		return filename + ".json"
	}
	return filename
}

// Get 获取指定名称的JSON文件内容
// 如果decodeInto不为nil，将JSON内容解码到该结构中
func (s *JSONStore) Get(name string, decodeInto interface{}) ([]byte, error) {
	filename := ensureJSONExtension(name)
	filePath := filepath.Join(s.Path, filename)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("文件不存在: %s", filename)
	}

	// 读取文件内容
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败 %s: %w", filename, err)
	}

	// 如果提供了解码目标，解码JSON
	if decodeInto != nil {
		if err := json.Unmarshal(data, decodeInto); err != nil {
			return data, fmt.Errorf("解析JSON失败 %s: %w", filename, err)
		}
	}

	return data, nil
}

// Set 设置或创建指定名称的JSON文件
// data可以是字节数组或任何可以编码为JSON的对象
func (s *JSONStore) Set(name string, data interface{}) error {
	filename := ensureJSONExtension(name)
	filePath := filepath.Join(s.Path, filename)

	var jsonData []byte
	var err error

	// 根据data类型进行处理
	switch v := data.(type) {
	case []byte:
		// 检查是否是有效的JSON
		if !json.Valid(v) {
			return fmt.Errorf("提供的数据不是有效的JSON")
		}
		jsonData = v
	case string:
		// 检查是否是有效的JSON
		if !json.Valid([]byte(v)) {
			return fmt.Errorf("提供的字符串不是有效的JSON")
		}
		jsonData = []byte(v)
	default:
		// 将对象编码为JSON
		jsonData, err = json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("编码为JSON失败: %w", err)
		}
	}

	// 写入文件
	if err := helper.WriteFile(filePath, jsonData); err != nil {
		return fmt.Errorf("写入文件失败 %s: %w", filename, err)
	}

	return nil
}

// Delete 删除指定名称的JSON文件
func (s *JSONStore) Delete(name string) error {
	filename := ensureJSONExtension(name)
	filePath := filepath.Join(s.Path, filename)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("文件不存在: %s", filename)
	}

	// 删除文件
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("删除文件失败 %s: %w", filename, err)
	}

	return nil
}

// List 列出所有JSON文件
// 返回不带.json扩展名的文件名列表
func (s *JSONStore) List() ([]string, error) {
	// 读取目录内容
	entries, err := os.ReadDir(s.Path)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败 %s: %w", s.Path, err)
	}

	var files []string
	for _, entry := range entries {
		// 跳过目录
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// 只处理.json文件
		if strings.HasSuffix(strings.ToLower(name), ".json") {
			// 移除.json扩展名
			name = strings.TrimSuffix(name, ".json")
			files = append(files, name)
		}
	}

	return files, nil
}

// Exists 检查指定名称的JSON文件是否存在
func (s *JSONStore) Exists(name string) bool {
	filename := ensureJSONExtension(name)
	filePath := filepath.Join(s.Path, filename)

	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// Update 更新现有JSON文件的部分内容
// 此方法会读取现有文件，应用更新，然后写回
func (s *JSONStore) Update(name string, updateFunc func(map[string]interface{}) error) error {
	filename := ensureJSONExtension(name)
	filePath := filepath.Join(s.Path, filename)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("文件不存在: %s", filename)
	}

	// 读取现有文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件失败 %s: %w", filename, err)
	}

	// 解析为map
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		return fmt.Errorf("解析JSON失败 %s: %w", filename, err)
	}

	// 应用更新
	if err := updateFunc(jsonMap); err != nil {
		return fmt.Errorf("更新JSON数据失败: %w", err)
	}

	// 编码回JSON
	updatedData, err := json.MarshalIndent(jsonMap, "", "  ")
	if err != nil {
		return fmt.Errorf("编码为JSON失败: %w", err)
	}

	// 写回文件
	if err := helper.WriteFile(filePath, updatedData); err != nil {
		return fmt.Errorf("写入文件失败 %s: %w", filename, err)
	}

	return nil
}

// GetString 获取JSON文件中的字符串值
// 如果文件不存在或键不存在，返回默认值
// 支持嵌套路径，如 "settings.theme"
func (s *JSONStore) GetString(name, key, defaultValue string) string {
	var data map[string]interface{}
	if _, err := s.Get(name, &data); err != nil {
		return defaultValue
	}

	// 处理嵌套路径
	value := getNestedValue(data, key)
	if value == nil {
		return defaultValue
	}

	// 尝试转换为字符串
	if strValue, ok := value.(string); ok {
		return strValue
	}

	return defaultValue
}

// GetInt 获取JSON文件中的整数值
// 如果文件不存在或键不存在，返回默认值
// 支持嵌套路径，如 "settings.fontSize"
func (s *JSONStore) GetInt(name, key string, defaultValue int) int {
	var data map[string]interface{}
	if _, err := s.Get(name, &data); err != nil {
		return defaultValue
	}

	// 处理嵌套路径
	value := getNestedValue(data, key)
	if value == nil {
		return defaultValue
	}

	// 处理不同类型的数字
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	}

	return defaultValue
}

// GetBool 获取JSON文件中的布尔值
// 如果文件不存在或键不存在，返回默认值
// 支持嵌套路径，如 "settings.showNotifications"
func (s *JSONStore) GetBool(name, key string, defaultValue bool) bool {
	var data map[string]interface{}
	if _, err := s.Get(name, &data); err != nil {
		return defaultValue
	}

	// 处理嵌套路径
	value := getNestedValue(data, key)
	if value == nil {
		return defaultValue
	}

	// 尝试转换为布尔值
	if boolValue, ok := value.(bool); ok {
		return boolValue
	}

	return defaultValue
}

// SetValue 设置JSON文件中的单个键值
// 支持嵌套路径，如 "settings.theme"
func (s *JSONStore) SetValue(name, key string, value interface{}) error {
	// 如果文件存在，更新它
	if s.Exists(name) {
		return s.Update(name, func(data map[string]interface{}) error {
			return setNestedValue(data, key, value)
		})
	}

	// 否则创建新文件，并处理嵌套路径
	data := make(map[string]interface{})
	if err := setNestedValue(data, key, value); err != nil {
		return err
	}
	return s.Set(name, data)
}

// Search 搜索匹配指定前缀的JSON文件
func (s *JSONStore) Search(prefix string) ([]string, error) {
	files, err := s.List()
	if err != nil {
		return nil, err
	}

	var matches []string
	for _, file := range files {
		if strings.HasPrefix(file, prefix) {
			matches = append(matches, file)
		}
	}

	return matches, nil
}

// DeleteKey 从JSON文件中删除指定键
// 支持嵌套路径，如 "settings.theme"
func (s *JSONStore) DeleteKey(name, key string) error {
	if !s.Exists(name) {
		return fmt.Errorf("文件不存在: %s", name)
	}

	return s.Update(name, func(data map[string]interface{}) error {
		return deleteNestedValue(data, key)
	})
}

// getNestedValue 获取嵌套JSON中的值
// 例如 key="settings.theme" 将返回 data["settings"]["theme"]
func getNestedValue(data map[string]interface{}, key string) interface{} {
	parts := strings.Split(key, ".")

	// 没有嵌套
	if len(parts) == 1 {
		return data[key]
	}

	// 处理嵌套路径
	current := data
	for i, part := range parts {
		if i == len(parts)-1 {
			// 最后一部分是实际的键
			return current[part]
		}

		// 获取下一级嵌套
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			// 路径中的某一部分不存在或不是对象
			return nil
		}
	}

	return nil
}

// setNestedValue 设置嵌套JSON中的值
// 例如 key="settings.theme" 将设置 data["settings"]["theme"] = value
func setNestedValue(data map[string]interface{}, key string, value interface{}) error {
	parts := strings.Split(key, ".")

	// 没有嵌套
	if len(parts) == 1 {
		data[key] = value
		return nil
	}

	// 处理嵌套路径
	current := data
	for i, part := range parts {
		if i == len(parts)-1 {
			// 最后一部分是实际的键
			current[part] = value
			return nil
		}

		// 获取或创建下一级嵌套
		next, ok := current[part].(map[string]interface{})
		if !ok {
			// 如果不存在或不是对象，创建一个新对象
			next = make(map[string]interface{})
			current[part] = next
		}
		current = next
	}

	return nil
}

// deleteNestedValue 删除嵌套JSON中的值
// 例如 key="settings.theme" 将删除 data["settings"]["theme"]
func deleteNestedValue(data map[string]interface{}, key string) error {
	parts := strings.Split(key, ".")

	// 没有嵌套
	if len(parts) == 1 {
		delete(data, key)
		return nil
	}

	// 处理嵌套路径
	current := data
	for i, part := range parts {
		if i == len(parts)-1 {
			// 最后一部分是实际的键
			delete(current, part)
			return nil
		}

		// 获取下一级嵌套
		next, ok := current[part].(map[string]interface{})
		if !ok {
			// 路径中的某一部分不存在或不是对象
			return fmt.Errorf("路径不存在: %s", key)
		}
		current = next
	}

	return nil
}
