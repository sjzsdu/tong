package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/share"
)

// LogHook 实现了 Hook 接口，用于打印请求和响应日志
type LogHook struct {
	prefix string
}

// NewLogHook 创建一个新的日志钩子
func NewLogHook(prefix string) *LogHook {
	return &LogHook{
		prefix: prefix,
	}
}

func (h *LogHook) BeforeRequest(ctx context.Context, method string, args interface{}) {
	if share.GetDebug() {
		if args == nil {
			helper.PrintWithLabel("BeforeRequest: "+h.prefix+"["+method+"]", "nil arguments")
		} else {
			helper.PrintWithLabel("BeforeRequest: "+h.prefix+"["+method+"]", args)
		}
	}
}

func (h *LogHook) AfterRequest(ctx context.Context, method string, response interface{}, err error) {
	if err != nil {
		if share.GetDebug() {
			if err != nil {
				helper.PrintWithLabel("AfterRequest: "+h.prefix+"["+method+"]", err)
			}
		}
	} else {
		if share.GetDebug() {
			if response != nil {
				helper.PrintWithLabel("AfterRequest: "+h.prefix+"["+method+"]", response)
			}
		}
	}
}

func (h *LogHook) OnNotification(notification mcp.JSONRPCNotification) {
	if share.GetDebug() {
		helper.PrintWithLabel(h.prefix+"["+"通知"+"]", notification)
	}
}

// CompositeHook 组合多个 Hook
type CompositeHook struct {
	hooks []Hook
}

// NewCompositeHook 创建一个新的组合钩子
func NewCompositeHook(hooks ...Hook) *CompositeHook {
	return &CompositeHook{
		hooks: hooks,
	}
}

func (h *CompositeHook) BeforeRequest(ctx context.Context, method string, args interface{}) {
	for _, hook := range h.hooks {
		hook.BeforeRequest(ctx, method, args)
	}
}

func (h *CompositeHook) AfterRequest(ctx context.Context, method string, response interface{}, err error) {
	for _, hook := range h.hooks {
		hook.AfterRequest(ctx, method, response, err)
	}
}

func (h *CompositeHook) OnNotification(notification mcp.JSONRPCNotification) {
	for _, hook := range h.hooks {
		hook.OnNotification(notification)
	}
}
