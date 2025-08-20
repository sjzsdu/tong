package project

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/rag"
	"github.com/sjzsdu/tong/schema"
	"github.com/spf13/cobra"
)

var AskCmd = &cobra.Command{
	Use:   "ask",
	Short: lang.T("对项目进行RAG问答"),
	Long:  lang.T("基于已构建的索引进行问答，未索引时尝试索引后再问答"),
	Run:   runAsk,
}

func init() {}

func runAsk(cmd *cobra.Command, args []string) {
	if sharedProject == nil {
		fmt.Println("错误: 未找到共享的项目实例")
		os.Exit(1)
	}
	cfg, err := schema.LoadMCPConfig(sharedProject.GetRootPath(), "")
	if err != nil {
		fmt.Printf("获取配置失败: %v\n", err)
		os.Exit(1)
	}

	// 使用默认 RAG 选项并指向项目根
	options := rag.GetDefaultOptions()
	options.Storage.CollectionName = sharedProject.GetName()
	options.DocsDir = sharedProject.GetRootPath()

	llm, embed, _ := initializeModels(cfg)
	ctx := context.Background()
	r, err := rag.InitializeFromConfig(ctx, llm, embed, options)
	if err != nil {
		fmt.Printf("RAG 初始化失败: %v\n", err)
		os.Exit(1)
	}

	indexed, _ := r.IsIndexed(options.Storage.CollectionName)
	if !indexed {
		fmt.Println(lang.T("尚未索引，开始索引..."))
		if err := r.IndexDocuments(ctx, options.DocsDir); err != nil {
			fmt.Printf("索引失败: %v\n", err)
			os.Exit(1)
		}
	}

	// 单次问答或交互模式
	if len(args) > 0 {
		ans, err := r.Query(ctx, args[0])
		if err != nil {
			fmt.Printf("查询失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(ans)
		return
	}

	fmt.Println(lang.T("进入交互式问答（Ctrl+C 退出）"))
	s := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !s.Scan() {
			break
		}
		q := s.Text()
		if q == "" {
			continue
		}
		ans, err := r.Query(ctx, q)
		if err != nil {
			fmt.Printf("查询失败: %v\n", err)
			continue
		}
		fmt.Println(ans)
	}
}
