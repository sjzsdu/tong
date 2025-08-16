package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/project"
)

// RegisterComputerTools 注册与计算机相关的工具
func RegisterComputerTools(s *server.MCPServer, proj *project.Project) {
	if s == nil || proj == nil {
		return
	}

	// 注册执行命令工具
	toolRunCommand := mcp.NewTool(
		"run_command",
		mcp.WithDescription("执行shell命令并返回结果"),
		mcp.WithString("command", mcp.Required(), mcp.Description("要执行的shell命令")),
		mcp.WithString("workDir", mcp.Description("命令执行的工作目录，默认返回'.'")),
	)

	hRunCommand := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return runCommand(ctx, proj, req)
	}
	s.AddTool(toolRunCommand, hRunCommand)
}

// runCommand 执行命令并返回结果
func runCommand(ctx context.Context, proj *project.Project, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 获取命令参数
	command, found := helper.GetStringFromRequest(req, "command", "")
	if !found {
		return mcp.NewToolResultError("missing or invalid command parameter: required argument \"command\" not found"), nil
	}

	// 获取工作目录参数（可选）
	workDir := proj.GetRootPath() // 默认为项目根目录
	wd, _ := helper.GetStringFromRequest(req, "workDir", "")
	if wd != "" {
		// 确保工作目录是项目内的目录
		wd = proj.NormalizePath(wd)
		if node, err := proj.FindNode(wd); err == nil && node.IsDir {
			// 转换为实际文件系统路径
			workDir = node.Path
			// 如果是相对路径，转换为绝对路径
			if !strings.HasPrefix(workDir, "/") {
				workDir = proj.GetRootPath() + "/" + workDir
			}
		} else {
			return mcp.NewToolResultError(fmt.Sprintf("无效的工作目录: %s", wd)), nil
		}
	}

	// 分割命令和参数
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return mcp.NewToolResultError("命令不能为空"), nil
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = workDir

	// 执行命令并获取输出
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	if err != nil {
		result := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"output":  outputStr,
		}
		// 使用JSON格式化结果
		jsonResult, jsonErr := json.Marshal(result)
		if jsonErr != nil {
			return mcp.NewToolResultError(fmt.Sprintf("执行失败: %v, 输出: %s", err, outputStr)), nil
		}
		return mcp.NewToolResultText(string(jsonResult)), nil
	}

	// 返回成功结果
	result := map[string]interface{}{
		"success": true,
		"output":  outputStr,
	}
	// 使用JSON格式化结果
	jsonResult, jsonErr := json.Marshal(result)
	if jsonErr != nil {
		return mcp.NewToolResultText(outputStr), nil
	}
	return mcp.NewToolResultText(string(jsonResult)), nil
}
