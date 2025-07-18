package analyzer

import (
	"bufio"
	"bytes"
	"path/filepath"
	"strings"

	"github.com/sjzsdu/tong/project"
)

// CodeStats 代码统计信息
type CodeStats struct {
	TotalFiles      int            // 文件总数
	TotalDirs       int            // 目录总数
	TotalLines      int            // 代码总行数
	TotalSize       int64          // 总大小（字节）
	LanguageStats   map[string]int // 各语言代码行数统计
	FileTypeStats   map[string]int // 各文件类型统计
	ComplexityStats map[string]int // 复杂度统计（可选）
}

// CodeAnalyzer 代码分析器接口
type CodeAnalyzer interface {
	// 分析代码并返回统计信息，可选的进度回调参数
	Analyze(project *project.Project, progressCallback ...project.ProgressCallback) (*CodeStats, error)
}

// DefaultCodeAnalyzer 默认代码分析器实现
type DefaultCodeAnalyzer struct {
	// 语言扩展名映射
	languageMap map[string]string
}

// NewDefaultCodeAnalyzer 创建一个新的默认代码分析器
func NewDefaultCodeAnalyzer() *DefaultCodeAnalyzer {
	return &DefaultCodeAnalyzer{
		languageMap: map[string]string{
			"go":    "Go",
			"py":    "Python",
			"js":    "JavaScript",
			"ts":    "TypeScript",
			"java":  "Java",
			"c":     "C",
			"cpp":   "C++",
			"h":     "C/C++ Header",
			"hpp":   "C++ Header",
			"cs":    "C#",
			"php":   "PHP",
			"rb":    "Ruby",
			"swift": "Swift",
			"kt":    "Kotlin",
			"rs":    "Rust",
			"html":  "HTML",
			"css":   "CSS",
			"scss":  "SCSS",
			"sass":  "Sass",
			"less":  "Less",
			"xml":   "XML",
			"json":  "JSON",
			"yaml":  "YAML",
			"yml":   "YAML",
			"md":    "Markdown",
			"txt":   "Text",
			"sh":    "Shell",
			"bat":   "Batch",
			"ps1":   "PowerShell",
		},
	}
}

// Analyze 实现 CodeAnalyzer 接口，支持可选的进度回调
func (d *DefaultCodeAnalyzer) Analyze(p *project.Project, progressCallback ...project.ProgressCallback) (*CodeStats, error) {
	stats := &CodeStats{
		LanguageStats:   make(map[string]int),
		FileTypeStats:   make(map[string]int),
		ComplexityStats: make(map[string]int),
	}

	// 创建访问者函数
	visitor := project.VisitorFunc(func(path string, node *project.Node, depth int) error {
		if node.IsDir {
			stats.TotalDirs++
			return nil
		}

		// 统计文件
		stats.TotalFiles++
		stats.TotalSize += int64(len(node.Content))

		// 获取文件扩展名
		ext := strings.TrimPrefix(filepath.Ext(node.Name), ".")
		stats.FileTypeStats[ext]++

		// 获取语言类型
		if lang, ok := d.languageMap[ext]; ok {
			// 统计代码行数
			lines := countCodeLines(node.Content, ext)
			stats.TotalLines += lines
			stats.LanguageStats[lang] += lines
		}

		return nil
	})

	// 创建遍历器
	traverser := project.NewTreeTraverser(p)

	// 如果提供了进度回调，则使用带进度的遍历
	if len(progressCallback) > 0 && progressCallback[0] != nil {
		traverser = traverser.WithProgressCallback(progressCallback[0])
	}

	err := traverser.TraverseTree(visitor)
	return stats, err
}

// countCodeLines 计算代码行数（排除空行和注释）
func countCodeLines(content []byte, ext string) int {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineCount := 0
	inComment := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行
		if line == "" {
			continue
		}

		// 根据不同语言处理注释
		switch ext {
		case "go", "c", "cpp", "java", "cs", "js", "ts":
			// 处理多行注释
			if inComment {
				if strings.Contains(line, "*/") {
					inComment = false
					line = strings.TrimSpace(strings.Split(line, "*/")[1])
					if line == "" {
						continue
					}
				} else {
					continue
				}
			}

			// 检查是否开始多行注释
			if strings.Contains(line, "/*") {
				parts := strings.Split(line, "/*")
				if !strings.Contains(parts[1], "*/") {
					inComment = true
					line = strings.TrimSpace(parts[0])
					if line == "" {
						continue
					}
				}
			}

			// 处理单行注释
			if strings.HasPrefix(line, "//") {
				continue
			}

		case "py", "rb":
			// 处理 Python/Ruby 注释
			if strings.HasPrefix(line, "#") {
				continue
			}

		case "html", "xml":
			// 处理 HTML/XML 注释
			if strings.HasPrefix(line, "<!--") && !strings.Contains(line, "-->") {
				inComment = true
				continue
			}
			if inComment {
				if strings.Contains(line, "-->") {
					inComment = false
				}
				continue
			}
		}

		lineCount++
	}

	return lineCount
}
