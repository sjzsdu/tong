package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/sjzsdu/langchaingo-cn/llms"
	"github.com/sjzsdu/tong/cmdio"
	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/mcp"
	"github.com/sjzsdu/tong/prompt"
	"github.com/sjzsdu/tong/schema"
	"github.com/sjzsdu/tong/share"
	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/tools"
)

var composeCmd = &cobra.Command{
	Use:   "compose",
	Short: lang.T("AI Compose"),
	Long:  lang.T("AI Compose"),
	Run:   runCompose,
}

func init() {
	initProjectArgs(composeCmd)
	// 添加streamMode标志
	composeCmd.Flags().BoolVarP(&streamMode, "stream", "s", true, lang.T("启用流式输出模式"))
	composeCmd.Flags().StringVarP(&agentType, "type", "t", "conversation", lang.T("Compose type"))
	composeCmd.Flags().StringVarP(&configFile, "config", "c", "tong.json", lang.T("Config file"))
	composeCmd.Flags().StringVarP(&promptName, "prompt", "p", "", lang.T("Prompt name"))

	rootCmd.AddCommand(composeCmd)
}

func runCompose(cmd *cobra.Command, args []string) {
	if promptName == "" {
		promptName = "coder"
	}

	// Handle compose-specific configuration
	if len(args) > 0 {
		composeName := args[0]
		if err := handleComposeConfig(composeName); err != nil {
			log.Printf("Warning: failed to handle compose config for '%s': %v", composeName, err)
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

			return agents.NewExecutor(agent,
				agents.WithMemory(chatMemory),
				agents.WithMaxIterations(20),
			)
		})
	case "oneShotZero":
		session = cmdio.CreateAgentAdapter(streamMode, func(processor *cmdio.AgentProcessor) *agents.Executor {
			agent := agents.NewOneShotAgent(llm, schemeTools,
				agents.WithCallbacksHandler(processor.Handler),
				openAIOption.WithSystemMessage(systemPrompt))

			return agents.NewExecutor(agent,
				agents.WithMemory(chatMemory),
				agents.WithMaxIterations(20),
			)
		})
	}

	// 启动交互式会话
	err = session.Start(ctx, project)
	if err != nil {
		log.Fatalf("会话错误: %v", err)
	}
}

func handleComposeConfig(composeName string) error {
	// Get the composes directory path
	composesDir := helper.GetPath("composes")
	composeDir := filepath.Join(composesDir, composeName)

	// Create the compose directory if it doesn't exist
	if _, err := os.Stat(composeDir); os.IsNotExist(err) {
		if err := os.MkdirAll(composeDir, 0755); err != nil {
			return fmt.Errorf("failed to create compose directory %s: %v", composeDir, err)
		}
		fmt.Printf("Created compose directory: %s\n", composeDir)
	}

	// Check for tong.json in the compose directory
	composeConfigPath := filepath.Join(composeDir, share.SCHEMA_CONFIG_FILE)

	if _, err := os.Stat(composeConfigPath); os.IsNotExist(err) {
		// tong.json doesn't exist, create a default one
		if err := createDefaultComposeConfig(composeConfigPath); err != nil {
			return fmt.Errorf("failed to create default config for compose '%s': %v", composeName, err)
		}
		fmt.Printf("Created default configuration for compose '%s' at %s\n", composeName, composeConfigPath)
	}

	configFile = composeConfigPath
	return nil
}

// createDefaultComposeConfig creates a default configuration file for a compose
func createDefaultComposeConfig(configPath string) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", dir, err)
	}

	defaultConfig := schema.DefaultSchemaConfig()

	if err := defaultConfig.ToJSON(configPath); err != nil {
		return fmt.Errorf("failed to save default config to %s: %v", configPath, err)
	}

	return nil
}
