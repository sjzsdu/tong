package cmdio_test

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sjzsdu/tong/cmdio"
	"github.com/sjzsdu/tong/helper"
)

// ExampleBatchProcessor 展示如何使用批量处理器
func ExampleBatchProcessor() {
	// 创建一个回显处理器
	processor := cmdio.NewEchoProcessor()

	// 创建交互式会话
	session := cmdio.NewInteractiveSession(
		processor,
		cmdio.WithWelcome("欢迎使用批量处理器示例"),
		cmdio.WithTip("输入文本将被回显"),
		cmdio.WithTip("输入 'quit' 退出"),
	)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 处理信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// 启动会话
	if err := session.Start(ctx); err != nil {
		fmt.Printf("会话错误: %v\n", err)
	}
}

// ExampleStreamProcessor 展示如何使用流式处理器
func ExampleStreamProcessor() {
	// 创建一个流式回显处理器
	processor := cmdio.NewStreamEchoProcessor()

	// 创建交互式会话
	session := cmdio.NewInteractiveSession(
		processor,
		cmdio.WithWelcome("欢迎使用流式处理器示例"),
		cmdio.WithTip("输入文本将被逐字符回显"),
		cmdio.WithTip("输入 'quit' 退出"),
	)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 处理信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// 启动会话
	if err := session.Start(ctx); err != nil {
		fmt.Printf("会话错误: %v\n", err)
	}
}

// ExampleDelayedProcessor 展示如何使用延迟处理器
func ExampleDelayedProcessor() {
	// 创建一个延迟处理器
	processor := cmdio.NewDelayedProcessor(time.Second*3, cmdio.BatchMode)

	// 创建交互式会话
	session := cmdio.NewInteractiveSession(
		processor,
		cmdio.WithWelcome("欢迎使用延迟处理器示例"),
		cmdio.WithTip("输入文本将在3秒后回显"),
		cmdio.WithTip("输入 'quit' 退出"),
		cmdio.WithRenderer(helper.NewRenderer("markdown")),
	)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 处理信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// 启动会话
	if err := session.Start(ctx); err != nil {
		fmt.Printf("会话错误: %v\n", err)
	}
}