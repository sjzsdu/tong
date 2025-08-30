package cmd

import (
	"fmt"
	"os"

	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/share"
	"github.com/spf13/cobra"
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
	rootCmd.PersistentFlags().BoolVarP(&debugMode, "debug", "v", false, lang.T("Debug mode"))
	// 设置全局 debug 模式
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		share.SetDebug(debugMode)
	}
}
