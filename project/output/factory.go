package output

import (
	"fmt"
	"path/filepath"

	"github.com/sjzsdu/tong/project"
)

// Output 将项目导出为指定格式的文件
func Output(doc *project.Project, outputFile string) error {
	// 获取导出器
	exporter, err := GetExporter(doc, outputFile)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return err
	}
	
	// 执行导出
	if err := exporter.Export(outputFile); err != nil {
		fmt.Printf("Error exporting to %s: %v\n", outputFile, err)
		return err
	}

	fmt.Printf("Successfully exported project to %s\n", outputFile)
	return nil
}

// GetExporter 根据输出文件类型返回对应的导出器
func GetExporter(doc *project.Project, outputFile string) (Exporter, error) {
	switch filepath.Ext(outputFile) {
	case ".md":
		return NewMarkdownExporter(doc), nil
	case ".pdf":
		return NewPDFExporter(doc)
	case ".xml":
		return NewXMLExporter(doc), nil
	default:
		return nil, fmt.Errorf("unsupported output format: %s", filepath.Ext(outputFile))
	}
}
