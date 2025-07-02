package renders

import (
	"fmt"
)

// TextRenderer 实现 Renderer 接口，提供纯文本渲染功能
type TextRenderer struct {
}

// NewTextRenderer 创建一个新的文本渲染器
func NewTextRenderer() *TextRenderer {
	return &TextRenderer{}
}

// WriteStream 实现 Renderer 接口，将内容写入缓冲区
// 如果是第一次写入，会显示 "output...." 提示
func (t *TextRenderer) WriteStream(content string) error {
	fmt.Print(content)
	return nil
}

// Done 实现 Renderer 接口，完成输出并显示文本
func (t *TextRenderer) Done() {
	fmt.Println()
}
