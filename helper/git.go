package helper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
)

// CloneProject 克隆指定的Git仓库到临时目录并返回克隆的路径
func CloneProject(gitURL string) (string, error) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "git-clone-")
	if err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}

	// 克隆仓库
	_, err = git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:      gitURL,
		Progress: os.Stdout, // 显示克隆进度
	})
	if err != nil {
		os.RemoveAll(tempDir) // 清理临时目录
		return "", fmt.Errorf("克隆仓库失败: %w", err)
	}

	fmt.Printf("仓库已成功克隆到临时目录: %s\n", tempDir)

	// 返回克隆的路径
	return tempDir, nil
}

func IsGitRoot(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}

	// 确认是目录而不是文件
	return info.IsDir()
}

func IsGitSubdir(path string) bool {
	// 获取绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// 从当前目录开始向上查找 .git 目录
	currentPath := absPath
	for {
		// 检查当前路径是否存在 .git 目录
		gitDir := filepath.Join(currentPath, ".git")
		info, err := os.Stat(gitDir)
		if err == nil && info.IsDir() {
			// 找到 .git 目录，确认是子目录
			return currentPath != absPath
		}

		// 移动到父目录
		parentPath := filepath.Dir(currentPath)
		// 如果已经到达根目录，则停止搜索
		if parentPath == currentPath {
			break
		}
		currentPath = parentPath
	}

	return false
}

// FindGitRoot 查找给定路径所属的git项目根目录
func FindGitRoot(path string) (string, bool) {
	// 获取绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}

	// 从当前目录开始向上查找 .git 目录
	currentPath := absPath
	for {
		// 检查当前路径是否存在 .git 目录
		gitDir := filepath.Join(currentPath, ".git")
		info, err := os.Stat(gitDir)
		if err == nil && info.IsDir() {
			// 找到 .git 目录，返回该目录的父目录（即git项目根目录）
			return currentPath, true
		}

		// 移动到父目录
		parentPath := filepath.Dir(currentPath)
		// 如果已经到达根目录，则停止搜索
		if parentPath == currentPath {
			break
		}
		currentPath = parentPath
	}

	return "", false
}

// GetRelativePathToGitRoot 获取指定路径相对于git项目根目录的相对路径
func GetRelativePathToGitRoot(path string) (string, bool) {
	// 获取绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}

	// 查找git根目录
	gitRoot, found := FindGitRoot(absPath)
	if !found {
		return "", false
	}

	// 计算相对路径
	relPath, err := filepath.Rel(gitRoot, absPath)
	if err != nil {
		return "", false
	}

	// 统一使用正斜杠作为路径分隔符
	relPath = strings.ReplaceAll(relPath, "\\", "/")

	return relPath, true
}
