package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/sjzsdu/langchaingo-cn/llms"
	"github.com/sjzsdu/tong/cmdio"
	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/mcp"
	"github.com/sjzsdu/tong/share"
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: lang.T("AI Agent"),
	Long:  lang.T("AI Agent"),
	Run:   runAgent,
}

func init() {
	// 添加streamMode标志
	agentCmd.Flags().BoolVarP(&streamMode, "stream", "s", true, lang.T("启用流式输出模式"))
	agentCmd.Flags().StringVarP(&agentType, "type", "t", "conversation", lang.T("Agent type"))
	agentCmd.Flags().StringVarP(&configFile, "config", "c", "tong.json", lang.T("Config file"))
	agentCmd.Flags().StringVarP(&workDir, "directory", "d", ".", lang.T("Work directory path"))
	agentCmd.Flags().StringVarP(&repoURL, "repository", "r", "", lang.T("Git repository URL to clone and pack"))
	promptCmd.Flags().StringVar(&promptName, "prompt", "coder", lang.T("Prompt name"))

	rootCmd.AddCommand(agentCmd)
}

func runAgent(cmd *cobra.Command, args []string) {

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

	host, err := mcp.NewHost(config)
	if err != nil {
		log.Fatal(err)
	}

	switch agentType {
	case "conversation":
		ctx := context.Background()
		// 创建基于 SchemeConfig 的工具
		schemeTools, err := host.GetTools(ctx)
		if err != nil {
			log.Fatal(err)
		}

		// 打印可用工具列表
		if share.GetDebug() && len(schemeTools) > 0 {
			fmt.Println(lang.T("可用工具列表:"))
			for _, tool := range schemeTools {
				fmt.Printf("- %s: %s\n", tool.Name(), tool.Description())
			}
			fmt.Println()
		}

		// 创建交互式会话适配器
		session := cmdio.CreateAgentAdapter(llm, promptName, schemeTools, streamMode)

		// 启动交互式会话
		err = session.Start(ctx)
		if err != nil {
			log.Fatalf("会话错误: %v", err)
		}
		return
	}
}
