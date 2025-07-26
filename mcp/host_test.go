package mcp

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMCPClient 是一个模拟的 MCPClient 实现
type MockMCPClient struct {
	mock.Mock
}

// ListTools 模拟 ListTools 方法
func (m *MockMCPClient) ListTools(ctx context.Context, req mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*mcp.ListToolsResult), args.Error(1)
}

// CallTool 模拟 CallTool 方法
func (m *MockMCPClient) CallTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*mcp.CallToolResult), args.Error(1)
}

// 实现其他必要的 MCPClient 接口方法
func (m *MockMCPClient) Initialize(ctx context.Context, req mcp.InitializeRequest) (*mcp.InitializeResult, error) {
	return nil, nil
}

func (m *MockMCPClient) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockMCPClient) ListResources(ctx context.Context, req mcp.ListResourcesRequest) (*mcp.ListResourcesResult, error) {
	return nil, nil
}

func (m *MockMCPClient) ListResourceTemplates(ctx context.Context, req mcp.ListResourceTemplatesRequest) (*mcp.ListResourceTemplatesResult, error) {
	return nil, nil
}

func (m *MockMCPClient) ReadResource(ctx context.Context, req mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	return nil, nil
}

func (m *MockMCPClient) Subscribe(ctx context.Context, req mcp.SubscribeRequest) error {
	return nil
}

// Unsubscribe 模拟 Unsubscribe 方法
func (m *MockMCPClient) Unsubscribe(ctx context.Context, request mcp.UnsubscribeRequest) error {
	return nil
}

// Close 模拟 Close 方法
func (m *MockMCPClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

// OnNotification 模拟 OnNotification 方法
func (m *MockMCPClient) OnNotification(handler func(notification mcp.JSONRPCNotification)) {
	// 空实现
}

// Complete 模拟 Complete 方法
func (m *MockMCPClient) Complete(ctx context.Context, request mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	return nil, nil
}

// GetPrompt 模拟 GetPrompt 方法
func (m *MockMCPClient) GetPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(*mcp.GetPromptResult), args.Error(1)
}

// ListPromptsByPage 模拟 ListPromptsByPage 方法
func (m *MockMCPClient) ListPromptsByPage(ctx context.Context, request mcp.ListPromptsRequest) (*mcp.ListPromptsResult, error) {
	return nil, nil
}

// ListToolsByPage 模拟 ListToolsByPage 方法
func (m *MockMCPClient) ListToolsByPage(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	return nil, nil
}

// ListResourcesByPage 模拟 ListResourcesByPage 方法
func (m *MockMCPClient) ListResourcesByPage(ctx context.Context, request mcp.ListResourcesRequest) (*mcp.ListResourcesResult, error) {
	return nil, nil
}

// ListResourceTemplatesByPage 模拟 ListResourceTemplatesByPage 方法
func (m *MockMCPClient) ListResourceTemplatesByPage(ctx context.Context, request mcp.ListResourceTemplatesRequest) (*mcp.ListResourceTemplatesResult, error) {
	return nil, nil
}

// SetLevel 模拟 SetLevel 方法
func (m *MockMCPClient) SetLevel(ctx context.Context, request mcp.SetLevelRequest) error {
	return nil
}

// ListPrompts 模拟 ListPrompts 方法
func (m *MockMCPClient) ListPrompts(ctx context.Context, request mcp.ListPromptsRequest) (*mcp.ListPromptsResult, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(*mcp.ListPromptsResult), args.Error(1)
}

func TestHostGetTools(t *testing.T) {
	// 创建上下文
	ctx := context.Background()

	// 创建模拟客户端
	mockClient := new(MockMCPClient)

	// 创建测试工具列表
	testTools := []mcp.Tool{
		{
			Name:        "test-tool-1",
			Description: "测试工具1",
		},
		{
			Name:        "test-tool-2",
			Description: "测试工具2",
		},
	}

	// 创建预期的 ListToolsResult
	expectedResult := &mcp.ListToolsResult{
		Tools: testTools,
	}

	// 设置模拟客户端的预期行为
	mockClient.On("ListTools", ctx, mock.Anything).Return(expectedResult, nil)

	// 创建 Host 实例
	host := &Host{
		Clients: map[string]*Client{
			"test-client": {
				conn: mockClient,
			},
		},
	}

	// 调用 GetTools 方法
	tools, err := host.GetTools(ctx)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, tools)
	// 注意：GetTools 方法会返回 MCP 客户端的工具 + 自定义工具
	// 自定义工具在 GetCustomTools() 中定义，包括 calculator 和 weather 两个工具
	assert.Len(t, tools, 4) // 2个测试工具 + 2个自定义工具

	// 验证前两个工具属性（来自 MCP 客户端）
	for i := 0; i < len(testTools); i++ {
		assert.Equal(t, testTools[i].Name, tools[i].Name())
		assert.Equal(t, testTools[i].Description, tools[i].Description())
	}

	// 验证后两个工具属性（自定义工具）
	assert.Equal(t, "calculator", tools[2].Name())
	assert.Equal(t, "weather", tools[3].Name())

	// 验证模拟客户端的方法被调用
	mockClient.AssertExpectations(t)
}

func TestMCPToolAdapterCall(t *testing.T) {
	// 创建上下文
	ctx := context.Background()

	// 创建模拟客户端
	mockClient := new(MockMCPClient)

	// 创建预期的 CallToolResult
	expectedResult := &mcp.CallToolResult{}
	// 设置 Result 字段
	expectedResult.Result.Meta = map[string]interface{}{
		"result": "测试结果",
	}

	// 设置模拟客户端的预期行为
	mockClient.On("CallTool", ctx, mock.Anything).Return(expectedResult, nil)

	// 创建 MCPToolAdapter 实例
	adapter := &MCPToolAdapter{
		ToolName:        "test-tool",
		ToolDescription: "测试工具",
		Client: &Client{
			conn: mockClient,
		},
		ClientName: "test-client",
	}

	// 调用 Call 方法
	result, err := adapter.Call(ctx, "测试输入")

	// 验证结果
	assert.NoError(t, err)
	assert.Contains(t, result, "map")

	// 验证模拟客户端的方法被调用
	mockClient.AssertExpectations(t)
}