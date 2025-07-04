package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGoProject 测试使用Go项目的功能
func TestGoProject(t *testing.T) {
	// 创建一个示例项目
	projectPath := CreateExampleGoProject(t)
	defer os.RemoveAll(projectPath) // 测试结束后清理

	// 使用示例项目创建 GoProject 实例
	goProject := GetSharedProject(t, projectPath)

	// 验证项目结构
	assert.NotNil(t, goProject.Project)
	assert.Equal(t, projectPath, goProject.RootPath)

	// 验证项目文件
	files, err := goProject.Project.GetAllFiles()
	assert.NoError(t, err)

	// 检查是否包含预期的文件
	expectedFiles := []string{
		"/go.mod",
		"/main.go",
		"/pkg/utils/greeting.go",
		"/pkg/utils/math.go",
		"/pkg/config/config.go",
		"/pkg/utils/greeting_test.go",
		"/pkg/utils/math_test.go",
		"/config.json",
		"/.env",
		"/README.md",
		"/docs/api.md",
	}

	// 检查每个预期文件是否存在
	for _, expectedFile := range expectedFiles {
		found := false
		for _, file := range files {
			// files 中的路径是相对于项目根目录的路径，不需要再次转换
			if filepath.ToSlash(file) == expectedFile {
				found = true
				break
			}
		}
		assert.True(t, found, "预期文件未找到: %s", expectedFile)
	}

	// 测试读取文件内容
	content, err := goProject.Project.ReadFile("/main.go")
	assert.NoError(t, err)
	assert.Contains(t, string(content), "package main")

	// 测试获取绝对路径
	absPath := goProject.GetAbsolutePath("/pkg/utils/greeting.go")
	expectedPath := filepath.Join(projectPath, "pkg", "utils", "greeting.go")
	assert.Equal(t, expectedPath, absPath)
}

// TestProjectBasicOperations 测试项目基本操作
func TestProjectBasicOperations(t *testing.T) {
	// 使用共享项目
	goProject := GetSharedProject(t, "")
	project := goProject.GetProject()

	// 测试 GetRootPath
	rootPath := project.GetRootPath()
	assert.NotEmpty(t, rootPath)

	// 测试 GetName
	name := project.GetName()
	assert.NotEmpty(t, name)

	// 测试项目不为空
	assert.False(t, project.IsEmpty())

	// 测试 GetTotalNodes
	totalNodes := project.GetTotalNodes()
	assert.Greater(t, totalNodes, 0)
}

// TestProjectFileOperations 测试项目文件操作
func TestProjectFileOperations(t *testing.T) {
	goProject := GetSharedProject(t, "")
	project := goProject.GetProject()

	// 测试读取存在的文件
	content, err := project.ReadFile("/main.go")
	assert.NoError(t, err)
	assert.Contains(t, string(content), "package main")

	// 测试读取不存在的文件
	_, err = project.ReadFile("/nonexistent.go")
	assert.Error(t, err)

	// 测试读取目录（应该报错）
	_, err = project.ReadFile("/pkg")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read directory")
}

// TestProjectWriteOperations 测试项目写入操作
func TestProjectWriteOperations(t *testing.T) {
	goProject := GetSharedProject(t, "")
	project := goProject.GetProject()

	// 测试写入现有文件
	newContent := []byte("// 修改后的内容\npackage main\n\nfunc main() {\n\tfmt.Println(\"Modified!\")\n}\n")
	err := project.WriteFile("/main.go", newContent)
	assert.NoError(t, err)

	// 验证写入是否成功
	content, err := project.ReadFile("/main.go")
	assert.NoError(t, err)
	assert.Equal(t, newContent, content)

	// 测试写入不存在的文件
	err = project.WriteFile("/nonexistent.go", []byte("test"))
	assert.Error(t, err)

	// 测试写入目录（应该报错）
	err = project.WriteFile("/pkg", []byte("test"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot write to directory")
}

// TestProjectFindNode 测试查找节点功能
func TestProjectFindNode(t *testing.T) {
	goProject := GetSharedProject(t, "")
	project := goProject.GetProject()

	// 测试查找根节点
	rootNode, err := project.FindNode("/")
	assert.NoError(t, err)
	assert.True(t, rootNode.IsDir)
	assert.Equal(t, "/", rootNode.Name)

	// 测试查找文件节点
	fileNode, err := project.FindNode("/main.go")
	assert.NoError(t, err)
	assert.False(t, fileNode.IsDir)
	assert.Equal(t, "main.go", fileNode.Name)

	// 测试查找目录节点
	dirNode, err := project.FindNode("/pkg")
	assert.NoError(t, err)
	assert.True(t, dirNode.IsDir)
	assert.Equal(t, "pkg", dirNode.Name)

	// 测试查找不存在的节点
	_, err = project.FindNode("/nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path not found")

	// 测试查找嵌套路径
	nestedNode, err := project.FindNode("/pkg/utils/greeting.go")
	assert.NoError(t, err)
	assert.False(t, nestedNode.IsDir)
	assert.Equal(t, "greeting.go", nestedNode.Name)
}

// TestProjectGetAllFiles 测试获取所有文件
func TestProjectGetAllFiles(t *testing.T) {
	goProject := GetSharedProject(t, "")
	project := goProject.GetProject()

	files, err := project.GetAllFiles()
	assert.NoError(t, err)
	assert.NotEmpty(t, files)

	// 验证所有返回的都是文件路径（不是目录）
	for _, file := range files {
		node, err := project.FindNode(file)
		assert.NoError(t, err)
		assert.False(t, node.IsDir, "返回的路径应该都是文件: %s", file)
	}

	// 验证包含主要文件
	fileSet := make(map[string]bool)
	for _, file := range files {
		fileSet[file] = true
	}

	assert.True(t, fileSet["/main.go"], "应该包含 main.go")
	assert.True(t, fileSet["/go.mod"], "应该包含 go.mod")
}

// TestProjectGetAbsolutePath 测试获取绝对路径
func TestProjectGetAbsolutePath(t *testing.T) {
	goProject := GetSharedProject(t, "")
	project := goProject.GetProject()

	// 测试获取文件的绝对路径
	absPath := project.GetAbsolutePath("main.go")
	expectedPath := filepath.Join(goProject.RootPath, "main.go")
	assert.Equal(t, expectedPath, absPath)

	// 测试获取目录的绝对路径
	dirAbsPath := project.GetAbsolutePath("pkg/utils")
	expectedDirPath := filepath.Join(goProject.RootPath, "pkg", "utils")
	assert.Equal(t, expectedDirPath, dirAbsPath)
}

// TestProjectNodeHashCalculation 测试节点哈希计算
func TestProjectNodeHashCalculation(t *testing.T) {
	goProject := GetSharedProject(t, "")
	project := goProject.GetProject()

	// 测试文件节点哈希计算
	fileNode, err := project.FindNode("/main.go")
	assert.NoError(t, err)

	hash1, err := fileNode.CalculateHash()
	assert.NoError(t, err)
	assert.NotEmpty(t, hash1)

	// 相同内容应该产生相同的哈希
	hash2, err := fileNode.CalculateHash()
	assert.NoError(t, err)
	assert.Equal(t, hash1, hash2)

	// 测试目录节点哈希计算
	dirNode, err := project.FindNode("/pkg")
	assert.NoError(t, err)

	dirHash, err := dirNode.CalculateHash()
	assert.NoError(t, err)
	assert.NotEmpty(t, dirHash)

	// 测试根节点哈希计算
	rootNode, err := project.FindNode("/")
	assert.NoError(t, err)

	rootHash, err := rootNode.CalculateHash()
	assert.NoError(t, err)
	assert.NotEmpty(t, rootHash)
}

// TestTreeTraverser 测试树遍历器
func TestTreeTraverser(t *testing.T) {
	goProject := GetSharedProject(t, "")
	project := goProject.GetProject()

	// 测试前序遍历
	var visitedPaths []string
	visitor := VisitorFunc(func(path string, node *Node, depth int) error {
		visitedPaths = append(visitedPaths, path)
		return nil
	})

	traverser := NewTreeTraverser(project)
	err := traverser.SetTraverseOrder(PreOrder).TraverseTree(visitor)
	assert.NoError(t, err)
	assert.NotEmpty(t, visitedPaths)

	// 验证根路径被访问
	assert.Contains(t, visitedPaths, "/")

	// 测试后序遍历
	visitedPaths = []string{}
	err = traverser.SetTraverseOrder(PostOrder).TraverseTree(visitor)
	assert.NoError(t, err)
	assert.NotEmpty(t, visitedPaths)

	// 测试中序遍历
	visitedPaths = []string{}
	err = traverser.SetTraverseOrder(InOrder).TraverseTree(visitor)
	assert.NoError(t, err)
	assert.NotEmpty(t, visitedPaths)
}

// TestTreeTraverserWithFilter 测试带过滤的遍历
func TestTreeTraverserWithFilter(t *testing.T) {
	goProject := GetSharedProject(t, "")
	project := goProject.GetProject()

	// 只访问 .go 文件
	var goFiles []string
	visitor := VisitorFunc(func(path string, node *Node, depth int) error {
		if !node.IsDir && filepath.Ext(node.Name) == ".go" {
			goFiles = append(goFiles, path)
		}
		return nil
	})

	traverser := NewTreeTraverser(project)
	err := traverser.TraverseTree(visitor)
	assert.NoError(t, err)
	assert.NotEmpty(t, goFiles)

	// 验证所有文件都是 .go 文件
	for _, file := range goFiles {
		assert.True(t, strings.HasSuffix(file, ".go"), "文件应该以 .go 结尾: %s", file)
	}
}

// TestBaseExporter 测试基础导出器
func TestBaseExporter(t *testing.T) {
	goProject := GetSharedProject(t, "")
	project := goProject.GetProject()

	// 创建一个简单的内容收集器用于测试
	collector := &MockContentCollector{
		titles:   []string{},
		contents: []string{},
		tocItems: []string{},
	}

	exporter := NewBaseExporter(project, collector)

	// 测试目录访问
	err := exporter.VisitDirectory(project.root, "/", 0)
	assert.NoError(t, err)

	// 测试文件访问
	fileNode, err := project.FindNode("/main.go")
	assert.NoError(t, err)

	err = exporter.VisitFile(fileNode, "/main.go", 1)
	assert.NoError(t, err)
}

// TestBuildProjectTree 测试构建项目树
func TestBuildProjectTree(t *testing.T) {
	// 创建一个临时项目用于测试
	tempDir := CreateExampleGoProject(t)
	defer os.RemoveAll(tempDir)

	// 测试默认选项
	options := DefaultWalkDirOptions()
	builtProject, err := BuildProjectTree(tempDir, options)
	assert.NoError(t, err)
	assert.NotNil(t, builtProject)
	assert.False(t, builtProject.IsEmpty())

	// 测试只包含特定扩展名的文件
	options.Extensions = []string{"go"}
	goOnlyProject, err := BuildProjectTree(tempDir, options)
	assert.NoError(t, err)
	assert.NotNil(t, goOnlyProject)

	files, err := goOnlyProject.GetAllFiles()
	assert.NoError(t, err)
	for _, file := range files {
		assert.True(t, strings.HasSuffix(file, ".go"), "应该只包含 .go 文件: %s", file)
	}

	// 测试禁用 .gitignore
	options.DisableGitIgnore = true
	options.Extensions = []string{"*"}
	fullProject, err := BuildProjectTree(tempDir, options)
	assert.NoError(t, err)
	assert.NotNil(t, fullProject)
}

// TestProjectWithEmptyDirectory 测试空目录处理
func TestProjectWithEmptyDirectory(t *testing.T) {
	// 创建一个临时空目录
	tempDir, err := os.MkdirTemp("", "empty-project-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 构建空项目
	options := DefaultWalkDirOptions()
	emptyProject, err := BuildProjectTree(tempDir, options)
	assert.NoError(t, err)
	assert.NotNil(t, emptyProject)
	assert.True(t, emptyProject.IsEmpty())

	// 测试空项目的方法
	files, err := emptyProject.GetAllFiles()
	assert.NoError(t, err)
	assert.Empty(t, files)

	totalNodes := emptyProject.GetTotalNodes()
	assert.Equal(t, 1, totalNodes) // 只有根节点
}

// MockContentCollector 用于测试的模拟内容收集器
type MockContentCollector struct {
	titles   []string
	contents []string
	tocItems []string
}

func (m *MockContentCollector) AddTitle(title string, level int) error {
	m.titles = append(m.titles, title)
	return nil
}

func (m *MockContentCollector) AddContent(content string) error {
	m.contents = append(m.contents, content)
	return nil
}

func (m *MockContentCollector) AddTOCItem(title string, level int) error {
	m.tocItems = append(m.tocItems, title)
	return nil
}

func (m *MockContentCollector) Render(outputPath string) error {
	// 简单的测试实现，不实际写入文件
	return nil
}
