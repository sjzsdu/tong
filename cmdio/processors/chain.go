package processors

import (
	"context"
	"io"

	"github.com/tmc/langchaingo/chains"
)

// ChainProcessor 是一个适配器，将 langchaingo 的 chain 适配为 InteractiveProcessor
type ChainProcessor struct {
	*BaseProcessor
	chain  chains.Chain
	input  string
	result string
}

// NewChainProcessor 创建一个新的 Chain 处理器
func NewChainProcessor(chain chains.Chain) *ChainProcessor {
	// 使用批量模式配置
	config := DefaultProcessorConfig()
	config.Mode = BatchMode

	processor := &ChainProcessor{
		chain: chain,
	}

	// 创建处理函数
	processFunc := func(ctx context.Context, input string) (string, error) {
		processor.input = input
		result, err := chains.Run(ctx, processor.chain, input)
		if err != nil {
			return "", err
		}
		processor.result = result
		return result, nil
	}

	// 初始化基础处理器
	processor.BaseProcessor = NewBaseProcessor(config, processFunc)
	return processor
}

// ProcessInput 处理输入
func (p *ChainProcessor) ProcessInput(ctx context.Context, input string) error {
	p.input = input
	return nil
}

// ProcessOutput 处理输出
func (p *ChainProcessor) ProcessOutput(ctx context.Context) error {
	result, err := chains.Run(ctx, p.chain, p.input)
	if err != nil {
		return err
	}

	p.result = result
	return p.writeOutput(result)
}

// NewStreamChainProcessor 创建一个新的流式 Chain 处理器
func NewStreamChainProcessor(chain chains.Chain) *CustomStreamProcessor {
	// 使用流式模式配置
	config := DefaultProcessorConfig()
	config.Mode = StreamMode

	// 创建自定义处理函数
	processFunc := func(ctx context.Context, input string, writer io.Writer) error {
		// 运行 chain，使用流式输出函数
		_, err := chains.Run(ctx, chain, input, chains.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			// 直接将每个流式输出的块写入到 writer
			_, err := writer.Write(chunk)
			return err
		}))

		return err
	}

	return NewCustomStreamProcessor(config, processFunc)
}
