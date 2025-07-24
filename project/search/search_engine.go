package search

import (
	"bufio"
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/sjzsdu/tong/project"
)

// SearchOptions 搜索选项
type SearchOptions struct {
	CaseSensitive bool     // 区分大小写
	WholeWord     bool     // 全词匹配
	RegexMode     bool     // 正则表达式模式
	FileTypes     []string // 限定文件类型
	MaxResults    int      // 最大结果数
}

// SearchResult 搜索结果
type SearchResult struct {
	FilePath    string // 文件路径
	LineNumber  int    // 行号
	ColumnStart int    // 列开始位置
	ColumnEnd   int    // 列结束位置
	LineContent string // 行内容
	Context     string // 上下文内容
}

// SearchEngine 搜索引擎接口
type SearchEngine interface {
	// 构建搜索索引
	BuildIndex(project *project.Project) error
	// 搜索关键词
	Search(query string, options SearchOptions) ([]SearchResult, error)
}

// DefaultSearchEngine 默认搜索引擎实现
type DefaultSearchEngine struct {
	project     *project.Project
	indexed     bool
	fileContent map[string][]byte
	mu          sync.RWMutex
}

// NewDefaultSearchEngine 创建一个新的默认搜索引擎
func NewDefaultSearchEngine() *DefaultSearchEngine {
	return &DefaultSearchEngine{
		fileContent: make(map[string][]byte),
		indexed:     false,
	}
}

// BuildIndex 实现 SearchEngine 接口
func (s *DefaultSearchEngine) BuildIndex(p *project.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.project = p
	s.fileContent = make(map[string][]byte)

	// 创建访问者函数
	visitor := project.VisitorFunc(func(path string, node *project.Node, depth int) error {
		if !node.IsDir {
			// 确保文件内容已加载
			content, err := node.ReadContent()
			if err != nil {
				return fmt.Errorf("无法读取文件 %s 内容: %v", path, err)
			}
			s.fileContent[path] = content
		}
		return nil
	})

	// 遍历项目树
	traverser := project.NewTreeTraverser(p)
	err := traverser.TraverseTree(visitor)
	if err != nil {
		return err
	}

	s.indexed = true
	return nil
}

// Search 实现 SearchEngine 接口
func (s *DefaultSearchEngine) Search(query string, options SearchOptions) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.indexed {
		return nil, fmt.Errorf("搜索引擎尚未建立索引")
	}

	var results []SearchResult
	var wg sync.WaitGroup
	var resultsMu sync.Mutex

	// 准备正则表达式
	var re *regexp.Regexp
	var err error
	if options.RegexMode {
		// 使用用户提供的正则表达式
		re, err = regexp.Compile(query)
		if err != nil {
			return nil, fmt.Errorf("无效的正则表达式: %v", err)
		}
	} else {
		// 构建搜索模式
		pattern := regexp.QuoteMeta(query)
		if options.WholeWord {
			pattern = fmt.Sprintf("\\b%s\\b", pattern)
		}
		if options.CaseSensitive {
			re, err = regexp.Compile(pattern)
		} else {
			re, err = regexp.Compile("(?i)" + pattern)
		}
		if err != nil {
			return nil, fmt.Errorf("无法编译搜索模式: %v", err)
		}
	}

	// 创建通道用于限制并发数
	semaphore := make(chan struct{}, 10) // 最多10个并发搜索

	// 遍历所有文件
	for path, content := range s.fileContent {
		// 检查文件类型
		if len(options.FileTypes) > 0 {
			ext := strings.TrimPrefix(filepath.Ext(path), ".")
			if !contains(options.FileTypes, ext) && !contains(options.FileTypes, "*") {
				continue
			}
		}

		wg.Add(1)
		go func(filePath string, fileContent []byte) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 搜索文件内容
			fileResults := s.searchInFile(filePath, fileContent, re, options)

			// 添加到结果集
			if len(fileResults) > 0 {
				resultsMu.Lock()
				results = append(results, fileResults...)
				resultsMu.Unlock()
			}
		}(path, content)
	}

	wg.Wait()

	// 限制结果数量
	if options.MaxResults > 0 && len(results) > options.MaxResults {
		results = results[:options.MaxResults]
	}

	return results, nil
}

// searchInFile 在单个文件中搜索
func (s *DefaultSearchEngine) searchInFile(filePath string, content []byte, re *regexp.Regexp, options SearchOptions) []SearchResult {
	var results []SearchResult

	// 按行读取文件内容
	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNumber := 0
	contextLines := make([]string, 0, 5) // 保存上下文行

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// 更新上下文行
		if len(contextLines) >= 4 {
			contextLines = append(contextLines[1:], line)
		} else {
			contextLines = append(contextLines, line)
		}

		// 查找匹配
		matches := re.FindAllStringIndex(line, -1)
		if len(matches) > 0 {
			for _, match := range matches {
				// 构建上下文
				context := strings.Join(contextLines[:len(contextLines)-1], "\n")

				// 创建搜索结果
				result := SearchResult{
					FilePath:    filePath,
					LineNumber:  lineNumber,
					ColumnStart: match[0] + 1, // 1-indexed
					ColumnEnd:   match[1] + 1, // 1-indexed
					LineContent: line,
					Context:     context,
				}
				results = append(results, result)
			}
		}
	}

	return results
}

// contains 检查切片是否包含指定元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// FormatSearchResults 格式化搜索结果
func FormatSearchResults(results []SearchResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("找到 %d 个结果:\n\n", len(results)))

	// 按文件分组
	fileGroups := make(map[string][]SearchResult)
	for _, result := range results {
		fileGroups[result.FilePath] = append(fileGroups[result.FilePath], result)
	}

	// 输出每个文件的结果
	for filePath, fileResults := range fileGroups {
		sb.WriteString(fmt.Sprintf("文件: %s (%d 个匹配)\n", filePath, len(fileResults)))
		for _, result := range fileResults {
			sb.WriteString(fmt.Sprintf("  行 %d, 列 %d-%d: %s\n", 
				result.LineNumber, 
				result.ColumnStart, 
				result.ColumnEnd,
				result.LineContent))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// MarkdownSearchFormatter Markdown格式的搜索结果格式化器
type MarkdownSearchFormatter struct{}

// Format 格式化搜索结果为Markdown
func (m *MarkdownSearchFormatter) Format(results []SearchResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# 搜索结果\n\n"))
	sb.WriteString(fmt.Sprintf("找到 **%d** 个结果\n\n", len(results)))

	// 按文件分组
	fileGroups := make(map[string][]SearchResult)
	for _, result := range results {
		fileGroups[result.FilePath] = append(fileGroups[result.FilePath], result)
	}

	// 输出每个文件的结果
	for filePath, fileResults := range fileGroups {
		sb.WriteString(fmt.Sprintf("## %s\n\n", filePath))
		sb.WriteString(fmt.Sprintf("*%d 个匹配*\n\n", len(fileResults)))

		for _, result := range fileResults {
			// 高亮显示匹配部分
			highlightedLine := result.LineContent
			prefix := highlightedLine[:result.ColumnStart-1]
			match := highlightedLine[result.ColumnStart-1:result.ColumnEnd-1]
			suffix := highlightedLine[result.ColumnEnd-1:]
			highlightedLine = fmt.Sprintf("%s**%s**%s", prefix, match, suffix)

			sb.WriteString(fmt.Sprintf("- 行 **%d**, 列 %d-%d:\n  ```\n  %s\n  ```\n\n", 
				result.LineNumber, 
				result.ColumnStart, 
				result.ColumnEnd,
				highlightedLine))
		}
	}

	return sb.String()
}

// HTMLSearchFormatter HTML格式的搜索结果格式化器
type HTMLSearchFormatter struct{}

// Format 格式化搜索结果为HTML
func (h *HTMLSearchFormatter) Format(results []SearchResult) string {
	var sb strings.Builder

	// 添加HTML头部
	sb.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	sb.WriteString("<title>搜索结果</title>\n")
	sb.WriteString("<style>\n")
	sb.WriteString("body { font-family: Arial, sans-serif; margin: 20px; }\n")
	sb.WriteString("h1, h2 { color: #333; }\n")
	sb.WriteString(".summary { margin-bottom: 20px; }\n")
	sb.WriteString(".file { margin-bottom: 30px; }\n")
	sb.WriteString(".file-path { font-weight: bold; color: #0066cc; }\n")
	sb.WriteString(".match-count { color: #666; font-style: italic; }\n")
	sb.WriteString(".result { margin: 10px 0; padding: 5px; border-left: 3px solid #ccc; }\n")
	sb.WriteString(".line-number { color: #999; margin-right: 10px; }\n")
	sb.WriteString(".line-content { font-family: monospace; white-space: pre; }\n")
	sb.WriteString(".highlight { background-color: #ffff00; font-weight: bold; }\n")
	sb.WriteString("</style>\n")
	sb.WriteString("</head>\n<body>\n")

	// 添加标题和摘要
	sb.WriteString("<h1>搜索结果</h1>\n")
	sb.WriteString(fmt.Sprintf("<div class=\"summary\">找到 <strong>%d</strong> 个结果</div>\n", len(results)))

	// 按文件分组
	fileGroups := make(map[string][]SearchResult)
	for _, result := range results {
		fileGroups[result.FilePath] = append(fileGroups[result.FilePath], result)
	}

	// 输出每个文件的结果
	for filePath, fileResults := range fileGroups {
		sb.WriteString(fmt.Sprintf("<div class=\"file\">\n"))
		sb.WriteString(fmt.Sprintf("<h2><span class=\"file-path\">%s</span> <span class=\"match-count\">(%d 个匹配)</span></h2>\n", filePath, len(fileResults)))

		for _, result := range fileResults {
			// 高亮显示匹配部分
			highlightedLine := result.LineContent
			prefix := highlightedLine[:result.ColumnStart-1]
			match := highlightedLine[result.ColumnStart-1:result.ColumnEnd-1]
			suffix := highlightedLine[result.ColumnEnd-1:]
			highlightedLine = fmt.Sprintf("%s<span class=\"highlight\">%s</span>%s", prefix, match, suffix)

			sb.WriteString(fmt.Sprintf("<div class=\"result\">\n"))
			sb.WriteString(fmt.Sprintf("<div><span class=\"line-number\">行 %d, 列 %d-%d:</span></div>\n", 
				result.LineNumber, 
				result.ColumnStart, 
				result.ColumnEnd))
			sb.WriteString(fmt.Sprintf("<div class=\"line-content\">%s</div>\n", highlightedLine))
			sb.WriteString("</div>\n")
		}

		sb.WriteString("</div>\n")
	}

	// 添加HTML尾部
	sb.WriteString("</body>\n</html>")

	return sb.String()
}