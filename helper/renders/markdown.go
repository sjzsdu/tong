package renders

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
)

// MarkdownRenderer 实现 Renderer 接口，提供 Markdown 渲染功能
type MarkdownRenderer struct {
	renderer    *glamour.TermRenderer
	buffer      strings.Builder
	mu          sync.Mutex
	isOutputing bool
}

// NewMarkdownRenderer 创建一个新的 Markdown 渲染器
func NewMarkdownRenderer() (*MarkdownRenderer, error) {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(120),
	)
	if err != nil {
		return nil, fmt.Errorf("初始化 Markdown 渲染器失败: %v", err)
	}

	return &MarkdownRenderer{
		renderer:    renderer,
		buffer:      strings.Builder{},
		isOutputing: false,
	}, nil
}

// WriteStream 实现 Renderer 接口，将内容写入缓冲区
// 当内容满足段落结束条件时，会立即渲染并输出
func (m *MarkdownRenderer) WriteStream(content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 标记为正在输出状态
	if !m.isOutputing {
		m.isOutputing = true
	}

	// 将内容添加到缓冲区
	m.buffer.WriteString(content)
	
	// 判断当前内容是否满足段落结束条件
	bufferContent := m.buffer.String()
	
	// 检查是否有完整段落需要渲染
	if m.isParagraphComplete(bufferContent) {
		// 查找最后一个换行符的位置
		lastNewlinePos := strings.LastIndex(bufferContent, "\n")
		if lastNewlinePos > 0 {
			// 只渲染到最后一个换行符
			contentToRender := bufferContent[:lastNewlinePos+1] // 包含换行符
			remaining := bufferContent[lastNewlinePos+1:]
			
			// 渲染并输出当前段落
			m.renderContent(contentToRender)
			
			// 重置缓冲区，保留剩余内容
			m.buffer.Reset()
			m.buffer.WriteString(remaining)
		} else {
			// 整个内容作为一个段落渲染
			m.renderContent(bufferContent)
			m.buffer.Reset()
		}
	}
	
	return nil
}

// Done 实现 Renderer 接口，完成输出并渲染 Markdown
func (m *MarkdownRenderer) Done() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果没有开始输出，直接返回
	if !m.isOutputing {
		return
	}

	// 获取缓冲区内容
	content := m.buffer.String()
	if content == "" {
		m.reset()
		return
	}

	// 渲染剩余内容
	m.renderContent(content)

	// 重置状态
	m.reset()
}

// reset 重置渲染器状态
func (m *MarkdownRenderer) reset() {
	m.buffer.Reset()
	m.isOutputing = false
}

// isParagraphComplete 判断内容是否构成完整段落
func (m *MarkdownRenderer) isParagraphComplete(content string) bool {
	// 如果内容为空，不是完整段落
	if content == "" {
		return false
	}
	
	// 判断是否在代码块内
	lines := strings.Split(content, "\n")
	inCodeBlock := false
	
	// 检查所有行，包括最后一行
	for _, line := range lines {
		// 检查是否是代码块标记
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "```") {
			inCodeBlock = !inCodeBlock
		}
	}
	
	// 如果当前在代码块内，则不认为是完整段落
	if inCodeBlock {
		return false
	}
	
	// 判断段落结束的条件：
	// 1. 内容以换行符结束，且不在代码块内，表示段落结束
	if strings.HasSuffix(content, "\n") {
		// 检查最后一个换行符前是否还有内容
		contentWithoutLastNewline := strings.TrimSuffix(content, "\n")
		if contentWithoutLastNewline != "" {
			return true
		}
	}
	
	// 2. 内容包含完整的 Markdown 块元素（如代码块）
	if strings.Contains(content, "```") {
		// 计算 ``` 出现的次数，如果是偶数，说明代码块是完整的
		count := strings.Count(content, "```")
		if count >= 2 && count%2 == 0 {
			return true
		}
	}
	
	// 3. 内容长度超过一定阈值，可以作为一个段落输出
	// 这里设置为500个字符，可以根据实际需求调整
	if len(content) > 500 {
		// 但要确保不会在单词中间截断
		lastNewline := strings.LastIndex(content, "\n")
		if lastNewline > 0 && lastNewline > len(content)-100 {
			return true
		}
	}
	
	return false
}

// renderContent 渲染并输出内容
func (m *MarkdownRenderer) renderContent(content string) {
	// 确保内容以换行符结束
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	// 渲染 Markdown 内容
	rendered, err := m.renderer.Render(content)
	if err != nil {
		// 渲染失败时，输出原始内容
		fmt.Print(content)
	} else {
		// 处理渲染结果
		// 注意：不使用 TrimSpace，以保留原始的换行符
		// 将连续的多个空行替换为单个空行
		for strings.Contains(rendered, "\n\n\n") {
			rendered = strings.ReplaceAll(rendered, "\n\n\n", "\n\n")
		}
		
		// 确保输出以换行符结束
		if !strings.HasSuffix(rendered, "\n") {
			rendered += "\n"
		}
		
		fmt.Print(rendered)
	}
}
