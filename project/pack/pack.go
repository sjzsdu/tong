package pack

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/project"
)

// PackNode 将节点下的所有文本文件打包成一个文件
func PackNode(node *project.Node, outputPath string, options *PackOptions) error {
	if node == nil {
		return fmt.Errorf("节点不能为空")
	}

	if options == nil {
		options = DefaultOptions()
	}

	if options.Formatter == nil {
		options.Formatter = GetFormatter("")
	}

	if node.IsDir {
		return packDirectory(node, outputPath, options)
	}

	return packFile(node, outputPath, options)
}

// PackToString 将节点下的所有文本文件打包成字符串
func PackToString(node *project.Node, options *PackOptions) (string, error) {
	if node == nil {
		return "", fmt.Errorf("节点不能为空")
	}

	if options == nil {
		options = DefaultOptions()
	}

	if options.Formatter == nil {
		options.Formatter = GetFormatter("")
	}

	if node.IsDir {
		return packDirectoryToString(node, options)
	}

	return packFileToString(node, options)
}

// packDirectory 打包目录及其子目录中的所有文本文件到文件
func packDirectory(dir *project.Node, outputPath string, options *PackOptions) error {
	content, err := packDirectoryToString(dir, options)
	if err != nil {
		return err
	}

	// 如果outputPath没有扩展名，添加默认扩展名
	if filepath.Ext(outputPath) == "" {
		outputPath = outputPath + options.Formatter.FileExtension()
	}

	// 确保输出目录存在
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	return os.WriteFile(outputPath, []byte(content), 0644)
}

// packDirectoryToString 打包目录及其子目录中的所有文本文件到字符串
func packDirectoryToString(dir *project.Node, options *PackOptions) (string, error) {
	if !dir.IsDir {
		return "", fmt.Errorf("节点不是目录")
	}

	var builder strings.Builder

	// 添加文档头部
	builder.WriteString(options.Formatter.Header(dir.Name))

	// 收集所有文本文件
	textFiles := collectTextFiles(dir, "", options)

	// 按路径排序，确保一致的输出顺序
	sort.Slice(textFiles, func(i, j int) bool {
		return textFiles[i].path < textFiles[j].path
	})

	// 打包每个文件
	for _, file := range textFiles {
		content, err := file.node.ReadContent()
		if err != nil {
			// 跳过无法读取的文件
			continue
		}

		formatted := options.Formatter.Format(file.node, string(content), file.path)
		builder.WriteString(formatted)
	}

	// 添加文档尾部
	builder.WriteString(options.Formatter.Footer())

	return builder.String(), nil
}

// packFile 打包单个文件到文件
func packFile(file *project.Node, outputPath string, options *PackOptions) error {
	content, err := packFileToString(file, options)
	if err != nil {
		return err
	}

	// 如果outputPath没有扩展名，添加默认扩展名
	if filepath.Ext(outputPath) == "" {
		outputPath = outputPath + options.Formatter.FileExtension()
	}

	// 确保输出目录存在
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	return os.WriteFile(outputPath, []byte(content), 0644)
}

// packFileToString 打包单个文件到字符串
func packFileToString(file *project.Node, options *PackOptions) (string, error) {
	if file.IsDir {
		return "", fmt.Errorf("节点不是文件")
	}

	// 检查文件是否应该被包含
	if !shouldIncludeFile(file, options) {
		return "", fmt.Errorf("文件类型不在允许范围内")
	}

	content, err := file.ReadContent()
	if err != nil {
		return "", fmt.Errorf("读取文件内容失败: %w", err)
	}

	var builder strings.Builder
	builder.WriteString(options.Formatter.Header(file.Name))
	builder.WriteString(options.Formatter.Format(file, string(content), file.Name))
	builder.WriteString(options.Formatter.Footer())

	return builder.String(), nil
}

// textFile 表示一个文本文件的信息
type textFile struct {
	node *project.Node
	path string
}

// collectTextFiles 收集所有文本文件
func collectTextFiles(node *project.Node, currentPath string, options *PackOptions) []textFile {
	if !node.IsDir {
		// 如果是文件，检查是否为文本文件(扩展名+内容探测)
		if shouldIncludeFile(node, options) && !isBinaryNode(node) {
			path := currentPath
			if path == "" {
				path = node.Name
			} else {
				path = filepath.Join(currentPath, node.Name)
			}
			// 记录包含
			options.IncludedFiles = append(options.IncludedFiles, path)
			return []textFile{{node: node, path: path}}
		}
		return []textFile{}
	}

	// 如果不是递归模式，只处理当前目录
	if !options.Recursive {
		var files []textFile
		for _, child := range node.Children {
			if !child.IsDir && shouldIncludeFile(child, options) && !isBinaryNode(child) {
				path := filepath.Join(currentPath, child.Name)
				options.IncludedFiles = append(options.IncludedFiles, path)
				files = append(files, textFile{node: child, path: path})
			}
		}
		return files
	}

	// 使用BFS策略并行收集所有文本文件
	var allFiles []textFile

	// 定义一个可递归的函数来处理节点并收集文件
	var processNode func(n *project.Node, path string) []textFile
	processNode = func(n *project.Node, path string) []textFile {
		if !n.IsDir {
			if shouldIncludeFile(n, options) && !isBinaryNode(n) {
				options.IncludedFiles = append(options.IncludedFiles, path)
				return []textFile{{node: n, path: path}}
			}
			return []textFile{}
		}

		var files []textFile
		for _, child := range n.Children {
			childPath := filepath.Join(path, child.Name)
			files = append(files, processNode(child, childPath)...)
		}
		return files
	}

	// 对于根目录，直接使用BFS遍历
	if currentPath == "" {
		// 使用并发处理第一层子节点
		ctx := context.Background()
		maxWorkers := 10 // 限制并发数

		processFunc := func(n *project.Node) (interface{}, error) {
			childPath := n.Name
			return processNode(n, childPath), nil
		}

		results := node.ProcessConcurrent(ctx, maxWorkers, processFunc)

		// 合并结果
		for _, result := range results {
			if result.Err == nil {
				if files, ok := result.Value.([]textFile); ok {
					allFiles = append(allFiles, files...)
				}
			}
		}
	} else {
		// 对于子目录，使用普通递归避免过多协程
		allFiles = processNode(node, currentPath)
	}

	return allFiles
}

// shouldIncludeFile 判断文件是否应该被包含在打包中
func shouldIncludeFile(node *project.Node, options *PackOptions) bool {
	if node.IsDir {
		return false
	}

	// 判断是否为隐藏文件
	isHidden := strings.HasPrefix(node.Name, ".")

	// 创建文件过滤选项
	filterOptions := &helper.FileFilterOptions{
		IncludeHidden: options.IncludeHidden,
		IncludeExts:   options.IncludeExts,
		ExcludeExts:   options.ExcludeExts,
	}

	return helper.ShouldIncludeFile(node.Name, isHidden, filterOptions)
}

// isBinaryNode 通过内容粗略判断是否是二进制文件
// 策略：读取前8192字节，统计不可打印字符(排除常见的换行/回车/制表)比例或是否出现0字节
func isBinaryNode(node *project.Node) bool {
	content, err := node.ReadContent()
	if err != nil || len(content) == 0 {
		return false // 读取失败时不当作二进制，交给扩展名过滤
	}

	limit := len(content)
	if limit > 8192 {
		limit = 8192
	}
	data := content[:limit]

	var nonPrintable int
	for _, b := range data {
		if b == 0 { // NULL 字节强烈指示二进制
			return true
		}
		// 允许的控制字符: \n, \r, \t
		if b < 32 && b != 10 && b != 13 && b != 9 {
			nonPrintable++
			continue
		}
		// 尝试按 UTF-8 解码首字节快速检测
		if b >= 0x80 { // 高位字节，尝试UTF-8
			// 简单：跳过完整UTF-8验证，只在无法解析成合法序列时记为不可打印。
			// 这里保守处理：不直接计入不可打印，除非后面整体UTF-8比例异常。
			continue
		}
	}

	// 如果不可打印字符比例 > 30% 判定为二进制
	if float64(nonPrintable)/float64(limit) > 0.30 {
		return true
	}

	// 额外：如果整体不是有效UTF-8且包含大量高位字节也可能是二进制
	if !utf8.Valid(data) {
		// 统计高位字节数量
		var highBytes int
		for _, b := range data {
			if b >= 0x80 {
				highBytes++
			}
		}
		if float64(highBytes)/float64(limit) > 0.50 { // 超过一半高位且非有效UTF-8
			return true
		}
	}

	return false
}
