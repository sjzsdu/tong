package cmdio

import (
	"context"
	"io"
	"sync"
)

// BaseProcessor 提供处理器的基础实现
type BaseProcessor struct {
	// 配置信息
	config ProcessorConfig
	// 输出写入器
	writer io.Writer
	// 互斥锁，保护数据访问
	mu sync.Mutex
	// 处理函数，由具体实现提供
	processFunc func(ctx context.Context, input string) (string, error)
}

// NewBaseProcessor 创建一个新的基础处理器
func NewBaseProcessor(config ProcessorConfig, processFunc func(ctx context.Context, input string) (string, error)) *BaseProcessor {
	return &BaseProcessor{
		config:      config,
		processFunc: processFunc,
		mu:          sync.Mutex{},
	}
}

// SetOutputWriter 设置输出写入器
func (p *BaseProcessor) SetOutputWriter(writer io.Writer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.writer = writer
}

// GetConfig 获取处理器配置
func (p *BaseProcessor) GetConfig() ProcessorConfig {
	return p.config
}

// writeOutput 写入输出内容
func (p *BaseProcessor) writeOutput(content string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.writer == nil {
		return nil
	}

	_, err := p.writer.Write([]byte(content))
	return err
}

// createTimeoutContext 创建带超时的上下文
func (p *BaseProcessor) createTimeoutContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if p.config.Timeout > 0 {
		return context.WithTimeout(ctx, p.config.Timeout)
	}
	return ctx, func() {}
}