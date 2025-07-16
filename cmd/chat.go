package cmd

import (
	"fmt"

	"github.com/sjzsdu/tong/lang"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: lang.T("Chat to the project"),
	Long:  lang.T("Chat to the project"),
	Run:   runChat,
}

func init() {
	rootCmd.AddCommand(chatCmd)
}

func runChat(cmd *cobra.Command, args []string) {

	_, err := GetProject()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
}
