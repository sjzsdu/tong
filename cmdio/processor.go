package cmdio

import "context"

// InteractiveProcessor 交互式处理器接口
type InteractiveProcessor interface {
	// ProcessInput 处理用户输入
	ProcessInput(ctx context.Context, input string) (string, error)
	ProcessInputStream(ctx context.Context, input string, callback func(content string, done bool)) error
}

// BaseProcessor 基础处理器实现，作为其他处理器的基类
type BaseProcessor struct{}

// NewBaseProcessor 创建新的基础处理器
func NewBaseProcessor() *BaseProcessor {
	return &BaseProcessor{}
}

// ProcessInput 处理用户输入的空实现
func (p *BaseProcessor) ProcessInput(ctx context.Context, input string) (string, error) {
	// 空实现，直接返回输入
	return input, nil
}

// ProcessInputStream 流式处理用户输入的空实现
func (p *BaseProcessor) ProcessInputStream(ctx context.Context, input string, callback func(content string, done bool)) error {
	// 空实现，直接回调完成
	callback(input, true)
	return nil
}


