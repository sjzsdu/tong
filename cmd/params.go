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
