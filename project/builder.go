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

// NewProject 创建一个新的文档树
func NewProject(rootPath string) *Project {
	root := &Node{
		Name:     "/",
		Path:     rootPath,
		IsDir:    true,
		Children: make(map[string]*Node),
	}
	
	nodes := make(map[string]*Node)
	nodes["/"] = root
	
	return &Project{
		root:     root,
		rootPath: rootPath,
		nodes:    nodes,
	}
}

// BuildProjectTree 构建项目树
func BuildProjectTree(targetPath string, options helper.WalkDirOptions) (*Project, error) {
	doc := NewProject(targetPath)
	// 将根节点添加到 nodes 映射中
	if doc.nodes == nil {
		doc.nodes = make(map[string]*Node)
	}
	doc.nodes["/"] = doc.root
	
	gitignoreRules := make(map[string][]string)
	targetPath = filepath.Clean(targetPath)

	// 添加一个选项，控制是否立即加载文件内容
	loadContent := options.LoadContent

	// 标记是否处理过任何文件或目录（除了根目录）
	processedAny := false

	err := filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 如果是根目录本身，跳过处理
		if path == targetPath {
			return nil
		}

		// 标记已处理文件或目录
		processedAny = true

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

		// 将相对路径转换为项目路径格式
		projPath := "/" + filepath.ToSlash(relPath)
		if relPath == "." {
			projPath = "/"
		}

		if info.IsDir() {
			if info.Name() == "." {
				return nil
			}
			// 创建目录节点
			return doc.CreateDir(projPath, info)
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

		// 创建文件节点，根据选项决定是否立即加载内容
		if loadContent {
			// 如果需要立即加载内容
			content, err := os.ReadFile(path)
			if err != nil {
				return nil // 跳过无法读取的文件
			}
			return doc.CreateFileWithContent(projPath, content, info)
		} else {
			// 只创建节点，不加载内容
			return doc.CreateFileNode(projPath, info)
		}
	})

	if err != nil {
		return nil, err
	}

	// 如果没有处理任何文件或目录（除了根目录），则清空根节点的子节点
	if !processedAny {
		doc.root.mu.Lock()
		doc.root.Children = make(map[string]*Node)
		doc.root.mu.Unlock()
	}

	return doc, nil
}
