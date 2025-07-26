package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// ClientOption 定义客户端选项函数类型
type ClientOption func(*Client)

// Hook 定义钩子函数的接口
type Hook interface {
	BeforeRequest(ctx context.Context, method string, args interface{})
	AfterRequest(ctx context.Context, method string, response interface{}, err error)
	OnNotification(notification mcp.JSONRPCNotification)
}

// Client 实现 MCPClient 接口
type Client struct {
	conn client.MCPClient
	hook Hook
}

// WithHook 设置客户端钩子
func WithHook(hook Hook) ClientOption {
	return func(c *Client) {
		c.hook = hook
	}
}

func NewClient(conn client.MCPClient, opts ...ClientOption) *Client {
	client := &Client{
		conn: conn,
	}

	// 应用选项
	for _, opt := range opts {
		opt(client)
	}

	client.Initialize(context.Background(), NewInitializeRequest())
	return client
}

func (c *Client) callHookBefore(ctx context.Context, method string, args interface{}) {
	if c.hook != nil {
		c.hook.BeforeRequest(ctx, method, args)
	}
}

func (c *Client) callHookAfter(ctx context.Context, method string, response interface{}, err error) {
	if c.hook != nil {
		c.hook.AfterRequest(ctx, method, response, err)
	}
}

// 实现 MCPClient 接口的所有方法
func (c *Client) Initialize(ctx context.Context, request mcp.InitializeRequest) (*mcp.InitializeResult, error) {
	c.callHookBefore(ctx, "Initialize", request)
	result, err := c.conn.Initialize(ctx, request)
	c.callHookAfter(ctx, "Initialize", result, err)
	return result, err
}

func (c *Client) Ping(ctx context.Context) error {
	c.callHookBefore(ctx, "Ping", nil)
	err := c.conn.Ping(ctx)
	c.callHookAfter(ctx, "Ping", nil, err)
	return err
}

func (c *Client) ListResources(ctx context.Context, request mcp.ListResourcesRequest) (*mcp.ListResourcesResult, error) {
	c.callHookBefore(ctx, "ListResources", request)
	result, err := c.conn.ListResources(ctx, request)
	c.callHookAfter(ctx, "ListResources", result, err)
	return result, err
}

func (c *Client) ListResourceTemplates(ctx context.Context, request mcp.ListResourceTemplatesRequest) (*mcp.ListResourceTemplatesResult, error) {
	c.callHookBefore(ctx, "ListResourceTemplates", request)
	result, err := c.conn.ListResourceTemplates(ctx, request)
	c.callHookAfter(ctx, "ListResourceTemplates", result, err)
	return result, err
}

func (c *Client) ReadResource(ctx context.Context, request mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	c.callHookBefore(ctx, "ReadResource", request)
	result, err := c.conn.ReadResource(ctx, request)
	c.callHookAfter(ctx, "ReadResource", result, err)
	return result, err
}

func (c *Client) Subscribe(ctx context.Context, request mcp.SubscribeRequest) error {
	c.callHookBefore(ctx, "Subscribe", request)
	err := c.conn.Subscribe(ctx, request)
	c.callHookAfter(ctx, "Subscribe", nil, err)
	return err
}

func (c *Client) Unsubscribe(ctx context.Context, request mcp.UnsubscribeRequest) error {
	c.callHookBefore(ctx, "Unsubscribe", request)
	err := c.conn.Unsubscribe(ctx, request)
	c.callHookAfter(ctx, "Unsubscribe", nil, err)
	return err
}

func (c *Client) ListPrompts(ctx context.Context, request mcp.ListPromptsRequest) (*mcp.ListPromptsResult, error) {
	c.callHookBefore(ctx, "ListPrompts", request)
	result, err := c.conn.ListPrompts(ctx, request)
	c.callHookAfter(ctx, "ListPrompts", result, err)
	return result, err
}

func (c *Client) GetPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	c.callHookBefore(ctx, "GetPrompt", request)
	result, err := c.conn.GetPrompt(ctx, request)
	c.callHookAfter(ctx, "GetPrompt", result, err)
	return result, err
}

func (c *Client) ListTools(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	c.callHookBefore(ctx, "ListTools", request)
	result, err := c.conn.ListTools(ctx, request)
	c.callHookAfter(ctx, "ListTools", result, err)
	return result, err
}

func (c *Client) CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	c.callHookBefore(ctx, "CallTool", request)
	result, err := c.conn.CallTool(ctx, request)
	c.callHookAfter(ctx, "CallTool", result, err)
	return result, err
}

func (c *Client) SetLevel(ctx context.Context, request mcp.SetLevelRequest) error {
	c.callHookBefore(ctx, "SetLevel", request)
	err := c.conn.SetLevel(ctx, request)
	c.callHookAfter(ctx, "SetLevel", nil, err)
	return err
}

func (c *Client) Complete(ctx context.Context, request mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	c.callHookBefore(ctx, "Complete", request)
	result, err := c.conn.Complete(ctx, request)
	c.callHookAfter(ctx, "Complete", result, err)
	return result, err
}

func (c *Client) Close() error {
	c.callHookBefore(context.Background(), "Close", nil)
	err := c.conn.Close()
	c.callHookAfter(context.Background(), "Close", nil, err)
	return err
}

func (c *Client) OnNotification(handler func(notification mcp.JSONRPCNotification)) {
	if c.hook != nil {
		originalHandler := handler
		handler = func(notification mcp.JSONRPCNotification) {
			c.hook.OnNotification(notification)
			if originalHandler != nil {
				originalHandler(notification)
			}
		}
	}
	c.conn.OnNotification(handler)
}
