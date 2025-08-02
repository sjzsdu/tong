package cmdio

import (
	"context"

	"github.com/sjzsdu/tong/helper/renders"
)

// InteractiveProcessor 交互式处理器接口
type InteractiveProcessor interface {
	ProcessInput(ctx context.Context, input string, stream bool, render renders.Renderer, loadingDone chan bool) error
}

// BaseProcessor 基础处理器实现，作为其他处理器的基类
type BaseProcessor struct{}

// NewBaseProcessor 创建新的基础处理器
func NewBaseProcessor() *BaseProcessor {
	return &BaseProcessor{}
}

// ProcessInput 处理用户输入的空实现
func (p *BaseProcessor) ProcessInput(ctx context.Context, input string, stream bool, render renders.Renderer, loadingDone chan bool) error {
	// 通知加载动画结束
	return nil
}
