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
	"github.com/sjzsdu/tong/prompt"
	"github.com/sjzsdu/tong/share"
	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/tools"
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
	agentCmd.Flags().StringVarP(&promptName, "prompt", "p", "", lang.T("Prompt name"))

	rootCmd.AddCommand(agentCmd)
}

func runAgent(cmd *cobra.Command, args []string) {
	if promptName == "" {
		promptName = "coder"
	}
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
		// 如果 MCP 初始化失败，打印错误但继续执行
		fmt.Printf("警告: MCP 服务初始化失败: %v\n", err)
		// 创建一个空的 Host 实例
		host = &mcp.Host{Clients: make(map[string]*mcp.Client)}
	}

	ctx := context.Background()
	// 创建基于 SchemeConfig 的工具
	schemeTools, err := host.GetTools(ctx)
	if err != nil {
		fmt.Printf("警告: 获取 MCP 工具失败: %v\n将继续执行但功能可能受限\n", err)
		schemeTools = []tools.Tool{}
	}

	systemPrompt := prompt.ShowPromptContent(promptName)

	// 打印可用工具列表
	if share.GetDebug() {
		helper.PrintWithLabel("可用工具:", schemeTools)
		helper.PrintWithLabel("提示词:", systemPrompt)
	}

	chatMemory := memory.NewConversationBuffer()
	openAIOption := agents.NewOpenAIOption()
	var session *cmdio.InteractiveSession

	switch agentType {
	case "conversation":
		session = cmdio.CreateAgentAdapter(streamMode, func(processor *cmdio.AgentProcessor) *agents.Executor {
			agent := agents.NewConversationalAgent(llm, schemeTools,
				agents.WithCallbacksHandler(processor.Handler),
				openAIOption.WithSystemMessage(systemPrompt))

			return agents.NewExecutor(agent, agents.WithMemory(chatMemory))
		})
	case "oneShotZero":
		session = cmdio.CreateAgentAdapter(streamMode, func(processor *cmdio.AgentProcessor) *agents.Executor {
			agent := agents.NewOneShotAgent(llm, schemeTools,
				agents.WithCallbacksHandler(processor.Handler),
				openAIOption.WithSystemMessage(systemPrompt))

			return agents.NewExecutor(agent, agents.WithMemory(chatMemory))
		})
	}

	// 启动交互式会话
	err = session.Start(ctx)
	if err != nil {
		log.Fatalf("会话错误: %v", err)
	}
}
