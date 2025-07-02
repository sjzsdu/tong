package helper

import (
	"fmt"
	"os"

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
