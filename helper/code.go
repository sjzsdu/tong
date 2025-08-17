package helper

import (
	"path/filepath"
	"strings"
)

// 常见的程序文件扩展名
var ProgramFileExtensions = map[string]bool{
	"go":    true,
	"py":    true,
	"js":    true,
	"ts":    true,
	"jsx":   true,
	"tsx":   true,
	"java":  true,
	"cpp":   true,
	"c":     true,
	"h":     true,
	"hpp":   true,
	"rs":    true,
	"rb":    true,
	"php":   true,
	"swift": true,
	"kt":    true,
	"scala": true,
	"cs":    true,
	"vue":   true,
	"sh":    true,
	"pl":    true,
	"r":     true,
	"m":     true,
	"mm":    true,
	"lua":   true,
}

// IsProgramFile 判断是否是程序文件
func IsProgramFile(file string) bool {
	ext := GetFileExt(file)
	return ProgramFileExtensions[ext]
}

// GetLanguageFromExtension 根据文件扩展名返回对应的语言标识
func GetLanguageFromExtension(ext string) string {
	ext = strings.ToLower(ext)
	langMap := map[string]string{
		".go":         "go",
		".py":         "python",
		".js":         "javascript",
		".ts":         "typescript",
		".jsx":        "jsx",
		".tsx":        "tsx",
		".java":       "java",
		".cpp":        "cpp",
		".c":          "c",
		".h":          "c",
		".hpp":        "cpp",
		".cs":         "csharp",
		".php":        "php",
		".rb":         "ruby",
		".rs":         "rust",
		".swift":      "swift",
		".kt":         "kotlin",
		".scala":      "scala",
		".sh":         "bash",
		".yaml":       "yaml",
		".yml":        "yaml",
		".json":       "json",
		".xml":        "xml",
		".html":       "html",
		".css":        "css",
		".scss":       "scss",
		".less":       "less",
		".sql":        "sql",
		".md":         "markdown",
		".txt":        "text",
		".cfg":        "ini",
		".ini":        "ini",
		".toml":       "toml",
		".dockerfile": "dockerfile",
	}

	if lang, ok := langMap[ext]; ok {
		return lang
	}
	return ""
}

// IsTextFile 判断是否为文本文件
func IsTextFile(filename string) bool {
	// 基于文件扩展名判断
	ext := strings.ToLower(filepath.Ext(filename))

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

// ShouldIncludeFile 判断文件是否应该被包含在打包中
type FileFilterOptions struct {
	IncludeHidden bool
	IncludeExts   []string
	ExcludeExts   []string
}

func ShouldIncludeFile(filename string, isHidden bool, options *FileFilterOptions) bool {
	// 检查是否为隐藏文件
	if !options.IncludeHidden && isHidden {
		return false
	}

	// 检查文件类型
	if !IsTextFile(filename) {
		return false
	}

	// 检查排除扩展名
	ext := strings.ToLower(filepath.Ext(filename))
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
	return IsTextFile(filename)
}
