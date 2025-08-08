package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSaveToFS(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 创建测试文件和目录
	err = proj.CreateDir("/dir1")
	assert.NoError(t, err)

	err = proj.CreateFileWithContent("/dir1/file.txt", []byte("test content"))
	assert.NoError(t, err)

	err = proj.CreateFileWithContent("/file.txt", []byte("root file content"))
	assert.NoError(t, err)

	// 保存到文件系统
	err = proj.SaveToFS()
	assert.NoError(t, err)

	// 验证文件是否已保存到文件系统
	filePath := filepath.Join(tempDir, "file.txt")
	assert.FileExists(t, filePath)

	dirPath := filepath.Join(tempDir, "dir1")
	assert.DirExists(t, dirPath)

	nestedFilePath := filepath.Join(tempDir, "dir1", "file.txt")
	assert.FileExists(t, nestedFilePath)

	// 验证文件内容
	content, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, "root file content", string(content))

	content, err = os.ReadFile(nestedFilePath)
	assert.NoError(t, err)
	assert.Equal(t, "test content", string(content))

	// 测试空项目
	emptyProj := &Project{
		rootPath: tempDir + "/empty",
	}
	err = emptyProj.SaveToFS()
	assert.Error(t, err)

	// 测试无根路径的项目
	noRootProj := &Project{
		root: &Node{
			Name:     "/",
			IsDir:    true,
			Children: make(map[string]*Node),
		},
	}
	err = noRootProj.SaveToFS()
	assert.Error(t, err)
}

func TestSyncFromFS(t *testing.T) {
	// 创建测试目录结构
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建测试文件和目录
	dirPath := filepath.Join(tempDir, "dir1")
	err = os.MkdirAll(dirPath, 0755)
	assert.NoError(t, err)

	filePath := filepath.Join(tempDir, "file.txt")
	err = os.WriteFile(filePath, []byte("root file content"), 0644)
	assert.NoError(t, err)

	nestedFilePath := filepath.Join(dirPath, "file.txt")
	err = os.WriteFile(nestedFilePath, []byte("test content"), 0644)
	assert.NoError(t, err)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 从文件系统同步
	err = proj.SyncFromFS()
	assert.NoError(t, err)

	// 验证项目结构
	node, err := proj.FindNode("/file.txt")
	assert.NoError(t, err)
	assert.False(t, node.IsDir)
	assert.Equal(t, "file.txt", node.Name)

	node, err = proj.FindNode("/dir1")
	assert.NoError(t, err)
	assert.True(t, node.IsDir)
	assert.Equal(t, "dir1", node.Name)

	node, err = proj.FindNode("/dir1/file.txt")
	assert.NoError(t, err)
	assert.False(t, node.IsDir)
	assert.Equal(t, "file.txt", node.Name)

	// 测试无根路径的项目
	noRootProj := &Project{}
	err = noRootProj.SyncFromFS()
	assert.Error(t, err)
}

func TestLoadFileContent(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建测试文件
	filePath := filepath.Join(tempDir, "file.txt")
	content := []byte("test content")
	err = os.WriteFile(filePath, content, 0644)
	assert.NoError(t, err)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 创建文件节点（不加载内容）
	err = proj.CreateFileNode("/file.txt")
	assert.NoError(t, err)

	// 验证内容未加载
	node, err := proj.FindNode("/file.txt")
	assert.NoError(t, err)
	assert.False(t, node.ContentLoaded)
	assert.Nil(t, node.Content)

	// 加载文件内容
	err = proj.LoadFileContent("/file.txt")
	assert.NoError(t, err)

	// 验证内容已加载
	node, err = proj.FindNode("/file.txt")
	assert.NoError(t, err)
	assert.True(t, node.ContentLoaded)
	assert.Equal(t, content, node.Content)

	// 测试加载不存在的文件
	err = proj.LoadFileContent("/nonexistent.txt")
	assert.Error(t, err)

	// 测试加载目录
	err = proj.CreateDir("/dir1")
	assert.NoError(t, err)

	err = proj.LoadFileContent("/dir1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot load content for directory")
}

func TestUnloadFileContent(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 创建带内容的文件
	content := []byte("test content")
	err = proj.CreateFileWithContent("/file.txt", content)
	assert.NoError(t, err)

	// 验证内容已加载
	node, err := proj.FindNode("/file.txt")
	assert.NoError(t, err)
	assert.True(t, node.ContentLoaded)
	assert.Equal(t, content, node.Content)

	// 卸载文件内容
	err = proj.UnloadFileContent("/file.txt")
	assert.NoError(t, err)

	// 验证内容已卸载
	node, err = proj.FindNode("/file.txt")
	assert.NoError(t, err)
	assert.False(t, node.ContentLoaded)
	assert.Nil(t, node.Content)

	// 测试卸载不存在的文件
	err = proj.UnloadFileContent("/nonexistent.txt")
	assert.Error(t, err)

	// 测试卸载目录
	err = proj.CreateDir("/dir1")
	assert.NoError(t, err)

	err = proj.UnloadFileContent("/dir1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot unload content for directory")
}
