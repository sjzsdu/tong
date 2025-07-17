package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
