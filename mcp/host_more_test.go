package mcp

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHostPing(t *testing.T) {
	// 创建上下文
	ctx := context.Background()

	// 创建模拟客户端
	mockClient := new(MockMCPClient)

	// 设置模拟客户端的预期行为
	mockClient.On("Ping", ctx).Return(nil)

	// 创建 Host 实例
	host := &Host{
		Clients: map[string]*Client{
			"test-client": {
				conn: mockClient,
			},
		},
	}

	// 调用 Ping 方法
	err := host.Ping(ctx)

	// 验证结果
	assert.NoError(t, err)

	// 验证模拟客户端的方法被调用
	mockClient.AssertExpectations(t)
}

func TestHostListPrompts(t *testing.T) {
	// 创建上下文
	ctx := context.Background()

	// 创建模拟客户端
	mockClient := new(MockMCPClient)

	// 创建预期的 ListPromptsResult
	expectedResult := &mcp.ListPromptsResult{
		Prompts: []mcp.Prompt{
			{
				Name:        "test-prompt-1",
				Description: "测试提示1",
			},
			{
				Name:        "test-prompt-2",
				Description: "测试提示2",
			},
		},
	}

	// 设置模拟客户端的预期行为
	mockClient.On("ListPrompts", ctx, mock.Anything).Return(expectedResult, nil)

	// 创建 Host 实例
	host := &Host{
		Clients: map[string]*Client{
			"test-client": {
				conn: mockClient,
			},
		},
	}

	// 调用 ListPrompts 方法
	result, err := host.ListPrompts(ctx, mcp.ListPromptsRequest{})

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedResult, result)

	// 验证模拟客户端的方法被调用
	mockClient.AssertExpectations(t)
}

func TestHostGetPrompt(t *testing.T) {
	// 创建上下文
	ctx := context.Background()

	// 创建模拟客户端
	mockClient := new(MockMCPClient)

	// 创建预期的 GetPromptResult
	expectedResult := &mcp.GetPromptResult{
		Messages: []mcp.PromptMessage{
			{
				Role:    "system",
				Content: mcp.NewTextContent("这是一个测试提示"),
			},
		},
	}

	// 设置模拟客户端的预期行为
	mockClient.On("GetPrompt", mock.Anything, mock.Anything).Return(expectedResult, nil)

	// 创建 Host 实例
	host := &Host{
		Clients: map[string]*Client{
			"test-client": {
				conn: mockClient,
			},
		},
	}

	// 调用 GetPrompt 方法
	result, err := host.GetPrompt(ctx, mcp.GetPromptRequest{})

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedResult, result)

	// 验证模拟客户端的方法被调用
	mockClient.AssertExpectations(t)
}

func TestHostCallTool(t *testing.T) {
	// 创建上下文
	ctx := context.Background()

	// 创建模拟客户端
	mockClient := new(MockMCPClient)

	// 创建预期的 CallToolResult
	expectedResult := &mcp.CallToolResult{}
	expectedResult.Result.Meta = map[string]interface{}{
		"result": "测试结果",
	}

	// 设置模拟客户端的预期行为
	mockClient.On("CallTool", ctx, mock.Anything).Return(expectedResult, nil)

	// 创建 Host 实例
	host := &Host{
		Clients: map[string]*Client{
			"test-client": {
				conn: mockClient,
			},
		},
	}

	// 调用 CallTool 方法
	result, err := host.CallTool(ctx, mcp.CallToolRequest{})

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedResult, result)

	// 验证模拟客户端的方法被调用
	mockClient.AssertExpectations(t)
}

func TestHostGetClient(t *testing.T) {
	// 创建模拟客户端
	mockClient := new(MockMCPClient)

	// 创建 Client 实例
	client := &Client{
		conn: mockClient,
	}

	// 创建 Host 实例
	host := &Host{
		Clients: map[string]*Client{
			"test-client": client,
		},
	}

	// 测试获取存在的客户端
	result := host.GetClient("test-client")
	assert.Equal(t, client, result)

	// 测试获取不存在的客户端
	result = host.GetClient("non-existent")
	assert.Nil(t, result)

	// 测试 nil Host
	var nilHost *Host
	result = nilHost.GetClient("test-client")
	assert.Nil(t, result)
}

func TestHostGetAllClients(t *testing.T) {
	// 创建模拟客户端
	mockClient := new(MockMCPClient)

	// 创建 Client 实例
	client := &Client{
		conn: mockClient,
	}

	// 创建 Host 实例
	host := &Host{
		Clients: map[string]*Client{
			"test-client": client,
		},
	}

	// 测试获取所有客户端
	result := host.GetAllClients()
	assert.Equal(t, host.Clients, result)

	// 测试 nil Host
	var nilHost *Host
	result = nilHost.GetAllClients()
	assert.Nil(t, result)
}

func TestHostClose(t *testing.T) {
	// 创建模拟客户端
	mockClient := new(MockMCPClient)

	// 设置模拟客户端的预期行为
	mockClient.On("Close").Return(nil)

	// 创建 Host 实例
	host := &Host{
		Clients: map[string]*Client{
			"test-client": {
				conn: mockClient,
			},
		},
	}

	// 调用 Close 方法
	err := host.Close()

	// 验证结果
	assert.NoError(t, err)

	// 验证模拟客户端的方法被调用
	mockClient.AssertExpectations(t)
}
