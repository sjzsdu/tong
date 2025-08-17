package project

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sjzsdu/tong/helper"
)

// HandleContext 处理用户输入中的上下文信息
// 支持 "#File:<filepath> <query>" 格式的输入
// 将自动解析文件/目录路径，并将内容作为上下文添加到输入中
func (pjt *Project) HandleContext(input string) (string, error) {
	// 使用正则表达式匹配 #File:<path> 模式
	re := regexp.MustCompile(`#File:([^\s]+)(\s|$)`)
	matches := re.FindAllStringSubmatchIndex(input, -1)

	// 如果没有匹配项，直接返回原始输入
	if len(matches) == 0 {
		return input, nil
	}

	// 从后向前处理，避免替换后的索引变化影响前面的匹配
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		// match[0]和match[1]是整个匹配的起始和结束位置
		// match[2]和match[3]是第一个捕获组(文件路径)的起始和结束位置

		// 提取文件路径
		filePath := input[match[2]:match[3]]

		// 获取文件或目录的内容
		content, err := pjt.getFileOrDirContent(filePath)
		if err != nil {
			return input, fmt.Errorf("处理文件上下文时出错: %w", err)
		}

		// 替换原始的 #File: 部分
		// 替换为空字符串，之后会将内容附加到输入后面
		input = input[:match[0]] + input[match[1]:]

		// 在输入的末尾添加文件内容作为上下文
		input = input + "\n\n" + content
	}

	return input, nil
}

// getFileOrDirContent 获取文件或目录的内容
// 利用已有的项目实例找到对应的节点
// 如果是文件，直接读取内容
// 如果是目录，递归处理目录内容
func (pjt *Project) getFileOrDirContent(filePath string) (string, error) {
	// 获取绝对路径
	absPath, err := helper.GetAbsPath(filePath)
	if err != nil {
		return "", fmt.Errorf("获取绝对路径失败: %w", err)
	}

	// 检查路径是否存在
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return "", fmt.Errorf("路径不存在: %s", absPath)
	} else if err != nil {
		return "", fmt.Errorf("检查路径失败: %w", err)
	}

	// 获取相对于项目的路径
	relPath, err := filepath.Rel(pjt.rootPath, absPath)
	if err != nil {
		return "", fmt.Errorf("获取相对路径失败: %w", err)
	}

	// 转换为项目路径格式
	projPath := "/" + filepath.ToSlash(relPath)
	if relPath == "." {
		projPath = "/"
	}

	// 从项目中查找节点
	node, err := pjt.FindNode(projPath)
	if err != nil || node == nil {
		return "", fmt.Errorf("在项目中找不到节点: %s", projPath)
	}

	// 根据节点类型处理
	if node.IsDir {
		// 是目录，递归处理其内容
		var contentBuilder strings.Builder
		contentBuilder.WriteString(fmt.Sprintf("# 目录: %s\n\n", node.Name))

		// 手动递归处理节点
		err = formatProjectNode(node, "", &contentBuilder)
		if err != nil {
			return "", fmt.Errorf("格式化目录内容失败: %w", err)
		}

		// 返回格式化的内容
		return fmt.Sprintf("目录内容(%s):\n```\n%s\n```", absPath, contentBuilder.String()), nil
	} else {
		// 是文件，直接读取内容
		content, err := node.ReadContent()
		if err != nil {
			return "", fmt.Errorf("读取文件内容失败: %w", err)
		}

		// 添加文件扩展名对应的语言标识
		ext := filepath.Ext(node.Name)
		lang := helper.GetLanguageFromExtension(ext)

		// 返回格式化的内容
		return fmt.Sprintf("文件内容(%s):\n```%s\n%s\n```", absPath, lang, content), nil
	}
}

// formatProjectNode 递归格式化项目节点
func formatProjectNode(node *Node, path string, builder *strings.Builder) error {
	if node == nil {
		return nil
	}

	if !node.IsDir {
		// 处理文件
		nodePath := path
		if path == "" {
			nodePath = node.Name
		} else {
			nodePath = filepath.Join(path, node.Name)
		}

		// 读取文件内容
		content, err := node.ReadContent()
		if err != nil {
			return nil // 跳过错误文件
		}

		// 添加文件头部信息
		builder.WriteString(fmt.Sprintf("## %s\n\n", nodePath))

		// 添加代码块
		ext := filepath.Ext(nodePath)
		lang := helper.GetLanguageFromExtension(ext)
		builder.WriteString(fmt.Sprintf("```%s\n%s\n```\n\n", lang, content))

		return nil
	}

	// 处理目录
	if node.Path != "/" {
		// 跳过根目录
		builder.WriteString(fmt.Sprintf("## 📁 %s\n\n", path))
	}

	// 获取子节点并排序
	children := node.GetChildrenNodes()

	// 递归处理子节点
	for _, child := range children {
		childPath := path
		if path == "" {
			childPath = child.Name
		} else {
			childPath = filepath.Join(path, child.Name)
		}

		err := formatProjectNode(child, childPath, builder)
		if err != nil {
			return err
		}
	}

	return nil
}
