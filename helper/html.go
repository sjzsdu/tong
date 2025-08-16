package helper

import (
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/sjzsdu/tong/share"
	"golang.org/x/net/html/charset"
)

// DecompressResponse 根据Content-Encoding头处理HTTP响应的解压缩
// 返回一个可读取解压后内容的Reader，使用后应当关闭返回的ReadCloser
func DecompressResponse(resp *http.Response) (io.Reader, io.ReadCloser, error) {
	var decoded io.ReadCloser
	var baseReader io.Reader = resp.Body
	encHeader := strings.ToLower(resp.Header.Get("Content-Encoding"))

	// Some servers may return multiple encodings separated by commas
	if encHeader != "" {
		if strings.Contains(encHeader, "gzip") {
			gz, gerr := gzip.NewReader(resp.Body)
			if gerr == nil {
				decoded = gz
				baseReader = decoded
			} else if share.GetDebug() {
				PrintWithLabel("gzip_decompress_error", gerr.Error())
			}
		} else if strings.Contains(encHeader, "deflate") {
			// Try zlib-wrapped deflate first
			zr, zerr := zlib.NewReader(resp.Body)
			if zerr == nil {
				decoded = zr
				baseReader = decoded
			} else {
				// Fallback to raw deflate
				fr := flate.NewReader(resp.Body)
				decoded = io.NopCloser(fr)
				baseReader = decoded
				if share.GetDebug() {
					PrintWithLabel("deflate_fallback", "used raw flate reader")
				}
			}
		}
	}

	return baseReader, decoded, nil
}

// ReadDecodedBody 读取HTTP响应，处理解压缩和字符集转换
func ReadDecodedBody(resp *http.Response) (string, error) {
	baseReader, decoded, _ := DecompressResponse(resp)

	// 确保关闭解压缩的reader
	if decoded != nil {
		defer decoded.Close()
	}

	// 自动转换到UTF-8
	utf8Reader, err := charset.NewReader(baseReader, resp.Header.Get("Content-Type"))
	if err != nil {
		// 回退到原始reader
		utf8Reader = baseReader
	}

	raw, err := io.ReadAll(utf8Reader)
	if err != nil {
		return "", err
	}

	return string(raw), nil
}

// DocumentToMarkdown 将已解析的goquery.Document转换为Markdown格式
// 使用更智能的内容提取算法以减少信息丢失风险
func DocumentToMarkdown(doc *goquery.Document) string {
	// 尝试多种策略提取内容，按优先级排序
	var contentSel *goquery.Selection

	// 1. 尝试查找常见内容容器
	contentSel = doc.Find("article, main, #content, .content, .article, .post, .entry, .post-content, .article-content").First()

	// 2. 尝试查找具有最多段落的div作为候选内容区域
	if contentSel.Length() == 0 {
		var maxPCount int
		var bestDiv *goquery.Selection

		doc.Find("div").Each(func(i int, s *goquery.Selection) {
			pCount := s.Find("p").Length()
			if pCount > maxPCount {
				maxPCount = pCount
				bestDiv = s
			}
		})

		if maxPCount > 3 { // 至少有3个段落才考虑为主要内容
			contentSel = bestDiv
		}
	}

	// 3. 最后回退到body
	if contentSel == nil || contentSel.Length() == 0 {
		contentSel = doc.Find("body")
	}

	// 创建一个副本以避免修改原始文档
	contentClone := contentSel.Clone()

	// 移除噪音元素
	contentClone.Find("script, style, noscript, iframe, [style*='display:none']").Remove()
	contentClone.Find("header, footer, nav, aside, [role='navigation'], .ads, .advert, .sidebar, .toolbar, .comments").Remove()

	// 如果移除噪音后内容太少，回退到使用整个body
	if strings.TrimSpace(contentClone.Text()) == "" || len(strings.TrimSpace(contentClone.Text())) < 100 {
		return htmlNodesToMarkdown(doc.Find("body"))
	}

	// 转换HTML到Markdown
	return htmlNodesToMarkdown(contentClone)
}

// HTMLToMarkdown 将HTML文本转换为Markdown格式
func HTMLToMarkdown(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	// 使用DocumentToMarkdown处理已解析的文档
	md := DocumentToMarkdown(doc)

	return md, nil
}

// processNestedList 处理嵌套列表结构
func processNestedList(b *strings.Builder, s *goquery.Selection, indent string, prefix string) {
	s.Children().Each(func(_ int, li *goquery.Selection) {
		if goquery.NodeName(li) == "li" {
			// 提取列表项的主要文本，但排除嵌套的ul/ol
			textParts := []string{}
			li.Contents().Each(func(_ int, content *goquery.Selection) {
				name := goquery.NodeName(content)
				if name != "ul" && name != "ol" {
					if name == "#text" {
						text := strings.TrimSpace(content.Text())
						if text != "" {
							textParts = append(textParts, text)
						}
					} else {
						// 对于非列表的其他元素，递归提取文本
						var contentBuilder strings.Builder
						traverseNode(&contentBuilder, content)
						text := strings.TrimSpace(contentBuilder.String())
						if text != "" {
							textParts = append(textParts, text)
						}
					}
				}
			})

			itemText := strings.Join(textParts, " ")
			itemText = strings.TrimSpace(itemText)

			if itemText != "" {
				b.WriteString(indent + prefix + itemText + "\n")
			}

			// 处理嵌套列表
			li.Find("> ul").Each(func(_ int, ul *goquery.Selection) {
				b.WriteString("\n") // 在嵌套列表前增加空行
				processNestedList(b, ul, indent+"  ", "- ")
			})

			li.Find("> ol").Each(func(_ int, ol *goquery.Selection) {
				b.WriteString("\n") // 在嵌套列表前增加空行
				processOrderedList(b, ol, indent+"  ", 1)
			})
		}
	})
}

// processOrderedList 处理有序列表
func processOrderedList(b *strings.Builder, s *goquery.Selection, indent string, startIndex int) {
	index := startIndex
	s.Children().Each(func(_ int, li *goquery.Selection) {
		if goquery.NodeName(li) == "li" {
			// 提取列表项的主要文本，但排除嵌套的ul/ol
			textParts := []string{}
			li.Contents().Each(func(_ int, content *goquery.Selection) {
				name := goquery.NodeName(content)
				if name != "ul" && name != "ol" {
					if name == "#text" {
						text := strings.TrimSpace(content.Text())
						if text != "" {
							textParts = append(textParts, text)
						}
					} else {
						// 对于非列表的其他元素，递归提取文本
						var contentBuilder strings.Builder
						traverseNode(&contentBuilder, content)
						text := strings.TrimSpace(contentBuilder.String())
						if text != "" {
							textParts = append(textParts, text)
						}
					}
				}
			})

			itemText := strings.Join(textParts, " ")
			itemText = strings.TrimSpace(itemText)

			if itemText != "" {
				b.WriteString(fmt.Sprintf("%s%d. %s\n", indent, index, itemText))
				index++
			}

			// 处理嵌套列表
			li.Find("> ul").Each(func(_ int, ul *goquery.Selection) {
				b.WriteString("\n") // 在嵌套列表前增加空行
				processNestedList(b, ul, indent+"  ", "- ")
			})

			li.Find("> ol").Each(func(_ int, ol *goquery.Selection) {
				b.WriteString("\n") // 在嵌套列表前增加空行
				processOrderedList(b, ol, indent+"  ", 1)
			})
		}
	})
}

// processTable 将HTML表格转换为Markdown表格
func processTable(b *strings.Builder, table *goquery.Selection) {
	// 获取表头
	headers := []string{}
	table.Find("thead th, tr:first-child th").Each(func(_ int, th *goquery.Selection) {
		headers = append(headers, strings.TrimSpace(th.Text()))
	})

	// 如果没有找到表头，尝试从第一行td获取
	if len(headers) == 0 {
		table.Find("tr:first-child td").Each(func(_ int, td *goquery.Selection) {
			headers = append(headers, strings.TrimSpace(td.Text()))
		})
	}

	// 如果仍然没有表头，创建一些默认列名
	if len(headers) == 0 {
		// 计算列数
		maxColumns := 0
		table.Find("tr").Each(func(_ int, tr *goquery.Selection) {
			columns := tr.Find("td").Length()
			if columns > maxColumns {
				maxColumns = columns
			}
		})

		for i := 0; i < maxColumns; i++ {
			headers = append(headers, fmt.Sprintf("Column %d", i+1))
		}
	}

	// 写入表头
	for i, header := range headers {
		if i > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(header)
	}
	b.WriteString("\n")

	// 写入分隔线
	for i := range headers {
		if i > 0 {
			b.WriteString(" | ")
		}
		b.WriteString("---")
	}
	b.WriteString("\n")

	// 写入表格内容
	table.Find("tbody tr, tr:not(:first-child)").Each(func(_ int, tr *goquery.Selection) {
		cells := []string{}

		tr.Find("td").Each(func(_ int, td *goquery.Selection) {
			cells = append(cells, strings.TrimSpace(td.Text()))
		})

		if len(cells) > 0 {
			for i, cell := range cells {
				if i >= len(headers) {
					break
				}

				if i > 0 {
					b.WriteString(" | ")
				}
				b.WriteString(cell)
			}

			// 补齐缺少的单元格
			for i := len(cells); i < len(headers); i++ {
				b.WriteString(" | ")
			}

			b.WriteString("\n")
		}
	})
}

// ExtractMainContent 从HTML中提取主要内容
// 注意: 此函数已被更强大的DocumentToMarkdown替代，保留此函数以保持向后兼容性
func ExtractMainContent(doc *goquery.Document) *goquery.Selection {
	sel := doc.Find("article, main, #content, .content, .article, .post, .entry, .post-content, .article-content").First()
	if sel.Length() == 0 {
		// 如果没有找到特定容器，回退到body
		sel = doc.Find("body")
	}

	// 移除噪音
	sel.Find("script, style, noscript, iframe, header, footer, nav, aside").Remove()
	sel.Find("[role='navigation'], .ads, .advert, .sidebar, .toolbar").Remove()

	return sel
}

// htmlNodesToMarkdown 为goquery.Selection执行全面的HTML到Markdown转换
// 支持更多HTML元素，包括表格、嵌套列表等
func htmlNodesToMarkdown(sel *goquery.Selection) string {
	var b strings.Builder
	sel.Each(func(_ int, s *goquery.Selection) {
		traverseNode(&b, s)
	})
	// 折叠过多的空行
	txt := b.String()
	txt = regexp.MustCompile("\n{3,}").ReplaceAllString(txt, "\n\n")
	// 确保列表项后有适当的空行
	txt = regexp.MustCompile("(\n[\\d-][^\n]+\n)([^\\d-\n])").ReplaceAllString(txt, "$1\n$2")
	return strings.TrimSpace(txt)
}

// traverseNode 递归地为选择内容编写markdown
func traverseNode(b *strings.Builder, s *goquery.Selection) {
	nodeName := goquery.NodeName(s)

	switch nodeName {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		level := nodeName[1:] // "1".."6"
		b.WriteString(strings.Repeat("#", int(level[0]-'0')))
		b.WriteString(" ")
		b.WriteString(strings.TrimSpace(s.Text()))
		b.WriteString("\n\n")
		return
	case "p":
		text := strings.TrimSpace(s.Text())
		if text != "" {
			b.WriteString(text)
			b.WriteString("\n\n")
		}
		return
	case "div", "section", "article":
		// 处理div、section和article作为内容容器
		hasChildren := false
		s.Contents().Each(func(_ int, c *goquery.Selection) {
			hasChildren = true
			traverseNode(b, c)
		})
		// 如果有内容，确保块结束有足够的空行
		if hasChildren {
			b.WriteString("\n")
		}
		return
	case "br":
		b.WriteString("\n")
		return
	case "ul":
		processNestedList(b, s, "", "- ")
		b.WriteString("\n")
		return
	case "ol":
		processOrderedList(b, s, "", 1)
		b.WriteString("\n")
		return
	case "li":
		// 此处应该由ul/ol的处理函数处理，不单独处理
		return
	case "pre", "code":
		text := strings.TrimSpace(s.Text())
		if text != "" {
			if nodeName == "pre" {
				// 尝试检测语言
				lang := ""
				if s.Find("code").Length() > 0 {
					// 如果pre内有code，尝试从class获取语言
					if class, exists := s.Find("code").First().Attr("class"); exists {
						// 从类似"language-python"提取语言
						langMatch := regexp.MustCompile(`language-(\w+)`).FindStringSubmatch(class)
						if len(langMatch) > 1 {
							lang = langMatch[1]
						}
					}
				}
				b.WriteString("```" + lang + "\n")
				b.WriteString(text)
				b.WriteString("\n```\n\n")
			} else {
				b.WriteString("`" + text + "`")
			}
		}
		return
	case "img":
		alt, _ := s.Attr("alt")
		src, _ := s.Attr("src")
		if src != "" {
			// 确保src是绝对URL
			if !strings.HasPrefix(src, "http") && !strings.HasPrefix(src, "https") {
				// 简单处理，实际应根据页面URL构建完整路径
				if strings.HasPrefix(src, "/") {
					src = "https:" + src
				} else {
					src = "https:/" + src
				}
			}
			b.WriteString(fmt.Sprintf("![%s](%s)\n\n", strings.TrimSpace(alt), src))
		}
		return
	case "a":
		href, _ := s.Attr("href")
		text := strings.TrimSpace(s.Text())
		if href != "" && text != "" {
			// 确保href是绝对URL（简化处理）
			if !strings.HasPrefix(href, "http") && !strings.HasPrefix(href, "https") && !strings.HasPrefix(href, "#") {
				if strings.HasPrefix(href, "/") {
					href = "https:" + href
				} else {
					href = "https:/" + href
				}
			}
			b.WriteString(fmt.Sprintf("[%s](%s)", text, href))
		} else if text != "" {
			b.WriteString(text)
		}
		return
	case "table":
		processTable(b, s)
		b.WriteString("\n\n")
		return
	case "blockquote":
		// 处理引用块
		var quoteBuilder strings.Builder
		s.Contents().Each(func(_ int, c *goquery.Selection) {
			traverseNode(&quoteBuilder, c)
		})
		// 在每行前添加>符号
		lines := strings.Split(quoteBuilder.String(), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				b.WriteString("> " + line + "\n")
			}
		}
		b.WriteString("\n")
		return
	case "hr":
		// 水平分割线
		b.WriteString("\n---\n\n")
		return
	}

	// 默认：处理子节点
	s.Contents().Each(func(_ int, c *goquery.Selection) {
		if goquery.NodeName(c) == "#text" {
			t := strings.TrimSpace(c.Text())
			if t != "" {
				b.WriteString(t + " ")
			}
			return
		}
		traverseNode(b, c)
	})
}
