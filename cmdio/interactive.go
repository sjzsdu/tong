package cmdio

import (
	"context"
	"fmt"
	"strings"

	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/lang"
)

// Start 启动交互式会话
func (s *InteractiveSession) Start(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}

	if s.Processor == nil {
		return fmt.Errorf("processor cannot be nil")
	}

	// 设置输出写入器
	SetProcessorWriter(s.Processor, s.renderer)

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
		input, err := helper.InputString(s.prompt)
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

		// 处理输入
		if err := s.Processor.ProcessInput(ctx, input); err != nil {
			if err == context.Canceled || strings.Contains(err.Error(), "context canceled") {
				return err
			}
			errMsg := fmt.Sprintf(lang.T("Error processing input")+": %v\n", err)
			if s.renderer != nil {
				s.renderer.WriteStream(errMsg)
			} else {
				fmt.Print(errMsg)
			}
			continue
		}

		// 处理输出
		if err := s.Processor.ProcessOutput(ctx); err != nil {
			if err == context.Canceled || strings.Contains(err.Error(), "context canceled") {
				return err
			}
			errMsg := fmt.Sprintf(lang.T("Error processing output")+": %v\n", err)
			if s.renderer != nil {
				s.renderer.WriteStream(errMsg)
			} else {
				fmt.Print(errMsg)
			}
			continue
		}

		// 完成输出
		s.renderer.Done()
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
