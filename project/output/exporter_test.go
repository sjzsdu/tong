package output

import (
	"os"
	"testing"

	"github.com/sjzsdu/tong/project"
	"github.com/stretchr/testify/assert"
)

// TestMarkdownExporter 测试Markdown导出器
func TestMarkdownExporter(t *testing.T) {
	// 创建一个示例项目
	projectPath := project.CreateExampleGoProject(t)
	defer os.RemoveAll(projectPath) // 测试结束后清理

	// 使用示例项目创建 GoProject 实例
	goProject := project.GetSharedProject(t, projectPath)
	proj := goProject.GetProject()

	// 创建并测试Markdown导出器
	exporter := NewMarkdownExporter(proj)
	assert.NotNil(t, exporter)

	// 导出到临时文件
	outputPath := "/tmp/test_output.md"
	err := exporter.Export(outputPath)
	assert.NoError(t, err)

	// 验证文件已创建
	_, err = os.Stat(outputPath)
	assert.NoError(t, err, "输出文件应该已创建")

	// 清理
	os.Remove(outputPath)
}

// TestPDFExporter 测试PDF导出器
func TestPDFExporter(t *testing.T) {
	// 使用共享项目
	goProject := project.GetSharedProject(t, "")
	proj := goProject.GetProject()

	// 创建并测试PDF导出器
	exporter, err := NewPDFExporter(proj)
	assert.NoError(t, err)
	assert.NotNil(t, exporter)

	// 导出到临时文件
	outputPath := "/tmp/test_output.pdf"
	err = exporter.Export(outputPath)
	assert.NoError(t, err)

	// 验证文件已创建
	_, err = os.Stat(outputPath)
	assert.NoError(t, err, "输出文件应该已创建")

	// 清理
	os.Remove(outputPath)
}

// TestXMLExporter 测试XML导出器
func TestXMLExporter(t *testing.T) {
	// 使用共享项目
	goProject := project.GetSharedProject(t, "")
	proj := goProject.GetProject()

	// 创建并测试XML导出器
	exporter := NewXMLExporter(proj)
	assert.NotNil(t, exporter)

	// 导出到临时文件
	outputPath := "/tmp/test_output.xml"
	err := exporter.Export(outputPath)
	assert.NoError(t, err)

	// 验证文件已创建
	_, err = os.Stat(outputPath)
	assert.NoError(t, err, "输出文件应该已创建")

	// 清理
	os.Remove(outputPath)
}

// TestExporterFactory 测试导出器工厂
func TestExporterFactory(t *testing.T) {
	// 使用共享项目
	goProject := project.GetSharedProject(t, "")
	proj := goProject.GetProject()

	// 测试创建各种导出器
	formats := []string{".md", ".pdf", ".xml"}

	for _, format := range formats {
		outputPath := "/tmp/test_output" + format
		exporter, err := GetExporter(proj, outputPath)
		assert.NoError(t, err, "应该能够创建 %s 导出器", format)
		assert.NotNil(t, exporter, "导出器不应为空")
	}

	// 测试无效格式
	_, err := GetExporter(proj, "/tmp/test_output.invalid")
	assert.Error(t, err, "应该对无效格式返回错误")
}

// TestExporterWithExampleProject 测试在示例项目上的导出器
func TestExporterWithExampleProject(t *testing.T) {
	// 创建一个示例项目
	projectPath := project.CreateExampleGoProject(t)
	defer os.RemoveAll(projectPath) // 测试结束后清理

	// 使用示例项目创建 GoProject 实例
	goProject := project.GetSharedProject(t, projectPath)
	proj := goProject.GetProject()

	// 测试Markdown导出
	mdExporter := NewMarkdownExporter(proj)
	mdOutputPath := "/tmp/example_project.md"
	err := mdExporter.Export(mdOutputPath)
	assert.NoError(t, err)

	// 验证文件已创建并包含项目信息
	_, err = os.Stat(mdOutputPath)
	assert.NoError(t, err, "Markdown输出文件应该已创建")

	// 清理
	os.Remove(mdOutputPath)
}
