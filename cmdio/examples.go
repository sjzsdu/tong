package cmdio

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

// EchoProcessor 简单的回显处理器，将输入回显给用户
type EchoProcessor struct {
	*BaseProcessor
}

// NewEchoProcessor 创建一个新的回显处理器
func NewEchoProcessor() *EchoProcessor {
	config := DefaultProcessorConfig()
	config.Mode = BatchMode

	processFunc := func(ctx context.Context, input string) (string, error) {
		// 模拟处理延迟
		time.Sleep(time.Second)
		return fmt.Sprintf("你输入了: %s", input), nil
	}

	return &EchoProcessor{
		BaseProcessor: NewBaseProcessor(config, processFunc),
	}
}

// ProcessInput 处理输入
func (p *EchoProcessor) ProcessInput(ctx context.Context, input string) error {
	// 创建批量处理器
	batchProcessor := NewBatchProcessor(p.config, p.processFunc)
	batchProcessor.SetOutputWriter(p.writer)
	return batchProcessor.ProcessInput(ctx, input)
}

// ProcessOutput 处理输出
func (p *EchoProcessor) ProcessOutput(ctx context.Context) error {
	// 创建批量处理器
	batchProcessor := NewBatchProcessor(p.config, p.processFunc)
	batchProcessor.SetOutputWriter(p.writer)
	return batchProcessor.ProcessOutput(ctx)
}

// StreamEchoProcessor 流式回显处理器，将输入逐字符回显给用户
type StreamEchoProcessor struct {
	*BaseProcessor
}

// NewStreamEchoProcessor 创建一个新的流式回显处理器
func NewStreamEchoProcessor() *StreamEchoProcessor {
	config := DefaultProcessorConfig()
	config.Mode = StreamMode
	config.StreamInterval = time.Millisecond * 50

	processFunc := func(ctx context.Context, input string) (string, error) {
		// 这个函数在流式处理器中不会被直接使用
		return input, nil
	}

	return &StreamEchoProcessor{
		BaseProcessor: NewBaseProcessor(config, processFunc),
	}
}

// ProcessInput 处理输入
func (p *StreamEchoProcessor) ProcessInput(ctx context.Context, input string) error {
	// 创建自定义流式处理器
	customProcessFunc := func(ctx context.Context, input string, writer io.Writer) error {
		// 添加前缀
		response := fmt.Sprintf("你输入了: %s\n\n字符逐个显示:\n", input)
		writer.Write([]byte(response))

		// 逐字符输出，模拟流式响应
		for _, char := range input {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				writer.Write([]byte(string(char)))
				time.Sleep(p.config.StreamInterval)
			}
		}

		// 添加后缀
		writer.Write([]byte("\n\n处理完成！"))
		return nil
	}

	streamProcessor := NewCustomStreamProcessor(p.config, customProcessFunc)
	streamProcessor.SetOutputWriter(p.writer)
	return streamProcessor.ProcessInput(ctx, input)
}

// ProcessOutput 处理输出
func (p *StreamEchoProcessor) ProcessOutput(ctx context.Context) error {
	// 创建流式处理器
	streamProcessor := NewStreamProcessor(p.config, p.processFunc)
	streamProcessor.SetOutputWriter(p.writer)
	return streamProcessor.ProcessOutput(ctx)
}

// DelayedProcessor 延迟处理器，模拟长时间处理
type DelayedProcessor struct {
	*BaseProcessor
	delayTime time.Duration
}

// NewDelayedProcessor 创建一个新的延迟处理器
func NewDelayedProcessor(delayTime time.Duration, mode ResponseMode) *DelayedProcessor {
	config := DefaultProcessorConfig()
	config.Mode = mode

	processFunc := func(ctx context.Context, input string) (string, error) {
		// 模拟长时间处理
		time.Sleep(delayTime)

		// 生成响应
		response := strings.Builder{}
		response.WriteString(fmt.Sprintf("处理完成，耗时 %v\n", delayTime))
		response.WriteString(fmt.Sprintf("你的输入是: %s\n", input))
		response.WriteString("处理模式: ")
		if mode == BatchMode {
			response.WriteString("批量模式")
		} else {
			response.WriteString("流式模式")
		}

		return response.String(), nil
	}

	return &DelayedProcessor{
		BaseProcessor: NewBaseProcessor(config, processFunc),
		delayTime:     delayTime,
	}
}

// ProcessInput 处理输入
func (p *DelayedProcessor) ProcessInput(ctx context.Context, input string) error {
	if p.config.Mode == BatchMode {
		batchProcessor := NewBatchProcessor(p.config, p.processFunc)
		batchProcessor.SetOutputWriter(p.writer)
		return batchProcessor.ProcessInput(ctx, input)
	} else {
		streamProcessor := NewStreamProcessor(p.config, p.processFunc)
		streamProcessor.SetOutputWriter(p.writer)
		return streamProcessor.ProcessInput(ctx, input)
	}
}

// ProcessOutput 处理输出
func (p *DelayedProcessor) ProcessOutput(ctx context.Context) error {
	if p.config.Mode == BatchMode {
		batchProcessor := NewBatchProcessor(p.config, p.processFunc)
		batchProcessor.SetOutputWriter(p.writer)
		return batchProcessor.ProcessOutput(ctx)
	} else {
		streamProcessor := NewStreamProcessor(p.config, p.processFunc)
		streamProcessor.SetOutputWriter(p.writer)
		return streamProcessor.ProcessOutput(ctx)
	}
}