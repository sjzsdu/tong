package helper

import (
	"fmt"
	"os"

	"github.com/sjzsdu/tong/helper/renders"
	"github.com/sjzsdu/tong/share"
)

func GetDefaultRenderer() renders.Renderer {
	render := os.Getenv("WN_RENDER")
	if render == "" {
		render = share.DEFAULT_RENDERER
	}
	return NewRenderer(render)
}

func NewRenderer(renderer string) renders.Renderer {
	switch renderer {
	case "text":
		return renders.NewTextRenderer()
	case "markdown":
		render, err := renders.NewMarkdownRenderer()
		if err != nil {
			fmt.Printf("初始化 Markdown 渲染器失败: %v，将使用文本渲染器\n", err)
			return renders.NewTextRenderer()
		}
		return render
	default:
		return renders.NewTextRenderer()
	}
}
