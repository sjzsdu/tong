package project

import (
	"github.com/sjzsdu/tong/helper"
)

// DefaultWalkDirOptions 返回默认的文件遍历选项
func DefaultWalkDirOptions() helper.WalkDirOptions {
	return helper.WalkDirOptions{
		DisableGitIgnore: false,
		Extensions:       []string{"*"}, // 所有文件类型
		Excludes:         []string{},    // 不排除任何文件
	}
}