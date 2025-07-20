package cmdio

import (
	"context"
	"fmt"

	"github.com/sjzsdu/tong/lang"
	"github.com/tmc/langchaingo/chains"
)

// ChainProcessor 是一个适配 langchaingo chain 的处理器
// 实现了 InteractiveProcessor 接口
type ChainProcessor struct {
	chain       chains.Chain // langchaingo 的 chain
	streamMode  bool         // 是否使用流式输出
	lastContent string       // 最后一次处理的内容
}

// NewChainProcessor 创建一个新的 ChainProcessor
func NewChainProcessor(chain chains.Chain, streamMode bool) *ChainProcessor {
	return &ChainProcessor{
		chain:      chain,
		streamMode: streamMode,
	}
}

// ProcessInput 处理用户输入，非流式模式
func (p *ChainProcessor) ProcessInput(ctx context.Context, input string) (string, error) {
	// 使用 chain 处理输入
	result, err := chains.Run(ctx, p.chain, input)
	if err != nil {
		return "", fmt.Errorf(lang.T("处理输入时出错")+": %v", err)
	}

	// 保存最后处理的内容
	p.lastContent = result
	return result, nil
}

// ProcessInputStream 流式处理用户输入
func (p *ChainProcessor) ProcessInputStream(ctx context.Context, input string, callback func(content string, done bool)) error {
	if !p.streamMode {
		// 如果不是流式模式，则使用非流式处理
		content, err := p.ProcessInput(ctx, input)
		if err != nil {
			return err
		}
		callback(content, true)
		return nil
	}

	// 创建一个累积内容的变量
	var accumulatedContent string

	// 创建一个流式回调函数
	streamingFunc := func(ctx context.Context, chunk []byte) error {
		// 将字节转换为字符串并回调
		content := string(chunk)
		if content != "" {
			// 累积内容
			accumulatedContent += content
			// 回调当前内容片段
			callback(content, false)
		}
		return nil
	}

	// 使用 WithStreamingFunc 选项创建流式处理
	options := []chains.ChainCallOption{
		chains.WithStreamingFunc(streamingFunc),
	}

	// 运行 chain
	result, err := chains.Run(ctx, p.chain, input, options...)
	if err != nil {
		return fmt.Errorf(lang.T("流式处理输入时出错")+": %v", err)
	}

	// 如果累积内容为空但结果不为空，使用结果
	if accumulatedContent == "" && result != "" {
		accumulatedContent = result
	}

	// 保存最后处理的内容
	p.lastContent = accumulatedContent

	// 标记处理完成，并传递累积的内容
	callback(accumulatedContent, true)
	return nil
}

// CreateChatAdapter 创建一个适配 langchaingo chain 的交互式会话
func CreateChatAdapter(chain chains.Chain, streamMode bool) *InteractiveSession {
	// 创建处理器
	processor := NewChainProcessor(chain, streamMode)

	// 创建交互式会话
	session := NewInteractiveSession(
		processor,
		WithWelcome(lang.T("欢迎使用 AI 助手，输入问题开始对话，输入 'quit' 或 'exit' 退出")),
		WithTip(lang.T("提示: 您可以询问任何问题，AI 将尽力回答")),
		WithStream(streamMode),
		WithPrompt("🤖 > "),
	)

	return session
}
