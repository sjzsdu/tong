package output

import (
	"encoding/xml"
	"os"

	"github.com/sjzsdu/tong/project"
)

type XMLNode struct {
	XMLName  xml.Name  `xml:"node"`
	Name     string    `xml:"name,attr"`
	Type     string    `xml:"type,attr"`
	Content  *string   `xml:"content,omitempty"`
	Children []XMLNode `xml:"nodes>node,omitempty"`
}

type XMLDocument struct {
	XMLName xml.Name  `xml:"document"`
	TOC     []XMLNode `xml:"toc>item,omitempty"`
	Nodes   []XMLNode `xml:"nodes>node"`
}

type XMLCollector struct {
	doc XMLDocument
}

func NewXMLCollector() *XMLCollector {
	return &XMLCollector{
		doc: XMLDocument{},
	}
}

func (x *XMLCollector) AddTitle(title string, level int) error {
	node := XMLNode{
		Name: title,
		Type: "directory",
	}
	x.doc.Nodes = append(x.doc.Nodes, node)
	return nil
}

func (x *XMLCollector) AddContent(content string) error {
	node := XMLNode{
		Type:    "file",
		Content: &content,
	}
	x.doc.Nodes = append(x.doc.Nodes, node)
	return nil
}

// AddTOCItem 实现可选的目录项添加
func (x *XMLCollector) AddTOCItem(title string, level int) error {
	node := XMLNode{
		Name: title,
		Type: "toc",
		// 由于 XMLNode 结构体中没有 Level 字段，需要将 level 信息存储在其他现有字段中
		// 这里我们可以考虑将 level 信息编码到 Name 或 Content 中，或者添加自定义属性
	}
	x.doc.TOC = append(x.doc.TOC, node)
	return nil
}

func (x *XMLCollector) Render(outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	file.WriteString(xml.Header)
	encoder := xml.NewEncoder(file)
	encoder.Indent("", "  ")
	return encoder.Encode(x.doc)
}

type XMLExporter struct {
	*BaseExporter
}

func NewXMLExporter(p *project.Project) *XMLExporter {
	return &XMLExporter{
		BaseExporter: NewBaseExporter(p, NewXMLCollector()),
	}
}
