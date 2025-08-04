package cmd

import (
	"fmt"

	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/prompt"
	"github.com/spf13/cobra"
)

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: lang.T("Prompt management"),
	Long:  lang.T("Prompt management"),
	Run:   runPrompt,
}

func init() {
	promptCmd.Flags().StringVar(&promptName, "name", "", lang.T("Prompt name"))
	promptCmd.Flags().StringVar(&content, "content", "", lang.T("Prompt content"))
	promptCmd.Flags().StringVar(&contentFile, "file", "", lang.T("Read content from file"))
	rootCmd.AddCommand(promptCmd)
}

func runPrompt(cmd *cobra.Command, args []string) {

	// 检查参数是否存在
	if len(args) == 0 {
		fmt.Println("请指定操作类型: list, add, delete, show")
		cmd.Help()
		return
	}

	if len(args) > 1 {
		promptName = args[1]
	}

	switch args[0] {
	case "list":
		prompt.ListPrompts()
	case "add":
		prompt.SavePrompt(promptName, getContent())
	case "delete":
		prompt.DeleteExistingPrompt(promptName)
	case "show":
		promptContent := prompt.ShowPromptContent(promptName)
		fmt.Print(promptContent)
	default:
		fmt.Println("未知的操作类型: " + args[0])
		cmd.Help()
	}
}
