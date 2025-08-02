package cmdio

import (
	"context"
	"fmt"

	"github.com/sjzsdu/tong/helper/renders"
	"github.com/sjzsdu/tong/lang"
	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/chains"
)

// ChainProcessor 是一个适配 langchaingo chain 的处理器
// 实现了 InteractiveProcessor 接口
type ChainProcessor struct {
	chain       chains.Chain // langchaingo 的 chain
	streamMode  bool         // 是否使用流式输出
	lastContent string       // 最后一次处理的内容
	Handler     callbacks.Handler
	handled     bool
	render      renders.Renderer
	loadingDone chan bool
}

// NewChainProcessor 创建一个新的 ChainProcessor
func NewChainProcessor(chain chains.Chain, streamMode bool) *ChainProcessor {
	processor := &ChainProcessor{
		chain:      chain,
		streamMode: streamMode,
	}
	processor.Handler = NewCallbackHandler(processor)
	return processor
}

func (p *ChainProcessor) StartProcess(stream bool, render renders.Renderer, loadingDone chan bool) {
	p.render = render
	p.loadingDone = loadingDone
	p.streamMode = stream
	p.handled = false
	p.lastContent = ""
}

// ProcessInput 处理用户输入
func (p *ChainProcessor) ProcessInput(ctx context.Context, input string, stream bool, render renders.Renderer, loadingDone chan bool) error {
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

		// 运行 chain
		_, err := chains.Call(ctx, p.chain, map[string]any{"input": input}, options...)
		if err != nil {
			return fmt.Errorf(lang.T("流式处理输入时出错")+": %v", err)
		}
		return nil
	} else {
		options := []chains.ChainCallOption{}
		result, err := chains.Call(ctx, p.chain, map[string]any{"input": input}, options...)
		if err != nil {
			return fmt.Errorf(lang.T("处理输入时出错")+": %v", err)
		}
		p.ProcessStreaming("", false)

		// 从结果中获取输出
		outputKeys := p.chain.GetOutputKeys()
		var output string
		if len(outputKeys) > 0 && result[outputKeys[0]] != nil {
			output = fmt.Sprintf("%v", result[outputKeys[0]])
		}
		p.ProcessStreaming(output, true)
		return nil
	}
}

func (p *ChainProcessor) ProcessStreaming(content string, done bool) error {
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
