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
func (m *MarkdownRenderer) WriteStream(content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 标记为正在输出状态
	if !m.isOutputing {
		m.isOutputing = true
	}

	// 将内容添加到缓冲区
	m.buffer.WriteString(content)
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

	// 渲染 Markdown 内容
	rendered, err := m.renderer.Render(content)
	if err != nil {
		// 渲染失败时，输出原始内容
		fmt.Print(content)
	} else {
		// 处理渲染结果，移除多余空行
		rendered = strings.TrimSpace(rendered)
		// 将连续的多个空行替换为单个空行
		for strings.Contains(rendered, "\n\n\n") {
			rendered = strings.ReplaceAll(rendered, "\n\n\n", "\n\n")
		}
		fmt.Print(rendered)
	}

	// 重置状态
	m.reset()
}

// reset 重置渲染器状态
func (m *MarkdownRenderer) reset() {
	m.buffer.Reset()
	m.isOutputing = false
}
