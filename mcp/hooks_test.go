package mcp

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

func TestLogHook(t *testing.T) {
	// 创建 LogHook 实例
	hook := NewLogHook("test")

	// 测试 BeforeRequest 方法
	// 这里主要是确保方法不会 panic
	hook.BeforeRequest(context.Background(), "TestMethod", nil)
	hook.BeforeRequest(context.Background(), "TestMethod", map[string]string{"key": "value"})

	// 测试 AfterRequest 方法
	hook.AfterRequest(context.Background(), "TestMethod", nil, nil)
	hook.AfterRequest(context.Background(), "TestMethod", map[string]string{"key": "value"}, nil)
	hook.AfterRequest(context.Background(), "TestMethod", nil, assert.AnError)

	// 测试 OnNotification 方法
	notification := mcp.JSONRPCNotification{
		Notification: mcp.Notification{
			Method: "test-method",
			Params: mcp.NotificationParams{
				Meta: map[string]any{
					"key": "value",
				},
			},
		},
	}
	hook.OnNotification(notification)

	// 由于 LogHook 主要是打印日志，这里只是确保方法不会 panic
	// 实际上没有返回值可以断言
	assert.True(t, true, "LogHook 方法应该不会 panic")
}

func TestCompositeHook(t *testing.T) {
	// 创建两个 mock hook
	hook1 := &mockHook{}
	hook2 := &mockHook{}

	// 创建 CompositeHook 实例
	compositeHook := NewCompositeHook(hook1, hook2)

	// 测试 BeforeRequest 方法
	compositeHook.BeforeRequest(context.Background(), "TestMethod", nil)
	assert.Equal(t, 1, hook1.beforeRequestCalled, "hook1 的 BeforeRequest 方法应该被调用一次")
	assert.Equal(t, 1, hook2.beforeRequestCalled, "hook2 的 BeforeRequest 方法应该被调用一次")

	// 测试 AfterRequest 方法
	compositeHook.AfterRequest(context.Background(), "TestMethod", nil, nil)
	assert.Equal(t, 1, hook1.afterRequestCalled, "hook1 的 AfterRequest 方法应该被调用一次")
	assert.Equal(t, 1, hook2.afterRequestCalled, "hook2 的 AfterRequest 方法应该被调用一次")

	// 测试 OnNotification 方法
	notification := mcp.JSONRPCNotification{
		Notification: mcp.Notification{
			Method: "test-method",
			Params: mcp.NotificationParams{
				Meta: map[string]any{
					"key": "value",
				},
			},
		},
	}
	compositeHook.OnNotification(notification)
	assert.Equal(t, 1, hook1.onNotificationCalled, "hook1 的 OnNotification 方法应该被调用一次")
	assert.Equal(t, 1, hook2.onNotificationCalled, "hook2 的 OnNotification 方法应该被调用一次")
}

// mockHook 是一个用于测试的 Hook 实现
type mockHook struct {
	beforeRequestCalled  int
	afterRequestCalled   int
	onNotificationCalled int
}

func (h *mockHook) BeforeRequest(ctx context.Context, method string, args interface{}) {
	h.beforeRequestCalled++
}

func (h *mockHook) AfterRequest(ctx context.Context, method string, response interface{}, err error) {
	h.afterRequestCalled++
}

func (h *mockHook) OnNotification(notification mcp.JSONRPCNotification) {
	h.onNotificationCalled++
}