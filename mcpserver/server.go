package mcpserver

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sjzsdu/tong/project"
)

// TongMCPServer 实现基于 Tong project 包的 MCP 服务器
type TongMCPServer struct {
	project     *project.Project
	projectPath string
	handlers    map[string]func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

// NewTongMCPServer 创建一个新的 Tong MCP 服务器
func NewTongMCPServer(project *project.Project) (*server.MCPServer, error) {
	return nil, nil
}
