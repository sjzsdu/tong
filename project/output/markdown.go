package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/sjzsdu/tong/project"
)

type MarkdownCollector struct {
	content strings.Builder
}

func NewMarkdownCollector() *MarkdownCollector {
	return &MarkdownCollector{
		content: strings.Builder{},
	}
}

func (m *MarkdownCollector) AddTOCItem(title string, level int) error {
	if level > 0 {
		indent := strings.Repeat("  ", level-1)
		anchor := strings.ToLower(strings.ReplaceAll(title, " ", "-"))
		m.content.WriteString(fmt.Sprintf("%s- [%s](#%s)\n",
			indent, title, anchor))
	}
	return nil
}

func (m *MarkdownCollector) AddTitle(title string, level int) error {
	m.content.WriteString(fmt.Sprintf("\n%s %s\n\n", strings.Repeat("#", level+1), title))
	return nil
}

func (m *MarkdownCollector) AddContent(content string) error {
	m.content.WriteString(content)
	m.content.WriteString("\n")
	return nil
}

func (m *MarkdownCollector) Render(outputPath string) error {
	return os.WriteFile(outputPath, []byte(m.content.String()), 0644)
}

type MarkdownExporter struct {
	*BaseExporter
}

func NewMarkdownExporter(p *project.Project) *MarkdownExporter {
	return &MarkdownExporter{
		BaseExporter: NewBaseExporter(p, NewMarkdownCollector()),
	}
}
