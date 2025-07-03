package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/project"
)

// PDFCollector 实现 ContentCollector 接口
type PDFCollector struct {
	pdf      *gofpdf.Fpdf
	tocItems []struct {
		title string
		page  int
		level int
	}
	fontName string
	tocPage  int
}

// 添加辅助函数用于清理文本
func cleanText(text string) string {
	runes := []rune(text)
	result := make([]rune, 0, len(runes))
	for _, r := range runes {
		if r <= 0xFFFF {
			result = append(result, r)
		}
	}
	return string(result)
}

func NewPDFCollector() (*PDFCollector, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 10)

	// 使用嵌入式字体
	fontPath, err := helper.UseEmbeddedFont("")
	if err != nil {
		return nil, fmt.Errorf("error loading embedded font: %v", err)
	}

	fontName := helper.FONT_NAME

	// 读取字体文件
	fontData, err := os.ReadFile(fontPath)
	if err != nil {
		return nil, fmt.Errorf("error reading font file: %v", err)
	}

	// 添加字体
	pdf.AddUTF8FontFromBytes(fontName, "", fontData)
	pdf.SetFont(fontName, "", 12)

	// 添加目录页
	pdf.AddPage()

	return &PDFCollector{
		pdf: pdf,
		tocItems: make([]struct {
			title string
			page  int
			level int
		}, 0),
		fontName: fontName,
		tocPage:  1,
	}, nil
}

// AddTitle 实现 ContentCollector 接口
func (p *PDFCollector) AddTitle(title string, level int) error {
	// 只有在不是第一页时才添加新页面
	if p.pdf.PageNo() > 0 {
		p.pdf.AddPage()
	}
	cleanTitle := cleanText(title)
	p.pdf.SetFont(p.fontName, "", 14+float64(4-level))
	p.pdf.CellFormat(190, 10, cleanTitle, "", 1, "L", false, 0, "")
	p.pdf.SetFont(p.fontName, "", 12)
	return nil
}

// AddContent 实现 ContentCollector 接口
func (p *PDFCollector) AddContent(content string) error {
	// 确保当前页面存在
	if p.pdf.PageNo() == 0 {
		p.pdf.AddPage()
	}
	cleanContent := cleanText(content)
	p.pdf.MultiCell(190, 5, cleanContent, "", "L", false)
	return nil
}

// AddTOCItem 实现 ContentCollector 接口
func (p *PDFCollector) AddTOCItem(title string, level int) error {
	cleanTitle := cleanText(title)
	p.tocItems = append(p.tocItems, struct {
		title string
		page  int
		level int
	}{
		title: cleanTitle,
		page:  p.pdf.PageNo() + 1, // 修正页码计算
		level: level,
	})
	return nil
}

// writeTOC 生成目录
func (p *PDFCollector) writeTOC() error {
	if len(p.tocItems) == 0 {
		return nil
	}

	p.pdf.SetPage(p.tocPage)
	p.pdf.SetY(30)
	p.pdf.SetFont(p.fontName, "", 16)
	p.pdf.CellFormat(190, 10, "目录", "", 1, "C", false, 0, "")
	p.pdf.SetFont(p.fontName, "", 12)

	for _, item := range p.tocItems {
		indent := strings.Repeat(" ", item.level)
		titleWidth := 150 - float64(item.level*5)

		x, y := p.pdf.GetXY()

		p.pdf.CellFormat(float64(item.level*5), 5, "", "", 0, "L", false, 0, "")
		p.pdf.CellFormat(titleWidth, 5, indent+item.title, "", 0, "L", false, 0, "")
		p.pdf.CellFormat(40, 5, fmt.Sprintf("%d", item.page), "", 1, "R", false, 0, "")

		link := p.pdf.AddLink()
		p.pdf.SetLink(link, 0, item.page) // 不需要再加1
		p.pdf.Link(x, y, 190, 5, link)
	}

	return nil
}

// Render 实现 ContentCollector 接口
func (p *PDFCollector) Render(outputPath string) error {
	if err := p.writeTOC(); err != nil {
		return err
	}

	// 确保输出目录存在
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	return p.pdf.OutputFileAndClose(outputPath)
}

// PDFExporter 使用 BaseExporter 和 PDFCollector
type PDFExporter struct {
	*project.BaseExporter
}

// NewPDFExporter 创建一个新的 PDF 导出器
func NewPDFExporter(p *project.Project) (*PDFExporter, error) {
	collector, err := NewPDFCollector()
	if err != nil {
		return nil, err
	}

	return &PDFExporter{
		BaseExporter: project.NewBaseExporter(p, collector),
	}, nil
}

// Export 实现Exporter接口
func (p *PDFExporter) Export(outputPath string) error {
	return p.BaseExporter.Export(outputPath)
}
