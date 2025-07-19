package cmdio

import (
	"context"
	"errors"
	"io"
	"time"
)

// ResponseMode 定义响应模式类型
type ResponseMode int

const (
	// BatchMode 批量响应模式 - 一次性返回所有结果
	BatchMode ResponseMode = iota
	// StreamMode 流式响应模式 - 逐步返回结果
	StreamMode
)

// ProcessorConfig 处理器配置
type ProcessorConfig struct {
	// 响应模式：批量或流式
	Mode ResponseMode
	// 批量模式下的最大等待时间
	MaxWaitTime time.Duration
	// 流式模式下的刷新间隔
	StreamInterval time.Duration
	// 处理超时时间
	Timeout time.Duration
}

// DefaultProcessorConfig 返回默认处理器配置
func DefaultProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		Mode:           BatchMode,
		MaxWaitTime:    time.Second * 30,
		StreamInterval: time.Millisecond * 100,
		Timeout:        time.Minute * 5,
	}
}

// InteractiveProcessor 定义交互式处理器接口
type InteractiveProcessor interface {
	// ProcessInput 处理输入字符串，返回处理是否成功
	ProcessInput(ctx context.Context, input string) error
	// ProcessOutput 处理输出，将结果写入渲染器
	ProcessOutput(ctx context.Context) error
	// SetOutputWriter 设置输出写入器
	SetOutputWriter(writer io.Writer)
	// GetConfig 获取处理器配置
	GetConfig() ProcessorConfig
}

// ErrProcessingCanceled 表示处理被取消
var ErrProcessingCanceled = errors.New("processing canceled")

// ErrProcessingTimeout 表示处理超时
var ErrProcessingTimeout = errors.New("processing timeout")
