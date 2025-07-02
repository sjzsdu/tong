package helper

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupTestDir(t *testing.T) string {
	log.Println("Setting up test directory")
	tempDir := t.TempDir()
	os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("test file 1"), 0644)
	os.WriteFile(filepath.Join(tempDir, "file2.log"), []byte("test file 2"), 0644)
	os.WriteFile(filepath.Join(tempDir, "subdir", "file3.txt"), []byte("test file 3"), 0644)
	os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte("*.log\n"), 0644)
	log.Println("Test directory setup complete")

	// 创建 .gitignore 文件
	gitignoreContent := []byte("*.log\n")
	err := os.WriteFile(filepath.Join(tempDir, ".gitignore"), gitignoreContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create .gitignore file: %v", err)
	}
	return tempDir
}

func TestWalkDir(t *testing.T) {
	tempDir := setupTestDir(t)

	tests := []struct {
		name     string
		options  WalkDirOptions
		expected []string
	}{
		{
			name: "正常遍历所有文件",
			options: WalkDirOptions{
				DisableGitIgnore: true,
			},
			expected: []string{
				filepath.Join(tempDir, "file1.txt"),
				filepath.Join(tempDir, "file2.log"),
				filepath.Join(tempDir, "subdir", "file3.txt"),
				filepath.Join(tempDir, ".gitignore"),
			},
		},
		{
			name: "启用.gitignore规则",
			options: WalkDirOptions{
				DisableGitIgnore: false,
			},
			expected: []string{
				filepath.Join(tempDir, "file1.txt"),
				filepath.Join(tempDir, "subdir", "file3.txt"),
				filepath.Join(tempDir, ".gitignore"),
			},
		},
		{
			name: "仅筛选特定扩展名文件",
			options: WalkDirOptions{
				DisableGitIgnore: true,
				Extensions:       []string{"txt"},
			},
			expected: []string{
				filepath.Join(tempDir, "file1.txt"),
				filepath.Join(tempDir, "subdir", "file3.txt"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result []string
			done := make(chan bool)
			go func() {
				log.Printf("Starting test: %s", tt.name)
				err := WalkDir(tempDir, func(fileInfo FileInfo) error {
					if !fileInfo.Info.IsDir() {
						result = append(result, fileInfo.Path)
					}
					return nil
				}, nil, tt.options)
				assert.NoError(t, err)
				log.Printf("Finished test: %s", tt.name)
				done <- true
			}()

			<-done
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestFilterReadableFiles(t *testing.T) {
	tempDir := setupTestDir(t)

	tests := []struct {
		name     string
		options  WalkDirOptions
		expected []string
	}{
		{
			name: "筛选可读文本文件",
			options: WalkDirOptions{
				DisableGitIgnore: true,
			},
			expected: []string{
				filepath.Join(tempDir, "file1.txt"),
				filepath.Join(tempDir, "subdir", "file3.txt"),
				filepath.Join(tempDir, ".gitignore"),
			},
		},
		{
			name: "启用.gitignore规则",
			options: WalkDirOptions{
				DisableGitIgnore: false,
			},
			expected: []string{
				filepath.Join(tempDir, "file1.txt"),
				filepath.Join(tempDir, "subdir", "file3.txt"),
				filepath.Join(tempDir, ".gitignore"),
			},
		},
		{
			name: "筛选特定扩展名文件",
			options: WalkDirOptions{
				DisableGitIgnore: true,
				Extensions:       []string{"txt"},
			},
			expected: []string{
				filepath.Join(tempDir, "file1.txt"),
				filepath.Join(tempDir, "subdir", "file3.txt"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log.Printf("Starting test: %s", tt.name)
			done := make(chan bool)
			var result []string
			var err error

			go func() {
				result, err = FilterReadableFiles(tempDir, tt.options)
				done <- true
			}()

			<-done
			assert.NoError(t, err)
			assert.ElementsMatch(t, tt.expected, result)
			log.Printf("Finished test: %s", tt.name)
		})
	}
}
