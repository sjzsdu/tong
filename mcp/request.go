package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sjzsdu/tong/share"
)

// NewInitializeRequest 创建一个新的初始化请求
func NewInitializeRequest() mcp.InitializeRequest {
	return mcp.InitializeRequest{
		Request: mcp.Request{
			Method: string(mcp.MethodInitialize),
		},
		Params: struct {
			ProtocolVersion string                 `json:"protocolVersion"`
			Capabilities    mcp.ClientCapabilities `json:"capabilities"`
			ClientInfo      mcp.Implementation     `json:"clientInfo"`
		}{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			Capabilities:    mcp.ClientCapabilities{},
			ClientInfo: mcp.Implementation{
				Name:    share.MCP_CLIENT_NAME,
				Version: share.VERSION,
			},
		},
	}
}

// NewReadResourceRequest 创建一个新的资源读取请求
func NewReadResourceRequest(uri string, args map[string]interface{}) mcp.ReadResourceRequest {
	return mcp.ReadResourceRequest{
		Request: mcp.Request{
			Method: string(mcp.MethodResourcesRead),
		},
		Params: struct {
			URI       string                 `json:"uri"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
		}{
			URI:       uri,
			Arguments: args,
		},
	}
}

func NewPromptRequest(name string, args map[string]string) mcp.GetPromptRequest {
	return mcp.GetPromptRequest{
		Request: mcp.Request{
			Method: string(mcp.MethodPromptsGet),
		},
		Params: struct {
			// The name of the prompt or prompt template.
			Name string `json:"name"`
			// Arguments to use for templating the prompt.
			Arguments map[string]string `json:"arguments,omitempty"`
		}{
			Name:      name,
			Arguments: args,
		},
	}
}

func NewToolCallRequest(name string, args map[string]interface{}, progressToken ...mcp.ProgressToken) mcp.CallToolRequest {
	// 创建请求结构
	req := mcp.CallToolRequest{
		Request: mcp.Request{
			Method: string(mcp.MethodToolsCall),
		},
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	}

	// 如果提供了进度令牌，则设置 Meta 字段
	if len(progressToken) > 0 {
		req.Params.Meta = &mcp.Meta{
			ProgressToken: progressToken[0],
		}
	}

	return req
}
