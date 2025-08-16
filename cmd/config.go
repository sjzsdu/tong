package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/sjzsdu/tong/config"
	"github.com/sjzsdu/tong/lang"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: lang.T("Configuration commands"),
	Long:  lang.T("Set or get global configuration"),
	Run:   handleConfigCommand,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: lang.T("Set configuration value"),
	Long:  lang.T("Set global configuration value for a specific key"),
	Args:  cobra.MinimumNArgs(1),
	Run:   handleConfigSetCommand,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: lang.T("Get configuration value"),
	Long:  lang.T("Get global configuration value for a specific key"),
	Args:  cobra.ExactArgs(1),
	Run:   handleConfigGetCommand,
}

func init() {
	if config.GetConfig(config.KeyLang) == "" {
		config.SetConfig(config.KeyLang, "en")
	}

	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.Flags().BoolVarP(&showAllConfigs, "list", "l", false, lang.T("List all configurations"))
	
	// 添加显示可用配置键的命令
	configCmd.Flags().BoolP("help-keys", "k", false, lang.T("Show all available configuration keys"))
	
	// 设置运行时的帮助
	configSetCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		cmd.Parent().Flags().Lookup("help-keys").Value.Set("true")
		cmd.Parent().Help()
	})
	
	configGetCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		cmd.Parent().Flags().Lookup("help-keys").Value.Set("true")
		cmd.Parent().Help()
	})

	// 准备所有配置键用于自动补全
	allConfigKeys := config.GetAllConfigKeys()

	// 为 set 和 get 子命令添加配置键的自动补全
	_ = configSetCmd.RegisterFlagCompletionFunc("key", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return allConfigKeys, cobra.ShellCompDirectiveNoFileComp
	})

	_ = configGetCmd.RegisterFlagCompletionFunc("key", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return append(allConfigKeys, "all"), cobra.ShellCompDirectiveNoFileComp
	})

	// 为 set 子命令添加配置值的自动补全
	_ = configSetCmd.RegisterFlagCompletionFunc("value", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			key := args[0]
			options := config.GetConfigOptions(key)
			if len(options) > 0 {
				return options, cobra.ShellCompDirectiveNoFileComp
			}
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	})
}

// 验证配置值是否在允许的范围内
func validateConfigValue(key, value string) error {
	// 使用 config 包中的 IsValidConfigOption 函数
	if !config.IsValidConfigOption(key, value) {
		options := config.GetConfigOptions(key)
		return fmt.Errorf("%s 的值必须是以下之一: %v", key, options)
	}
	return nil
}

// 处理特殊配置项，根据配置类型进行处理
func handleSpecialConfig(key, value string) (string, error) {
	// 获取配置项的类型
	configType := config.GetConfigType(key)
	
	// 根据配置类型进行处理
	switch configType {
	case "json":
		// 验证是否是有效的JSON
		var jsonMap map[string]interface{}
		if err := json.Unmarshal([]byte(value), &jsonMap); err != nil {
			return "", fmt.Errorf("%s 必须是有效的JSON格式: %v", key, err)
		}
		// 返回格式化后的JSON字符串
		formattedJSON, err := json.Marshal(jsonMap)
		if err != nil {
			return "", err
		}
		return string(formattedJSON), nil
	// 可以在这里添加其他类型的处理
	default:
		return value, nil
	}
}

func handleConfigCommand(cmd *cobra.Command, args []string) {
	if err := config.LoadConfig(); err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	// 如果没有指定子命令，且使用了 --list 标志，则显示所有配置
	if showAllConfigs {
		showAllConfigValues()
		return
	}

	// 如果使用了 --help-keys 标志，显示所有可用的配置键
	helpKeys, _ := cmd.Flags().GetBool("help-keys")
	if helpKeys {
		cmd.Help()
		fmt.Println("\n" + lang.T("Available configuration keys:"))
		showAvailableConfigKeys()
		return
	}

	// 没有子命令且没有指定标志，显示使用说明
	cmd.Help()
}

// 显示所有可用的配置键
func showAvailableConfigKeys() {
	for key, info := range config.ConfigKeys {
		usage := lang.T(info.Description)
		
		// 为有限制值的配置项添加说明
		if len(info.Options) > 0 {
			usage = fmt.Sprintf("%s (可选值: %v)", usage, info.Options)
		}
		
		// 为特殊类型的配置项添加额外说明
		if info.Type != "string" {
			switch info.Type {
			case "json":
				usage = fmt.Sprintf("%s (JSON格式)", usage)
			}
		}
		
		fmt.Printf("  %s - %s\n", key, usage)
	}
}

// 显示所有配置项的值
func showAllConfigValues() {
	fmt.Println(lang.T("Current configurations:"))
	for key := range config.ConfigKeys {
		value := config.GetConfig(key)
		if value != "" {
			// 根据配置类型格式化输出
			configType := config.GetConfigType(key)
			if configType != "string" {
				switch configType {
				case "json":
					var jsonMap map[string]interface{}
					if json.Unmarshal([]byte(value), &jsonMap) == nil {
						prettyJSON, _ := json.MarshalIndent(jsonMap, "", "  ")
						fmt.Printf("%s=\n%s\n", key, string(prettyJSON))
						continue
					}
					// 可以在这里添加其他类型的格式化输出
				}
			}
			fmt.Printf("%s=%s\n", key, value)
		}
	}
}

// 处理 config set 命令
func handleConfigSetCommand(cmd *cobra.Command, args []string) {
	if err := config.LoadConfig(); err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	// 需要至少一个参数：键名
	if len(args) < 1 {
		fmt.Println("Error: key name is required")
		cmd.Help()
		return
	}

	key := args[0]
	var value string

	// 如果有第二个参数，则为值
	if len(args) >= 2 {
		value = args[1]
	}

	// 验证键名是否有效
	if _, exists := config.ConfigKeys[key]; !exists {
		fmt.Printf("Error: unknown config key '%s'\n", key)
		fmt.Println(lang.T("Available configuration keys:"))
		showAvailableConfigKeys()
		return
	}

	// 验证配置值
	if err := validateConfigValue(key, value); err != nil {
		fmt.Println("Error:", err)
		return
	}

	// 处理特殊配置项
	processedValue, err := handleSpecialConfig(key, value)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// 设置配置值
	config.SetConfig(key, processedValue)

	// 保存配置
	if err := config.SaveConfig(); err != nil {
		fmt.Println("Error saving config:", err)
		return
	}

	fmt.Printf("%s has been set to %s\n", key, value)
}

// 处理 config get 命令
func handleConfigGetCommand(cmd *cobra.Command, args []string) {
	if err := config.LoadConfig(); err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	// 需要一个参数：键名
	if len(args) != 1 {
		fmt.Println("Error: key name is required")
		cmd.Help()
		return
	}

	key := args[0]

	// 如果请求的是所有配置项
	if key == "all" {
		showAllConfigValues()
		return
	}

	// 验证键名是否有效
	if _, exists := config.ConfigKeys[key]; !exists {
		fmt.Printf("Error: unknown config key '%s'\n", key)
		fmt.Println(lang.T("Available configuration keys:"))
		showAvailableConfigKeys()
		return
	}

	// 获取配置值
	value := config.GetConfig(key)

	// 根据配置类型格式化输出
	configType := config.GetConfigType(key)
	if configType != "string" && value != "" {
		switch configType {
		case "json":
			var jsonMap map[string]interface{}
			if json.Unmarshal([]byte(value), &jsonMap) == nil {
				prettyJSON, _ := json.MarshalIndent(jsonMap, "", "  ")
				fmt.Printf("%s=\n%s\n", key, string(prettyJSON))
				return
			}
		// 可以在这里添加其他类型的格式化输出
		}
	}
	
	if value == "" {
		fmt.Printf("%s is not set\n", key)
	} else {
		fmt.Printf("%s=%s\n", key, value)
	}
}
