package helper

import (
	"fmt"
)

func GetTargetPath(cmdPath string, gitURL string) (string, error) {
	var targetPath string

	if gitURL != "" {
		// 创建临时目录
		tempDir, err := CloneProject(gitURL)
		if err != nil {
			return "", fmt.Errorf("error cloning repository: %v", err)
		}
		targetPath = tempDir
	} else {
		absPath, err := GetAbsPath(cmdPath)
		if err != nil {
			return "", err
		}
		targetPath = absPath
	}

	return targetPath, nil
}
