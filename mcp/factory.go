package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/client"
	"github.com/sjzsdu/tong/config"
	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/share"
)

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

		wrapClient := NewClient(mcpClient, WithHook(NewLogHook(name)))
		// 添加超时初始化
		ctx, cancel := context.WithTimeout(context.Background(), share.TIMEOUT_MCP)
		defer cancel()
		_, initErr := wrapClient.Initialize(ctx, NewInitializeRequest())
		if initErr != nil {
			fmt.Printf("警告：客户端 %s 初始化超时或失败: %v\n", name, initErr)
			continue
		}

		Host.Clients[name] = wrapClient
	}

	return Host, nil
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
