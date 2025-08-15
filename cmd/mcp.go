package cmd

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/server"
	configPackage "github.com/sjzsdu/tong/config"
	"github.com/sjzsdu/tong/lang"
	mcpHost "github.com/sjzsdu/tong/mcp"
	"github.com/sjzsdu/tong/mcpserver"
	"github.com/spf13/cobra"
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
	mcpDebug     bool
)

func init() {
	rootCmd.AddCommand(mcpCmd)

	// 添加命令行标志
	mcpCmd.Flags().StringVar(&mcpTransport, "transport", "", "传输方式 (stdio, http, sse)，默认为 stdio")
	mcpCmd.Flags().StringVar(&mcpPortFlag, "port", "8080", "HTTP/SSE 服务器端口，默认为 8080")
	mcpCmd.Flags().BoolVar(&mcpDebug, "debug", false, "启用调试日志")
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
	default:
		fmt.Println("未知的操作类型: " + args[0])
		cmd.Help()
	}
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

	// 如果启用了调试模式
	if mcpDebug {
		fmt.Printf("调试模式已启用\n")
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
	for name, serverConfig := range configPackage.PopularMCPServers {
		services[name] = serverConfig
	}

	// 打印服务列表，并提供格式化函数显示命令
	printMCPServices("可用的 MCP 服务:", services, func(name string) string {
		serverConfig := configPackage.PopularMCPServers[name]
		return fmt.Sprintf("%s (命令: %s %s)", name, serverConfig.Command, strings.Join(serverConfig.Args, " "))
	})
}
