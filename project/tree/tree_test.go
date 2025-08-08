package tree

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sjzsdu/tong/project"
)

func TestNodeTree(t *testing.T) {
	// 创建临时目录结构用于测试
	tempDir, err := os.MkdirTemp("", "tree_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试文件结构
	testStructure := map[string]string{
		"README.md":           "# Test Project\nThis is a test project.",
		"src/main.go":         "package main\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}",
		"src/utils/helper.go": "package utils\n\nfunc Helper() string {\n\treturn \"helper\"\n}",
		"docs/api.md":         "# API Documentation",
		"docs/guide.md":       "# User Guide",
		".gitignore":          "*.log\n*.tmp",
		"config.json":         "{\"name\": \"test\", \"version\": \"1.0.0\"}",
	}

	// 创建文件和目录
	for filePath, content := range testStructure {
		fullPath := filepath.Join(tempDir, filePath)
		dir := filepath.Dir(fullPath)
		
		// 创建目录
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		
		// 创建文件
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	// 创建项目并加载
	proj := project.NewProject(tempDir)

	if err := proj.SyncFromFS(); err != nil {
		t.Fatalf("Failed to sync project: %v", err)
	}

	// 通过查找根路径获取根节点
	root, err := proj.FindNode("/")
	if err != nil {
		t.Fatalf("Failed to find root node: %v", err)
	}
	if root == nil {
		t.Fatal("Root node is nil")
	}

	// 测试基本 Tree 功能
	t.Run("BasicTree", func(t *testing.T) {
		treeOutput := Tree(root)
		fmt.Println("=== Basic Tree Output ===")
		fmt.Print(treeOutput)
		
		// 验证输出包含预期的文件和目录
		if !containsAll(treeOutput, []string{"README.md", "src/", "docs/", "main.go", "helper.go"}) {
			t.Error("Tree output missing expected files or directories")
		}
	})

	// 测试带选项的 Tree 功能
	t.Run("TreeWithOptions", func(t *testing.T) {
		// 只显示目录，不显示隐藏文件
		treeOutput := TreeWithOptions(root, false, false, 0)
		fmt.Println("\n=== Tree Output (Directories Only, No Hidden) ===")
		fmt.Print(treeOutput)
		
		// 验证不包含文件，但包含目录
		if containsAny(treeOutput, []string{"README.md", "main.go", ".gitignore"}) {
			t.Error("Tree output should not contain files when showFiles=false")
		}
		if !containsAll(treeOutput, []string{"src/", "docs/"}) {
			t.Error("Tree output missing expected directories")
		}
	})

	// 测试深度限制
	t.Run("TreeWithDepthLimit", func(t *testing.T) {
		treeOutput := TreeWithOptions(root, true, true, 2)
		fmt.Println("\n=== Tree Output (Max Depth 2) ===")
		fmt.Print(treeOutput)
		
		// 验证包含第一层内容（根目录下的文件和目录）
		if !containsAll(treeOutput, []string{"src/", "docs/", "README.md"}) {
			t.Error("Tree output missing expected content at depth 2")
		}
		// 验证不包含第三层内容（src目录下的文件）
		if containsAny(treeOutput, []string{"main.go", "helper.go"}) {
			t.Error("Tree output should not contain depth 3 content when maxDepth=2")
		}
	})

	// 测试显示隐藏文件
	t.Run("TreeWithHiddenFiles", func(t *testing.T) {
		treeOutput := TreeWithOptions(root, true, true, 0)
		fmt.Println("\n=== Tree Output (With Hidden Files) ===")
		fmt.Print(treeOutput)
		
		// 验证包含隐藏文件
		if !containsAll(treeOutput, []string{".gitignore"}) {
			t.Error("Tree output missing hidden files when showHidden=true")
		}
	})

	// 测试统计信息
	t.Run("TreeStats", func(t *testing.T) {
		stats := Stats(root)
		fmt.Printf("\n=== Tree Statistics ===\n%s\n", stats.String())
		
		// 验证统计信息
		if stats.DirectoryCount < 3 { // 至少有 root, src, docs, utils
			t.Errorf("Expected at least 3 directories, got %d", stats.DirectoryCount)
		}
		if stats.FileCount < 5 { // 至少有 5 个文件
			t.Errorf("Expected at least 5 files, got %d", stats.FileCount)
		}
		if stats.TotalSize == 0 {
			t.Error("Expected total size > 0")
		}
	})
}

// 辅助函数：检查字符串是否包含所有指定的子字符串
func containsAll(text string, substrings []string) bool {
	for _, substr := range substrings {
		if !contains(text, substr) {
			return false
		}
	}
	return true
}

// 辅助函数：检查字符串是否包含任何指定的子字符串
func containsAny(text string, substrings []string) bool {
	for _, substr := range substrings {
		if contains(text, substr) {
			return true
		}
	}
	return false
}

// 辅助函数：检查字符串是否包含子字符串
func contains(text, substr string) bool {
	return len(text) >= len(substr) && 
		   (text == substr || 
		    (len(text) > len(substr) && 
		     (text[:len(substr)] == substr || 
		      text[len(text)-len(substr):] == substr || 
		      containsSubstring(text, substr))))
}

// 辅助函数：在字符串中查找子字符串
func containsSubstring(text, substr string) bool {
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// 演示函数：展示如何在实际代码中使用 Tree 功能
func ExampleTree() {
	// 假设我们有一个已加载的项目
	// proj, _ := project.NewProject("/path/to/project")
	// proj.SyncFromFS()
	// root, _ := proj.FindNode("/")

	// 基本用法：显示完整的树结构
	// fmt.Println("Project Structure:")
	// fmt.Print(Tree(root))

	// 高级用法：只显示目录结构，不显示隐藏文件，限制深度为3
	// fmt.Println("Directory Structure (Max Depth 3):")
	// fmt.Print(TreeWithOptions(root, false, false, 3))

	// 获取统计信息
	// stats := Stats(root)
	// fmt.Printf("Project contains: %s\n", stats.String())
}