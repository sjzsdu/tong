package cmd

import (
	"fmt"

	"github.com/sjzsdu/tong/config"
	"github.com/sjzsdu/tong/lang"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: lang.T("Set config"),
	Long:  lang.T("Set global configuration"),
	Run:   runConfig,
}

var (
	flagKeys = map[string]string{
		"lang":             "Set language",
		"render":           "Set llm response render type",
		"default_provider": "Set default LLM provider",
		"default_agent":    "Set default agent",
		"deepseek_apikey":  "Set DeepSeek API Key",
		"deepseek_model":   "Set DeepSeek default model",
		"openai_apikey":    "Set Openai API Key",
		"openai_model":     "Set Openai default model",
		"claude_apikey":    "Set Claude API Key",
		"claude_model":     "Set Claude default model",
		"qwen_apikey":      "Set Qwen API Key",
		"qwen_model":       "Set Qwen default model",
	}
	listFlag bool
)

func init() {
	if config.GetConfig("lang") == "" {
		config.SetConfig("lang", "en")
	}

	rootCmd.AddCommand(configCmd)
	configCmd.Flags().BoolVar(&listFlag, "list", false, lang.T("List all configurations"))

	// 通过遍历 flagKeys 自动添加所有配置项
	for key, desc := range flagKeys {
		configCmd.Flags().String(key, config.GetConfig(key), lang.T(desc))
	}
}

func runConfig(cmd *cobra.Command, args []string) {
	if err := config.LoadConfig(); err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	if listFlag {
		fmt.Println(lang.T("Current configurations:"))
		for key := range flagKeys {
			value := config.GetConfig(key)
			if value != "" {
				fmt.Printf("%s=%s\n", config.GetEnvKey(key), value)
			}
		}
		return
	}

	configChanged := false
	// 处理 flagKeys 中的标准配置项
	for key := range flagKeys {
		flag := cmd.Flag(key)
		if flag != nil && flag.Changed {
			value, _ := cmd.Flags().GetString(key)
			config.SetConfig(key, value)
			configChanged = true
		}
	}

	// 特殊处理 default_provider 标志，将其映射到 llm 配置项
	defaultProviderFlag := cmd.Flag("default_provider")
	if defaultProviderFlag != nil && defaultProviderFlag.Changed {
		value, err := cmd.Flags().GetString("default_provider")
		if err == nil {
			envKey := config.GetEnvKey("default_provider")
			if envKey != "" {
				config.SetConfig(envKey, value)
				configChanged = true
			}
		}
	}

	if configChanged {
		if err := config.SaveConfig(); err != nil {
			fmt.Println("Error saving config:", err)
			return
		}
	}
}
