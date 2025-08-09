package cmd

import (
	"fmt"
	"os"

	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/lang"
)

var (
	workDir         string
	extensions      []string
	excludePatterns []string
	repoURL         string
	skipGitIgnore   bool
	debugMode       bool

	promptName  string
	content     string
	contentFile string

	configOptions = map[string]string{
		"lang":                 "Set language",
		"renderer":             "Set llm response render type",
		"master_llm":           "Set master llm",
		"master_llm_params":    "Set master llm params",
		"embedding_llm":        "Set master llm",
		"embedding_llm_params": "Set embedding model params",
	}

	// 配置项的有效值映射
	configValidValues = map[string][]string{
		"master_llm":    {"deepseek", "qwen", "kimi"},
		"embedding_llm": {"deepseek", "qwen", "kimi"},
		"renderer":      {"text", "markdown"},
	}

	// 配置项的类型映射，用于特殊处理不同类型的配置
	configTypes = map[string]string{
		"master_llm_params":    "json",
		"embedding_llm_params": "json",
	}

	showAllConfigs bool
	configFile     string
	streamMode     bool
	agentType      string

	mcpPort   int
	showTools bool
)

func getContent() string {
	// 如果指定了文件，从文件读取
	if contentFile != "" {
		data, err := os.ReadFile(contentFile)
		if err != nil {
			fmt.Printf(lang.T("Failed to read file: %v\n"), err)
			return ""
		}
		return string(data)
	}

	// 如果指定了内容，直接使用
	if content != "" {
		return content
	}

	input, _ := helper.InputString("> ")
	return input
}
