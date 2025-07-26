package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sjzsdu/tong/config"
	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/share"
	"github.com/tmc/langchaingo/tools"
)

type Host struct {
	Clients map[string]*Client
}

func createMCPClient(config config.MCPServerConfig) (client.MCPClient, error) {
	switch config.TransportType {
	case "sse":
		return client.NewSSEMCPClient(config.Url)
	case "stdio":
		return client.NewStdioMCPClient(
			config.Command,
			config.Env,
			config.Args...,
		)
	default:
		return nil, fmt.Errorf("不支持的传输类型: %s", config.TransportType)
	}
}

func NewHost(config *config.SchemaConfig) (*Host, error) {
	if config == nil {
		return nil, nil
	}
	if share.GetDebug() {
		helper.PrintWithLabel("Mcp Host confg:", config)
	}

	Host := &Host{
		Clients: make(map[string]*Client),
	}

	for name, serverConfig := range config.MCPServers {
		if serverConfig.Disabled {
			continue
		}

		mcpClient, err := createMCPClient(serverConfig)
		if err != nil {
			fmt.Printf("创建客户端 %s 失败: %v\n", name, err)
			continue
		}

		Host.Clients[name] = NewClient(mcpClient, WithHook(NewLogHook(name)))
	}

	return Host, nil
}

func (c *Host) Ping(ctx context.Context) error {
	var lastErr error
	for name, client := range c.Clients {
		if err := client.Ping(ctx); err != nil {
			fmt.Printf("客户端 %s Ping 失败: %v\n", name, err)
			lastErr = err
		}
	}
	return lastErr
}

func (c *Host) ListResources(ctx context.Context, request mcp.ListResourcesRequest) (*mcp.ListResourcesResult, error) {
	var lastErr error
	var lastResult *mcp.ListResourcesResult

	for name, client := range c.Clients {
		result, err := client.ListResources(ctx, request)
		if err != nil {
			fmt.Printf("客户端 %s 获取资源列表失败: %v\n", name, err)
			lastErr = err
			continue
		}
		lastResult = result
	}
	return lastResult, lastErr
}

func (c *Host) ListResourceTemplates(ctx context.Context, request mcp.ListResourceTemplatesRequest) (*mcp.ListResourceTemplatesResult, error) {
	var lastErr error
	var lastResult *mcp.ListResourceTemplatesResult

	for name, client := range c.Clients {
		result, err := client.ListResourceTemplates(ctx, request)
		if err != nil {
			fmt.Printf("客户端 %s 获取资源模板列表失败: %v\n", name, err)
			lastErr = err
			continue
		}
		lastResult = result
	}
	return lastResult, lastErr
}

func (c *Host) ReadResource(ctx context.Context, request mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	var lastErr error
	var lastResult *mcp.ReadResourceResult

	for name, client := range c.Clients {
		result, err := client.ReadResource(ctx, request)
		if err != nil {
			fmt.Printf("客户端 %s 读取资源失败: %v\n", name, err)
			lastErr = err
			continue
		}
		lastResult = result
	}
	return lastResult, lastErr
}

func (c *Host) Subscribe(ctx context.Context, request mcp.SubscribeRequest) error {
	var lastErr error
	for name, client := range c.Clients {
		if err := client.Subscribe(ctx, request); err != nil {
			fmt.Printf("客户端 %s 订阅失败: %v\n", name, err)
			lastErr = err
		}
	}
	return lastErr
}

func (c *Host) Unsubscribe(ctx context.Context, request mcp.UnsubscribeRequest) error {
	var lastErr error
	for name, client := range c.Clients {
		if err := client.Unsubscribe(ctx, request); err != nil {
			fmt.Printf("客户端 %s 取消订阅失败: %v\n", name, err)
			lastErr = err
		}
	}
	return lastErr
}

func (c *Host) ListPrompts(ctx context.Context, request mcp.ListPromptsRequest) (*mcp.ListPromptsResult, error) {
	var lastErr error
	var lastResult *mcp.ListPromptsResult

	for name, client := range c.Clients {
		result, err := client.ListPrompts(ctx, request)
		if err != nil {
			fmt.Printf("客户端 %s 获取提示列表失败: %v\n", name, err)
			lastErr = err
			continue
		}
		lastResult = result
	}
	return lastResult, lastErr
}

func (c *Host) GetPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	var lastErr error
	var lastResult *mcp.GetPromptResult

	for name, client := range c.Clients {
		result, err := client.GetPrompt(ctx, request)
		if err != nil {
			fmt.Printf("客户端 %s 获取提示失败: %v\n", name, err)
			lastErr = err
			continue
		}
		lastResult = result
	}
	return lastResult, lastErr
}

func (c *Host) ListTools(ctx context.Context, request mcp.ListToolsRequest) ([]*mcp.ListToolsResult, error) {
	var lastErr error
	var results []*mcp.ListToolsResult

	for name, client := range c.Clients {
		result, err := client.ListTools(ctx, request)
		if err != nil {
			fmt.Printf("客户端 %s 获取工具列表失败: %v\n", name, err)
			lastErr = err
			continue
		}
		results = append(results, result)
	}
	return results, lastErr
}

func (c *Host) CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var lastErr error
	var lastResult *mcp.CallToolResult

	for name, client := range c.Clients {
		result, err := client.CallTool(ctx, request)
		if err != nil {
			fmt.Printf("客户端 %s 调用工具失败: %v\n", name, err)
			lastErr = err
			continue
		}
		lastResult = result
	}
	return lastResult, lastErr
}

func (c *Host) SetLevel(ctx context.Context, request mcp.SetLevelRequest) error {
	var lastErr error
	for name, client := range c.Clients {
		if err := client.SetLevel(ctx, request); err != nil {
			fmt.Printf("客户端 %s 设置日志级别失败: %v\n", name, err)
			lastErr = err
		}
	}
	return lastErr
}

func (c *Host) Complete(ctx context.Context, request mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	var lastErr error
	var lastResult *mcp.CompleteResult

	for name, client := range c.Clients {
		result, err := client.Complete(ctx, request)
		if err != nil {
			fmt.Printf("客户端 %s 自动完成失败: %v\n", name, err)
			lastErr = err
			continue
		}
		lastResult = result
	}
	return lastResult, lastErr
}

func (c *Host) OnNotification(handler func(notification mcp.JSONRPCNotification)) {
	for _, client := range c.Clients {
		client.OnNotification(handler)
	}
}

func (c *Host) GetClient(name string) *Client {
	if c == nil {
		return nil
	}
	return c.Clients[name]
}

func (c *Host) GetAllClients() map[string]*Client {
	if c == nil {
		return nil
	}
	return c.Clients
}

// GetTools 返回所有可用工具的列表，实现 tools.Tool 接口
func (c *Host) GetTools(ctx context.Context) ([]tools.Tool, error) {
	// 创建工具列表请求
	request := mcp.ListToolsRequest{}

	// 获取工具列表
	results, err := c.ListTools(ctx, request)
	if err != nil {
		return nil, err
	}

	// 创建工具适配器列表
	var toolsList []tools.Tool

	// 遍历所有客户端的工具
	for clientName, client := range c.Clients {
		// 遍历该客户端的所有工具结果
		for _, result := range results {
			for _, tool := range result.Tools {
				// 创建适配器
				adapter := &MCPToolAdapter{
					ToolName:        tool.Name,
					ToolDescription: tool.Description,
					Client:          client,
					ClientName:      clientName,
				}
				// 添加到列表
				toolsList = append(toolsList, adapter)
			}
		}
	}

	// 添加自定义工具
	customTools := GetCustomTools()
	toolsList = append(toolsList, customTools...)

	return toolsList, nil
}

func (c *Host) Close() error {
	var lastErr error
	for name, client := range c.Clients {
		if err := client.Close(); err != nil {
			fmt.Printf("客户端 %s 关闭失败: %v\n", name, err)
			lastErr = err
		}
	}
	return lastErr
}
