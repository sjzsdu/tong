package cmd

import (
	"fmt"
	"os"

	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/share"
	"github.com/spf13/cobra"
)

var (
	workDir         string
	extensions      []string
	outputFile      string
	excludePatterns []string
	repoURL         string
	skipGitIgnore   bool
	debugMode       bool
)

var RootCmd = rootCmd

var rootCmd = &cobra.Command{
	Use:   share.BUILDNAME,
	Short: lang.T("Tong command line tool"),
	Long:  lang.T("A versatile command line tool for development"),
	// 移除 Args 限制，允许无参数调用
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Run: func(cmd *cobra.Command, args []string) {
		// 如果没有参数，显示帮助信息
		if len(args) == 0 {
			cmd.Help()
			return
		}
		fmt.Fprintln(os.Stderr, lang.T("Invalid arguments")+": ", args)
		os.Exit(1)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// 确保在初始化时已经加载了语言包
	rootCmd.PersistentFlags().StringVarP(&workDir, "directory", "d", "", lang.T("Work directory path"))
	rootCmd.PersistentFlags().StringSliceVarP(&extensions, "extensions", "e", []string{"*"}, lang.T("File extensions to include"))
	rootCmd.PersistentFlags().StringVarP(&outputFile, "out", "o", "", lang.T("Output file name"))
	rootCmd.PersistentFlags().StringSliceVarP(&excludePatterns, "exclude", "x", []string{}, lang.T("Glob patterns to exclude"))
	rootCmd.PersistentFlags().StringVarP(&repoURL, "repository", "r", "", lang.T("Git repository URL to clone and pack"))
	rootCmd.PersistentFlags().BoolVarP(&skipGitIgnore, "no-gitignore", "n", false, lang.T("Disable .gitignore rules"))
	rootCmd.PersistentFlags().BoolVarP(&debugMode, "debug", "v", false, lang.T("Debug mode"))
	// 设置全局 debug 模式
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		share.SetDebug(debugMode)
	}
}
