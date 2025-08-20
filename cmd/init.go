package cmd

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/schema"
	"github.com/sjzsdu/tong/share"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: lang.T("Init tong project"),
	Long:  lang.T("Init tong project"),
	Run:   handleInitCommand,
}

func init() {
	// 添加streamMode标志
	initCmd.PersistentFlags().StringVarP(&workDir, "directory", "d", ".", lang.T("Work directory path"))
	rootCmd.AddCommand(initCmd)
}

func handleInitCommand(cmd *cobra.Command, args []string) {

	// 检查是否已存在配置文件
	configPath := filepath.Join(workDir, share.SCHEMA_CONFIG_FILE)

	// 获取默认配置
	config, err := GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	// 处理命令行参数，添加对应的 MCP 配置
	for _, key := range args {
		if mcp, ok := schema.PopularMCPServers[key]; ok {
			config.MCPServers[key] = mcp
		}
	}

	// 使用 ToJSON 方法将配置写入文件
	err = config.ToJSON(configPath)
	if err != nil {
		fmt.Printf(lang.T("写入配置文件失败")+": %v\n", err)
		return
	}

	fmt.Println(lang.T("配置文件已创建") + ": " + configPath)
}
