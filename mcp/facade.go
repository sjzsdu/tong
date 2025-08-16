package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sjzsdu/tong/share"
)

func (host *Host) PrintListTools() {
	fmt.Println("当前已配置的 MCP 服务详细信息:")
	fmt.Println("==========================================")

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), share.TIMEOUT_MCP)
	defer cancel()

	// 获取工具列表请求
	request := mcp.ListToolsRequest{}

	// 获取所有客户端
	clients := host.GetAllClients()
	if len(clients) == 0 {
		fmt.Println("当前没有已配置的 MCP 服务")
		return
	}

	// 遍历每个客户端并显示其工具信息
	for clientName, client := range clients {
		fmt.Printf("\n服务器: %s\n", clientName)
		fmt.Println("------------------------------------------")

		clientResult, err := client.ListTools(ctx, request)
		if err != nil {
			fmt.Printf("❌ 获取工具列表失败: %v\n", err)
			continue
		}

		tools := clientResult.Tools
		if len(tools) == 0 {
			fmt.Println("📋 该服务器暂无可用工具")
			continue
		}

		fmt.Printf("🔧 可用工具 (%d 个):\n", len(tools))
		for _, tool := range tools {
			fmt.Printf("  • %s\n", tool.Name)
			if tool.Description != "" {
				fmt.Printf("    描述: %s\n", tool.Description)
			}

			// 显示参数信息
			if tool.InputSchema.Type != "" || len(tool.InputSchema.Properties) > 0 {
				fmt.Println("    参数:")
				if len(tool.InputSchema.Properties) > 0 {
					// 按字母顺序排序参数名
					paramNames := make([]string, 0, len(tool.InputSchema.Properties))
					for paramName := range tool.InputSchema.Properties {
						paramNames = append(paramNames, paramName)
					}
					sort.Strings(paramNames)

					for _, paramName := range paramNames {
						paramInfo := tool.InputSchema.Properties[paramName]
						paramType := "unknown"
						paramDesc := ""
						required := false

						// 检查参数类型和描述（paramInfo 是 interface{}/any 类型）
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
						for _, req := range tool.InputSchema.Required {
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

						fmt.Printf("      - %s (%s)%s", paramName, paramType, requiredText)
						if paramDesc != "" {
							fmt.Printf(": %s", paramDesc)
						}
						fmt.Println()
					}
				} else {
					fmt.Println("      无特定参数")
				}
			} else {
				fmt.Println("    参数: 无")
			}
			fmt.Println()
		}
	}
}

func (host *Host) GetToolSchema(toolName string) (string, error) {
	// 创建工具列表请求
	request := mcp.ListToolsRequest{}
	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), share.TIMEOUT_MCP)
	defer cancel()
	// 获取工具列表
	results, err := host.ListTools(ctx, request)
	if err != nil {
		return "", err
	}

	// 查找指定工具的信息
	for _, result := range results {
		for _, tool := range result.Tools {
			if tool.Name == toolName {
				// 尝试获取参数信息 - 使用反射检查结构
				toolVal := reflect.ValueOf(tool)

				// 如果是指针，获取其指向的值
				if toolVal.Kind() == reflect.Ptr && !toolVal.IsNil() {
					toolVal = toolVal.Elem()
				}

				// 只有当值是结构体时才尝试获取字段
				if toolVal.Kind() == reflect.Struct {
					// 尝试查找可能的参数字段
					paramFields := []string{"Parameters", "Schema", "Args", "Params"}
					for _, fieldName := range paramFields {
						field := toolVal.FieldByName(fieldName)
						if field.IsValid() && !field.IsZero() {
							// 找到了参数字段，尝试序列化
							paramsBytes, err := json.MarshalIndent(field.Interface(), "", "  ")
							if err == nil && len(paramsBytes) > 0 {
								return string(paramsBytes), nil
							}
						}
					}
				}

				// 如果没有找到参数信息，尝试序列化整个工具对象
				toolBytes, err := json.MarshalIndent(tool, "", "  ")
				if err == nil && len(toolBytes) > 0 {
					return string(toolBytes), nil
				}

				// 最后返回工具描述
				return fmt.Sprintf("工具 %s 的参数说明: %s", toolName, tool.Description), nil
			}
		}
	}

	// 检查是否是自定义工具
	customTools := GetCustomTools()
	for _, tool := range customTools {
		if tool.Name() == toolName {
			// 这里我们尝试从描述中解析参数信息
			return fmt.Sprintf("自定义工具参数信息: %s", tool.Description()), nil
		}
	}

	return "", fmt.Errorf("找不到工具 %s 的参数信息", toolName)
}
