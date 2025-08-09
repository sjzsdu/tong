package pack

import (
	"path/filepath"
	"strings"

	"github.com/sjzsdu/tong/project"
)

// getLanguageFromExtension 根据文件扩展名返回对应的语言标识
func getLanguageFromExtension(ext string) string {
	ext = strings.ToLower(ext)
	langMap := map[string]string{
		".go":   "go",
		".py":   "python",
		".js":   "javascript",
		".ts":   "typescript",
		".jsx":  "jsx",
		".tsx":  "tsx",
		".java": "java",
		".cpp":  "cpp",
		".c":    "c",
		".h":    "c",
		".hpp":  "cpp",
		".cs":   "csharp",
		".php":  "php",
		".rb":   "ruby",
		".rs":   "rust",
		".swift": "swift",
		".kt":   "kotlin",
		".scala": "scala",
		".sh":   "bash",
		".yaml": "yaml",
		".yml":  "yaml",
		".json": "json",
		".xml":  "xml",
		".html": "html",
		".css":  "css",
		".scss": "scss",
		".less": "less",
		".sql":  "sql",
		".md":   "markdown",
		".txt":  "text",
		".cfg":  "ini",
		".ini":  "ini",
		".toml": "toml",
		".dockerfile": "dockerfile",
	}
	
	if lang, ok := langMap[ext]; ok {
		return lang
	}
	return ""
}

// isTextFile 判断是否为文本文件
func isTextFile(node *project.Node) bool {
	if node.IsDir {
		return false
	}
	
	// 基于文件扩展名判断
	ext := strings.ToLower(filepath.Ext(node.Name))
	
	// 如果没有扩展名，默认为文本文件
	if ext == "" {
		return true
	}
	
	textExtensions := map[string]bool{
		".txt": true, ".md": true, ".go": true, ".py": true, ".js": true,
		".ts": true, ".java": true, ".cpp": true, ".c": true, ".h": true,
		".cs": true, ".php": true, ".rb": true, ".rs": true, ".swift": true,
		".kt": true, ".scala": true, ".sh": true, ".yaml": true, ".yml": true,
		".json": true, ".xml": true, ".html": true, ".css": true, ".scss": true,
		".less": true, ".sql": true, ".cfg": true, ".ini": true, ".toml": true,
		".dockerfile": true, ".gitignore": true, ".gitattributes": true,
		".editorconfig": true, ".babelrc": true, ".eslintrc": true,
		".prettierrc": true, ".npmignore": true, ".yarnrc": true,
	}
	
	return textExtensions[ext]
}

// shouldIncludeFile 判断文件是否应该被包含在打包中
func shouldIncludeFile(node *project.Node, options *PackOptions) bool {
	if node.IsDir {
		return false
	}

	// 检查是否为隐藏文件
	if !options.IncludeHidden && strings.HasPrefix(node.Name, ".") {
		return false
	}

	// 检查文件类型
	if !isTextFile(node) {
		return false
	}

	// 检查排除扩展名
	ext := strings.ToLower(filepath.Ext(node.Name))
	if len(options.ExcludeExts) > 0 {
		for _, excludeExt := range options.ExcludeExts {
			if strings.ToLower(excludeExt) == ext {
				return false
			}
		}
	}
	
	// 检查包含扩展名
	if len(options.IncludeExts) > 0 {
		for _, includeExt := range options.IncludeExts {
			if strings.ToLower(includeExt) == ext {
				return true
			}
		}
		return false
	}
	
	// 如果没有指定包含扩展名，则使用默认的文本文件判断
	return isTextFile(node)
}