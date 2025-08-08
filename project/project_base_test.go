package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRootPath(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 测试 GetRootPath 方法
	assert.Equal(t, tempDir, proj.GetRootPath())
}

func TestGetName(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 测试 GetName 方法
	expectedName := filepath.Base(tempDir)
	assert.Equal(t, expectedName, proj.GetName())
}

func TestIsEmpty(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 新创建的项目应该是空的
	assert.True(t, proj.IsEmpty())

	// 添加一个子节点
	proj.root.Children["test"] = &Node{
		Name:     "test",
		IsDir:    true,
		Children: make(map[string]*Node),
	}

	// 添加子节点后项目不应该是空的
	assert.False(t, proj.IsEmpty())
}

func TestGetTotalNodes(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 新创建的项目应该只有一个节点（根节点）
	assert.Equal(t, 1, proj.GetTotalNodes())

	// 添加一个子节点
	proj.root.Children["test"] = &Node{
		Name:     "test",
		IsDir:    true,
		Children: make(map[string]*Node),
	}

	// 添加子节点后应该有两个节点
	assert.Equal(t, 2, proj.GetTotalNodes())

	// 再添加一个子节点
	proj.root.Children["test2"] = &Node{
		Name:     "test2",
		IsDir:    false,
		Children: make(map[string]*Node),
	}

	// 添加第二个子节点后应该有三个节点
	assert.Equal(t, 3, proj.GetTotalNodes())
}

func TestIsInGit(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 测试 IsInGit 方法
	assert.Equal(t, proj.inGit, proj.IsInGit())
}

func TestSetInGit(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 设置 inGit 为 true
	proj.SetInGit(true)
	assert.True(t, proj.IsInGit())

	// 设置 inGit 为 false
	proj.SetInGit(false)
	assert.False(t, proj.IsInGit())
}

func TestGetAbsolutePath(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 测试相对路径转绝对路径
	testCases := []struct {
		relativePath string
		expectedPath string
	}{
		{"/file.txt", filepath.Join(tempDir, "file.txt")},
		{"file.txt", filepath.Join(tempDir, "file.txt")},
		{"/dir/file.txt", filepath.Join(tempDir, "dir", "file.txt")},
		{"dir/file.txt", filepath.Join(tempDir, "dir", "file.txt")},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.expectedPath, proj.GetAbsolutePath(tc.relativePath))
	}
}