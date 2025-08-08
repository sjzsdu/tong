package helper

import "strings"

// helper.StandardizePath 标准化路径
func StandardizePath(path string) string {
	// 标准化路径
	cleanPath := path
	if len(cleanPath) > 0 && cleanPath[0] != '/' {
		cleanPath = "/" + cleanPath
	}

	// 处理 Windows 路径分隔符
	cleanPath = strings.ReplaceAll(cleanPath, "\\", "/")

	// 处理多余的 /
	// 使用更安全的方式替换连续的 /，避免可能的死循环
	prevPath := ""
	for prevPath != cleanPath {
		prevPath = cleanPath
		cleanPath = strings.ReplaceAll(cleanPath, "//", "/")
	}

	return cleanPath
}
