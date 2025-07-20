package cmdio

import (
	"context"
	"fmt"
	"strings"

	"github.com/sjzsdu/tong/helper/renders"
)

// MockInteractiveSession 是一个用于测试的交互式会话实现
// 它不依赖于实际的终端输入，而是使用预定义的输入
type MockInteractiveSession struct {
	Processor    InteractiveProcessor
	renderer     renders.Renderer
	welcome      string
	tips         []string
	exitCommands []string
	// 预定义的输入和预期的输出，用于测试
	mockInputs []string
}

// NewMockInteractiveSession 创建一个新的模拟交互式会话
func NewMockInteractiveSession(processor InteractiveProcessor, mockInputs []string, opts ...SessionOption) *MockInteractiveSession {
	// 默认选项
	options := &MockInteractiveSession{
		Processor:    processor,
		renderer:     &MockRenderer{},
		welcome:      "",
		tips:         []string{},
		exitCommands: []string{"quit", "q", "exit"},
		mockInputs:   mockInputs,
	}

	// 应用函数式选项
	for _, opt := range opts {
		applyMockOption(opt, options)
	}

	return options
}

// 将标准会话选项应用到模拟会话
func applyMockOption(opt SessionOption, mock *MockInteractiveSession) {
	// 创建一个临时的标准会话来应用选项
	temp := &InteractiveSession{
		Processor:    mock.Processor,
		renderer:     mock.renderer,
		welcome:      mock.welcome,
		tips:         mock.tips,
		exitCommands: mock.exitCommands,
	}

	// 应用选项到临时会话
	opt(temp)

	// 将更新后的值复制回模拟会话
	mock.Processor = temp.Processor
	mock.renderer = temp.renderer
	mock.welcome = temp.welcome
	mock.tips = temp.tips
	mock.exitCommands = temp.exitCommands
}

// Start 启动模拟交互式会话
func (s *MockInteractiveSession) Start(ctx context.Context) error {
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

	// 处理模拟输入
	for _, input := range s.mockInputs {
		// 检查退出命令
		if s.isExitCommand(input) {
			// 在测试环境中，不输出退出消息
			return nil
		}

		// 处理输入
		if err := s.Processor.ProcessInput(ctx, input); err != nil {
			if err == context.Canceled || strings.Contains(err.Error(), "context canceled") {
				return err
			}
			s.renderer.WriteStream(fmt.Sprintf("Error processing input: %v\n", err))
			continue
		}

		// 处理输出
		if err := s.Processor.ProcessOutput(ctx); err != nil {
			if err == context.Canceled || strings.Contains(err.Error(), "context canceled") {
				return err
			}
			s.renderer.WriteStream(fmt.Sprintf("Error processing output: %v\n", err))
			continue
		}

		// 完成输出
		s.renderer.Done()
	}

	return nil
}

// isExitCommand 检查输入是否为退出命令，不区分大小写
func (s *MockInteractiveSession) isExitCommand(input string) bool {
	input = strings.TrimSpace(strings.ToLower(input))
	for _, cmd := range s.exitCommands {
		if input == strings.TrimSpace(strings.ToLower(cmd)) {
			return true
		}
	}
	return false
}

// MockRenderer 是一个用于测试的渲染器实现
type MockRenderer struct {
	Output strings.Builder
}

// WriteStream 实现 Renderer 接口
func (r *MockRenderer) WriteStream(content string) error {
	r.Output.WriteString(content)
	// 同时输出到标准输出，以便测试可以捕获输出
	fmt.Print(content)
	return nil
}

// Done 实现 Renderer 接口
func (r *MockRenderer) Done() {
	// 在测试环境中不需要做任何特殊处理
}
