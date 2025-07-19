package cmdio

import (
	"context"
	"sync"
	"time"
)

// BatchProcessor 批量处理器，等待处理完成后一次性返回所有结果
type BatchProcessor struct {
	*BaseProcessor
	// 当前处理的结果
	result string
	// 处理状态
	processing bool
	// 处理错误
	processErr error
	// 条件变量，用于等待处理完成
	cond *sync.Cond
}

// NewBatchProcessor 创建一个新的批量处理器
func NewBatchProcessor(config ProcessorConfig, processFunc func(ctx context.Context, input string) (string, error)) *BatchProcessor {
	if config.Mode != BatchMode {
		config.Mode = BatchMode
	}

	base := NewBaseProcessor(config, processFunc)
	return &BatchProcessor{
		BaseProcessor: base,
		cond:          sync.NewCond(&sync.Mutex{}),
	}
}

// ProcessInput 处理输入，启动异步处理任务
func (p *BatchProcessor) ProcessInput(ctx context.Context, input string) error {
	// 创建带超时的上下文
	timeoutCtx, cancel := p.createTimeoutContext(ctx)
	defer cancel()

	// 设置处理状态
	p.cond.L.Lock()
	p.processing = true
	p.result = ""
	p.processErr = nil
	p.cond.L.Unlock()

	// 启动异步处理
	go func() {
		defer func() {
			p.cond.L.Lock()
			p.processing = false
			p.cond.Signal() // 通知等待的 ProcessOutput
			p.cond.L.Unlock()
		}()

		// 调用处理函数
		result, err := p.processFunc(timeoutCtx, input)

		// 保存结果和错误
		p.cond.L.Lock()
		p.result = result
		p.processErr = err
		p.cond.L.Unlock()
	}()

	return nil
}

// ProcessOutput 等待处理完成并输出结果
func (p *BatchProcessor) ProcessOutput(ctx context.Context) error {
	// 等待处理完成或超时
	p.cond.L.Lock()
	deadline := time.Now().Add(p.config.MaxWaitTime)

	// 如果正在处理，等待处理完成或超时
	for p.processing && p.processErr == nil {
		// 检查上下文是否已取消
		if ctx.Err() != nil {
			p.cond.L.Unlock()
			return ErrProcessingCanceled
		}

		// 设置等待超时
		waitDuration := time.Until(deadline)
		if waitDuration <= 0 {
			p.cond.L.Unlock()
			return ErrProcessingTimeout
		}

		// 等待条件变量通知或超时
		timer := time.AfterFunc(waitDuration, func() {
			p.cond.Signal() // 超时时通知
		})
		p.cond.Wait()
		timer.Stop()
	}

	// 获取结果和错误
	result := p.result
	err := p.processErr
	p.cond.L.Unlock()

	// 如果有错误，返回错误
	if err != nil {
		return err
	}

	// 写入结果
	return p.writeOutput(result)
}