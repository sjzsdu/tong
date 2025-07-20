package processors

import (
	"context"
	"errors"
	"io"
	"time"
)

// ResponseMode 响应模式类型
type ResponseMode int

const (
	// BatchMode 批量处理模式，一次性返回所有结果
	BatchMode ResponseMode = iota
	// StreamMode 流式处理模式，逐步返回结果
	StreamMode
)

// ProcessorConfig 处理器配置
type ProcessorConfig struct {
	// Mode 处理模式：批量或流式
	Mode ResponseMode
	// Timeout 处理超时时间
	Timeout time.Duration
	// MaxTokens 最大令牌数
	MaxTokens int
	// Temperature 温度参数
	Temperature float64
	// StreamInterval 流式处理间隔
	StreamInterval time.Duration
	// MaxWaitTime 最大等待时间
	MaxWaitTime time.Duration
}

// defaultProcessorConfig 默认处理器配置
var defaultProcessorConfig = &ProcessorConfig{
	Mode:           BatchMode,
	Timeout:        time.Second * 60,
	MaxTokens:      2000,
	Temperature:    0.7,
	StreamInterval: time.Millisecond * 50,
	MaxWaitTime:    time.Second * 60,
}

// DefaultProcessorConfig 创建默认处理器配置的副本
func DefaultProcessorConfig() ProcessorConfig {
	return *defaultProcessorConfig
}

// InteractiveProcessor 交互式处理器接口
type InteractiveProcessor interface {
	// ProcessInput 处理输入
	ProcessInput(ctx context.Context, input string) error
	// ProcessOutput 处理输出
	ProcessOutput(ctx context.Context) error
	// SetOutputWriter 设置输出写入器
	SetOutputWriter(writer io.Writer)
	// GetConfig 获取处理器配置
	GetConfig() ProcessorConfig
	// SetConfig 设置处理器配置
	SetConfig(config ProcessorConfig)
}

// ProcessFunc 处理函数类型
type ProcessFunc func(ctx context.Context, input string) (string, error)

// CustomStreamProcessFunc 自定义流式处理函数类型
type CustomStreamProcessFunc func(ctx context.Context, input string, writer io.Writer) error

// DelayDuration 延迟时间类型
type DelayDuration time.Duration

// 错误定义
var (
	// ErrProcessingCanceled 处理被取消错误
	ErrProcessingCanceled = errors.New("processing canceled")
	// ErrProcessingTimeout 处理超时错误
	ErrProcessingTimeout = errors.New("processing timeout")
)
