package cmdio

import (
	"context"
	"fmt"
	"log"

	"github.com/sjzsdu/langchaingo-cn/llms"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/memory"
)

// ExampleChainProcessor 展示如何使用批量处理器
func ExampleChainProcessor() {
	// 初始化 LLM
	llm, err := llms.CreateLLM(llms.DeepSeekLLM, nil)
	if err != nil {
		log.Fatal(err)
	}

	// 创建对话记忆
	chatMemory := memory.NewConversationBuffer()

	// 创建对话链
	chain := chains.NewConversation(llm, chatMemory)

	// 创建批量处理器
	config := DefaultProcessorConfig()
	config.Mode = BatchMode
	processor := NewChainProcessor(chain, config)

	// 创建模拟交互式会话，使用预定义的输入
	mockInputs := []string{"quit"} // 只有一个退出命令，避免实际调用 LLM
	session := NewMockInteractiveSession(
		processor,
		mockInputs,
		WithWelcome("欢迎使用批量处理模式的 AI 聊天助手！输入 'quit' 退出。"),
		WithTip("提示：在批量模式下，AI 会等待处理完成后一次性返回所有结果。"),
		WithPrompt("您: "),
		WithExitCommands("quit", "exit", "q", "退出"),
	)

	// 启动交互式会话
	ctx := context.Background()
	err = session.Start(ctx)
	if err != nil {
		fmt.Printf("会话错误: %v\n", err)
	}

	// Output:
	// 欢迎使用批量处理模式的 AI 聊天助手！输入 'quit' 退出。
	// 提示：在批量模式下，AI 会等待处理完成后一次性返回所有结果。
}

// ExampleNewStreamChainProcessor 展示如何使用流式处理器
func ExampleNewStreamChainProcessor() {
	// 初始化 LLM
	llm, err := llms.CreateLLM(llms.DeepSeekLLM, nil)
	if err != nil {
		log.Fatal(err)
	}

	// 创建对话记忆
	chatMemory := memory.NewConversationBuffer()

	// 创建对话链
	chain := chains.NewConversation(llm, chatMemory)

	// 创建流式处理器
	config := DefaultProcessorConfig()
	config.Mode = StreamMode
	processor := NewStreamChainProcessor(chain, config)

	// 创建模拟交互式会话，使用预定义的输入
	mockInputs := []string{"quit"} // 只有一个退出命令，避免实际调用 LLM
	session := NewMockInteractiveSession(
		processor,
		mockInputs,
		WithWelcome("欢迎使用流式处理模式的 AI 聊天助手！输入 'quit' 退出。"),
		WithTip("提示：在流式模式下，AI 会在处理过程中不断返回部分结果。"),
		WithPrompt("您: "),
		WithExitCommands("quit", "exit", "q", "退出"),
	)

	// 启动交互式会话
	ctx := context.Background()
	err = session.Start(ctx)
	if err != nil {
		fmt.Printf("会话错误: %v\n", err)
	}

	// Output:
	// 欢迎使用流式处理模式的 AI 聊天助手！输入 'quit' 退出。
	// 提示：在流式模式下，AI 会在处理过程中不断返回部分结果。
}

// ExampleCreateChatAdapter 展示如何使用聊天适配器
func ExampleCreateChatAdapter() {
	// 初始化 LLM
	llm, err := llms.CreateLLM(llms.DeepSeekLLM, nil)
	if err != nil {
		log.Fatal(err)
	}

	// 创建对话记忆
	chatMemory := memory.NewConversationBuffer()

	// 创建对话链
	chain := chains.NewConversation(llm, chatMemory)

	// 获取交互式会话适配器（流式模式）
	originalSession := CreateChatAdapter(chain, true)
	
	// 从原始会话中获取处理器，创建模拟会话
	mockInputs := []string{"quit"} // 只有一个退出命令，避免实际调用 LLM
	session := NewMockInteractiveSession(
		originalSession.Processor,
		mockInputs,
		WithWelcome("欢迎使用 AI 聊天助手！输入 'quit' 退出。"),
		WithTip("提示：您可以询问任何问题，AI 将尽力回答。"),
		WithPrompt("您: "),
		WithExitCommands("quit", "exit", "q", "退出"),
	)

	// 启动交互式会话
	ctx := context.Background()
	err = session.Start(ctx)
	if err != nil {
		fmt.Printf("会话错误: %v\n", err)
	}

	// Output:
	// 欢迎使用 AI 聊天助手！输入 'quit' 退出。
	// 提示：您可以询问任何问题，AI 将尽力回答。
}