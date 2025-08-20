package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/client"
	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/helper/coroutine"
	"github.com/sjzsdu/tong/schema"
	"github.com/sjzsdu/tong/share"
)

func NewHost(cfg *schema.SchemaConfig) (*Host, error) {
	if cfg == nil {
		return nil, nil
	}
	if share.GetDebug() {
		helper.PrintWithLabel("Mcp Host confg:", cfg)
	}

	Host := &Host{
		Clients: make(map[string]*Client),
	}

	// 过滤出启用的服务器配置
	enabledServers := make(map[string]schema.MCPServerConfig)
	for name, serverConfig := range cfg.MCPServers {
		if !serverConfig.Disabled {
			enabledServers[name] = serverConfig
		}
	}

	// 使用MapDict并行创建和初始化客户端
	ctx, cancel := context.WithTimeout(context.Background(), share.TIMEOUT_MCP)
	defer cancel()

	results := coroutine.MapDict(ctx, 0, enabledServers, func(name string, serverConfig schema.MCPServerConfig) (*Client, error) {
		// 创建MCP客户端
		mcpClient, err := createMCPClient(serverConfig)
		if err != nil {
			return nil, fmt.Errorf("创建客户端失败: %v", err)
		}

		// 包装客户端
		wrapClient := NewClient(mcpClient, WithHook(NewLogHook(name)))

		// 初始化客户端
		_, initErr := wrapClient.Initialize(ctx, NewInitializeRequest())
		if initErr != nil {
			return nil, fmt.Errorf("初始化超时或失败: %v", initErr)
		}

		return wrapClient, nil
	})

	// 处理结果
	for name, result := range results {
		if result.Err != nil {
			fmt.Printf("客户端 %s 处理失败: %v\n", name, result.Err)
			continue
		}

		Host.Clients[name] = result.Value
	}

	return Host, nil
}

func createMCPClient(config schema.MCPServerConfig) (client.MCPClient, error) {
	switch config.TransportType {
	case "sse":
		return client.NewSSEMCPClient(config.Url)
	case "stdio":
		return client.NewStdioMCPClient(
			config.Command,
			config.Env,
			config.Args...,
		)
	case "url":
		return client.NewStreamableHttpClient(config.Url)
	case "oauth-sse":
		// 使用配置中的 OAuth 信息
		oauthConfig := client.OAuthConfig{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			Scopes:       config.Scopes,
		}
		return client.NewOAuthSSEClient(config.Url, oauthConfig)
	case "oauth-http":
		// 使用配置中的 OAuth 信息
		oauthConfig := client.OAuthConfig{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			Scopes:       config.Scopes,
		}
		return client.NewOAuthStreamableHttpClient(config.Url, oauthConfig)
	default:
		return nil, fmt.Errorf("不支持的传输类型: %s", config.TransportType)
	}
}
