package cmdio_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/sjzsdu/tong/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// 模拟渲染器
type MockRenderer struct {
	mock.Mock
}

func (m *MockRenderer) WriteStream(content string) error {
	args := m.Called(content)
	return args.Error(0)
}

func (m *MockRenderer) Done() {
	m.Called()
}

// 测试 InteractiveSession 的创建和配置
func TestNewInteractiveSession(t *testing.T) {
	// 创建模拟处理器
	mockProc := new(MockProcessor)

	// 创建模拟渲染器
	mockRenderer := new(MockRenderer)

	t.Run("DefaultOptions", func(t *testing.T) {
		// 使用默认选项创建会话
		session := cmdio.NewInteractiveSession(mockProc)

		// 验证会话不为空
		assert.NotNil(t, session, "会话不应为空")
	})

	t.Run("CustomOptions", func(t *testing.T) {
		// 使用自定义选项创建会话
		session := cmdio.NewInteractiveSession(
			mockProc,
			cmdio.WithRenderer(mockRenderer),
			cmdio.WithWelcome("欢迎使用"),
			cmdio.WithTip("输入命令"),
			cmdio.WithPrompt(">"),
			cmdio.WithExitCommands("exit", "quit"),
			cmdio.WithStream(true),
		)

		// 验证会话不为空
		assert.NotNil(t, session, "会话不应为空")
	})
}

// 测试 IsExitCommand 方法
func TestIsExitCommand(t *testing.T) {
	// 创建模拟处理器
	mockProc := new(MockProcessor)

	// 创建会话，设置退出命令为 "exit" 和 "quit"
	session := cmdio.NewInteractiveSession(
		mockProc,
		cmdio.WithExitCommands("exit", "quit"),
	)

	// 测试退出命令
	assert.True(t, session.IsExitCommand("exit"), "'exit' 应被识别为退出命令")
	assert.True(t, session.IsExitCommand("quit"), "'quit' 应被识别为退出命令")
	assert.False(t, session.IsExitCommand("hello"), "'hello' 不应被识别为退出命令")
}

// 模拟 InputString 函数，用于测试 Start 方法
type mockInputStringFunc struct {
	inputs []string
	errors []error
	index  int
}

// 创建一个模拟的输入函数
func createMockInputString(inputs []string, errs []error) func(string) (string, error) {
	mock := &mockInputStringFunc{
		inputs: inputs,
		errors: errs,
		index:  0,
	}

	return func(prompt string) (string, error) {
		if mock.index >= len(mock.inputs) {
			return "", fmt.Errorf("no more inputs")
		}

		input := mock.inputs[mock.index]
		err := mock.errors[mock.index]
		mock.index++

		return input, err
	}
}

// 创建一个模拟的加载动画函数
func createMockShowLoadingAnimation() func(chan bool) {
	return func(done chan bool) {
		// 简单实现，立即返回
		go func() {
			<-done
			done <- false
		}()
	}
}

// 测试 Start 方法（非流式模式）
func TestStart_NonStream(t *testing.T) {
	// 创建模拟处理器
	mockProc := new(MockProcessor)

	// 创建模拟渲染器
	mockRenderer := new(MockRenderer)

	// 设置模拟处理器的行为
	mockProc.On("ProcessInput", mock.Anything, "hello").Return("Hello, World!", nil)

	// 设置模拟渲染器的行为
	mockRenderer.On("WriteStream", "Hello, World!").Return(nil)
	mockRenderer.On("Done").Return()
	// 添加对退出消息的期望
	mockRenderer.On("WriteStream", "Chat session terminated, thanks for using!\n").Return(nil)

	// 创建模拟函数
	mockInputFunc := createMockInputString([]string{"hello", "exit"}, []error{nil, nil})
	mockLoadingFunc := createMockShowLoadingAnimation()

	// 创建会话，注入模拟函数
	session := cmdio.NewInteractiveSession(
		mockProc,
		cmdio.WithRenderer(mockRenderer),
		cmdio.WithExitCommands("exit"),
		cmdio.WithInputStringFunc(mockInputFunc),
		cmdio.WithShowLoadingAnimationFunc(mockLoadingFunc),
	)

	// 启动会话
	session.Start(context.Background())

	// 验证模拟对象的调用
	mockProc.AssertExpectations(t)
	mockRenderer.AssertExpectations(t)
}

// 测试 Start 方法（流式模式）
func TestStart_Stream(t *testing.T) {
	// 创建模拟处理器
	mockProc := new(MockProcessor)

	// 创建模拟渲染器
	mockRenderer := new(MockRenderer)

	// 用于捕获回调函数的变量
	var capturedCallback func(content string, done bool)

	// 设置模拟处理器的流式处理行为
	mockProc.On("ProcessInputStream", mock.Anything, "hello", mock.AnythingOfType("func(string, bool)")).Run(func(args mock.Arguments) {
		capturedCallback = args.Get(2).(func(string, bool))
	}).Return(func() {
		// 模拟流式输出
		if capturedCallback != nil {
			capturedCallback("Hello", false)
			capturedCallback(", World!", true)
		}
	}, nil)

	// 设置模拟渲染器的行为
	mockRenderer.On("WriteStream", "Hello").Return(nil)
	mockRenderer.On("WriteStream", ", World!").Return(nil)
	mockRenderer.On("Done").Return()
	// 添加对退出消息的期望
	mockRenderer.On("WriteStream", "Chat session terminated, thanks for using!\n").Return(nil)

	// 创建模拟函数
	mockInputFunc := createMockInputString([]string{"hello", "exit"}, []error{nil, nil})
	mockLoadingFunc := createMockShowLoadingAnimation()

	// 创建会话（流式模式）
	session := cmdio.NewInteractiveSession(
		mockProc,
		cmdio.WithRenderer(mockRenderer),
		cmdio.WithExitCommands("exit"),
		cmdio.WithStream(true),
		cmdio.WithInputStringFunc(mockInputFunc),
		cmdio.WithShowLoadingAnimationFunc(mockLoadingFunc),
	)

	// 启动会话
	session.Start(context.Background())

	// 验证模拟对象的调用
	mockProc.AssertExpectations(t)
	mockRenderer.AssertExpectations(t)
}

// 测试处理错误的情况
func TestStart_Error(t *testing.T) {
	// 创建模拟处理器
	mockProc := new(MockProcessor)

	// 创建模拟渲染器
	mockRenderer := new(MockRenderer)

	// 设置模拟处理器返回错误
	mockProc.On("ProcessInput", mock.Anything, "error").Return("", fmt.Errorf("处理错误"))

	// 设置模拟渲染器的行为（错误情况下的调用）
	// 使用具体的错误消息而不是 AnythingOfType
	mockRenderer.On("WriteStream", "Error processing input: 处理错误\n").Return(nil)
	// 添加对退出消息的期望
	mockRenderer.On("WriteStream", "Chat session terminated, thanks for using!\n").Return(nil)
	mockRenderer.On("Done").Return()

	// 创建模拟函数
	mockInputFunc := createMockInputString([]string{"error", "exit"}, []error{nil, nil})
	mockLoadingFunc := createMockShowLoadingAnimation()

	// 创建会话
	session := cmdio.NewInteractiveSession(
		mockProc,
		cmdio.WithRenderer(mockRenderer),
		cmdio.WithExitCommands("exit"),
		cmdio.WithInputStringFunc(mockInputFunc),
		cmdio.WithShowLoadingAnimationFunc(mockLoadingFunc),
	)

	// 启动会话
	session.Start(context.Background())

	// 验证模拟对象的调用
	mockProc.AssertExpectations(t)
	mockRenderer.AssertExpectations(t)
}