package mcp

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sjzsdu/tong/share"
	"github.com/stretchr/testify/assert"
)

func TestNewInitializeRequest(t *testing.T) {
	// 调用函数创建请求
	req := NewInitializeRequest()

	// 验证基本字段
	assert.Equal(t, string(mcp.MethodInitialize), req.Method)
	assert.Equal(t, mcp.LATEST_PROTOCOL_VERSION, req.Params.ProtocolVersion)
	assert.Equal(t, share.MCP_CLIENT_NAME, req.Params.ClientInfo.Name)
	assert.Equal(t, share.VERSION, req.Params.ClientInfo.Version)
}

func TestNewReadResourceRequest(t *testing.T) {
	// 测试参数
	uri := "test-uri"
	args := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	// 调用函数创建请求
	req := NewReadResourceRequest(uri, args)

	// 验证基本字段
	assert.Equal(t, string(mcp.MethodResourcesRead), req.Method)
	assert.Equal(t, uri, req.Params.URI)
	assert.Equal(t, args, req.Params.Arguments)
}

func TestNewPromptRequest(t *testing.T) {
	// 测试参数
	name := "test-prompt"
	args := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	// 调用函数创建请求
	req := NewPromptRequest(name, args)

	// 验证基本字段
	assert.Equal(t, string(mcp.MethodPromptsGet), req.Method)
	assert.Equal(t, name, req.Params.Name)
	assert.Equal(t, args, req.Params.Arguments)
}
