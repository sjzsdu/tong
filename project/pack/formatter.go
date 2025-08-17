package pack

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/project"
)

// Formatter 定义打包格式的接口
type Formatter interface {
	Format(node *project.Node, content string, relativePath string) string
	Header(title string) string
	Footer() string
	FileExtension() string
}

// MarkdownFormatter Markdown格式的打包器
type MarkdownFormatter struct{}

// Format 格式化单个文件内容为Markdown格式
func (m *MarkdownFormatter) Format(node *project.Node, content string, relativePath string) string {
	var builder strings.Builder

	// 添加文件头部信息
	builder.WriteString(fmt.Sprintf("## 📄 %s\n\n", relativePath))
	builder.WriteString(fmt.Sprintf("**路径:** `%s`  \n", relativePath))
	if node.Info != nil {
		builder.WriteString(fmt.Sprintf("**大小:** %d bytes  \n", node.Info.Size()))
	}
	builder.WriteString("\n")

	// 添加代码块
	fileExt := filepath.Ext(relativePath)
	lang := helper.GetLanguageFromExtension(fileExt)
	builder.WriteString(fmt.Sprintf("```%s\n", lang))
	builder.WriteString(content)
	if !strings.HasSuffix(content, "\n") {
		builder.WriteString("\n")
	}
	builder.WriteString("```\n\n")

	return builder.String()
}

// Header 生成文档头部
func (m *MarkdownFormatter) Header(title string) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("# 📦 项目打包: %s\n\n", title))
	builder.WriteString("> 此文档由 tong 工具自动生成，包含项目中的所有文本文件内容\n\n")
	builder.WriteString("---\n\n")
	return builder.String()
}

// Footer 生成文档尾部
func (m *MarkdownFormatter) Footer() string {
	var builder strings.Builder
	builder.WriteString("\n---\n")
	builder.WriteString("*文档由 [tong](https://github.com/sjzsdu/tong) 工具自动生成*\n")
	return builder.String()
}

// FileExtension 返回文件扩展名
func (m *MarkdownFormatter) FileExtension() string {
	return ".md"
}

// GetFormatter 根据格式名称获取对应的格式化器
func GetFormatter(format string) Formatter {
	switch strings.ToLower(format) {
	case "markdown", "md":
		return &MarkdownFormatter{}
	default:
		return &MarkdownFormatter{} // 默认使用Markdown格式
	}
}
