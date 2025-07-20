package cmdio_test

import (
	"reflect"
	"testing"

	"github.com/sjzsdu/tong/cmdio"
	"github.com/stretchr/testify/assert"
)

// 测试 WithRenderer 选项
func TestWithRenderer(t *testing.T) {
	// 创建模拟处理器
	mockProc := new(MockProcessor)

	// 创建模拟渲染器
	mockRenderer := new(MockRenderer)

	// 设置模拟渲染器的期望
	mockRenderer.On("WriteStream", "test").Return(nil)

	// 创建会话，使用 WithRenderer 选项
	session := cmdio.NewInteractiveSession(
		mockProc,
		cmdio.WithRenderer(mockRenderer),
	)

	// 使用反射获取私有字段的值，仅验证字段存在
	rendererField := reflect.ValueOf(session).Elem().FieldByName("renderer")
	assert.True(t, rendererField.IsValid(), "renderer 字段应存在")
	
	// 由于无法直接访问私有字段的值，我们通过行为验证渲染器是否正确设置
	// 调用 WriteStream 方法，如果渲染器设置正确，mockRenderer 的 WriteStream 方法应该被调用
	session.Renderer().WriteStream("test")
	
	// 验证 mockRenderer 的方法被调用
	mockRenderer.AssertExpectations(t)
}

// 测试 WithWelcome 选项
func TestWithWelcome(t *testing.T) {
	// 创建模拟处理器
	mockProc := new(MockProcessor)

	// 欢迎信息
	welcomeMsg := "欢迎使用测试系统"

	// 创建会话，使用 WithWelcome 选项
	session := cmdio.NewInteractiveSession(
		mockProc,
		cmdio.WithWelcome(welcomeMsg),
	)

	// 使用反射获取私有字段的值
	welcomeField := reflect.ValueOf(session).Elem().FieldByName("welcome")
	assert.True(t, welcomeField.IsValid(), "welcome 字段应存在")
	assert.Equal(t, welcomeMsg, welcomeField.String(), "欢迎信息应正确设置")
}

// 测试 WithTip 选项
func TestWithTip(t *testing.T) {
	// 创建模拟处理器
	mockProc := new(MockProcessor)

	// 提示信息
	tipMsg := "输入命令开始交互"

	// 创建会话，使用 WithTip 选项
	session := cmdio.NewInteractiveSession(
		mockProc,
		cmdio.WithTip(tipMsg),
	)

	// 使用反射获取私有字段的值
	tipsField := reflect.ValueOf(session).Elem().FieldByName("tips")
	assert.True(t, tipsField.IsValid(), "tips 字段应存在")
	assert.Equal(t, 1, tipsField.Len(), "应有一条提示信息")
	assert.Equal(t, tipMsg, tipsField.Index(0).String(), "提示信息应正确设置")
}

// 测试 WithTips 选项
func TestWithTips(t *testing.T) {
	// 创建模拟处理器
	mockProc := new(MockProcessor)

	// 多条提示信息
	tips := []string{"提示1", "提示2", "提示3"}

	// 创建会话，使用 WithTips 选项
	session := cmdio.NewInteractiveSession(
		mockProc,
		cmdio.WithTips(tips...),
	)

	// 使用反射获取私有字段的值
	tipsField := reflect.ValueOf(session).Elem().FieldByName("tips")
	assert.True(t, tipsField.IsValid(), "tips 字段应存在")
	assert.Equal(t, len(tips), tipsField.Len(), "提示信息数量应正确")
	
	// 验证每条提示信息
	for i, tip := range tips {
		assert.Equal(t, tip, tipsField.Index(i).String(), "提示信息应正确设置")
	}
}

// 测试 WithStream 选项
func TestWithStream(t *testing.T) {
	// 创建模拟处理器
	mockProc := new(MockProcessor)

	// 创建会话，使用 WithStream 选项
	session := cmdio.NewInteractiveSession(
		mockProc,
		cmdio.WithStream(true),
	)

	// 使用反射获取私有字段的值
	streamField := reflect.ValueOf(session).Elem().FieldByName("stream")
	assert.True(t, streamField.IsValid(), "stream 字段应存在")
	assert.True(t, streamField.Bool(), "流模式应正确设置为 true")

	// 创建会话，使用 WithStream 选项设置为 false
	session = cmdio.NewInteractiveSession(
		mockProc,
		cmdio.WithStream(false),
	)

	// 验证流模式已正确设置
	streamField = reflect.ValueOf(session).Elem().FieldByName("stream")
	assert.False(t, streamField.Bool(), "流模式应正确设置为 false")
}

// 测试 WithPrompt 选项
func TestWithPrompt(t *testing.T) {
	// 创建模拟处理器
	mockProc := new(MockProcessor)

	// 命令提示符
	prompt := ">> "

	// 创建会话，使用 WithPrompt 选项
	session := cmdio.NewInteractiveSession(
		mockProc,
		cmdio.WithPrompt(prompt),
	)

	// 使用反射获取私有字段的值
	promptField := reflect.ValueOf(session).Elem().FieldByName("prompt")
	assert.True(t, promptField.IsValid(), "prompt 字段应存在")
	assert.Equal(t, prompt, promptField.String(), "命令提示符应正确设置")
}

// 测试 WithExitCommands 选项
func TestWithExitCommands(t *testing.T) {
	// 创建模拟处理器
	mockProc := new(MockProcessor)

	// 退出命令列表
	exitCmds := []string{"bye", "goodbye", "end"}

	// 创建会话，使用 WithExitCommands 选项
	session := cmdio.NewInteractiveSession(
		mockProc,
		cmdio.WithExitCommands(exitCmds...),
	)

	// 使用反射获取私有字段的值
	exitCommandsField := reflect.ValueOf(session).Elem().FieldByName("exitCommands")
	assert.True(t, exitCommandsField.IsValid(), "exitCommands 字段应存在")
	assert.Equal(t, len(exitCmds), exitCommandsField.Len(), "退出命令数量应正确")
	
	// 将反射值转换为字符串切片
	actualCmds := make([]string, exitCommandsField.Len())
	for i := 0; i < exitCommandsField.Len(); i++ {
		actualCmds[i] = exitCommandsField.Index(i).String()
	}
	
	assert.ElementsMatch(t, exitCmds, actualCmds, "退出命令列表应正确设置")

	// 验证 IsExitCommand 方法能正确识别退出命令
	assert.True(t, session.IsExitCommand("bye"), "'bye' 应被识别为退出命令")
	assert.True(t, session.IsExitCommand("goodbye"), "'goodbye' 应被识别为退出命令")
	assert.True(t, session.IsExitCommand("end"), "'end' 应被识别为退出命令")
	assert.False(t, session.IsExitCommand("hello"), "'hello' 不应被识别为退出命令")
}