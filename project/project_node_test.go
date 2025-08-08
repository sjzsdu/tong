package project

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindNode(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 创建测试目录结构
	// /dir1/dir2/file.txt
	dir1 := &Node{
		Name:     "dir1",
		IsDir:    true,
		Children: make(map[string]*Node),
		Parent:   proj.root,
	}
	proj.root.Children["dir1"] = dir1

	dir2 := &Node{
		Name:     "dir2",
		IsDir:    true,
		Children: make(map[string]*Node),
		Parent:   dir1,
	}
	dir1.Children["dir2"] = dir2

	file := &Node{
		Name:     "file.txt",
		IsDir:    false,
		Children: make(map[string]*Node),
		Parent:   dir2,
	}
	dir2.Children["file.txt"] = file

	// 添加到 nodes 映射中
	proj.nodes = make(map[string]*Node)
	proj.nodes["/"] = proj.root
	proj.nodes["/dir1"] = dir1
	proj.nodes["/dir1/dir2"] = dir2
	proj.nodes["/dir1/dir2/file.txt"] = file

	// 测试用例
	testCases := []struct {
		path        string
		expectedNode *Node
		expectError  bool
	}{
		{"/", proj.root, false},
		{"/dir1", dir1, false},
		{"/dir1/dir2", dir2, false},
		{"/dir1/dir2/file.txt", file, false},
		{"/nonexistent", nil, true},      // 不存在的路径应该返回错误
		{"/dir1/nonexistent", nil, true}, // 不存在的子路径应该返回错误
	}

	for _, tc := range testCases {
		node, err := proj.FindNode(tc.path)
		if tc.expectError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedNode, node)
		}
	}
}

func TestListFiles(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 创建测试目录结构
	// /dir1/file1.txt
	// /dir1/file2.txt
	dir1 := &Node{
		Name:     "dir1",
		IsDir:    true,
		Children: make(map[string]*Node),
		Parent:   proj.root,
	}
	proj.root.Children["dir1"] = dir1

	file1 := &Node{
		Name:     "file1.txt",
		IsDir:    false,
		Children: make(map[string]*Node),
		Parent:   dir1,
	}
	dir1.Children["file1.txt"] = file1

	file2 := &Node{
		Name:     "file2.txt",
		IsDir:    false,
		Children: make(map[string]*Node),
		Parent:   dir1,
	}
	dir1.Children["file2.txt"] = file2

	// 添加到 nodes 映射中
	proj.nodes = make(map[string]*Node)
	proj.nodes["/"] = proj.root
	proj.nodes["/dir1"] = dir1
	proj.nodes["/dir1/file1.txt"] = file1
	proj.nodes["/dir1/file2.txt"] = file2

	// 测试根目录
	files, err := proj.ListFiles("/")
	assert.NoError(t, err)
	assert.Equal(t, []string{"dir1"}, files)

	// 测试子目录
	files, err = proj.ListFiles("/dir1")
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"file1.txt", "file2.txt"}, files)

	// 测试不存在的目录
	_, err = proj.ListFiles("/nonexistent")
	assert.Error(t, err)

	// 测试文件而不是目录
	_, err = proj.ListFiles("/dir1/file1.txt")
	assert.Error(t, err)
}

func TestGetAllFiles(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 创建测试目录结构
	// /file1.txt
	// /dir1/file2.txt
	// /dir1/dir2/file3.txt
	file1 := &Node{
		Name:     "file1.txt",
		IsDir:    false,
		Children: make(map[string]*Node),
		Parent:   proj.root,
	}
	proj.root.Children["file1.txt"] = file1

	dir1 := &Node{
		Name:     "dir1",
		IsDir:    true,
		Children: make(map[string]*Node),
		Parent:   proj.root,
	}
	proj.root.Children["dir1"] = dir1

	file2 := &Node{
		Name:     "file2.txt",
		IsDir:    false,
		Children: make(map[string]*Node),
		Parent:   dir1,
	}
	dir1.Children["file2.txt"] = file2

	dir2 := &Node{
		Name:     "dir2",
		IsDir:    true,
		Children: make(map[string]*Node),
		Parent:   dir1,
	}
	dir1.Children["dir2"] = dir2

	file3 := &Node{
		Name:     "file3.txt",
		IsDir:    false,
		Children: make(map[string]*Node),
		Parent:   dir2,
	}
	dir2.Children["file3.txt"] = file3

	// 添加到 nodes 映射中
	proj.nodes = make(map[string]*Node)
	proj.nodes["/"] = proj.root
	proj.nodes["/file1.txt"] = file1
	proj.nodes["/dir1"] = dir1
	proj.nodes["/dir1/file2.txt"] = file2
	proj.nodes["/dir1/dir2"] = dir2
	proj.nodes["/dir1/dir2/file3.txt"] = file3

	// 测试获取所有文件
	files, err := proj.GetAllFiles()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"/file1.txt", "/dir1/file2.txt", "/dir1/dir2/file3.txt"}, files)

	// 测试空项目
	emptyProj := &Project{
		rootPath: tempDir,
	}
	_, err = emptyProj.GetAllFiles()
	assert.Error(t, err)
}

func TestVisit(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建项目实例
	proj := NewProject(tempDir)

	// 创建测试目录结构
	// /file1.txt
	// /dir1/file2.txt
	file1 := &Node{
		Name:     "file1.txt",
		IsDir:    false,
		Children: make(map[string]*Node),
		Parent:   proj.root,
	}
	proj.root.Children["file1.txt"] = file1

	dir1 := &Node{
		Name:     "dir1",
		IsDir:    true,
		Children: make(map[string]*Node),
		Parent:   proj.root,
	}
	proj.root.Children["dir1"] = dir1

	file2 := &Node{
		Name:     "file2.txt",
		IsDir:    false,
		Children: make(map[string]*Node),
		Parent:   dir1,
	}
	dir1.Children["file2.txt"] = file2

	// 添加到 nodes 映射中
	proj.nodes = make(map[string]*Node)
	proj.nodes["/"] = proj.root
	proj.nodes["/file1.txt"] = file1
	proj.nodes["/dir1"] = dir1
	proj.nodes["/dir1/file2.txt"] = file2

	// 测试访问者模式
	visitedPaths := make([]string, 0)
	visitedDepths := make([]int, 0)

	err = proj.Visit(func(path string, node *Node, depth int) error {
		visitedPaths = append(visitedPaths, path)
		visitedDepths = append(visitedDepths, depth)
		return nil
	})

	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"/", "/file1.txt", "/dir1", "/dir1/file2.txt"}, visitedPaths)

	// 测试访问者返回错误
	err = proj.Visit(func(path string, node *Node, depth int) error {
		if path == "/dir1" {
			return errors.New("test error")
		}
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "test error")

	// 测试空项目
	emptyProj := &Project{
		rootPath: tempDir,
	}
	err = emptyProj.Visit(func(path string, node *Node, depth int) error {
		return nil
	})

	assert.Error(t, err)
}