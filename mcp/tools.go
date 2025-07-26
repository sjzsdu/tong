package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tmc/langchaingo/tools"
)

// CustomTool 是一个自定义工具实现
type CustomTool struct {
	ToolName        string
	ToolDescription string
	Handler         func(ctx context.Context, input string) (string, error)
}

// Name 返回工具名称
func (t *CustomTool) Name() string {
	return t.ToolName
}

// Description 返回工具描述
func (t *CustomTool) Description() string {
	return t.ToolDescription
}

// Call 执行工具调用
func (t *CustomTool) Call(ctx context.Context, input string) (string, error) {
	return t.Handler(ctx, input)
}

// NewCustomTool 创建一个新的自定义工具
func NewCustomTool(name, description string, handler func(ctx context.Context, input string) (string, error)) *CustomTool {
	return &CustomTool{
		ToolName:        name,
		ToolDescription: description,
		Handler:         handler,
	}
}

// ToolInput 表示工具输入的结构
type ToolInput struct {
	Args map[string]interface{} `json:"args"`
}

// ParseToolInput 解析工具输入
func ParseToolInput(input string) (map[string]interface{}, error) {
	var toolInput ToolInput
	err := json.Unmarshal([]byte(input), &toolInput)
	if err != nil {
		return nil, fmt.Errorf("解析工具输入失败: %v", err)
	}
	return toolInput.Args, nil
}

// GetCustomTools 返回自定义工具列表
func GetCustomTools() []tools.Tool {
	var customTools []tools.Tool

	// 添加一个简单的计算器工具
	calculatorTool := NewCustomTool(
		"calculator",
		"一个简单的计算器工具，可以执行基本的数学运算。输入格式: {\"args\": {\"operation\": \"add|subtract|multiply|divide\", \"a\": number, \"b\": number}}",
		func(ctx context.Context, input string) (string, error) {
			args, err := ParseToolInput(input)
			if err != nil {
				return "", err
			}

			operation, ok := args["operation"].(string)
			if !ok {
				return "", fmt.Errorf("缺少 'operation' 参数或类型错误")
			}

			a, ok := args["a"].(float64)
			if !ok {
				return "", fmt.Errorf("缺少 'a' 参数或类型错误")
			}

			b, ok := args["b"].(float64)
			if !ok {
				return "", fmt.Errorf("缺少 'b' 参数或类型错误")
			}

			var result float64
			switch operation {
			case "add":
				result = a + b
			case "subtract":
				result = a - b
			case "multiply":
				result = a * b
			case "divide":
				if b == 0 {
					return "", fmt.Errorf("除数不能为零")
				}
				result = a / b
			default:
				return "", fmt.Errorf("不支持的操作: %s", operation)
			}

			return fmt.Sprintf("%.2f", result), nil
		},
	)

	// 添加一个天气查询工具（模拟）
	weatherTool := NewCustomTool(
		"weather",
		"一个天气查询工具，可以查询指定城市的天气。输入格式: {\"args\": {\"city\": \"城市名\"}}",
		func(ctx context.Context, input string) (string, error) {
			args, err := ParseToolInput(input)
			if err != nil {
				return "", err
			}

			city, ok := args["city"].(string)
			if !ok {
				return "", fmt.Errorf("缺少 'city' 参数或类型错误")
			}

			// 这里只是模拟，实际应用中应该调用真实的天气 API
			weatherInfo := map[string]string{
				"北京":  "晴天，温度 25°C",
				"上海":  "多云，温度 28°C",
				"广州":  "小雨，温度 30°C",
				"深圳":  "阴天，温度 29°C",
				"成都":  "晴天，温度 26°C",
			}

			if weather, exists := weatherInfo[city]; exists {
				return fmt.Sprintf("%s 的天气: %s", city, weather), nil
			}

			return fmt.Sprintf("没有找到 %s 的天气信息", city), nil
		},
	)

	customTools = append(customTools, calculatorTool, weatherTool)
	return customTools
}