package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/rag"
	"github.com/sjzsdu/tong/schema"
	"github.com/spf13/cobra"
)

var (
	indexSubdir  string
	forceRebuild bool
)

var IndexCmd = &cobra.Command{
	Use:   "index",
	Short: lang.T("构建/更新项目语义索引"),
	Long:  lang.T("对项目源代码和文档进行向量索引（基于RAG）"),
	Run:   runIndex,
}

func init() {
	IndexCmd.Flags().StringVar(&indexSubdir, "subdir", "", lang.T("限定索引的子目录（相对项目根）"))
	IndexCmd.Flags().BoolVar(&forceRebuild, "force", false, lang.T("强制重新索引"))
}

func runIndex(cmd *cobra.Command, args []string) {
	if sharedProject == nil {
		fmt.Println("错误: 未找到共享的项目实例")
		os.Exit(1)
	}
	projectRoot := sharedProject.GetRootPath()
	cfg, err := schema.LoadMCPConfig(projectRoot, "")
	if err != nil {
		fmt.Printf("获取配置失败: %v\n", err)
		os.Exit(1)
	}

	// 复用 rag 选项解析
	options := resolveOptions(cmd, cfg, projectRoot)
	if indexSubdir != "" {
		if filepath.IsAbs(indexSubdir) {
			options.DocsDir = indexSubdir
		} else {
			options.DocsDir = filepath.Join(projectRoot, indexSubdir)
		}
	}
	if forceRebuild {
		options.Sync.ForceReindex = true
	}

	llm, embed, _ := initializeModels(cfg)
	ctx := context.Background()
	r, err := rag.InitializeFromConfig(ctx, llm, embed, options)
	if err != nil {
		fmt.Printf("初始化RAG失败: %v\n", err)
		os.Exit(1)
	}

	// 索引
	fmt.Printf("%s %s\n", lang.T("开始索引目录:"), options.DocsDir)
	if err := r.IndexDocuments(ctx, options.DocsDir); err != nil {
		fmt.Printf("索引失败: %v\n", err)
		os.Exit(1)
	}

	helper.PrintWithLabel(lang.T("索引完成"), map[string]any{"docsDir": options.DocsDir, "collection": options.Storage.CollectionName})
}
