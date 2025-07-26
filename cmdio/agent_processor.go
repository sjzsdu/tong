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
	// 使用 chains.Call 处理输入
	result, err := chains.Call(ctx, p.executor, map[string]any{"input": input})
	if err != nil {
		return "", fmt.Errorf(lang.T("处理输入时出错")+": %v", err)
	}

	// 从结果中获取输出
	outputKeys := p.executor.GetOutputKeys()
	var output string
	if len(outputKeys) > 0 && result[outputKeys[0]] != nil {
		output = fmt.Sprintf("%v", result[outputKeys[0]])
	}

	// 保存最后处理的内容
	p.lastContent = output
	return output, nil
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
	// 标记是否已经通过流式回调输出了内容
	var streamingDone bool

	// 创建一个流式回调函数
	streamingFunc := func(ctx context.Context, chunk []byte) error {
		// 将字节转换为字符串并回调
		content := string(chunk)
		if content != "" {
			// 累积内容
			accumulatedContent += content
			// 回调当前内容片段
			callback(content, false)
			// 标记已经输出了内容
			streamingDone = true
		}
		return nil
	}

	// 使用 WithStreamingFunc 选项创建流式处理
	options := []chains.ChainCallOption{
		chains.WithStreamingFunc(streamingFunc),
	}

	// 运行 agent executor 通过 chains.Call
	result, err := chains.Call(ctx, p.executor, map[string]any{"input": input}, options...)
	if err != nil {
		return fmt.Errorf(lang.T("流式处理输入时出错")+": %v", err)
	}

	// 从结果中获取输出
	outputKeys := p.executor.GetOutputKeys()
	var output string
	if len(outputKeys) > 0 && result[outputKeys[0]] != nil {
		output = fmt.Sprintf("%v", result[outputKeys[0]])
	}

	// 如果没有通过流式回调输出任何内容，但有最终输出，则发送一次
	if !streamingDone && output != "" {
		callback(output, false)
		accumulatedContent = output
	}

	// 保存最后处理的内容
	p.lastContent = accumulatedContent

	// 标记处理完成
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