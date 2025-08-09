package pack

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

// packDirectory 打包目录及其子目录中的所有文本文件
func packDirectory(dir *project.Node, outputPath string, options *PackOptions) error {
	if !dir.IsDir {
		return fmt.Errorf("节点不是目录")
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
	
	// 写入输出文件
	output := builder.String()
	
	// 如果outputPath没有扩展名，添加默认扩展名
	if filepath.Ext(outputPath) == "" {
		outputPath = outputPath + options.Formatter.FileExtension()
	}
	
	// 确保输出目录存在
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}
	
	return os.WriteFile(outputPath, []byte(output), 0644)
}

// packFile 打包单个文件
func packFile(file *project.Node, outputPath string, options *PackOptions) error {
	if file.IsDir {
		return fmt.Errorf("节点不是文件")
	}
	
	// 检查文件是否应该被包含
	if !shouldIncludeFile(file, options) {
		return fmt.Errorf("文件类型不在允许范围内")
	}
	
	content, err := file.ReadContent()
	if err != nil {
		return fmt.Errorf("读取文件内容失败: %w", err)
	}
	
	var builder strings.Builder
	builder.WriteString(options.Formatter.Header(file.Name))
	builder.WriteString(options.Formatter.Format(file, string(content), file.Name))
	builder.WriteString(options.Formatter.Footer())
	
	output := builder.String()
	
	// 如果outputPath没有扩展名，添加默认扩展名
	if filepath.Ext(outputPath) == "" {
		outputPath = outputPath + options.Formatter.FileExtension()
	}
	
	// 确保输出目录存在
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}
	
	return os.WriteFile(outputPath, []byte(output), 0644)
}

// textFile 表示一个文本文件的信息
type textFile struct {
	node *project.Node
	path string
}

// collectTextFiles 递归收集所有文本文件
func collectTextFiles(node *project.Node, currentPath string, options *PackOptions) []textFile {
	var files []textFile
	
	if !node.IsDir {
		// 如果是文件，检查是否为文本文件
		if shouldIncludeFile(node, options) {
			path := currentPath
			if path == "" {
				path = node.Name
			} else {
				path = filepath.Join(currentPath, node.Name)
			}
			files = append(files, textFile{node: node, path: path})
		}
		return files
	}
	
	// 如果不是递归模式，只处理当前目录
	if !options.Recursive {
		return files
	}
	
	// 处理目录
	for _, child := range node.Children {
		childPath := filepath.Join(currentPath, child.Name)
		files = append(files, collectTextFiles(child, childPath, options)...)
	}
	
	return files
}