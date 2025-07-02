package cmd

import (
	"fmt"

	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/share"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: lang.T("Print version information"),
	Long:  lang.T("Print detailed version information of tong"),
	Run: func(cmd *cobra.Command, args []string) {
		// 使用简单的字符串拼接替代模板
		fmt.Printf("%s: %s\n", lang.T("tong version"), share.VERSION)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
