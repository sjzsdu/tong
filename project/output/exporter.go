package output

import (
	"github.com/sjzsdu/tong/project"
)

// Exporter 定义了导出器接口，与project.Exporter保持一致
type Exporter interface {
	// Export 将项目导出到指定路径
	Export(outputPath string) error
}

type BaseExporter struct {
	*project.BaseExporter
}

// NewBaseExporter 创建一个基本导出器
func NewBaseExporter(p *project.Project, collector project.ContentCollector) *BaseExporter {
	return &BaseExporter{
		BaseExporter: project.NewBaseExporter(p, collector),
	}
}

// Export 实现Exporter接口
func (b *BaseExporter) Export(outputPath string) error {
	return b.BaseExporter.Export(outputPath)
}
