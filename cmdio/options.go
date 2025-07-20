package cmdio

import (
	"github.com/sjzsdu/tong/helper/renders"
)

// 函数类型定义，用于依赖注入
type InputStringFunc func(string) (string, error)
type ShowLoadingAnimationFunc func(chan bool)

// 这个文件包含 InteractiveSession 的选项函数

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
func WithStream(stream bool) SessionOption {
	return func(opts *InteractiveSession) {
		opts.stream = stream
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

// WithInputStringFunc 注入自定义的输入函数，用于测试
func WithInputStringFunc(fn InputStringFunc) SessionOption {
	return func(opts *InteractiveSession) {
		opts.inputStringFunc = fn
	}
}

// WithShowLoadingAnimationFunc 注入自定义的加载动画函数，用于测试
func WithShowLoadingAnimationFunc(fn ShowLoadingAnimationFunc) SessionOption {
	return func(opts *InteractiveSession) {
		opts.showLoadingAnimationFunc = fn
	}
}
