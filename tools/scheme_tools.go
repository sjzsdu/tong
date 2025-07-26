package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/sjzsdu/tong/config"
	"github.com/tmc/langchaingo/tools"
)

// SchemeTool 实现了 langchaingo/tools.Tool 接口
type SchemeTool struct {
	name        string
	description string
	config      *config.SchemaConfig
	scheme      config.SchemeConfig
}

// NewSchemeTool 创建一个新的 SchemeTool
func NewSchemeTool(name string, scheme config.SchemeConfig, config *config.SchemaConfig) *SchemeTool {
	return &SchemeTool{
		name:        name,
		description: fmt.Sprintf("Tool for executing %s command", name),
		config:      config,
		scheme:      scheme,
	}
}

// Name 返回工具名称
func (t *SchemeTool) Name() string {
	return t.name
}

// Description 返回工具描述
func (t *SchemeTool) Description() string {
	return t.description
}

// Call 执行工具
func (t *SchemeTool) Call(ctx context.Context, input string) (string, error) {
	// 解析输入参数
	args := make(map[string]interface{})
	args["input"] = input

	// 获取命令参数
	command := t.scheme.Command
	cmdArgs := make([]string, len(t.scheme.Args))
	copy(cmdArgs, t.scheme.Args)

	// 替换参数中的占位符
	for i, arg := range cmdArgs {
		for k, v := range args {
			placeholder := fmt.Sprintf("{%s}", k)
			if strings.Contains(arg, placeholder) {
				cmdArgs[i] = strings.ReplaceAll(arg, placeholder, fmt.Sprintf("%v", v))
			}
		}
	}

	// 创建命令
	cmd := exec.CommandContext(ctx, command, cmdArgs...)

	// 设置环境变量
	if len(t.scheme.Env) > 0 {
		cmd.Env = t.scheme.Env
	}

	// 设置超时
	timeout := t.scheme.Timeout
	if timeout <= 0 {
		timeout = 30 // 默认30秒超时
	}

	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// 使用带超时的上下文创建新命令
	cmd = exec.CommandContext(timeoutCtx, command, cmdArgs...)
	
	// 执行命令并获取输出
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("执行命令失败: %v, 输出: %s", err, string(output))
	}

	return string(output), nil
}

// CreateSchemeTools 从 SchemaConfig 创建工具列表
func CreateSchemeTools(config *config.SchemaConfig) []tools.Tool {
	if config == nil || len(config.MCPServers) == 0 {
		return nil
	}

	var schemeTools []tools.Tool

	// 遍历所有服务器配置
	for name, scheme := range config.MCPServers {
		// 跳过禁用的工具
		if scheme.Disabled {
			continue
		}

		// 创建工具并添加到列表
		tool := NewSchemeTool(name, scheme, config)
		schemeTools = append(schemeTools, tool)
	}

	return schemeTools
}