package mcpserver

import (
	"errors"

	"github.com/mark3labs/mcp-go/server"
	"github.com/sjzsdu/tong/project"
	"github.com/sjzsdu/tong/share"
)

// TongMCPServer 实现基于 Tong project 包的 MCP 服务器
// （保留占位结构，后续可扩展为有状态的会话管理）
type TongMCPServer struct {
	project     *project.Project
	projectPath string
}

// NewTongMCPServer 创建一个新的 Tong MCP 服务器
func NewTongMCPServer(proj *project.Project) (*server.MCPServer, error) {
	if proj == nil {
		return nil, errors.New("project is nil")
	}

	// 创建 MCP 服务器（启用工具能力）
	s := server.NewMCPServer(
		"Tong MCP Server",
		share.VERSION,
		server.WithToolCapabilities(false),
	)

	// 注册文件系统工具
	RegisterFileTools(s, proj)

	// 注册Web相关工具
	RegisterWebTools(s)

	// 注册计算机相关工具
	RegisterComputerTools(s, proj)

	return s, nil
}
