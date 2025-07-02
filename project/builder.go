package project

import (
	"os"
	"path/filepath"

	"github.com/sjzsdu/tong/helper"
)

// 需要排除的系统和开发工具目录
var excludedDirs = map[string]bool{
	".git":         true,
	".vscode":      true,
	".idea":        true,
	"node_modules": true,
	".svn":         true,
	".hg":          true,
	".DS_Store":    true,
	"__pycache__":  true,
	"bin":          true,
	"obj":          true,
	"dist":         true,
	"build":        true,
	"target":       true,
	"fonts":        true,
}

// BuildProjectTree 构建项目树
func BuildProjectTree(targetPath string, options helper.WalkDirOptions) (*Project, error) {
	doc := NewProject(targetPath)
	gitignoreRules := make(map[string][]string)
	targetPath = filepath.Clean(targetPath)

	err := filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 检查是否是需要排除的目录
		if info.IsDir() {
			name := info.Name()
			// 排除 . 和 .. 目录
			if name == "." || name == ".." {
				return nil
			}

			// 对于非根目录的情况才检查排除规则
			if path != targetPath && excludedDirs[name] {
				return filepath.SkipDir
			}

			if excludedDirs[name] {
				return filepath.SkipDir
			}

			// 处理 .gitignore 规则
			if !options.DisableGitIgnore {
				rules, err := helper.ReadGitignore(path)
				if err == nil && rules != nil {
					gitignoreRules[path] = rules
				}
			}
		}

		// 处理 .gitignore 规则
		if !options.DisableGitIgnore {
			excluded, excludeErr := helper.IsPathExcludedByGitignore(path, targetPath, gitignoreRules)
			if excludeErr != nil {
				return excludeErr
			}
			if excluded {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// 获取相对路径
		relPath, err := filepath.Rel(targetPath, path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			if info.Name() == "." {
				return nil
			}
			// 创建目录节点
			return doc.CreateDir(relPath, info)
		}

		// 检查文件扩展名
		if len(options.Extensions) > 0 {
			ext := filepath.Ext(path)
			if len(ext) > 0 {
				ext = ext[1:] // 移除开头的点
			}
			if !helper.StringSliceContains(options.Extensions, ext) && !helper.StringSliceContains(options.Extensions, "*") {
				return nil
			}
		}

		// 检查排除规则
		if helper.IsPathExcluded(path, options.Excludes, targetPath) {
			return nil
		}

		// 读取文件内容
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // 跳过无法读取的文件
		}

		// 创建文件节点
		return doc.CreateFile(relPath, content, info)
	})

	if err != nil {
		return nil, err
	}

	return doc, nil
}
