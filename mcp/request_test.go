package mcp

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

func TestNewToolCallRequest(t *testing.T) {
	// 测试不带 progressToken 的情况
	name := "test-tool"
	args := map[string]interface{}{
		"key": "value",
	}

	req := NewToolCallRequest(name, args)

	// 验证基本字段
	assert.Equal(t, string(mcp.MethodToolsCall), req.Method)
	assert.Equal(t, name, req.Params.Name)
	assert.Equal(t, args, req.Params.Arguments)
	// 验证 Meta 字段为 nil
	assert.Nil(t, req.Params.Meta)

	// 测试带 progressToken 的情况
	progressToken := mcp.ProgressToken("test-token")
	reqWithToken := NewToolCallRequest(name, args, progressToken)

	// 验证基本字段
	assert.Equal(t, string(mcp.MethodToolsCall), reqWithToken.Method)
	assert.Equal(t, name, reqWithToken.Params.Name)
	assert.Equal(t, args, reqWithToken.Params.Arguments)
	// 验证 Meta 字段不为 nil 且包含正确的 progressToken
	assert.NotNil(t, reqWithToken.Params.Meta)
	assert.Equal(t, progressToken, reqWithToken.Params.Meta.ProgressToken)
}