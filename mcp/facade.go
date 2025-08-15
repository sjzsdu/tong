package mcp

import (
	"context"
	"fmt"
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
