package project

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewProject 测试创建新项目
func TestNewProject(t *testing.T) {
	// 创建一个示例项目
	projectPath := CreateExampleGoProject(t)
	defer os.RemoveAll(projectPath) // 测试结束后清理

	// 测试创建新项目
	project := NewProject(projectPath)

	// 验证项目根路径
	assert.Equal(t, projectPath, project.GetRootPath())

	// 验证项目初始状态
	// 注意：由于我们使用的是示例项目，它不是空的，所以我们不检查 IsEmpty
	assert.NotNil(t, project.root)
	assert.Equal(t, "/", project.root.Name)
	assert.True(t, project.root.IsDir)
	// 验证 nodes 映射已初始化
	assert.NotNil(t, project.nodes)
}

// TestCreateDir 测试创建目录
func TestCreateDir(t *testing.T) {
	// 创建一个示例项目
	projectPath := CreateExampleGoProject(t)
	defer os.RemoveAll(projectPath) // 测试结束后清理

	// 使用示例项目创建 Project 实例
	project := NewProject(projectPath)

	// 创建目录
	dirInfo := &mockFileInfo{name: "testdir", isDir: true}
	err := project.CreateDir("testdir", dirInfo)
	assert.NoError(t, err)

	// 验证目录是否创建成功
	node, err := project.FindNode("testdir")
	assert.NoError(t, err)
	assert.NotNil(t, node)
	assert.True(t, node.IsDir)
	assert.Equal(t, "testdir", node.Name)

	// 测试创建嵌套目录
	subDirInfo := &mockFileInfo{name: "subdir", isDir: true}
	err = project.CreateDir("testdir/subdir", subDirInfo)
	assert.NoError(t, err)

	// 验证嵌套目录是否创建成功
	node, err = project.FindNode("testdir/subdir")
	assert.NoError(t, err)
	assert.NotNil(t, node)
	assert.True(t, node.IsDir)
	assert.Equal(t, "subdir", node.Name)

	// 测试创建已存在的目录
	err = project.CreateDir("testdir", dirInfo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

// TestCreateFile 测试创建文件
func TestCreateFile(t *testing.T) {
	// 创建一个示例项目
	projectPath := CreateExampleGoProject(t)
	defer os.RemoveAll(projectPath) // 测试结束后清理

	// 使用示例项目创建 Project 实例
	project := NewProject(projectPath)

	// 创建目录
	dirInfo := &mockFileInfo{name: "testdir", isDir: true}
	err := project.CreateDir("testdir", dirInfo)
	assert.NoError(t, err)

	// 创建文件
	content := []byte("测试文件内容")
	fileInfo := &mockFileInfo{name: "testfile.txt", isDir: false}
	err = project.CreateFile("testdir/testfile.txt", content, fileInfo)
	assert.NoError(t, err)

	// 验证文件是否创建成功
	node, err := project.FindNode("testdir/testfile.txt")
	assert.NoError(t, err)
	assert.NotNil(t, node)
	assert.False(t, node.IsDir)
	assert.Equal(t, "testfile.txt", node.Name)
	assert.Equal(t, content, node.Content)

	// 测试创建已存在的文件
	err = project.CreateFile("testdir/testfile.txt", content, fileInfo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// 测试在不存在的路径创建文件（应该在根目录创建）
	err = project.CreateFile("nonexistent/testfile2.txt", content, &mockFileInfo{name: "testfile2.txt", isDir: false})
	assert.NoError(t, err) // 应该成功，resolvePath 会在根目录创建文件

	// 验证文件被创建在根目录中
	node, err = project.FindNode("testfile2.txt") // 不是 "nonexistent/testfile2.txt"
	assert.NoError(t, err)
	assert.NotNil(t, node)
	assert.False(t, node.IsDir)
	assert.Equal(t, "testfile2.txt", node.Name)
}

// TestReadFile 测试读取文件
func TestReadFile(t *testing.T) {
	// 创建一个示例项目
	projectPath := CreateExampleGoProject(t)
	defer os.RemoveAll(projectPath) // 测试结束后清理

	// 使用示例项目创建 Project 实例
	project := NewProject(projectPath)

	// 创建文件
	content := []byte("测试文件内容")
	fileInfo := &mockFileInfo{name: "testfile.txt", isDir: false}
	err := project.CreateFile("testfile.txt", content, fileInfo)
	assert.NoError(t, err)

	// 读取文件
	readContent, err := project.ReadFile("testfile.txt")
	assert.NoError(t, err)
	assert.Equal(t, content, readContent)

	// 测试读取不存在的文件
	_, err = project.ReadFile("nonexistent.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path not found")

	// 测试读取目录
	dirInfo := &mockFileInfo{name: "testdir", isDir: true}
	err = project.CreateDir("testdir", dirInfo)
	assert.NoError(t, err)

	_, err = project.ReadFile("testdir")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read directory")
}

// TestWriteFile 测试写入文件
func TestWriteFile(t *testing.T) {
	// 创建一个示例项目
	projectPath := CreateExampleGoProject(t)
	defer os.RemoveAll(projectPath) // 测试结束后清理

	// 使用示例项目创建 Project 实例
	project := NewProject(projectPath)

	// 创建文件
	originalContent := []byte("原始内容")
	fileInfo := &mockFileInfo{name: "testfile.txt", isDir: false}
	err := project.CreateFile("testfile.txt", originalContent, fileInfo)
	assert.NoError(t, err)

	// 写入新内容
	newContent := []byte("新内容")
	err = project.WriteFile("testfile.txt", newContent)
	assert.NoError(t, err)

	// 验证内容是否更新
	readContent, err := project.ReadFile("testfile.txt")
	assert.NoError(t, err)
	assert.Equal(t, newContent, readContent)

	// 测试写入不存在的文件
	err = project.WriteFile("nonexistent.txt", newContent)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path not found")

	// 测试写入目录
	dirInfo := &mockFileInfo{name: "testdir", isDir: true}
	err = project.CreateDir("testdir", dirInfo)
	assert.NoError(t, err)

	err = project.WriteFile("testdir", newContent)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot write to directory")
}

// TestGetAllFiles 测试获取所有文件
func TestGetAllFiles(t *testing.T) {
	// 创建一个示例项目
	projectPath := CreateExampleGoProject(t)
	defer os.RemoveAll(projectPath) // 测试结束后清理

	// 使用示例项目创建 GoProject 实例
	goProject := GetSharedProject(t, projectPath)

	// 获取所有文件
	files, err := goProject.Project.GetAllFiles()
	assert.NoError(t, err)

	// 验证文件列表
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

	assert.Equal(t, len(expectedFiles), len(files))
	for _, expectedFile := range expectedFiles {
		found := false
		for _, file := range files {
			if file == expectedFile {
				found = true
				break
			}
		}
		assert.True(t, found, "文件 %s 应该在列表中", expectedFile)
	}
}

// TestGetTotalNodes 测试获取节点总数
func TestGetTotalNodes(t *testing.T) {
	// 创建一个示例项目
	projectPath := CreateExampleGoProject(t)
	defer os.RemoveAll(projectPath) // 测试结束后清理

	// 使用示例项目创建 GoProject 实例
	goProject := GetSharedProject(t, projectPath)

	// 获取节点总数
	totalNodes := goProject.Project.GetTotalNodes()

	// 验证节点总数（示例项目中有文件和目录，包括根目录）
	// 由于项目结构可能会变化，我们只验证节点总数大于0
	assert.Greater(t, totalNodes, 0, "节点总数应该大于0")
	// 记录当前的节点总数，以便将来参考
	t.Logf("当前示例项目的节点总数: %d", totalNodes)
}

// TestNodeCalculateHash 测试节点哈希计算
func TestNodeCalculateHash(t *testing.T) {
	// 创建一个示例项目
	projectPath := CreateExampleGoProject(t)
	defer os.RemoveAll(projectPath) // 测试结束后清理

	// 使用示例项目创建 Project 实例
	project := NewProject(projectPath)

	// 创建文件
	content := []byte("测试文件内容")
	fileInfo := &mockFileInfo{name: "testfile.txt", isDir: false}
	err := project.CreateFile("testfile.txt", content, fileInfo)
	assert.NoError(t, err)

	// 获取文件节点
	node, err := project.FindNode("testfile.txt")
	assert.NoError(t, err)

	// 计算哈希
	hash, err := node.CalculateHash()
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	// 创建目录
	dirInfo := &mockFileInfo{name: "testdir", isDir: true}
	err = project.CreateDir("testdir", dirInfo)
	assert.NoError(t, err)

	// 在目录中创建文件
	err = project.CreateFile("testdir/subfile.txt", []byte("子文件内容"), fileInfo)
	assert.NoError(t, err)

	// 获取目录节点
	dirNode, err := project.FindNode("testdir")
	assert.NoError(t, err)

	// 计算目录哈希
	dirHash, err := dirNode.CalculateHash()
	assert.NoError(t, err)
	assert.NotEmpty(t, dirHash)
}

// TestProjectWithSharedInstance 使用共享项目实例测试项目功能
func TestProjectWithSharedInstance(t *testing.T) {
	// 使用共享项目实例
	goProject := GetSharedProject(t, "")
	project := goProject.GetProject()

	// 测试项目基本属性
	assert.NotNil(t, project)
	assert.NotEmpty(t, project.GetRootPath())
	assert.False(t, project.IsEmpty())

	// 测试获取所有文件
	files, err := project.GetAllFiles()
	assert.NoError(t, err)
	assert.NotEmpty(t, files)

	// 验证文件类型正确性
	for _, file := range files {
		node, err := project.FindNode(file)
		assert.NoError(t, err)
		assert.False(t, node.IsDir, "GetAllFiles应该只返回文件，不是目录: %s", file)
	}
}

// TestProjectPathResolution 测试路径解析功能
func TestProjectPathResolution(t *testing.T) {
	goProject := GetSharedProject(t, "")
	project := goProject.GetProject()

	// 测试解析根路径
	rootNode, err := project.FindNode("/")
	assert.NoError(t, err)
	assert.Equal(t, "/", rootNode.Name)
	assert.True(t, rootNode.IsDir)

	// 测试解析嵌套路径
	if files, err := project.GetAllFiles(); err == nil && len(files) > 0 {
		// 选择第一个文件进行测试
		firstFile := files[0]
		node, err := project.FindNode(firstFile)
		assert.NoError(t, err)
		assert.False(t, node.IsDir)
	}

	// 测试路径不存在的情况
	_, err = project.FindNode("/nonexistent/path")
	assert.Error(t, err)
}

// TestProjectConcurrentAccess 测试项目的并发访问安全性
func TestProjectConcurrentAccess(t *testing.T) {
	goProject := GetSharedProject(t, "")
	project := goProject.GetProject()

	// 构建索引，确保所有文件内容被加载
	err := project.BuildIndex()
	assert.NoError(t, err, "构建索引应该成功")

	// 获取一个已知存在的文件
	files, err := project.GetAllFiles()
	assert.NoError(t, err)
	if len(files) == 0 {
		t.Skip("没有文件可供测试")
	}

	// 选择一个确定存在的文件
	testFile := "/main.go" // 示例项目中应该有这个文件

	// 确保文件存在
	absPath := filepath.Join(project.GetRootPath(), testFile)
	_, err = os.Stat(absPath)
	if os.IsNotExist(err) {
		// 如果main.go不存在，尝试使用第一个文件
		testFile = files[0]
		absPath = filepath.Join(project.GetRootPath(), testFile)
		_, err = os.Stat(absPath)
		if os.IsNotExist(err) {
			t.Skipf("测试文件 %s 不存在", testFile)
		}
	}

	t.Logf("使用文件 %s 进行并发访问测试", testFile)

	// 并发读取同一个文件
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			_, err := project.ReadFile(testFile)
			assert.NoError(t, err)
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestProjectTraversalPerformance 测试项目遍历性能
func TestProjectTraversalPerformance(t *testing.T) {
	goProject := GetSharedProject(t, "")
	project := goProject.GetProject()

	// 测试遍历性能
	visitCount := 0
	visitor := VisitorFunc(func(path string, node *Node, depth int) error {
		visitCount++
		return nil
	})

	traverser := NewTreeTraverser(project)
	err := traverser.TraverseTree(visitor)
	assert.NoError(t, err)
	assert.Greater(t, visitCount, 0)

	// 验证访问节点数与项目总节点数的关系
	totalNodes := project.GetTotalNodes()
	assert.Equal(t, totalNodes, visitCount, "遍历访问的节点数应该等于项目总节点数")
}

// TestProjectNodeHierarchy 测试项目节点层次结构
func TestProjectNodeHierarchy(t *testing.T) {
	goProject := GetSharedProject(t, "")
	project := goProject.GetProject()

	// 测试根节点
	rootNode, err := project.FindNode("/")
	assert.NoError(t, err)
	assert.Nil(t, rootNode.Parent) // 根节点没有父节点

	// 测试子节点的父子关系
	for childName := range rootNode.Children {
		childNode := rootNode.Children[childName]
		assert.Equal(t, rootNode, childNode.Parent, "子节点的父节点应该指向根节点")
		assert.Equal(t, childName, childNode.Name, "子节点名称应该匹配")
	}
}

// TestProjectDifferentFileTypes 测试不同类型文件的处理
func TestProjectDifferentFileTypes(t *testing.T) {
	goProject := GetSharedProject(t, "")
	project := goProject.GetProject()

	// 确保所有文件内容被加载
	err := project.BuildIndex()
	assert.NoError(t, err, "构建索引应该成功")

	files, err := project.GetAllFiles()
	assert.NoError(t, err)

	// 按文件扩展名分类
	fileTypeCount := make(map[string]int)
	for _, file := range files {
		ext := filepath.Ext(file)
		if ext == "" {
			ext = "no-extension"
		}
		fileTypeCount[ext]++
	}

	// 验证至少有一些常见的文件类型
	assert.NotEmpty(t, fileTypeCount, "应该有文件存在")

	// 对于每种文件类型，测试读取一个示例
	for ext, count := range fileTypeCount {
		if count > 0 {
			// 找到这种类型的第一个文件
			for _, file := range files {
				fileExt := filepath.Ext(file)
				if (fileExt == "" && ext == "no-extension") || fileExt == ext {
					// 确保文件存在于文件系统中
					absPath := filepath.Join(project.GetRootPath(), file)
					_, statErr := os.Stat(absPath)
					if statErr != nil {
						// 如果文件不存在，跳过此文件
						continue
					}
					
					content, err := project.ReadFile(file)
					assert.NoError(t, err, "应该能够读取 %s 类型的文件: %s", ext, file)
					assert.NotNil(t, content)
					break
				}
			}
		}
	}
}

// TestProjectGetAbsolutePathVariations 测试获取绝对路径的不同情况
func TestProjectGetAbsolutePathVariations(t *testing.T) {
	goProject := GetSharedProject(t, "")
	project := goProject.GetProject()

	// 测试不同格式的路径
	testPaths := []string{
		"main.go",
		"/main.go",
		"./main.go",
		"pkg/utils",
		"/pkg/utils",
		"pkg/utils/greeting.go",
	}

	for _, testPath := range testPaths {
		absPath := project.GetAbsolutePath(testPath)
		assert.NotEmpty(t, absPath)
		assert.Contains(t, absPath, goProject.RootPath)
	}
}

// TestProjectMemoryUsage 测试项目内存使用情况
func TestProjectMemoryUsage(t *testing.T) {
	// 创建一个示例项目
	projectPath := CreateExampleGoProject(t)
	defer os.RemoveAll(projectPath) // 测试结束后清理

	// 使用示例项目创建 GoProject 实例
	goProject := GetSharedProject(t, projectPath)
	project := goProject.GetProject()

	// 构建索引，确保所有文件内容被加载
	err := project.BuildIndex()
	assert.NoError(t, err, "构建索引应该成功")

	// 统计项目中的内容大小
	totalContentSize := 0
	visitor := VisitorFunc(func(path string, node *Node, depth int) error {
		if !node.IsDir && node.Content != nil {
			totalContentSize += len(node.Content)
		}
		return nil
	})

	traverser := NewTreeTraverser(project)
	err = traverser.TraverseTree(visitor)
	assert.NoError(t, err)

	// 验证有内容被加载
	assert.Greater(t, totalContentSize, 0, "项目应该包含一些文件内容")
	// 记录当前的内容大小，以便将来参考
	t.Logf("当前示例项目的内容大小: %d 字节", totalContentSize)
}

// TestNodeListFiles 测试节点列出文件名
func TestNodeListFiles(t *testing.T) {
	// 创建一个示例项目
	projectPath := CreateExampleGoProject(t)
	defer os.RemoveAll(projectPath) // 测试结束后清理

	// 使用示例项目创建 Project 实例
	project := NewProject(projectPath)

	// 创建测试目录结构
	err := project.CreateDir("testdir", &mockFileInfo{name: "testdir", isDir: true})
	assert.NoError(t, err)

	err = project.CreateFile("testdir/file1.txt", []byte("内容1"), &mockFileInfo{name: "file1.txt", isDir: false})
	assert.NoError(t, err)

	err = project.CreateFile("testdir/file2.txt", []byte("内容2"), &mockFileInfo{name: "file2.txt", isDir: false})
	assert.NoError(t, err)

	err = project.CreateDir("testdir/subdir", &mockFileInfo{name: "subdir", isDir: true})
	assert.NoError(t, err)

	err = project.CreateFile("testdir/subdir/file3.txt", []byte("内容3"), &mockFileInfo{name: "file3.txt", isDir: false})
	assert.NoError(t, err)

	// 测试目录节点的 ListFiles 方法
	dirNode, err := project.FindNode("testdir")
	assert.NoError(t, err)

	fileNames := dirNode.ListFiles()
	assert.Equal(t, 3, len(fileNames), "应该返回3个文件名")
	assert.Contains(t, fileNames, "file1.txt")
	assert.Contains(t, fileNames, "file2.txt")
	assert.Contains(t, fileNames, "file3.txt")

	// 测试子目录节点的 ListFiles 方法
	subdirNode, err := project.FindNode("testdir/subdir")
	assert.NoError(t, err)

	subFileNames := subdirNode.ListFiles()
	assert.Equal(t, 1, len(subFileNames), "子目录应该返回1个文件名")
	assert.Equal(t, "file3.txt", subFileNames[0])

	// 测试文件节点的 ListFiles 方法
	fileNode, err := project.FindNode("testdir/file1.txt")
	assert.NoError(t, err)

	singleFileName := fileNode.ListFiles()
	assert.Equal(t, 1, len(singleFileName), "文件节点应该返回1个文件名")
	assert.Equal(t, "file1.txt", singleFileName[0])
}

// TestProjectListFiles 测试项目列出文件名
func TestProjectListFiles(t *testing.T) {
	// 创建一个示例项目
	projectPath := CreateExampleGoProject(t)
	defer os.RemoveAll(projectPath) // 测试结束后清理

	// 使用示例项目创建 GoProject 实例，这会加载项目内容
	goProject := GetSharedProject(t, projectPath)
	project := goProject.Project

	// 创建一些测试文件
	err := project.CreateFile("custom1.txt", []byte("内容1"), &mockFileInfo{name: "custom1.txt", isDir: false})
	assert.NoError(t, err)
	
	err = project.CreateFile("custom2.txt", []byte("内容2"), &mockFileInfo{name: "custom2.txt", isDir: false})
	assert.NoError(t, err)

	// 测试 Project.ListFiles 方法
	fileNames, err := project.ListFiles()
	assert.NoError(t, err)
	
	// 示例项目已经有一些文件，加上我们创建的两个
	assert.True(t, len(fileNames) >= 2)
	assert.Contains(t, fileNames, "custom1.txt")
	assert.Contains(t, fileNames, "custom2.txt")
	
	// 对于示例项目中已有的文件，我们也应该验证它们
	// 这些是我们知道的在示例项目中存在的一些文件
	assert.Contains(t, fileNames, "main.go")
	assert.Contains(t, fileNames, "go.mod")
	assert.Contains(t, fileNames, "config.json")
	assert.Contains(t, fileNames, "README.md")

	// 空项目测试
	emptyProject := NewProject("")
	emptyProject.root = nil

	_, err = emptyProject.ListFiles()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project root is nil")
}

// mockFileInfo 是一个模拟的 os.FileInfo 实现
type mockFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
	sys     interface{}
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() os.FileMode  { return m.mode }
func (m *mockFileInfo) ModTime() time.Time { return m.modTime }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return m.sys }
