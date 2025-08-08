package project

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateDir(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 测试创建目录
	err = proj.CreateDir("/dir1")
	assert.NoError(t, err)

	// 验证目录是否已创建
	node, err := proj.FindNode("/dir1")
	assert.NoError(t, err)
	assert.True(t, node.IsDir)
	assert.Equal(t, "dir1", node.Name)

	// 测试创建嵌套目录
	err = proj.CreateDir("/dir1/dir2")
	assert.NoError(t, err)

	// 验证嵌套目录是否已创建
	node, err = proj.FindNode("/dir1/dir2")
	assert.NoError(t, err)
	assert.True(t, node.IsDir)
	assert.Equal(t, "dir2", node.Name)

	// 测试创建已存在的目录
	err = proj.CreateDir("/dir1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// 测试在不存在的父目录中创建目录
	err = proj.CreateDir("/nonexistent/dir")
	assert.Error(t, err)
}

func TestCreateFile(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 测试创建文件
	content := []byte("test content")
	err = proj.CreateFile("/file.txt", content)
	assert.NoError(t, err)

	// 验证文件是否已创建
	node, err := proj.FindNode("/file.txt")
	assert.NoError(t, err)
	assert.False(t, node.IsDir)
	assert.Equal(t, "file.txt", node.Name)
	assert.Equal(t, content, node.Content)
	assert.True(t, node.ContentLoaded)

	// 测试在目录中创建文件
	err = proj.CreateDir("/dir1")
	assert.NoError(t, err)

	content2 := []byte("test content 2")
	err = proj.CreateFile("/dir1/file.txt", content2)
	assert.NoError(t, err)

	// 验证目录中的文件是否已创建
	node, err = proj.FindNode("/dir1/file.txt")
	assert.NoError(t, err)
	assert.False(t, node.IsDir)
	assert.Equal(t, "file.txt", node.Name)
	assert.Equal(t, content2, node.Content)
	assert.True(t, node.ContentLoaded)

	// 测试创建已存在的文件
	err = proj.CreateFile("/file.txt", content)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// 测试在不存在的父目录中创建文件
	err = proj.CreateFile("/nonexistent/file.txt", content)
	assert.Error(t, err)
}

func TestCreateFileNode(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 测试创建文件节点
	err = proj.CreateFileNode("/file.txt")
	assert.NoError(t, err)

	// 验证文件节点是否已创建
	node, err := proj.FindNode("/file.txt")
	assert.NoError(t, err)
	assert.False(t, node.IsDir)
	assert.Equal(t, "file.txt", node.Name)
	assert.False(t, node.ContentLoaded)
	assert.Nil(t, node.Content)

	// 测试在目录中创建文件节点
	err = proj.CreateDir("/dir1")
	assert.NoError(t, err)

	err = proj.CreateFileNode("/dir1/file.txt")
	assert.NoError(t, err)

	// 验证目录中的文件节点是否已创建
	node, err = proj.FindNode("/dir1/file.txt")
	assert.NoError(t, err)
	assert.False(t, node.IsDir)
	assert.Equal(t, "file.txt", node.Name)
	assert.False(t, node.ContentLoaded)
	assert.Nil(t, node.Content)

	// 测试创建已存在的文件节点
	err = proj.CreateFileNode("/file.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// 测试在不存在的父目录中创建文件节点
	err = proj.CreateFileNode("/nonexistent/file.txt")
	assert.Error(t, err)
}

func TestCreateFileWithContent(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 测试创建带内容的文件
	content := []byte("test content")
	err = proj.CreateFileWithContent("/file.txt", content)
	assert.NoError(t, err)

	// 验证文件是否已创建
	node, err := proj.FindNode("/file.txt")
	assert.NoError(t, err)
	assert.False(t, node.IsDir)
	assert.Equal(t, "file.txt", node.Name)
	assert.Equal(t, content, node.Content)
	assert.True(t, node.ContentLoaded)

	// 测试在目录中创建带内容的文件
	err = proj.CreateDir("/dir1")
	assert.NoError(t, err)

	content2 := []byte("test content 2")
	err = proj.CreateFileWithContent("/dir1/file.txt", content2)
	assert.NoError(t, err)

	// 验证目录中的文件是否已创建
	node, err = proj.FindNode("/dir1/file.txt")
	assert.NoError(t, err)
	assert.False(t, node.IsDir)
	assert.Equal(t, "file.txt", node.Name)
	assert.Equal(t, content2, node.Content)
	assert.True(t, node.ContentLoaded)

	// 测试创建已存在的文件
	err = proj.CreateFileWithContent("/file.txt", content)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// 测试在不存在的父目录中创建文件
	err = proj.CreateFileWithContent("/nonexistent/file.txt", content)
	assert.Error(t, err)
}

func TestReadFile(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 创建测试文件
	content := []byte("test content")
	err = proj.CreateFile("/file.txt", content)
	assert.NoError(t, err)

	// 测试读取文件
	readContent, err := proj.ReadFile("/file.txt")
	assert.NoError(t, err)
	assert.Equal(t, content, readContent)

	// 测试读取不存在的文件
	_, err = proj.ReadFile("/nonexistent.txt")
	assert.Error(t, err)

	// 测试读取目录
	err = proj.CreateDir("/dir1")
	assert.NoError(t, err)
	_, err = proj.ReadFile("/dir1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read directory")

	// 测试读取未加载内容的文件（应该自动从文件系统加载）
	err = proj.CreateFileNode("/file2.txt")
	assert.NoError(t, err)
	readContent2, err := proj.ReadFile("/file2.txt")
	assert.NoError(t, err)
	assert.Equal(t, []byte{}, readContent2) // CreateFileNode 创建的是空文件
}

func TestWriteFile(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 测试写入新文件
	content := []byte("test content")
	err = proj.WriteFile("/file.txt", content)
	assert.NoError(t, err)

	// 验证文件是否已创建
	node, err := proj.FindNode("/file.txt")
	assert.NoError(t, err)
	assert.False(t, node.IsDir)
	assert.Equal(t, "file.txt", node.Name)
	assert.Equal(t, content, node.Content)
	assert.True(t, node.ContentLoaded)

	// 测试更新现有文件
	newContent := []byte("new content")
	err = proj.WriteFile("/file.txt", newContent)
	assert.NoError(t, err)

	// 验证文件内容是否已更新
	node, err = proj.FindNode("/file.txt")
	assert.NoError(t, err)
	assert.Equal(t, newContent, node.Content)

	// 测试写入目录
	err = proj.CreateDir("/dir1")
	assert.NoError(t, err)
	err = proj.WriteFile("/dir1", content)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot write to directory")

	// 测试写入嵌套文件
	err = proj.WriteFile("/dir1/file.txt", content)
	assert.NoError(t, err)

	// 验证嵌套文件是否已创建
	node, err = proj.FindNode("/dir1/file.txt")
	assert.NoError(t, err)
	assert.False(t, node.IsDir)
	assert.Equal(t, "file.txt", node.Name)
	assert.Equal(t, content, node.Content)

	// 测试写入不存在父目录的文件
	err = proj.WriteFile("/nonexistent/file.txt", content)
	assert.Error(t, err)
}

func TestDeleteNode(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 创建测试文件和目录
	err = proj.CreateDir("/dir1")
	assert.NoError(t, err)

	err = proj.CreateFile("/dir1/file1.txt", []byte("content1"))
	assert.NoError(t, err)

	err = proj.CreateFile("/file2.txt", []byte("content2"))
	assert.NoError(t, err)

	// 测试删除文件
	err = proj.DeleteNode("/file2.txt")
	assert.NoError(t, err)

	// 验证文件是否已删除
	_, err = proj.FindNode("/file2.txt")
	assert.Error(t, err)

	// 测试删除目录
	err = proj.DeleteNode("/dir1")
	assert.NoError(t, err)

	// 验证目录及其内容是否已删除
	_, err = proj.FindNode("/dir1")
	assert.Error(t, err)
	_, err = proj.FindNode("/dir1/file1.txt")
	assert.Error(t, err)

	// 测试删除不存在的节点
	err = proj.DeleteNode("/nonexistent")
	assert.Error(t, err)

	// 测试删除根节点
	err = proj.DeleteNode("/")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete root node")
}
