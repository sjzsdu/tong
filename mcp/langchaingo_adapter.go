package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	mcppkg "github.com/mark3labs/mcp-go/mcp"
	"github.com/sjzsdu/tong/helper"
)

// MCPToolAdapter 是一个适配器，将 mcp.Tool 转换为 tools.Tool 接口
type MCPToolAdapter struct {
	ToolName        string
	ToolDescription string
	InputSchema     mcppkg.ToolInputSchema
	Client          *Client
	ClientName      string
}

// Name 返回工具名称
func (t *MCPToolAdapter) Name() string {
	return t.ToolName
}

// Description 返回工具描述，包含参数信息
func (t *MCPToolAdapter) Description() string {
	// 基础描述
	description := t.ToolDescription

	// 添加参数信息
	if t.InputSchema.Type != "" || len(t.InputSchema.Properties) > 0 {
		description += "\n参数:"
		if len(t.InputSchema.Properties) > 0 {
			// 按字母顺序排序参数名
			paramNames := make([]string, 0, len(t.InputSchema.Properties))
			for paramName := range t.InputSchema.Properties {
				paramNames = append(paramNames, paramName)
			}
			sort.Strings(paramNames)

			for _, paramName := range paramNames {
				paramInfo := t.InputSchema.Properties[paramName]
				paramType := "unknown"
				paramDesc := ""
				required := false

				// 检查参数类型和描述
				if paramMap, ok := paramInfo.(map[string]interface{}); ok {
					if typeVal, exists := paramMap["type"]; exists {
						if typeStr, ok := typeVal.(string); ok {
							paramType = typeStr
						}
					}
					if descVal, exists := paramMap["description"]; exists {
						if descStr, ok := descVal.(string); ok {
							paramDesc = descStr
						}
					}
				}

				// 检查是否必需
				for _, req := range t.InputSchema.Required {
					if req == paramName {
						required = true
						break
					}
				}

				// 格式化输出
				requiredText := ""
				if required {
					requiredText = " [必需]"
				}

				description += fmt.Sprintf("\n  - %s (%s)%s", paramName, paramType, requiredText)
				if paramDesc != "" {
					description += fmt.Sprintf(": %s", paramDesc)
				}
			}
		} else {
			description += "\n  无特定参数"
		}
	}

	return description
}

// Call 执行工具调用
func (t *MCPToolAdapter) Call(ctx context.Context, input string) (string, error) {

	// 解析输入参数
	var params map[string]interface{}
	err := json.Unmarshal([]byte(input), &params)
	if err != nil {
		// 如果无法解析为 JSON，则将整个输入作为 input 参数传递
		params = map[string]interface{}{
			"input": input,
		}
	}
	// 创建工具调用请求
	req := NewToolCallRequest(t.ToolName, params)

	// 调用工具

	t.Client.callHookBefore(ctx, "CallTool", req)
	result, err := t.Client.CallTool(ctx, req)
	t.Client.callHookAfter(ctx, "CallTool", result, err)

	if err != nil {
		helper.PrintWithLabel("CallTool Error", err.Error(), fmt.Sprintf("Tool: %s, Error: %v", t.ToolName, err))
		return "", fmt.Errorf("调用工具 %s 失败: %v", t.ToolName, err)
	}

	// 返回结果
	// 处理 Content 字段
	if len(result.Content) > 0 {
		// 提取文本内容
		var textParts []string
		for _, content := range result.Content {
			if textContent, ok := content.(mcppkg.TextContent); ok {
				textParts = append(textParts, textContent.Text)
			}
		}

		if len(textParts) > 0 {
			resultStr := textParts[0]
			// 如果结果看起来像 JSON 对象或数组，尝试美化输出
			if len(resultStr) > 0 && (resultStr[0] == '{' || resultStr[0] == '[') {
				var jsonObj interface{}
				if err := json.Unmarshal([]byte(resultStr), &jsonObj); err == nil {
					if prettyJSON, err := json.MarshalIndent(jsonObj, "", "  "); err == nil {
						return string(prettyJSON), nil
					}
				}
			}
			return resultStr, nil
		}
	}
	// 兼容旧版本，如果 Content 为空，尝试使用 Result
	resultStr := fmt.Sprintf("%v", result.Result)
	return resultStr, nil
}
