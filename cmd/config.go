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
	Run:   handleConfigCommand,
}

func init() {
	if config.GetConfig("lang") == "" {
		config.SetConfig("lang", "en")
	}

	rootCmd.AddCommand(configCmd)
	configCmd.Flags().BoolVarP(&showAllConfigs, "list", "l", false, lang.T("List all configurations"))

	// 通过遍历 configOptions 自动添加所有配置项
	for key, desc := range configOptions {
		configCmd.Flags().String(key, config.GetConfig(key), lang.T(desc))
	}
}

func handleConfigCommand(cmd *cobra.Command, args []string) {
	if err := config.LoadConfig(); err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	if showAllConfigs {
		fmt.Println(lang.T("Current configurations:"))
		for key := range configOptions {
			value := config.GetConfig(key)
			if value != "" {
				fmt.Printf("%s=%s\n", config.GetEnvKey(key), value)
			}
		}
		return
	}

	configChanged := false
	// 处理 configOptions 中的标准配置项
	for key := range configOptions {
		flag := cmd.Flag(key)
		if flag != nil && flag.Changed {
			value, _ := cmd.Flags().GetString(key)
			config.SetConfig(key, value)
			configChanged = true
		}
	}

	// 特殊处理 provider 标志，将其映射到 llm 配置项
	providerFlag := cmd.Flag("provider")
	if providerFlag != nil && providerFlag.Changed {
		value, err := cmd.Flags().GetString("provider")
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
