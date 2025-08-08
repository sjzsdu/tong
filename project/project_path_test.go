package project

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolvePath(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 创建测试目录和文件
	err = proj.CreateDir("/dir1")
	assert.NoError(t, err)

	err = proj.CreateFileWithContent("/dir1/file.txt", []byte("test content"))
	assert.NoError(t, err)

	// 测试解析绝对路径
	parentNode, name, err := proj.resolvePath("/dir1/file.txt")
	assert.NoError(t, err)
	assert.Equal(t, "file.txt", name)
	assert.NotNil(t, parentNode)
	assert.Equal(t, "dir1", parentNode.Name)

	// 测试解析目录路径
	parentNode, name, err = proj.resolvePath("/dir1")
	assert.NoError(t, err)
	assert.Equal(t, "dir1", name)
	assert.Equal(t, proj.root, parentNode)

	// 测试解析根路径
	_, _, err = proj.resolvePath("/")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot resolve root path")

	// 测试解析非规范化路径
	parentNode, name, err = proj.resolvePath("dir1/file.txt")
	assert.NoError(t, err)
	assert.Equal(t, "file.txt", name)
	assert.NotNil(t, parentNode)
	assert.Equal(t, "dir1", parentNode.Name)

	// 测试解析不存在的路径
	_, _, err = proj.resolvePath("/nonexistent/file.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestGetNodePath(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 创建测试目录和文件
	err = proj.CreateDir("/dir1")
	assert.NoError(t, err)

	err = proj.CreateFileWithContent("/dir1/file.txt", []byte("test content"))
	assert.NoError(t, err)

	// 获取文件节点
	fileNode, err := proj.FindNode("/dir1/file.txt")
	assert.NoError(t, err)

	// 测试获取节点路径
	path := proj.GetNodePath(fileNode)
	assert.Equal(t, "/dir1/file.txt", path)

	// 获取目录节点
	dirNode, err := proj.FindNode("/dir1")
	assert.NoError(t, err)

	// 测试获取目录节点路径
	path = proj.GetNodePath(dirNode)
	assert.Equal(t, "/dir1", path)

	// 获取根节点
	rootNode, err := proj.FindNode("/")
	assert.NoError(t, err)

	// 测试获取根节点路径
	path = proj.GetNodePath(rootNode)
	assert.Equal(t, "/", path)
}

func TestNormalizePath(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 测试规范化绝对路径
	path := proj.NormalizePath("/dir1/file.txt")
	assert.Equal(t, "/dir1/file.txt", path)

	// 测试规范化相对路径
	path = proj.NormalizePath("dir1/file.txt")
	assert.Equal(t, "/dir1/file.txt", path)

	// 测试规范化包含 .. 的路径
	path = proj.NormalizePath("/dir1/../dir2/file.txt")
	assert.Equal(t, "/dir2/file.txt", path)

	// 测试规范化包含 . 的路径
	path = proj.NormalizePath("/dir1/./file.txt")
	assert.Equal(t, "/dir1/file.txt", path)

	// 测试规范化包含多个斜杠的路径
	path = proj.NormalizePath("/dir1//file.txt")
	assert.Equal(t, "/dir1/file.txt", path)

	// 测试规范化根路径
	path = proj.NormalizePath("/")
	assert.Equal(t, "/", path)

	// 测试规范化空路径
	path = proj.NormalizePath("")
	assert.Equal(t, "/", path)
}
