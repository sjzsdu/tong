package analyzer

import (
	"bufio"
	"bytes"
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sjzsdu/tong/project"
)

// DependencyNode 依赖节点
type DependencyNode struct {
	Name    string // 依赖名称
	Version string // 依赖版本
	Type    string // 依赖类型（直接依赖/间接依赖）
}

// DependencyGraph 依赖关系图
type DependencyGraph struct {
	Nodes map[string]*DependencyNode // 依赖节点
	Edges map[string][]string        // 依赖关系
}

// DependencyAnalyzer 依赖分析器接口
type DependencyAnalyzer interface {
	// 分析项目依赖关系
	AnalyzeDependencies(project *project.Project) (*DependencyGraph, error)
}

// LanguageDependencyAnalyzer 特定语言的依赖分析器
type LanguageDependencyAnalyzer interface {
	// 分析特定语言的依赖
	Analyze(content []byte, filePath string) ([]*DependencyNode, []string, error)
}

// DefaultDependencyAnalyzer 默认依赖分析器实现
type DefaultDependencyAnalyzer struct {
	languageAnalyzers map[string]LanguageDependencyAnalyzer
}

// NewDefaultDependencyAnalyzer 创建一个新的默认依赖分析器
func NewDefaultDependencyAnalyzer() *DefaultDependencyAnalyzer {
	return &DefaultDependencyAnalyzer{
		languageAnalyzers: map[string]LanguageDependencyAnalyzer{
			".go":     &GoDependencyAnalyzer{},
			".js":     &JSDependencyAnalyzer{},
			".ts":     &TypeScriptDependencyAnalyzer{},
			".tsx":    &TypeScriptDependencyAnalyzer{},
			".json":   &JSONDependencyAnalyzer{},
			".py":     &PythonDependencyAnalyzer{},
			".java":   &JavaDependencyAnalyzer{},
			".gradle": &GradleDependencyAnalyzer{},
		},
	}
}

// AnalyzeDependencies 实现 DependencyAnalyzer 接口
func (d *DefaultDependencyAnalyzer) AnalyzeDependencies(p *project.Project) (*DependencyGraph, error) {
	graph := &DependencyGraph{
		Nodes: make(map[string]*DependencyNode),
		Edges: make(map[string][]string),
	}

	// 创建访问者函数
	visitor := project.VisitorFunc(func(path string, node *project.Node, depth int) error {
		if node.IsDir {
			return nil
		}

		// 获取文件扩展名
		ext := filepath.Ext(node.Name)
		analyzer, ok := d.languageAnalyzers[ext]
		if !ok {
			return nil
		}

		// 分析依赖
		nodes, edges, err := analyzer.Analyze(node.Content, path)
		if err != nil {
			return err
		}

		// 添加到图中
		for _, node := range nodes {
			graph.Nodes[node.Name] = node
		}

		// 添加边
		for i := 0; i < len(edges)-1; i += 2 {
			src := edges[i]
			dst := edges[i+1]
			graph.Edges[src] = append(graph.Edges[src], dst)
		}

		return nil
	})

	// 遍历项目树
	traverser := project.NewTreeTraverser(p)
	err := traverser.TraverseTree(visitor)
	return graph, err
}

// GoDependencyAnalyzer Go语言依赖分析器
type GoDependencyAnalyzer struct{}

// Analyze 实现 LanguageDependencyAnalyzer 接口
func (g *GoDependencyAnalyzer) Analyze(content []byte, filePath string) ([]*DependencyNode, []string, error) {
	var nodes []*DependencyNode
	var edges []string

	// 检查是否是 go.mod 文件
	if strings.HasSuffix(filePath, "go.mod") {
		return g.analyzeGoMod(content)
	}

	// 分析 Go 源文件中的导入
	scanner := bufio.NewScanner(bytes.NewReader(content))
	packageRegex := regexp.MustCompile(`package\s+(\w+)`)

	var packageName string
	var inImportBlock bool
	var importBlock string

	for scanner.Scan() {
		line := scanner.Text()

		// 获取包名
		if packageName == "" {
			matches := packageRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				packageName = matches[1]
			}
		}

		// 检查是否是单行导入
		singleImportMatches := regexp.MustCompile(`import\s+"([^"]+)"`).FindStringSubmatch(line)
		if len(singleImportMatches) > 1 {
			importPath := singleImportMatches[1]
			if importPath != "" {
				nodes = append(nodes, &DependencyNode{
					Name: importPath,
					Type: "import",
				})
				if packageName != "" {
					edges = append(edges, packageName, importPath)
				}
			}
			continue
		}

		// 检查是否开始导入块
		if strings.Contains(line, "import (") {
			inImportBlock = true
			continue
		}

		// 检查是否结束导入块
		if inImportBlock && strings.Contains(line, ")") {
			inImportBlock = false

			// 处理收集到的导入块
			importLines := strings.Split(importBlock, "\n")
			for _, impLine := range importLines {
				impLine = strings.TrimSpace(impLine)
				if impLine == "" || strings.HasPrefix(impLine, "//") {
					continue
				}

				// 提取引号中的导入路径
				importMatches := regexp.MustCompile(`"([^"]+)"`).FindStringSubmatch(impLine)
				if len(importMatches) > 1 {
					importPath := importMatches[1]
					if importPath != "" {
						nodes = append(nodes, &DependencyNode{
							Name: importPath,
							Type: "import",
						})
						if packageName != "" {
							edges = append(edges, packageName, importPath)
						}
					}
				}
			}

			importBlock = "" // 重置导入块
			continue
		}

		// 收集导入块内的内容
		if inImportBlock {
			importBlock += line + "\n"
		}
	}

	return nodes, edges, nil
}

// analyzeGoMod 分析 go.mod 文件
func (g *GoDependencyAnalyzer) analyzeGoMod(content []byte) ([]*DependencyNode, []string, error) {
	var nodes []*DependencyNode
	var edges []string

	scanner := bufio.NewScanner(bytes.NewReader(content))
	moduleRegex := regexp.MustCompile(`module\s+(.+)`)
	requireRegex := regexp.MustCompile(`require\s+(.+)\s+(.+)`)
	requireBlockRegex := regexp.MustCompile(`require\s+\(`)

	var moduleName string
	inRequireBlock := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// 获取模块名
		if moduleName == "" {
			matches := moduleRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				moduleName = matches[1]
			}
		}

		// 检查是否进入 require 块
		if requireBlockRegex.MatchString(line) {
			inRequireBlock = true
			continue
		}

		// 检查是否退出 require 块
		if inRequireBlock && line == ")" {
			inRequireBlock = false
			continue
		}

		// 处理 require 块内的依赖
		if inRequireBlock {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				depName := parts[0]
				depVersion := parts[1]
				nodes = append(nodes, &DependencyNode{
					Name:    depName,
					Version: depVersion,
					Type:    "direct",
				})
				if moduleName != "" {
					edges = append(edges, moduleName, depName)
				}
			}
			continue
		}

		// 处理单行 require
		matches := requireRegex.FindStringSubmatch(line)
		if len(matches) > 2 {
			depName := matches[1]
			depVersion := matches[2]
			nodes = append(nodes, &DependencyNode{
				Name:    depName,
				Version: depVersion,
				Type:    "direct",
			})
			if moduleName != "" {
				edges = append(edges, moduleName, depName)
			}
		}
	}

	return nodes, edges, nil
}

// JSDependencyAnalyzer JavaScript依赖分析器
type JSDependencyAnalyzer struct{}

// Analyze 实现 LanguageDependencyAnalyzer 接口
func (j *JSDependencyAnalyzer) Analyze(content []byte, filePath string) ([]*DependencyNode, []string, error) {
	var nodes []*DependencyNode
	var edges []string

	// 分析 JS 源文件中的导入
	scanner := bufio.NewScanner(bytes.NewReader(content))
	// 改进正则表达式，更精确地匹配JS导入语句
	importRegex := regexp.MustCompile(`import\s+.*?from\s+['"]([^'"]+)['"]|require\(\s*['"]([^'"]+)['"]\s*\)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过注释和空行
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
			continue
		}

		// 查找导入
		matches := importRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			var importPath string
			if match[1] != "" {
				importPath = match[1]
			} else if match[2] != "" {
				importPath = match[2]
			}

			if importPath != "" && !strings.Contains(importPath, "{") && !strings.Contains(importPath, "(") {
				nodes = append(nodes, &DependencyNode{
					Name: importPath,
					Type: "import",
				})
				// 使用文件路径作为源节点
				edges = append(edges, filePath, importPath)
			}
		}
	}

	return nodes, edges, nil
}

// JSONDependencyAnalyzer JSON依赖分析器（主要用于package.json）
type JSONDependencyAnalyzer struct{}

// Analyze 实现 LanguageDependencyAnalyzer 接口
func (j *JSONDependencyAnalyzer) Analyze(content []byte, filePath string) ([]*DependencyNode, []string, error) {
	var nodes []*DependencyNode
	var edges []string

	// 只处理 package.json 文件
	if !strings.HasSuffix(filePath, "package.json") {
		return nodes, edges, nil
	}

	// 解析 JSON
	var packageJSON struct {
		Name            string            `json:"name"`
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	if err := json.Unmarshal(content, &packageJSON); err != nil {
		return nil, nil, err
	}

	// 处理依赖
	for name, version := range packageJSON.Dependencies {
		nodes = append(nodes, &DependencyNode{
			Name:    name,
			Version: version,
			Type:    "direct",
		})
		if packageJSON.Name != "" {
			edges = append(edges, packageJSON.Name, name)
		}
	}

	// 处理开发依赖
	for name, version := range packageJSON.DevDependencies {
		nodes = append(nodes, &DependencyNode{
			Name:    name,
			Version: version,
			Type:    "dev",
		})
		if packageJSON.Name != "" {
			edges = append(edges, packageJSON.Name, name)
		}
	}

	return nodes, edges, nil
}

// PythonDependencyAnalyzer Python依赖分析器
type PythonDependencyAnalyzer struct{}

// Analyze 实现 LanguageDependencyAnalyzer 接口
func (p *PythonDependencyAnalyzer) Analyze(content []byte, filePath string) ([]*DependencyNode, []string, error) {
	var nodes []*DependencyNode
	var edges []string

	// 检查是否是 requirements.txt 文件
	if strings.HasSuffix(filePath, "requirements.txt") {
		return p.analyzeRequirementsTxt(content, filePath)
	}

	// 分析 Python 源文件中的导入
	scanner := bufio.NewScanner(bytes.NewReader(content))
	// 改进正则表达式，更精确地匹配Python导入语句
	importRegex := regexp.MustCompile(`^\s*import\s+([\w\.]+)\s*$|^\s*from\s+([\w\.]+)\s+import`)

	for scanner.Scan() {
		line := scanner.Text()

		// 跳过注释和空行
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// 查找导入
		matches := importRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			var importPath string
			if matches[1] != "" {
				importPath = matches[1]
			} else if matches[2] != "" {
				importPath = matches[2]
			}

			if importPath != "" && !strings.Contains(importPath, " ") {
				// 获取顶级包名
				topLevelPackage := strings.Split(importPath, ".")[0]
				if topLevelPackage != "" {
					nodes = append(nodes, &DependencyNode{
						Name: topLevelPackage,
						Type: "import",
					})
					// 使用文件路径作为源节点
					edges = append(edges, filePath, topLevelPackage)
				}
			}
		}
	}

	return nodes, edges, nil
}

// analyzeRequirementsTxt 分析 requirements.txt 文件
func (p *PythonDependencyAnalyzer) analyzeRequirementsTxt(content []byte, filePath string) ([]*DependencyNode, []string, error) {
	var nodes []*DependencyNode
	var edges []string

	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 解析依赖
		parts := strings.Split(line, "==")
		if len(parts) >= 2 {
			name := strings.TrimSpace(parts[0])
			version := strings.TrimSpace(parts[1])
			nodes = append(nodes, &DependencyNode{
				Name:    name,
				Version: version,
				Type:    "direct",
			})
			// 使用文件路径作为源节点
			edges = append(edges, filePath, name)
		} else {
			// 处理没有版本的依赖
			name := strings.TrimSpace(line)
			nodes = append(nodes, &DependencyNode{
				Name: name,
				Type: "direct",
			})
			// 使用文件路径作为源节点
			edges = append(edges, filePath, name)
		}
	}

	return nodes, edges, nil
}

// JavaDependencyAnalyzer Java依赖分析器
type JavaDependencyAnalyzer struct{}

// Analyze 实现 LanguageDependencyAnalyzer 接口
func (j *JavaDependencyAnalyzer) Analyze(content []byte, filePath string) ([]*DependencyNode, []string, error) {
	var nodes []*DependencyNode
	var edges []string

	// 分析 Java 源文件中的导入
	scanner := bufio.NewScanner(bytes.NewReader(content))
	importRegex := regexp.MustCompile(`^\s*import\s+(static\s+)?([\w\.\*]+);`)
	packageRegex := regexp.MustCompile(`^\s*package\s+([\w\.]+);`)

	var packageName string
	for scanner.Scan() {
		line := scanner.Text()

		// 跳过注释和空行
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "//") || strings.HasPrefix(trimmedLine, "/*") {
			continue
		}

		// 获取包名
		if packageName == "" {
			matches := packageRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				packageName = matches[1]
			}
		}

		// 查找导入
		matches := importRegex.FindStringSubmatch(line)
		if len(matches) > 2 {
			importPath := matches[2]
			if importPath != "" && !strings.Contains(importPath, "(") {
				// 获取顶级包名
				topLevelPackage := strings.Split(importPath, ".")[0]
				if topLevelPackage != "" {
					nodes = append(nodes, &DependencyNode{
						Name: topLevelPackage,
						Type: "import",
					})
					if packageName != "" {
						edges = append(edges, packageName, topLevelPackage)
					}
				}
			}
		}
	}

	return nodes, edges, nil
}

// GradleDependencyAnalyzer Gradle依赖分析器
type GradleDependencyAnalyzer struct{}

// Analyze 实现 LanguageDependencyAnalyzer 接口
func (g *GradleDependencyAnalyzer) Analyze(content []byte, filePath string) ([]*DependencyNode, []string, error) {
	var nodes []*DependencyNode
	var edges []string

	// 只处理 build.gradle 文件
	if !strings.HasSuffix(filePath, "build.gradle") {
		return nodes, edges, nil
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	dependencyRegex := regexp.MustCompile(`\s*([\w]+)\s*['"]([^'"]+):([^'"]+):([^'"]+)['"]`)
	inDependenciesBlock := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 检查是否进入 dependencies 块
		if strings.HasPrefix(line, "dependencies {") {
			inDependenciesBlock = true
			continue
		}

		// 检查是否退出 dependencies 块
		if inDependenciesBlock && line == "}" {
			inDependenciesBlock = false
			continue
		}

		// 处理 dependencies 块内的依赖
		if inDependenciesBlock {
			matches := dependencyRegex.FindStringSubmatch(line)
			if len(matches) > 4 {
				depType := matches[1]  // implementation, api, etc.
				group := matches[2]    // group ID
				artifact := matches[3] // artifact ID
				version := matches[4]  // version

				depName := group + ":" + artifact
				nodes = append(nodes, &DependencyNode{
					Name:    depName,
					Version: version,
					Type:    depType,
				})
				// 使用文件路径作为源节点
				edges = append(edges, filePath, depName)
			}
		}
	}

	return nodes, edges, nil
}

// TypeScriptDependencyAnalyzer TypeScript依赖分析器
type TypeScriptDependencyAnalyzer struct {
	// 复用JavaScript分析器的大部分逻辑
	jsAnalyzer JSDependencyAnalyzer
}

// Analyze 实现 LanguageDependencyAnalyzer 接口
func (t *TypeScriptDependencyAnalyzer) Analyze(content []byte, filePath string) ([]*DependencyNode, []string, error) {
	// TypeScript的导入语法与JavaScript基本相同，但还支持类型导入语法
	var nodes []*DependencyNode
	var edges []string

	// 先使用JavaScript分析器处理基本的导入语句
	jsNodes, jsEdges, err := t.jsAnalyzer.Analyze(content, filePath)
	if err != nil {
		return nil, nil, err
	}

	nodes = append(nodes, jsNodes...)
	edges = append(edges, jsEdges...)

	// 分析TypeScript特有的导入语法，如 import type { ... } from '...'
	scanner := bufio.NewScanner(bytes.NewReader(content))
	typeImportRegex := regexp.MustCompile(`import\s+type\s+.*?from\s+['"]([^'"]+)['"]`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过注释和空行
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
			continue
		}

		// 查找类型导入
		typeMatches := typeImportRegex.FindStringSubmatch(line)
		if len(typeMatches) > 1 {
			importPath := typeMatches[1]
			if importPath != "" && !strings.Contains(importPath, "{") && !strings.Contains(importPath, "(") {
				// 检查是否已经添加过该依赖
				alreadyAdded := false
				for _, node := range nodes {
					if node.Name == importPath {
						alreadyAdded = true
						break
					}
				}

				if !alreadyAdded {
					nodes = append(nodes, &DependencyNode{
						Name: importPath,
						Type: "import",
					})
					edges = append(edges, filePath, importPath)
				}
			}
		}
	}

	return nodes, edges, nil
}
