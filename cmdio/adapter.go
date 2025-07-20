package cmdio

import (
	"io"

	"github.com/sjzsdu/tong/helper/renders"
)

// RendererWriter 是一个适配器，使渲染器能够作为io.Writer使用
type RendererWriter struct {
	renderer renders.Renderer
}

// NewRendererWriter 创建一个新的渲染器写入器
func NewRendererWriter(renderer renders.Renderer) *RendererWriter {
	return &RendererWriter{renderer: renderer}
}

// Write 实现io.Writer接口
func (w *RendererWriter) Write(p []byte) (n int, err error) {
	err = w.renderer.WriteStream(string(p))
	return len(p), err
}

// SetProcessorWriter 设置处理器的输出写入器
// 如果写入器是渲染器，则使用RendererWriter适配器
func SetProcessorWriter(processor InteractiveProcessor, writer interface{}) {
	switch w := writer.(type) {
	case io.Writer:
		processor.SetOutputWriter(w)
	case renders.Renderer:
		processor.SetOutputWriter(NewRendererWriter(w))
	}
}