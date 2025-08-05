package cmd

import (
	"fmt"
	"log"
	"sort"
	"strings"

	configPackage "github.com/sjzsdu/tong/config"
	"github.com/sjzsdu/tong/lang"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: lang.T("MCP Server"),
	Long:  lang.T("MCP Server for managing project files"),
	Run:   runMCP,
}

func init() {
	rootCmd.AddCommand(mcpCmd)

	// 添加命令行参数
	mcpCmd.Flags().IntVarP(&mcpPort, "port", "p", 8080, "Port to run the MCP server on")
	mcpCmd.Flags().BoolVarP(&showTools, "list", "l", false, "List all available MCP tools")
}

func runMCP(cmd *cobra.Command, args []string) {
	// 检查参数是否存在
	if len(args) == 0 {
		fmt.Println("请指定操作类型: available, list")
		cmd.Help()
		return
	}

	switch args[0] {
	case "list":
		// 列出当前配置的 MCP 服务
		listConfiguredMCPServers()
	case "available":
		// 列出所有可用的 MCP 服务
		listAvailableMCPServers()
	default:
		fmt.Println("未知的操作类型: " + args[0])
		cmd.Help()
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
