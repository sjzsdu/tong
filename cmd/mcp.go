package cmd

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/server"
	llms "github.com/sjzsdu/langchaingo-cn/llms"
	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/lang"
	mcpHost "github.com/sjzsdu/tong/mcp"
	"github.com/sjzsdu/tong/mcpserver"
	"github.com/sjzsdu/tong/schema"
	"github.com/sjzsdu/tong/share"
	"github.com/spf13/cobra"
	llmsPack "github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: lang.T("MCP Server"),
	Long:  lang.T("MCP Server for managing project files"),
	Run:   runMCP,
}

var (
	mcpTransport string
	mcpPortFlag  string
)

func init() {
	rootCmd.AddCommand(mcpCmd)

	// 添加命令行标志
	mcpCmd.Flags().StringVar(&mcpTransport, "transport", "", "传输方式 (stdio, http, sse)，默认为 stdio")
	mcpCmd.Flags().StringVar(&mcpPortFlag, "port", "8080", "HTTP/SSE 服务器端口，默认为 8080")
}

func runMCP(cmd *cobra.Command, args []string) {
	// 检查参数是否存在
	if len(args) == 0 {
		fmt.Println("请指定操作类型:")
		fmt.Println("  available - 列出所有可用的 MCP 服务")
		fmt.Println("  list      - 列出当前配置的 MCP 服务")
		fmt.Println("  detail    - 列出当前配置的 MCP 服务及其工具详情")
		fmt.Println("  server    - 启动 Tong MCP 服务器")
		fmt.Println()
		fmt.Println("启动服务器示例:")
		fmt.Println("  tong mcp server                    # 使用 STDIO 传输启动")
		fmt.Println("  tong mcp server --transport http   # 使用 HTTP 传输启动")
		fmt.Println("  tong mcp server --transport sse    # 使用 SSE 传输启动")
		fmt.Println("  tong mcp server --port 9000        # 指定端口启动")
		fmt.Println("  tong mcp server --debug            # 启用调试模式")
		cmd.Help()
		return
	}

	switch args[0] {
	case "list":
		// 列出当前配置的 MCP 服务
		listConfiguredMCPServers()
	case "detail":
		// 列出当前配置的 MCP 服务
		listMCPServersDetail()
	case "available":
		// 列出所有可用的 MCP 服务
		listAvailableMCPServers()
	case "server":
		// 这里运行自己的mcpserver
		runMCPServer()
	case "test":
		// 测试指定的工具
		if len(args) == 1 {
			fmt.Println("请指定要测试的工具名称:")
			fmt.Println("例如: tong mcp test run_command")
			fmt.Println("支持的工具:")
			fmt.Println("- run_command: 执行shell命令并返回结果")
			fmt.Println("- fs_list: 列出目录内容")
			fmt.Println("- fs_read: 读取文件内容")
			fmt.Println("- fs_write: 写入文件内容")
			fmt.Println("更多工具请运行 'tong mcp detail' 查看")
			cmd.Help()
			return
		}
		testToolCall(args[1])
	default:
		fmt.Println("未知的操作类型: " + args[0])
		cmd.Help()
	}
}

func testToolCall(toolName string) {
	// 获取配置
	config, err := GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize LLM
	llm, err := llms.CreateLLM(config.MasterLLM.Type, config.MasterLLM.Params)

	if err != nil {
		log.Fatal(err)
	}
	if share.GetDebug() {
		helper.PrintWithLabel("配置信息:", config)
	}

	host, err := mcpHost.NewHost(config)
	if err != nil {
		log.Fatalf("创建 MCP Host 失败: %v", err)
	}

	ctx := context.Background()
	schemeTools, err := host.GetTools(ctx)
	if err != nil {
		fmt.Printf("警告: 获取 MCP 工具失败: %v\n将继续执行但功能可能受限\n", err)
		schemeTools = []tools.Tool{}
	}

	// 找到指定的工具
	var targetTool tools.Tool
	var foundTool bool
	for _, tool := range schemeTools {
		if tool.Name() == toolName {
			targetTool = tool
			foundTool = true
			break
		}
	}

	if !foundTool {
		fmt.Printf("找不到工具: %s\n", toolName)
		fmt.Println("可用的工具有:")
		for _, tool := range schemeTools {
			fmt.Printf("- %s: %s\n", tool.Name(), tool.Description())
		}
		return
	}

	// 打印工具信息
	fmt.Printf("===== 工具信息 =====\n")
	fmt.Printf("名称: %s\n", targetTool.Name())
	fmt.Printf("描述: %s\n", targetTool.Description())

	// 获取工具的参数信息（如果可用）
	paramInfo := ""

	// 尝试从工具的源代码中提取参数信息
	toolSource, err := host.GetToolSchema(toolName)
	if err == nil && toolSource != "" {
		paramInfo = fmt.Sprintf("参数结构: %s", toolSource)
	}

	// 构建提示词，让LLM生成参数
	prompt := fmt.Sprintf(`请为以下工具生成调用参数：
工具名称: %s
工具描述: %s
%s

你需要生成一个有效的JSON对象作为工具的调用参数。必须是有效的JSON格式，只返回JSON对象，不要有其他说明文字。
`, targetTool.Name(), targetTool.Description(), paramInfo)

	// 询问用户是否要自行输入参数
	userWantsInput, err := helper.PromptYesNo("\n是否要自行输入参数? (y/n) \n", false)
	if err != nil {
		fmt.Printf("读取用户输入时出错: %v\n", err)
		return
	}

	var generatedParams string

	if userWantsInput {
		// 用户自行输入参数
		fmt.Println("请输入JSON格式的参数:")
		reader := bufio.NewReader(os.Stdin)
		generatedParams, _ = reader.ReadString('\n')
		generatedParams = strings.TrimSpace(generatedParams)
	} else {
		// 使用 GenerateContent 方法替代弃用的 Call 方法
		msgs := []llmsPack.MessageContent{
			{
				Role:  llmsPack.ChatMessageTypeSystem,
				Parts: []llmsPack.ContentPart{llmsPack.TextPart(prompt)},
			},
		}

		response, err := llm.GenerateContent(ctx, msgs)
		if err != nil {
			fmt.Printf("生成参数失败: %v\n", err)
			return
		}

		// 获取生成的内容
		llmResult := ""
		if len(response.Choices) > 0 {
			llmResult = response.Choices[0].Content
		}

		// 提取LLM生成的参数（从字符串中提取JSON）
		// 查找第一个 { 和最后一个 }
		var startIdx, endIdx int
		startIdx = strings.Index(llmResult, "{")
		endIdx = strings.LastIndex(llmResult, "}")

		if startIdx >= 0 && endIdx > startIdx {
			generatedParams = llmResult[startIdx : endIdx+1]
		} else {
			fmt.Println("LLM未能生成有效的JSON参数，将使用空参数")
			generatedParams = "{}"
		}
	}

	fmt.Printf("\n===== 生成的参数 =====\n%s\n", generatedParams)

	// 执行工具调用
	fmt.Printf("\n===== 执行工具调用 =====\n")
	toolResult, toolErr := targetTool.Call(ctx, generatedParams)
	if toolErr != nil {
		fmt.Printf("调用工具失败: %v\n", toolErr)
		return
	}

	// 打印结果
	fmt.Printf("\n===== 工具调用结果 =====\n%s\n", toolResult)
}

func runMCPServer() {
	project, err := GetProject()
	if err != nil {
		fmt.Printf("创建项目实例失败: %v\n", err)
		return
	}

	// 创建 Tong MCP 服务器
	mcpSrv, err := mcpserver.NewTongMCPServer(project)
	if err != nil {
		fmt.Printf("创建 MCP 服务器失败: %v\n", err)
		return
	}

	// 使用命令行参数或环境变量
	transport := mcpTransport
	if transport == "" {
		transport = os.Getenv("MCP_TRANSPORT")
	}

	port := mcpPortFlag
	if envPort := os.Getenv("MCP_PORT"); envPort != "" {
		port = envPort
	}

	fmt.Printf("启动 Tong MCP 服务器...\n")
	fmt.Printf("项目路径: %s\n", project.Root().Path)

	switch transport {
	case "http":
		fmt.Printf("使用 HTTP 传输，端口: %s\n", port)
		fmt.Printf("访问地址: http://localhost:%s\n", port)
		httpServer := server.NewStreamableHTTPServer(mcpSrv)
		if err := httpServer.Start(":" + port); err != nil {
			log.Fatalf("HTTP 服务器启动失败: %v", err)
		}
	case "sse":
		fmt.Printf("使用 SSE 传输，端口: %s\n", port)
		fmt.Printf("访问地址: http://localhost:%s\n", port)
		sseServer := server.NewSSEServer(mcpSrv)
		if err := sseServer.Start(":" + port); err != nil {
			log.Fatalf("SSE 服务器启动失败: %v", err)
		}
	default:
		fmt.Println("使用 STDIO 传输")
		fmt.Println("服务器已启动，等待客户端连接...")
		if err := server.ServeStdio(mcpSrv); err != nil {
			log.Fatalf("STDIO 服务器启动失败: %v", err)
		}
	}
}

// printMCPServices 打印 MCP 服务列表，提供通用的输出格式
func printMCPServices(title string, services map[string]interface{}, formatter func(string) string) {
	// 如果没有服务
	if len(services) == 0 {
		fmt.Println(title + "\n当前没有服务")
		return
	}

	// 按字母顺序排序服务名称
	serviceNames := make([]string, 0, len(services))
	for name := range services {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)

	// 输出服务
	fmt.Println(title)
	for _, name := range serviceNames {
		output := name
		if formatter != nil {
			output = formatter(name)
		}
		fmt.Printf("- %s\n", output)
	}
}

// 列出每一个mcp的所有的tool名称，参数和描述
func listMCPServersDetail() {
	schemaConfig, err := GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	// 创建 MCP Host 来连接所有服务器
	host, err := mcpHost.NewHost(schemaConfig)
	if err != nil {
		fmt.Printf("创建 MCP Host 失败: %v\n", err)
		return
	}
	defer host.Close()

	// 使用 Host 的方法显示详细工具信息
	host.PrintListTools()
}

// listConfiguredMCPServers 列出当前配置的 MCP 服务
func listConfiguredMCPServers() {
	schemaConfig, err := GetConfig()
	if err != nil {
		log.Fatal(err)
	}
	// 打印服务列表
	services := make(map[string]interface{})
	for name, serverConfig := range schemaConfig.MCPServers {
		if !serverConfig.Disabled {
			services[name] = serverConfig
		}
	}
	printMCPServices("当前已配置的 MCP 服务:", services, nil)
}

// listAvailableMCPServers 列出所有可用的 MCP 服务
func listAvailableMCPServers() {
	// 获取 PopularMCPServers 中的所有服务
	services := make(map[string]interface{})
	for name, serverConfig := range schema.PopularMCPServers {
		services[name] = serverConfig
	}

	// 打印服务列表，并提供格式化函数显示命令
	printMCPServices("可用的 MCP 服务:", services, func(name string) string {
		serverConfig := schema.PopularMCPServers[name]
		return fmt.Sprintf("%s (命令: %s %s)", name, serverConfig.Command, strings.Join(serverConfig.Args, " "))
	})
}
