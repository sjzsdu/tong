package cmdio

import (
	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/helper/renders"
)

// InteractiveSession 交互式会话结构体
type InteractiveSession struct {
	Processor    InteractiveProcessor // 改为公开字段，方便测试
	renderer     renders.Renderer
	welcome      string
	tips         []string
	prompt       string
	exitCommands []string
}

// SessionOption 会话选项函数类型
type SessionOption func(*InteractiveSession)

// NewInteractiveSession 创建新的交互式会话
func NewInteractiveSession(processor InteractiveProcessor, opts ...SessionOption) *InteractiveSession {
	// 默认选项
	options := &InteractiveSession{
		Processor:    processor,
		renderer:     helper.GetDefaultRenderer(),
		welcome:      "",
		tips:         []string{},
		prompt:       "> ",
		exitCommands: []string{"quit", "q", "exit"},
	}

	// 应用函数式选项
	for _, opt := range opts {
		opt(options)
	}

	return options
}

// WithRenderer 设置渲染器
func WithRenderer(renderer renders.Renderer) SessionOption {
	return func(opts *InteractiveSession) {
		opts.renderer = renderer
	}
}

// WithWelcome 设置欢迎信息
func WithWelcome(welcome string) SessionOption {
	return func(opts *InteractiveSession) {
		opts.welcome = welcome
	}
}

// WithTip 添加单个提示信息
func WithTip(tip string) SessionOption {
	return func(opts *InteractiveSession) {
		opts.tips = append(opts.tips, tip)
	}
}

// WithTips 设置多个提示信息
func WithTips(tips ...string) SessionOption {
	return func(opts *InteractiveSession) {
		opts.tips = append(opts.tips, tips...)
	}
}

// WithPrompt 设置命令提示符
func WithPrompt(prompt string) SessionOption {
	return func(opts *InteractiveSession) {
		opts.prompt = prompt
	}
}

// WithExitCommands 设置退出命令列表
func WithExitCommands(commands ...string) SessionOption {
	return func(opts *InteractiveSession) {
		opts.exitCommands = commands
	}
}
