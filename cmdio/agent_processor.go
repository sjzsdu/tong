package cmdio

import (
	"context"
	"fmt"

	"github.com/sjzsdu/tong/lang"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
)

// AgentProcessor 是一个适配 langchaingo agent 的处理器
// 实现了 InteractiveProcessor 接口
type AgentProcessor struct {
	executor    *agents.Executor // langchaingo 的 agent executor
	streamMode  bool            // 是否使用流式输出
	lastContent string          // 最后一次处理的内容
}

// NewAgentProcessor 创建一个新的 AgentProcessor
func NewAgentProcessor(executor *agents.Executor, streamMode bool) *AgentProcessor {
	return &AgentProcessor{
		executor:    executor,
		streamMode: streamMode,
	}
}

// ProcessInput 处理用户输入，非流式模式
func (p *AgentProcessor) ProcessInput(ctx context.Context, input string) (string, error) {
	// 使用 chains.Run 处理输入
	result, err := chains.Run(ctx, p.executor, input)
	if err != nil {
		return "", fmt.Errorf(lang.T("处理输入时出错")+": %v", err)
	}

	// 保存最后处理的内容
	p.lastContent = result
	return result, nil
}

// ProcessInputStream 流式处理用户输入
func (p *AgentProcessor) ProcessInputStream(ctx context.Context, input string, callback func(content string, done bool)) error {
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

	// 运行 agent executor 通过 chains.Run
	result, err := chains.Run(ctx, p.executor, input, options...)
	if err != nil {
		return fmt.Errorf(lang.T("流式处理输入时出错")+": %v", err)
	}

	// 如果累积内容为空但结果不为空，使用结果
	if accumulatedContent == "" && result != "" {
		accumulatedContent = result
	}

	// 保存最后处理的内容
	p.lastContent = accumulatedContent

	// 标记处理完成，但不再传递累积的内容，避免重复输出
	callback("", true)
	return nil
}

// CreateAgentAdapter 创建一个适配 langchaingo agent 的交互式会话
func CreateAgentAdapter(executor *agents.Executor, streamMode bool) *InteractiveSession {
	// 创建处理器
	processor := NewAgentProcessor(executor, streamMode)

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