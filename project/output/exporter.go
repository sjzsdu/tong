package output

import (
	"github.com/sjzsdu/tong/project"
)

type BaseExporter struct {
	*project.BaseExporter
}

// NewBaseExporter 创建一个基本导出器
func NewBaseExporter(p *project.Project, collector project.ContentCollector) *BaseExporter {
	return &BaseExporter{
		BaseExporter: project.NewBaseExporter(p, collector),
	}
}
