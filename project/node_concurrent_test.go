package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNodeConcurrentProcessing(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-concurrent-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建测试文件结构
	testFiles := map[string]string{
		"src/main.go":         "package main\n\nfunc main() {}\n",
		"src/utils/helper.go": "package utils\n\nfunc Helper() {}\n",
		"docs/README.md":      "# Project\n\nThis is a test project.\n",
		"config/app.json":     `{"name": "test", "version": "1.0.0"}`,
	}

	for filePath, content := range testFiles {
		fullPath := filepath.Join(tempDir, filePath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		assert.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(content), 0644)
		assert.NoError(t, err)
	}

	// 创建项目实例
	project := NewProject(tempDir)
	err = project.SyncFromFS()
	assert.NoError(t, err)

	ctx := context.Background()

	t.Run("ProcessConcurrent", func(t *testing.T) {
		// 测试基本的并行处理
		results := project.ProcessConcurrent(ctx, 4, func(node *Node) (interface{}, error) {
			if node.IsDir {
				return fmt.Sprintf("DIR: %s", node.Path), nil
			}
			return fmt.Sprintf("FILE: %s", node.Path), nil
		})

		// 验证结果
		assert.NotEmpty(t, results)
		
		// 检查根节点
		rootResult, exists := results["/"]
		assert.True(t, exists)
		assert.NoError(t, rootResult.Err)
		assert.Equal(t, "DIR: /", rootResult.Value)

		// 检查文件节点
		for filePath := range testFiles {
			projPath := "/" + strings.ReplaceAll(filePath, "\\", "/")
			result, exists := results[projPath]
			assert.True(t, exists, "Missing result for %s", projPath)
			assert.NoError(t, result.Err)
			assert.Contains(t, result.Value, "FILE:")
		}
	})

	t.Run("ProcessConcurrentBFS", func(t *testing.T) {
		// 测试BFS策略的并行处理
		results := project.ProcessConcurrentBFS(ctx, 2, func(node *Node) (interface{}, error) {
			// 模拟一些处理时间
			time.Sleep(10 * time.Millisecond)
			return node.Name, nil
		})

		// 验证结果
		assert.NotEmpty(t, results)
		
		// 检查根节点
		rootResult, exists := results["/"]
		assert.True(t, exists)
		assert.NoError(t, rootResult.Err)
	})

	t.Run("ProcessConcurrentTyped", func(t *testing.T) {
		// 测试泛型版本的并行处理
		results := ProcessProjectConcurrentTyped(ctx, project, 3, func(node *Node) (int, error) {
			if node.IsDir {
				return len(node.Children), nil
			}
			// 对于文件，返回内容长度
			if err := node.EnsureContentLoaded(); err != nil {
				return 0, err
			}
			return len(node.Content), nil
		})

		// 验证结果
		assert.NotEmpty(t, results)
		
		// 检查根节点（应该有子目录）
		rootResult, exists := results["/"]
		assert.True(t, exists)
		assert.NoError(t, rootResult.Err)
		assert.Greater(t, rootResult.Value, 0)

		// 检查文件节点（应该有内容长度）
		for filePath := range testFiles {
			projPath := "/" + strings.ReplaceAll(filePath, "\\", "/")
			result, exists := results[projPath]
			assert.True(t, exists, "Missing result for %s", projPath)
			assert.NoError(t, result.Err)
			assert.Greater(t, result.Value, 0)
		}
	})

	t.Run("ProcessConcurrentBFSTyped", func(t *testing.T) {
		// 测试泛型版本的BFS并行处理
		results := ProcessProjectConcurrentBFSTyped(ctx, project, 2, func(node *Node) (string, error) {
			return fmt.Sprintf("%s:%s", node.Name, node.Path), nil
		})

		// 验证结果
		assert.NotEmpty(t, results)
		
		// 检查所有节点都有结果
		for filePath := range testFiles {
			projPath := "/" + strings.ReplaceAll(filePath, "\\", "/")
			result, exists := results[projPath]
			assert.True(t, exists, "Missing result for %s", projPath)
			assert.NoError(t, result.Err)
			assert.Contains(t, result.Value, ":")
		}
	})
}

func TestNodeConcurrentProcessingWithErrors(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-concurrent-error-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建简单的文件结构
	err = os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("test content"), 0644)
	assert.NoError(t, err)

	// 创建项目实例
	project := NewProject(tempDir)
	err = project.SyncFromFS()
	assert.NoError(t, err)

	ctx := context.Background()

	t.Run("ProcessWithErrors", func(t *testing.T) {
		// 测试处理函数返回错误的情况
		results := project.ProcessConcurrent(ctx, 2, func(node *Node) (interface{}, error) {
			if node.Path == "/test.txt" {
				return nil, fmt.Errorf("simulated error for %s", node.Path)
			}
			return node.Name, nil
		})

		// 验证结果
		assert.NotEmpty(t, results)
		
		// 检查错误节点
		errorResult, exists := results["/test.txt"]
		assert.True(t, exists)
		assert.Error(t, errorResult.Err)
		assert.Contains(t, errorResult.Err.Error(), "simulated error")

		// 检查正常节点
		rootResult, exists := results["/"]
		assert.True(t, exists)
		assert.NoError(t, rootResult.Err)
	})
}

func TestNodeConcurrentProcessingPerformance(t *testing.T) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-concurrent-perf-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建较多的文件来测试性能
	for i := 0; i < 50; i++ {
		dirPath := filepath.Join(tempDir, fmt.Sprintf("dir%d", i))
		err := os.MkdirAll(dirPath, 0755)
		assert.NoError(t, err)
		
		for j := 0; j < 5; j++ {
			filePath := filepath.Join(dirPath, fmt.Sprintf("file%d.txt", j))
			content := fmt.Sprintf("Content of file %d in dir %d", j, i)
			err := os.WriteFile(filePath, []byte(content), 0644)
			assert.NoError(t, err)
		}
	}

	// 创建项目实例
	project := NewProject(tempDir)
	err = project.SyncFromFS()
	assert.NoError(t, err)

	ctx := context.Background()

	t.Run("PerformanceComparison", func(t *testing.T) {
		// 测试串行处理时间
		start := time.Now()
		serialResults := make(map[string]string)
		err := project.Visit(func(path string, node *Node, depth int) error {
			// 模拟一些处理时间
			time.Sleep(1 * time.Millisecond)
			serialResults[path] = node.Name
			return nil
		})
		assert.NoError(t, err)
		serialTime := time.Since(start)

		// 测试并行处理时间
		start = time.Now()
		concurrentResults := project.ProcessConcurrent(ctx, 8, func(node *Node) (interface{}, error) {
			// 模拟相同的处理时间
			time.Sleep(1 * time.Millisecond)
			return node.Name, nil
		})
		concurrentTime := time.Since(start)

		// 验证结果数量相同
		assert.Equal(t, len(serialResults), len(concurrentResults))

		// 并行处理应该更快（在有足够节点的情况下）
		t.Logf("Serial time: %v, Concurrent time: %v", serialTime, concurrentTime)
		
		// 验证所有节点都被处理了
		assert.Greater(t, len(concurrentResults), 250) // 50 dirs + 250 files + root
	})
}

func BenchmarkNodeConcurrentProcessing(b *testing.B) {
	// 创建测试项目
	tempDir, err := os.MkdirTemp("", "project-concurrent-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// 创建文件结构
	for i := 0; i < 20; i++ {
		dirPath := filepath.Join(tempDir, fmt.Sprintf("dir%d", i))
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			b.Fatal(err)
		}
		
		for j := 0; j < 10; j++ {
			filePath := filepath.Join(dirPath, fmt.Sprintf("file%d.txt", j))
			content := fmt.Sprintf("Content %d-%d", i, j)
			err := os.WriteFile(filePath, []byte(content), 0644)
			if err != nil {
				b.Fatal(err)
			}
		}
	}

	// 创建项目实例
	project := NewProject(tempDir)
	err = project.SyncFromFS()
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.Run("Concurrent", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			results := project.ProcessConcurrent(ctx, 4, func(node *Node) (interface{}, error) {
				return len(node.Name), nil
			})
			_ = results
		}
	})

	b.Run("Serial", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			count := 0
			err := project.Visit(func(path string, node *Node, depth int) error {
				count += len(node.Name)
				return nil
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}