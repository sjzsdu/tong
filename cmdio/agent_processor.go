package cmdio

import (
	"context"
	"fmt"

	"github.com/sjzsdu/tong/helper/renders"
	"github.com/sjzsdu/tong/lang"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/chains"
)

// AgentProcessor 是一个适配 langchaingo agent 的处理器
// 实现了 InteractiveProcessor 接口
type AgentProcessor struct {
	executor    *agents.Executor // langchaingo 的 agent executor
	streamMode  bool             // 是否使用流式输出
	lastContent string           // 最后一次处理的内容
	Handler     callbacks.Handler
	handled     bool
	render      renders.Renderer
	loadingDone chan bool
}

// NewAgentProcessor 创建一个新的 AgentProcessor
func NewAgentProcessor(streamMode bool) *AgentProcessor {
	// 创建处理器实例
	processor := &AgentProcessor{
		streamMode: streamMode,
	}

	// 创建并设置回调处理器
	processor.Handler = NewCallbackHandler(processor)

	return processor
}

func (p *AgentProcessor) SetExecutor(executor *agents.Executor) {
	p.executor = executor
}

func (p *AgentProcessor) StartProcess(stream bool, render renders.Renderer, loadingDone chan bool) {
	p.render = render
	p.loadingDone = loadingDone
	p.streamMode = stream
	p.handled = false
	p.lastContent = ""
}

// ProcessInput 处理用户输入
func (p *AgentProcessor) ProcessInput(ctx context.Context, input string, stream bool, render renders.Renderer, loadingDone chan bool) error {
	p.StartProcess(stream, render, loadingDone)

	if stream {
		streamingFunc := func(ctx context.Context, chunk []byte) error {
			content := string(chunk)
			done := false
			if content == "" {
				done = true
			}
			return p.ProcessStreaming(content, done)
		}

		// 使用 WithStreamingFunc 选项创建流式处理
		options := []chains.ChainCallOption{
			chains.WithStreamingFunc(streamingFunc),
		}

		// 运行 agent executor
		_, err := chains.Call(ctx, p.executor, map[string]any{"input": input}, options...)
		if err != nil {
			return fmt.Errorf(lang.T("流式处理输入时出错")+": %v", err)
		}
		return nil
	} else {
		options := []chains.ChainCallOption{}
		result, err := chains.Call(ctx, p.executor, map[string]any{"input": input}, options...)
		if err != nil {
			return fmt.Errorf(lang.T("处理输入时出错")+": %v", err)
		}
		p.ProcessStreaming("", false)

		// 从结果中获取输出
		outputKeys := p.executor.GetOutputKeys()
		var output string
		if len(outputKeys) > 0 && result[outputKeys[0]] != nil {
			output = fmt.Sprintf("%v", result[outputKeys[0]])
		}
		p.ProcessStreaming(output, true)
		return nil
	}
}

func (p *AgentProcessor) ProcessStreaming(content string, done bool) error {
	if !p.handled {
		p.handled = true
		p.loadingDone <- true
		<-p.loadingDone
	}
	p.lastContent += content
	p.render.WriteStream(content)
	if done {
		p.render.Done()
	}
	return nil
}

// AgentCreator 是一个创建agent executor的函数类型
type AgentCreator func(processor *AgentProcessor) *agents.Executor

// CreateAgentAdapter 创建一个适配 langchaingo agent 的交互式会话
func CreateAgentAdapter(streamMode bool, createAgentFunc AgentCreator) *InteractiveSession {

	processor := NewAgentProcessor(streamMode)

	// 使用提供的函数创建executor
	executor := createAgentFunc(processor)

	// 创建处理器
	processor.SetExecutor(executor)

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
