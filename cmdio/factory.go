package cmdio

import (
	"time"

	"github.com/sjzsdu/tong/cmdio/processors"
	"github.com/tmc/langchaingo/chains"
)

// 类型别名，使外部调用更简洁
type (
	// ResponseMode 响应模式类型
	ResponseMode = processors.ResponseMode
	// ProcessorConfig 处理器配置
	ProcessorConfig = processors.ProcessorConfig
	// InteractiveProcessor 交互式处理器接口
	InteractiveProcessor = processors.InteractiveProcessor
	// ProcessFunc 处理函数类型
	ProcessFunc = processors.ProcessFunc
	// CustomStreamProcessFunc 自定义流式处理函数类型
	CustomStreamProcessFunc = processors.CustomStreamProcessFunc
)

// 常量别名
const (
	// BatchMode 批量处理模式
	BatchMode = processors.BatchMode
	// StreamMode 流式处理模式
	StreamMode = processors.StreamMode
)

// DefaultProcessorConfig 创建默认处理器配置的副本
func DefaultProcessorConfig() ProcessorConfig {
	return processors.DefaultProcessorConfig()
}

// NewBatchProcessor 创建一个新的批量处理器
func NewBatchProcessor(processFunc ProcessFunc) InteractiveProcessor {
	config := DefaultProcessorConfig()
	config.Mode = BatchMode
	return processors.NewBatchProcessor(config, processFunc)
}

// NewStreamProcessor 创建一个新的流式处理器
func NewStreamProcessor(processFunc ProcessFunc) InteractiveProcessor {
	config := DefaultProcessorConfig()
	config.Mode = StreamMode
	return processors.NewStreamProcessor(config, processFunc)
}

// NewCustomStreamProcessor 创建一个新的自定义流式处理器
func NewCustomStreamProcessor(processFunc CustomStreamProcessFunc) InteractiveProcessor {
	config := DefaultProcessorConfig()
	config.Mode = StreamMode
	return processors.NewCustomStreamProcessor(config, processFunc)
}

// NewChainProcessor 创建一个新的 Chain 处理器
func NewChainProcessor(chain chains.Chain) InteractiveProcessor {
	processor := processors.NewChainProcessor(chain)
	return processor
}

// NewStreamChainProcessor 创建一个新的流式 Chain 处理器
func NewStreamChainProcessor(chain chains.Chain) InteractiveProcessor {
	processor := processors.NewStreamChainProcessor(chain)
	return processor
}

// NewEchoProcessor 创建一个新的回显处理器
func NewEchoProcessor() InteractiveProcessor {
	return processors.NewEchoProcessor()
}

// NewStreamEchoProcessor 创建一个新的流式回显处理器
func NewStreamEchoProcessor() InteractiveProcessor {
	return processors.NewStreamEchoProcessor()
}

// NewDelayedProcessor 创建一个新的延迟处理器
func NewDelayedProcessor(delayTime time.Duration, mode ResponseMode) InteractiveProcessor {
	return processors.NewDelayedProcessor(delayTime, mode)
}

// CreateChatAdapter 创建一个聊天适配器，将 chain 转换为 InteractiveSession
func CreateChatAdapter(chain chains.Chain, streamMode bool) *InteractiveSession {
	var processor InteractiveProcessor

	if streamMode {
		processor = NewStreamChainProcessor(chain)
	} else {
		processor = NewChainProcessor(chain)
	}

	return NewInteractiveSession(
		processor,
		WithWelcome("欢迎使用 AI 聊天助手！输入 'quit' 退出。"),
		WithTip("提示：您可以询问任何问题，AI 将尽力回答。"),
		WithPrompt(">"),
		WithExitCommands("quit", "exit", "q", "退出"),
	)
}
