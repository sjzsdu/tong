package cmdio

import (
	"context"
	"io"
	"sync"
	"time"
)

// StreamProcessor 流式处理器，在处理过程中不断返回部分结果
type StreamProcessor struct {
	*BaseProcessor
	// 输入通道
	inputCh chan string
	// 输出通道
	outputCh chan string
	// 错误通道
	errCh chan error
	// 处理状态
	processing bool
	// 互斥锁
	mu sync.Mutex
	// 处理完成通道
	doneCh chan struct{}
}

// NewStreamProcessor 创建一个新的流式处理器
func NewStreamProcessor(config ProcessorConfig, processFunc func(ctx context.Context, input string) (string, error)) *StreamProcessor {
	if config.Mode != StreamMode {
		config.Mode = StreamMode
	}

	base := NewBaseProcessor(config, processFunc)
	return &StreamProcessor{
		BaseProcessor: base,
		inputCh:       make(chan string, 1),
		outputCh:      make(chan string, 100),
		errCh:         make(chan error, 1),
		doneCh:        make(chan struct{}),
	}
}

// ProcessInput 处理输入，启动异步处理任务
func (p *StreamProcessor) ProcessInput(ctx context.Context, input string) error {
	p.mu.Lock()
	if p.processing {
		p.mu.Unlock()
		return nil // 已经在处理中，忽略新输入
	}
	p.processing = true
	p.mu.Unlock()

	// 创建带超时的上下文
	timeoutCtx, cancel := p.createTimeoutContext(ctx)

	// 启动处理协程
	go func() {
		defer func() {
			cancel()
			close(p.outputCh)
			close(p.errCh)
			close(p.doneCh)
			p.mu.Lock()
			p.processing = false
			p.mu.Unlock()
		}()

		// 模拟流式处理，将结果分批次输出
		result, err := p.processFunc(timeoutCtx, input)
		if err != nil {
			p.errCh <- err
			return
		}

		// 模拟流式输出，将结果分成多个部分发送
		chunkSize := 50 // 每次发送的字符数
		for i := 0; i < len(result); i += chunkSize {
			// 检查上下文是否已取消
			if timeoutCtx.Err() != nil {
				p.errCh <- ErrProcessingCanceled
				return
			}

			end := i + chunkSize
			if end > len(result) {
				end = len(result)
			}

			chunk := result[i:end]
			select {
			case p.outputCh <- chunk:
				// 成功发送
			case <-timeoutCtx.Done():
				p.errCh <- ErrProcessingCanceled
				return
			}

			// 模拟处理延迟
			time.Sleep(p.config.StreamInterval)
		}
	}()

	return nil
}

// ProcessOutput 处理输出，将结果写入渲染器
func (p *StreamProcessor) ProcessOutput(ctx context.Context) error {
	var lastError error

	// 创建一个定时器，用于定期检查是否有新的输出
	ticker := time.NewTicker(p.config.StreamInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ErrProcessingCanceled

		case err := <-p.errCh:
			lastError = err
			if err != nil {
				return err
			}

		case output, ok := <-p.outputCh:
			if !ok {
				// 输出通道已关闭，处理完成
				return lastError
			}

			// 写入输出
			if err := p.writeOutput(output); err != nil {
				return err
			}

		case <-p.doneCh:
			// 处理已完成
			return lastError

		case <-ticker.C:
			// 检查处理状态
			p.mu.Lock()
			isProcessing := p.processing
			p.mu.Unlock()

			if !isProcessing {
				// 处理已完成，但没有错误
				return nil
			}
		}
	}
}

// 实现自定义的流式处理器，用于实际场景

// StreamWriter 流式写入器，实现io.Writer接口
type StreamWriter struct {
	outputCh chan<- string
}

// NewStreamWriter 创建一个新的流式写入器
func NewStreamWriter(outputCh chan<- string) *StreamWriter {
	return &StreamWriter{outputCh: outputCh}
}

// Write 实现io.Writer接口
func (w *StreamWriter) Write(p []byte) (n int, err error) {
	w.outputCh <- string(p)
	return len(p), nil
}

// CustomStreamProcessor 自定义流式处理器，支持实时流式输出
type CustomStreamProcessor struct {
	*StreamProcessor
	// 自定义处理函数
	customProcessFunc func(ctx context.Context, input string, writer io.Writer) error
}

// NewCustomStreamProcessor 创建一个新的自定义流式处理器
func NewCustomStreamProcessor(config ProcessorConfig, processFunc func(ctx context.Context, input string, writer io.Writer) error) *CustomStreamProcessor {
	sp := &CustomStreamProcessor{
		StreamProcessor:    NewStreamProcessor(config, nil),
		customProcessFunc: processFunc,
	}

	return sp
}

// ProcessInput 处理输入，启动异步处理任务
func (p *CustomStreamProcessor) ProcessInput(ctx context.Context, input string) error {
	p.mu.Lock()
	if p.processing {
		p.mu.Unlock()
		return nil // 已经在处理中，忽略新输入
	}
	p.processing = true
	p.mu.Unlock()

	// 创建带超时的上下文
	timeoutCtx, cancel := p.createTimeoutContext(ctx)

	// 创建流式写入器
	streamWriter := NewStreamWriter(p.outputCh)

	// 启动处理协程
	go func() {
		defer func() {
			cancel()
			close(p.outputCh)
			close(p.errCh)
			close(p.doneCh)
			p.mu.Lock()
			p.processing = false
			p.mu.Unlock()
		}()

		// 调用自定义处理函数
		err := p.customProcessFunc(timeoutCtx, input, streamWriter)
		if err != nil {
			p.errCh <- err
		}
	}()

	return nil
}