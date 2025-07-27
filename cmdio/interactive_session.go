package cmdio

import (
	"context"
	"fmt"
	"strings"

	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/helper/renders"
	"github.com/sjzsdu/tong/lang"
)

// InteractiveSession 交互式会话结构体
type InteractiveSession struct {
	Processor                InteractiveProcessor     // 处理器
	renderer                 renders.Renderer         // 渲染器
	welcome                  string                   // 欢迎信息
	stream                   bool                     // 是否流失输出
	tips                     []string                 // 提示信息
	prompt                   string                   // 命令提示符
	exitCommands             []string                 // 退出命令列表
	inputStringFunc          InputStringFunc          // 输入函数，用于依赖注入和测试
	showLoadingAnimationFunc ShowLoadingAnimationFunc // 加载动画函数，用于依赖注入和测试
}

// SessionOption 会话选项函数类型
type SessionOption func(*InteractiveSession)

// NewInteractiveSession 创建新的交互式会话
func NewInteractiveSession(processor InteractiveProcessor, opts ...SessionOption) *InteractiveSession {
	// 默认选项
	options := &InteractiveSession{
		Processor:                processor,
		renderer:                 helper.GetDefaultRenderer(),
		welcome:                  "",
		tips:                     []string{},
		prompt:                   "> ",
		exitCommands:             []string{"quit", "q", "exit"},
		inputStringFunc:          helper.InputString,
		showLoadingAnimationFunc: helper.ShowLoadingAnimation,
	}

	// 应用函数式选项
	for _, opt := range opts {
		opt(options)
	}

	return options
}

// Start 启动交互式会话
func (s *InteractiveSession) Start(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}

	if s.Processor == nil {
		return fmt.Errorf("processor cannot be nil")
	}

	// 显示欢迎信息
	if s.welcome != "" {
		s.renderer.WriteStream(s.welcome + "\n")
	}

	// 显示提示信息
	for _, tip := range s.tips {
		s.renderer.WriteStream(tip + "\n")
	}

	// 主循环
	for {
		// 获取用户输入
		input, err := s.inputStringFunc(s.prompt)
		if err != nil {
			fmt.Printf(lang.T("Error reading input")+": %v\n", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// 检查退出命令
		if s.IsExitCommand(input) {
			msg := lang.T("Chat session terminated, thanks for using!")
			s.renderer.WriteStream(msg + "\n")
			return nil
		}

		responseStarted := false
		loadingDone := make(chan bool, 1)
		go s.showLoadingAnimationFunc(loadingDone)

		if s.stream {
			// 使用流式处理
			err := s.Processor.ProcessInputStream(ctx, input, func(content string, done bool) {
				if !responseStarted {
					loadingDone <- true
					responseStarted = true
					<-loadingDone
				}
				s.renderer.WriteStream(content)
				if done {
					s.renderer.Done()
				}
			})

			if !responseStarted {
				loadingDone <- true
				<-loadingDone // 确保加载动画完全结束
			}

			if err != nil {
				if err == context.Canceled || strings.Contains(err.Error(), "context canceled") {
					return err
				}
				errMsg := fmt.Sprintf(lang.T("Error processing input")+": %v\n", err)
				s.renderer.WriteStream(errMsg)
				s.renderer.Done()
				continue
			}
		} else {
			content, err := s.Processor.ProcessInput(ctx, input)
			if err != nil {
				if err == context.Canceled || strings.Contains(err.Error(), "context canceled") {
					return err
				}
				errMsg := fmt.Sprintf(lang.T("Error processing input")+": %v\n", err)
				// 通知加载动画结束
				loadingDone <- true
				<-loadingDone
				s.renderer.WriteStream(errMsg)
				s.renderer.Done()
				continue
			}
			// 通知加载动画结束
			loadingDone <- true
			<-loadingDone
			// 写入内容，不添加换行符
			s.renderer.WriteStream(content)
			s.renderer.Done()
		}
	}
}

// IsExitCommand 检查输入是否为退出命令，不区分大小写
func (s *InteractiveSession) IsExitCommand(input string) bool {
	input = strings.TrimSpace(strings.ToLower(input))
	for _, cmd := range s.exitCommands {
		if input == strings.TrimSpace(strings.ToLower(cmd)) {
			return true
		}
	}
	return false
}

// Renderer 返回渲染器，用于测试
func (s *InteractiveSession) Renderer() renders.Renderer {
	return s.renderer
}
