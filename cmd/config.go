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
		// 构建使用说明
		usage := lang.T(desc)
		
		// 为有限制值的配置项添加说明
		if validValues, exists := configValidValues[key]; exists {
			usage = fmt.Sprintf("%s (可选值: %v)", lang.T(desc), validValues)
		}
		
		// 为特殊类型的配置项添加额外说明
		if configType, exists := configTypes[key]; exists {
			switch configType {
			case "json":
				usage = fmt.Sprintf("%s (JSON格式，例如: '{\"key\": \"value\"}')", lang.T(desc))
				// 可以在这里添加其他类型的说明
			}
		}
		
		// 添加标志
		configCmd.Flags().String(key, config.GetConfig(key), usage)
		
		// 为有限制值的配置项添加自动补全功能
		if _, exists := configValidValues[key]; exists {
			// 注册自动补全函数
			_ = configCmd.RegisterFlagCompletionFunc(key, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				return configValidValues[key], cobra.ShellCompDirectiveNoFileComp
			})
		}
	}
}

// 验证配置值是否在允许的范围内
func validateConfigValue(key, value string) error {
	validValues, exists := configValidValues[key]
	if !exists {
		// 如果没有定义有效值列表，则认为所有值都有效
		return nil
	}

	for _, validValue := range validValues {
		if value == validValue {
			return nil
		}
	}

	return fmt.Errorf("%s 的值必须是以下之一: %v", key, validValues)
}

// 处理特殊配置项，根据配置类型进行处理
func handleSpecialConfig(key, value string) (string, error) {
	// 获取配置项的类型
	configType, exists := configTypes[key]
	if !exists {
		// 如果没有定义特殊类型，则按原样返回
		return value, nil
	}
	
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

	if showAllConfigs {
		fmt.Println(lang.T("Current configurations:"))
		for key := range configOptions {
			value := config.GetConfig(key)
			if value != "" {
				// 根据配置类型格式化输出
				configType, isSpecialType := configTypes[key]
				if isSpecialType {
					switch configType {
					case "json":
						var jsonMap map[string]interface{}
						if json.Unmarshal([]byte(value), &jsonMap) == nil {
							prettyJSON, _ := json.MarshalIndent(jsonMap, "", "  ")
							fmt.Printf("%s=\n%s\n", config.GetEnvKey(key), string(prettyJSON))
							continue
						}
					// 可以在这里添加其他类型的格式化输出
					}
				}
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
			
			config.SetConfig(key, processedValue)
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
