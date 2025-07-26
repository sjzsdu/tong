package mcp

import (
	"context"
	"fmt"
)

// MCPToolAdapter 是一个适配器，将 mcp.Tool 转换为 tools.Tool 接口
type MCPToolAdapter struct {
	ToolName        string
	ToolDescription string
	Client          *Client
	ClientName      string
}

// Name 返回工具名称
func (t *MCPToolAdapter) Name() string {
	return t.ToolName
}

// Description 返回工具描述
func (t *MCPToolAdapter) Description() string {
	return t.ToolDescription
}

// Call 执行工具调用
func (t *MCPToolAdapter) Call(ctx context.Context, input string) (string, error) {
	// 创建工具调用请求
	args := map[string]interface{}{
		"input": input,
	}
	// 创建工具调用请求
	req := NewToolCallRequest(t.ToolName, args)

	// 调用工具
	result, err := t.Client.CallTool(ctx, req)
	if err != nil {
		return "", fmt.Errorf("调用工具 %s 失败: %v", t.ToolName, err)
	}

	// 返回结果
	return fmt.Sprintf("%v", result.Result), nil
}

// GetTools 方法已移至 host.go 文件中实现
