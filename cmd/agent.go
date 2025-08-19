package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/sjzsdu/langchaingo-cn/llms"
	"github.com/sjzsdu/tong/cmdio"
	"github.com/sjzsdu/tong/config"
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
	initProjectArgs(agentCmd)
	// 添加streamMode标志
	agentCmd.Flags().BoolVarP(&streamMode, "stream", "s", true, lang.T("启用流式输出模式"))
	agentCmd.Flags().StringVarP(&agentType, "type", "t", "conversation", lang.T("Agent type"))
	agentCmd.Flags().StringVarP(&configFile, "config", "c", "tong.json", lang.T("Config file"))
	agentCmd.Flags().StringVarP(&promptName, "prompt", "p", "", lang.T("Prompt name"))

	rootCmd.AddCommand(agentCmd)
}

func runAgent(cmd *cobra.Command, args []string) {
	if promptName == "" {
		promptName = "coder"
	}

	// Handle agent-specific configuration
	if len(args) > 0 {
		agentName := args[0]
		if err := handleAgentConfig(agentName); err != nil {
			log.Printf("Warning: failed to handle agent config for '%s': %v", agentName, err)
		}
	}

	// 获取配置
	config, err := GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	// 获取项目
	project, err := GetProject()
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
	err = session.Start(ctx, project)
	if err != nil {
		log.Fatalf("会话错误: %v", err)
	}
}

func handleAgentConfig(agentName string) error {
	// Get the agents directory path
	agentsDir := helper.GetPath("agents")
	agentDir := filepath.Join(agentsDir, agentName)

	// Create the agent directory if it doesn't exist
	if _, err := os.Stat(agentDir); os.IsNotExist(err) {
		if err := os.MkdirAll(agentDir, 0755); err != nil {
			return fmt.Errorf("failed to create agent directory %s: %v", agentDir, err)
		}
		fmt.Printf("Created agent directory: %s\n", agentDir)
	}

	// Check for tong.json in the agent directory
	agentConfigPath := filepath.Join(agentDir, share.SCHEMA_CONFIG_FILE)

	if _, err := os.Stat(agentConfigPath); os.IsNotExist(err) {
		// tong.json doesn't exist, create a default one
		if err := createDefaultAgentConfig(agentConfigPath); err != nil {
			return fmt.Errorf("failed to create default config for agent '%s': %v", agentName, err)
		}
		fmt.Printf("Created default configuration for agent '%s' at %s\n", agentName, agentConfigPath)
	}

	configFile = agentConfigPath
	return nil
}

// createDefaultAgentConfig creates a default configuration file for an agent
func createDefaultAgentConfig(configPath string) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", dir, err)
	}

	defaultConfig := config.DefaultSchemaConfig()

	if err := defaultConfig.ToJSON(configPath); err != nil {
		return fmt.Errorf("failed to save default config to %s: %v", configPath, err)
	}

	return nil
}
