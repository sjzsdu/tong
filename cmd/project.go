package cmd

import (
	"fmt"

	projectSubcommand "github.com/sjzsdu/tong/cmd/project"
	"github.com/sjzsdu/tong/lang"
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "项目管理工具",
	Long: `project 命令提供了一系列项目管理功能，包括文件树状结构显示、项目打包等。

可用的子命令：
  tree    显示项目目录的树状结构
  pack    打包项目文件
  search  搜索项目节点
  blame   统计作者/时间粒度的提交变更
  rag     基于项目节点索引并检索文档
  profile 项目画像与规模分级
  index   构建/更新语义索引
  doc     生成基础文档
  ask     基于RAG的问答

示例：
  tong project tree                    # 显示当前目录的树状结构
  tong project tree --stats            # 显示树状结构和统计信息`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// 在执行任何子命令之前，先创建项目实例
		proj, err := GetProject()
		if err != nil {
			fmt.Printf("创建项目实例失败: %v\n", err)
			return
		}
		// 将项目实例设置到子命令的 project 包中
		projectSubcommand.SetSharedProject(proj)
	},
	Run: runproject,
}

func init() {
	rootCmd.AddCommand(projectCmd)

	// 添加子命令
	projectCmd.AddCommand(projectSubcommand.TreeCmd)
	projectCmd.AddCommand(projectSubcommand.PackCmd)
	projectCmd.AddCommand(projectSubcommand.SearchCmd)
	projectCmd.AddCommand(projectSubcommand.BlameCmd)
	projectCmd.AddCommand(projectSubcommand.ProfileCmd)
	projectCmd.AddCommand(projectSubcommand.IndexCmd)
	projectCmd.AddCommand(projectSubcommand.DocCmd)
	projectCmd.AddCommand(projectSubcommand.AskCmd)

	initProjectArgs(projectCmd)
}

func initProjectArgs(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&workDir, "directory", "d", ".", lang.T("Work directory path"))
	cmd.PersistentFlags().StringSliceVarP(&extensions, "extensions", "e", []string{"*"}, lang.T("File extensions to include"))
	cmd.PersistentFlags().StringSliceVarP(&excludePatterns, "exclude", "x", []string{}, lang.T("Glob patterns to exclude"))
	cmd.PersistentFlags().StringVarP(&repoURL, "repository", "r", "", lang.T("Git repository URL to clone and pack"))
	cmd.PersistentFlags().BoolVarP(&skipGitIgnore, "no-gitignore", "n", false, lang.T("Disable .gitignore rules"))
}

func runproject(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		// 如果没有参数，显示帮助信息
		cmd.Help()
		return
	}

	switch args[0] {
	default:
		fmt.Println("支持的操作: pack, code, deps, quality, search, blame, rag, profile, index, doc, ask")
	}
}
