package pack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/project"
)

func TestMarkdownFormatter(t *testing.T) {
	formatter := &MarkdownFormatter{}

	// 测试Header
	header := formatter.Header("test-project")
	if !strings.Contains(header, "test-project") {
		t.Error("Header should contain project name")
	}

	// 测试Footer
	footer := formatter.Footer()
	if !strings.Contains(footer, "tong") {
		t.Error("Footer should contain tong reference")
	}

	// 测试FileExtension
	ext := formatter.FileExtension()
	if ext != ".md" {
		t.Errorf("Expected .md, got %s", ext)
	}
}

func TestGetFormatter(t *testing.T) {
	// 测试默认返回MarkdownFormatter
	formatter := GetFormatter("")
	if _, ok := formatter.(*MarkdownFormatter); !ok {
		t.Error("Default formatter should be MarkdownFormatter")
	}

	// 测试markdown返回MarkdownFormatter
	formatter = GetFormatter("markdown")
	if _, ok := formatter.(*MarkdownFormatter); !ok {
		t.Error("markdown format should return MarkdownFormatter")
	}

	// 测试md返回MarkdownFormatter
	formatter = GetFormatter("md")
	if _, ok := formatter.(*MarkdownFormatter); !ok {
		t.Error("md format should return MarkdownFormatter")
	}
}

func TestIsTextFile(t *testing.T) {
	// 创建测试节点
	testCases := []struct {
		name     string
		isDir    bool
		expected bool
	}{
		{"test.go", false, true},
		{"test.py", false, true},
		{"test.txt", false, true},
		{"test.md", false, true},
		{"test.json", false, true},
		{"test.exe", false, false},
		{"test.jpg", false, false},
		{"test", false, true}, // 无扩展名，默认为文本文件
		{"dir", true, false},  // 目录
	}

	for _, tc := range testCases {
		// 使用 helper 包的 IsTextFile 函数
		result := helper.IsTextFile(tc.name)
		if tc.isDir {
			result = false // 目录总是返回 false
		}
		if result != tc.expected {
			t.Errorf("helper.IsTextFile(%s) = %v, expected %v", tc.name, result, tc.expected)
		}
	}
}

func TestGetLanguageFromExtension(t *testing.T) {
	testCases := []struct {
		ext      string
		expected string
	}{
		{".go", "go"},
		{".py", "python"},
		{".js", "javascript"},
		{".ts", "typescript"},
		{".java", "java"},
		{".cpp", "cpp"},
		{".c", "c"},
		{".txt", "text"},     // helper包中txt返回text
		{".md", "markdown"},
		{".unknown", ""},
		{"", ""},
	}

	for _, tc := range testCases {
		result := helper.GetLanguageFromExtension(tc.ext)
		if result != tc.expected {
			t.Errorf("helper.GetLanguageFromExtension(%s) = %s, expected %s", tc.ext, result, tc.expected)
		}
	}
}

func TestShouldIncludeFile(t *testing.T) {
	node := &project.Node{
		Name:  "test.go",
		IsDir: false,
	}

	// 测试默认包含
	options := &PackOptions{}
	if !shouldIncludeFile(node, options) {
		t.Error("go文件应该被默认包含")
	}

	// 测试包含扩展名
	options = &PackOptions{
		IncludeExts: []string{".go", ".py"},
	}
	if !shouldIncludeFile(node, options) {
		t.Error("go文件应该在包含扩展名列表中")
	}

	// 测试排除扩展名
	options = &PackOptions{
		ExcludeExts: []string{".go"},
	}
	if shouldIncludeFile(node, options) {
		t.Error("go文件应该被排除扩展名排除")
	}

	// 测试目录
	nodeDir := &project.Node{
		Name:  "testdir",
		IsDir: true,
	}
	if shouldIncludeFile(nodeDir, options) {
		t.Error("目录不应该被包含")
	}
}

func TestPackNode(t *testing.T) {
	// 创建临时项目用于测试
	tempDir, err := os.MkdirTemp("", "pack-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试文件结构
	os.WriteFile(filepath.Join(tempDir, "file1.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("hello world"), 0644)
	os.WriteFile(filepath.Join(tempDir, "file3.jpg"), []byte("binary data"), 0644)

	subDir := filepath.Join(tempDir, "subdir")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "file4.py"), []byte("print('hello')"), 0644)

	// 创建项目实例
	proj := project.NewProject(tempDir)
	err = proj.SyncFromFS()
	if err != nil {
		t.Fatal(err)
	}

	// 获取根节点
	root := proj.Root()

	// 创建输出文件
	outputFile := filepath.Join(tempDir, "test_output.md")

	// 执行打包
	err = PackNode(root, outputFile, DefaultOptions())
	if err != nil {
		t.Fatalf("PackNode failed: %v", err)
	}

	// 验证输出文件存在
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("输出文件应该存在")
	}

	// 验证输出文件内容
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("读取输出文件失败: %v", err)
	}

	outputContent := string(content)
	if !strings.Contains(outputContent, "file1.go") {
		t.Error("输出文件应该包含file1.go")
	}
	if !strings.Contains(outputContent, "file2.txt") {
		t.Error("输出文件应该包含file2.txt")
	}
	if !strings.Contains(outputContent, "subdir/file4.py") {
		t.Error("输出文件应该包含subdir/file4.py")
	}
	if strings.Contains(outputContent, "file3.jpg") {
		t.Error("输出文件不应该包含file3.jpg")
	}
}

func TestPackNodeSingleFile(t *testing.T) {
	// 创建临时项目用于测试
	tempDir, err := os.MkdirTemp("", "pack-test")
	if err != nil {
		 t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试文件
	testFile := filepath.Join(tempDir, "test.go")
	os.WriteFile(testFile, []byte("package main"), 0644)

	// 创建项目实例
	proj := project.NewProject(tempDir)
	err = proj.SyncFromFS()
	if err != nil {
		 t.Fatal(err)
	}

	// 获取文件节点
	node := proj.Root()

	// 创建输出文件
	outputFile := filepath.Join(tempDir, "single_file_output.md")

	// 执行打包
	err = PackNode(node, outputFile, DefaultOptions())
	if err != nil {
		 t.Fatalf("PackNode failed: %v", err)
	}

	// 验证输出文件存在
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		 t.Error("输出文件应该存在")
	}

	// 验证输出文件内容
	content, err := os.ReadFile(outputFile)
	if err != nil {
		 t.Fatalf("读取输出文件失败: %v", err)
	}

	outputContent := string(content)
	if !strings.Contains(outputContent, "test.go") {
		 t.Error("输出文件应该包含test.go")
	}
	if !strings.Contains(outputContent, "package main") {
		 t.Error("输出文件应该包含文件内容")
	}
}

func TestPackNodeWithNilNode(t *testing.T) {
	// 测试空节点
	outputFile := filepath.Join(os.TempDir(), "nil_node_output.md")
	err := PackNode(nil, outputFile, DefaultOptions())
	if err == nil {
		 t.Error("PackNode with nil node should return error")
	} else if !strings.Contains(err.Error(), "节点不能为空") {
		 t.Errorf("Expected error message '节点不能为空', got '%v'", err)
	}
}

func TestPackNodeWithInvalidOutputDir(t *testing.T) {
	// 创建临时项目用于测试
	tempDir, err := os.MkdirTemp("", "pack-test")
	if err != nil {
		 t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := project.NewProject(tempDir)
	err = proj.SyncFromFS()
	if err != nil {
		 t.Fatal(err)
	}

	// 测试无效的输出目录
	invalidOutputDir := filepath.Join(tempDir, "non_existent_dir", "output.md")
	err = PackNode(proj.Root(), invalidOutputDir, DefaultOptions())
	if err != nil {
		 t.Fatalf("PackNode with invalid output dir failed: %v", err)
	}

	// 验证输出文件存在
	if _, err := os.Stat(invalidOutputDir); os.IsNotExist(err) {
		 t.Error("输出文件应该存在，即使输出目录不存在")
	}
}

func TestPackNodeWithUnreadableFile(t *testing.T) {
	// 创建临时项目用于测试
	tempDir, err := os.MkdirTemp("", "pack-test")
	if err != nil {
		 t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试文件
	testFile := filepath.Join(tempDir, "unreadable.go")
	os.WriteFile(testFile, []byte("package main"), 0000) // 不可读权限
	defer os.Chmod(testFile, 0644) // 测试后恢复权限以便删除

	// 创建项目实例
	proj := project.NewProject(tempDir)
	err = proj.SyncFromFS()
	if err != nil {
		 t.Fatal(err)
	}

	// 创建输出文件
	outputFile := filepath.Join(tempDir, "unreadable_output.md")

	// 执行打包
	err = PackNode(proj.Root(), outputFile, DefaultOptions())
	if err != nil {
		 t.Fatalf("PackNode with unreadable file failed: %v", err)
	}

	// 验证输出文件存在
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		 t.Error("输出文件应该存在")
	}
}

func TestPackNodeWithNilFormatter(t *testing.T) {
	// 创建临时项目用于测试
	tempDir, err := os.MkdirTemp("", "pack-test")
	if err != nil {
		 t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := project.NewProject(tempDir)
	err = proj.SyncFromFS()
	if err != nil {
		 t.Fatal(err)
	}

	// 创建输出文件
	outputFile := filepath.Join(tempDir, "nil_formatter_output.md")

	// 测试nil格式化器
	options := DefaultOptions()
	options.Formatter = nil
	err = PackNode(proj.Root(), outputFile, options)
	if err != nil {
		 t.Fatalf("PackNode with nil formatter failed: %v", err)
	}

	// 验证输出文件存在
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		 t.Error("输出文件应该存在")
	}
}

func TestPackNodeWithNonTextFile(t *testing.T) {
	// 创建临时项目用于测试
	tempDir, err := os.MkdirTemp("", "pack-test")
	if err != nil {
		 t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// 创建非文本文件
	binaryFile := filepath.Join(tempDir, "test.jpg")
	os.WriteFile(binaryFile, []byte{0x00, 0x01, 0x02}, 0644)

	// 创建项目实例
	proj := project.NewProject(tempDir)
	err = proj.SyncFromFS()
	if err != nil {
		 t.Fatal(err)
	}

	// 创建输出文件
	outputFile := filepath.Join(tempDir, "non_text_file_output.md")

	// 执行打包
	err = PackNode(proj.Root(), outputFile, DefaultOptions())
	if err != nil {
		 t.Fatalf("PackNode with non-text file failed: %v", err)
	}

	// 验证输出文件内容不包含二进制文件
	content, err := os.ReadFile(outputFile)
	if err != nil {
		 t.Fatalf("读取输出文件失败: %v", err)
	}

	outputContent := string(content)
	if strings.Contains(outputContent, "test.jpg") {
		 t.Error("输出文件不应该包含非文本文件")
	}
}
