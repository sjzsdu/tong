package cmdio_test

import (
	"context"
	"testing"

	"github.com/sjzsdu/tong/cmdio"
	"github.com/stretchr/testify/assert"
)

// TestBaseProcessor 测试基础处理器的功能
func TestBaseProcessor(t *testing.T) {
	// 创建测试上下文
	ctx := context.Background()

	// 创建基础处理器
	processor := cmdio.NewBaseProcessor()

	t.Run("ProcessInput", func(t *testing.T) {
		// 测试输入处理
		input := "test input"
		output, err := processor.ProcessInput(ctx, input)

		// 验证结果
		assert.NoError(t, err, "处理输入不应返回错误")
		assert.Equal(t, input, output, "基础处理器应原样返回输入")
	})

	t.Run("ProcessInputStream", func(t *testing.T) {
		// 测试流式输入处理
		input := "test stream input"
		var receivedContent string
		var receivedDone bool

		// 定义回调函数
		callback := func(content string, done bool) {
			receivedContent = content
			receivedDone = done
		}

		// 调用流式处理方法
		err := processor.ProcessInputStream(ctx, input, callback)

		// 验证结果
		assert.NoError(t, err, "流式处理输入不应返回错误")
		assert.Equal(t, input, receivedContent, "基础处理器应原样返回输入")
		assert.True(t, receivedDone, "基础处理器应标记处理完成")
	})
}