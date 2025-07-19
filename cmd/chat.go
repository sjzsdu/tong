package cmd

import (
	"context"
	"log"

	"github.com/sjzsdu/langchaingo-cn/llms"
	"github.com/sjzsdu/tong/cmdio"
	"github.com/sjzsdu/tong/lang"
	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/memory"
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

	// Initialize LLM
	llm, err := llms.CreateLLM(llms.DeepSeekLLM, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Create conversation memory
	chatMemory := memory.NewConversationBuffer()

	// Create conversation chain
	chain := chains.NewConversation(llm, chatMemory)

	// 创建交互式会话适配器
	session := cmdio.CreateChatAdapter(chain, true) // 使用流式模式

	// 启动交互式会话
	ctx := context.Background()
	err = session.Start(ctx)
	if err != nil {
		log.Fatalf("会话错误: %v", err)
	}

	// _, err := GetProject()
	// if err != nil {
	// 	fmt.Printf("%v\n", err)
	// 	return
	// }
}
