package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/mcpserver"
	"github.com/spf13/cobra"
)

var (
	mcpPort   int
	showTools bool
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
	// 如果设置了显示工具选项，则输出可用工具列表并退出
	if showTools {
		displayAvailableTools()
		return
	}

	doc, err := GetProject()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	// 创建 MCP 服务器
	mcpServer, err := mcpserver.NewTongMCPServer(doc)
	if err != nil {
		log.Fatalf("Error creating MCP server: %v", err)
	}

	// 创建 HTTP 服务器
	httpServer := server.NewStreamableHTTPServer(mcpServer,
		server.WithEndpointPath("/mcp"),
	)

	// 创建取消上下文用于优雅退出
	cancel := make(chan struct{})
	defer close(cancel)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping server...")
		// 关闭服务器
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error shutting down server: %v", err)
		}
		close(cancel)
	}()

	// 启动服务器
	log.Printf("MCP Server started on http://localhost:%d/mcp", mcpPort)
	log.Printf("提示: 使用 'tong mcp --list' 查看所有可用的 MCP 工具")
	if err := httpServer.Start(fmt.Sprintf(":%d", mcpPort)); err != nil {
		if err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}

	log.Println("Server stopped gracefully")
}

// displayAvailableTools 显示MCP服务器提供的所有工具
func displayAvailableTools() {
	tools := mcpserver.GetAvailableTools()

	fmt.Println("\n=== MCP 服务器提供的工具 ===\n")

	// 按类别组织工具
	fileTools := []map[string]string{}
	editorTools := []map[string]string{}
	projectTools := []map[string]string{}

	// 分类工具
	for _, tool := range tools {
		name := tool["name"]
		switch {
		case name == "listFiles" || name == "readFile" || name == "writeFile" ||
			name == "createFile" || name == "createDirectory" || name == "deleteFile":
			fileTools = append(fileTools, tool)
		case name == "findText" || name == "replaceText" || name == "formatCode":
			editorTools = append(editorTools, tool)
		default:
			projectTools = append(projectTools, tool)
		}
	}

	// 显示文件操作工具
	fmt.Println("文件操作工具:")
	for _, tool := range fileTools {
		fmt.Printf("  - %-15s: %s\n", tool["name"], tool["description"])
	}

	// 显示编辑器工具
	fmt.Println("\n编辑器工具:")
	for _, tool := range editorTools {
		fmt.Printf("  - %-15s: %s\n", tool["name"], tool["description"])
	}

	// 显示项目工具
	fmt.Println("\n项目工具:")
	for _, tool := range projectTools {
		fmt.Printf("  - %-15s: %s\n", tool["name"], tool["description"])
	}

	fmt.Println("\n使用示例:")
	fmt.Println("  1. 启动 MCP 服务器:")
	fmt.Printf("     tong mcp -p %d\n", mcpPort)
	fmt.Println("  2. MCP 客户端可以通过以下地址访问服务器:")
	fmt.Printf("     http://localhost:%d/mcp\n", mcpPort)
	fmt.Println("  3. 工具使用方式取决于 MCP 客户端的实现")
	fmt.Println()
}
