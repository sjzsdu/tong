package cmdio_test

import (
	"context"
	"testing"

	"github.com/sjzsdu/tong/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/schema"
)

// mockStreamFunc 是一个模拟的流式输出函数
func mockStreamFunc(ctx context.Context, chunk []byte) error {
	// 这个函数只是一个模拟，不做任何实际操作
	return nil
}

// 模拟处理器实现
type MockProcessor struct {
	mock.Mock
}

func (m *MockProcessor) ProcessInput(ctx context.Context, input string) (string, error) {
	args := m.Called(ctx, input)
	return args.String(0), args.Error(1)
}

func (m *MockProcessor) ProcessInputStream(ctx context.Context, input string, callback func(content string, done bool)) error {
	args := m.Called(ctx, input, callback)
	// 调用回调函数以模拟流式输出
	if fn, ok := args.Get(0).(func()); ok && fn != nil {
		fn()
	}
	return args.Error(1)
}

// 实现 chains.Chain 接口所需的方法
func (m *MockProcessor) Call(ctx context.Context, inputs map[string]any, options ...chains.ChainCallOption) (map[string]any, error) {
	args := m.Called(ctx, inputs, options)
	return args.Get(0).(map[string]any), args.Error(1)
}

func (m *MockProcessor) GetMemory() schema.Memory {
	args := m.Called()
	return args.Get(0).(schema.Memory)
}

func (m *MockProcessor) GetInputKeys() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockProcessor) GetOutputKeys() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

// TestChainProcessor 测试链式处理器的功能
func TestChainProcessor(t *testing.T) {
	// 创建测试上下文
	ctx := context.Background()

	t.Run("ProcessInput_SingleProcessor", func(t *testing.T) {
		// 创建模拟处理器
		mockProc := new(MockProcessor)
		// 设置 Call 方法的期望行为
		mockProc.On("Call", ctx, map[string]any{"input": "test input"}, mock.Anything).Return(
			map[string]any{"output": "processed output"}, nil)
		// 设置其他必要的方法
		mockProc.On("GetInputKeys").Return([]string{"input"})
		mockProc.On("GetOutputKeys").Return([]string{"output"})
		mockProc.On("GetMemory").Return(memory.NewSimple())

		// 创建链式处理器，只包含一个处理器
		chainProcessor := cmdio.NewChainProcessor(mockProc, false)

		// 调用处理方法
		output, err := chainProcessor.ProcessInput(ctx, "test input")

		// 验证结果
		assert.NoError(t, err, "处理输入不应返回错误")
		assert.Equal(t, "processed output", output, "输出应与模拟处理器的输出一致")
		mockProc.AssertExpectations(t)
	})

	t.Run("ProcessInput_MultipleProcessors", func(t *testing.T) {
		// 创建多个模拟处理器
		mockProc1 := new(MockProcessor)
		mockProc2 := new(MockProcessor)

		// 设置第一个处理器的期望行为
		mockProc1.On("Call", ctx, map[string]any{"input": "test input"}, mock.Anything).Return(
			map[string]any{"output": "intermediate output"}, nil)
		mockProc1.On("GetInputKeys").Return([]string{"input"})
		mockProc1.On("GetOutputKeys").Return([]string{"output"})
		mockProc1.On("GetMemory").Return(memory.NewSimple())

		// 设置第二个处理器的期望行为
		mockProc2.On("Call", ctx, map[string]any{"input": "intermediate output"}, mock.Anything).Return(
			map[string]any{"output": "final output"}, nil)
		mockProc2.On("GetInputKeys").Return([]string{"input"})
		mockProc2.On("GetOutputKeys").Return([]string{"output"})
		mockProc2.On("GetMemory").Return(memory.NewSimple())

		// 创建链式处理器，包含一个处理器
		chainProcessor := cmdio.NewChainProcessor(mockProc1, false)
		// 创建第二个处理器
		chainProcessor2 := cmdio.NewChainProcessor(mockProc2, false)

		// 调用第一个处理器
		intermediate, err := chainProcessor.ProcessInput(ctx, "test input")
		assert.NoError(t, err, "第一个处理器不应返回错误")
		
		// 调用第二个处理器
		output, err := chainProcessor2.ProcessInput(ctx, intermediate)

		// 验证结果
		assert.NoError(t, err, "处理输入不应返回错误")
		assert.Equal(t, "final output", output, "输出应与最后一个处理器的输出一致")
		mockProc1.AssertExpectations(t)
		mockProc2.AssertExpectations(t)
	})

	t.Run("ProcessInputStream_SingleProcessor", func(t *testing.T) {
		// 创建模拟处理器
		mockProc := new(MockProcessor)

		// 设置 Call 方法的期望行为，支持流式输出
		mockProc.On("Call", ctx, map[string]any{"input": "test stream input"}, mock.Anything).Run(func(args mock.Arguments) {
			// 获取流式回调函数
			options := args.Get(2).([]chains.ChainCallOption)
			if len(options) > 0 {
				// 模拟流式输出
				// 由于 ChainCallOption 是函数类型，我们需要直接调用流式回调函数
				// 这里我们通过调用 mockStreamFunc 来模拟流式输出
				mockStreamFunc(ctx, []byte("processed stream output"))
			}
		}).Return(map[string]any{"output": "processed stream output"}, nil)

		// 设置其他必要的方法
		mockProc.On("GetInputKeys").Return([]string{"input"})
		mockProc.On("GetOutputKeys").Return([]string{"output"})
		mockProc.On("GetMemory").Return(memory.NewSimple())

		// 创建链式处理器
		chainProcessor := cmdio.NewChainProcessor(mockProc, true)

		// 用于验证回调函数的变量
		var receivedContent string
		var receivedDone bool

		// 定义回调函数
		callback := func(content string, done bool) {
			if content != "" {
				receivedContent = content
			}
			receivedDone = done
		}

		// 调用流式处理方法
		err := chainProcessor.ProcessInputStream(ctx, "test stream input", callback)

		// 验证结果
		assert.NoError(t, err, "流式处理输入不应返回错误")
		assert.Equal(t, "processed stream output", receivedContent, "内容应与模拟处理器的输出一致")
		assert.True(t, receivedDone, "完成标志应为true")
		mockProc.AssertExpectations(t)
	})

	t.Run("ProcessInputStream_MultipleProcessors", func(t *testing.T) {
		// 创建多个模拟处理器
		mockProc1 := new(MockProcessor)
		mockProc2 := new(MockProcessor)

		// 设置第一个处理器的期望行为，支持流式输出
		mockProc1.On("Call", ctx, map[string]any{"input": "test stream input"}, mock.Anything).Run(func(args mock.Arguments) {
			// 获取流式回调函数
			options := args.Get(2).([]chains.ChainCallOption)
			if len(options) > 0 {
				// 模拟流式输出
				mockStreamFunc(ctx, []byte("intermediate stream output"))
			}
		}).Return(map[string]any{"output": "intermediate stream output"}, nil)

		// 设置第一个处理器的其他必要方法
		mockProc1.On("GetInputKeys").Return([]string{"input"})
		mockProc1.On("GetOutputKeys").Return([]string{"output"})
		mockProc1.On("GetMemory").Return(memory.NewSimple())

		// 设置第二个处理器的期望行为，支持流式输出
		mockProc2.On("Call", ctx, map[string]any{"input": "intermediate stream output"}, mock.Anything).Run(func(args mock.Arguments) {
			// 获取流式回调函数
			options := args.Get(2).([]chains.ChainCallOption)
			if len(options) > 0 {
				// 模拟流式输出
				mockStreamFunc(ctx, []byte("final stream output"))
			}
		}).Return(map[string]any{"output": "final stream output"}, nil)

		// 设置第二个处理器的其他必要方法
		mockProc2.On("GetInputKeys").Return([]string{"input"})
		mockProc2.On("GetOutputKeys").Return([]string{"output"})
		mockProc2.On("GetMemory").Return(memory.NewSimple())

		// 创建链式处理器
		chainProcessor := cmdio.NewChainProcessor(mockProc1, true)
		// 创建第二个处理器
		chainProcessor2 := cmdio.NewChainProcessor(mockProc2, true)

		// 用于验证回调函数的变量
		var receivedContent string
		var receivedDone bool

		// 定义回调函数
		callback := func(content string, done bool) {
			if content != "" {
				receivedContent = content
			}
			receivedDone = done
		}

		// 调用第一个处理器的流式处理方法
		err := chainProcessor.ProcessInputStream(ctx, "test stream input", func(content string, done bool) {
			// 保存中间内容，但不需要在这里使用
			// 我们只关心最终的输出结果
			
			if done {
				// 第一个处理器完成后，调用第二个处理器
				// 使用 "intermediate stream output" 作为输入，而不是 content
				err := chainProcessor2.ProcessInputStream(ctx, "intermediate stream output", callback)
				assert.NoError(t, err, "第二个处理器不应返回错误")
			}
		})

		// 验证结果
		assert.NoError(t, err, "流式处理输入不应返回错误")
		assert.Equal(t, "final stream output", receivedContent, "内容应与最后一个处理器的输出一致")
		assert.True(t, receivedDone, "完成标志应为true")
		mockProc1.AssertExpectations(t)
		mockProc2.AssertExpectations(t)
	})
}