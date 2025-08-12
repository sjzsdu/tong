package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	cnllms "github.com/sjzsdu/langchaingo-cn/llms"
	"github.com/sjzsdu/tong/config"
	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/rag"
	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
)

var (
	qdrantURL      string
	collectionName string
	chunkSize      int
	chunkOverlap   int
	docsDir        string
	syncOnly       bool
	autoSync       bool
	syncInterval   int
	forceReindex   bool
)

var ragCmd = &cobra.Command{
	Use:   "rag",
	Short: lang.T("Rag application"),
	Long:  lang.T("Rag application"),
	Run:   runRag,
}

func init() {
	// 获取当前目录名称作为默认集合名称
	defaultCollection := GetProjectName()

	// 添加streamMode标志
	ragCmd.Flags().BoolVarP(&streamMode, "stream", "s", true, lang.T("启用流式输出模式"))
	// 添加Qdrant URL标志
	ragCmd.Flags().StringVarP(&qdrantURL, "qdrant", "q", "http://localhost:6333", lang.T("Qdrant服务URL"))
	// 添加集合名称标志
	ragCmd.Flags().StringVarP(&collectionName, "collection", "c", defaultCollection, lang.T("Qdrant集合名称"))
	// 添加文本分块大小标志
	ragCmd.Flags().IntVarP(&chunkSize, "chunk-size", "", 1000, lang.T("文本分块大小"))
	// 添加文本分块重叠标志
	ragCmd.Flags().IntVarP(&chunkOverlap, "chunk-overlap", "", 200, lang.T("文本分块重叠大小"))
	// 添加文档目录标志
	ragCmd.Flags().StringVarP(&docsDir, "docs-dir", "d", ".", lang.T("文档目录路径"))

	// 添加同步相关标志
	ragCmd.Flags().BoolVarP(&syncOnly, "sync", "", false, lang.T("仅同步文档，不启动交互会话"))
	ragCmd.Flags().BoolVarP(&autoSync, "auto-sync", "", false, lang.T("启用自动同步"))
	ragCmd.Flags().IntVarP(&syncInterval, "sync-interval", "", 300, lang.T("自动同步间隔（秒）"))
	ragCmd.Flags().BoolVarP(&forceReindex, "force-reindex", "", false, lang.T("强制重新索引所有文档"))

	rootCmd.AddCommand(ragCmd)
}

func runRag(cmd *cobra.Command, args []string) {
	// 获取配置
	cfg, err := GetConfig()
	if err != nil {
		log.Fatalf("获取配置失败: %v", err)
	}

	// 初始化模型
	llmModel, embeddingModel, err := initializeModels(cfg)
	if err != nil {
		log.Fatalf("初始化模型失败: %v", err)
	}

	// 创建RAG选项
	options := rag.RAGOptions{
		Storage: rag.StorageOptions{
			URL:            qdrantURL,
			CollectionName: collectionName,
		},
		Splitter: rag.SplitterOptions{
			ChunkSize:    chunkSize,
			ChunkOverlap: chunkOverlap,
		},
		Retriever: rag.RetrieverOptions{
			TopK: 4,
		},
		Session: rag.SessionOptions{
			Stream: streamMode,
		},
		DocsDir: docsDir,
		Sync: rag.SyncOptions{
			ForceReindex: forceReindex,
			SyncInterval: time.Duration(syncInterval) * time.Second,
		},
	}

	// 初始化RAG系统
	ctx := context.Background()
	ragSystem, err := rag.InitializeFromConfig(ctx, llmModel, embeddingModel, options)
	if err != nil {
		log.Fatalf("初始化RAG系统失败: %v", err)
	}

	// 如果只需同步文档
	if syncOnly {
		fmt.Println(lang.T("执行文档同步..."))
		err = ragSystem.SyncDocuments(ctx)
		if err != nil {
			log.Fatalf("文档同步失败: %v", err)
		}
		fmt.Println(lang.T("文档同步完成"))
		return
	}

	// 检查是否需要索引或强制重新索引
	indexed, err := rag.IsIndexed(options.Storage.CollectionName)
	if err != nil {
		log.Fatalf("检查索引状态失败: %v", err)
	}

	if !indexed || forceReindex {
		// 文档尚未索引或强制重新索引
		log.Println(lang.T("开始执行文档索引..."))
		err = ragSystem.IndexDocuments(ctx, options.DocsDir)
		if err != nil {
			log.Fatalf("文档索引失败: %v", err)
		}
	} else {
		// 询问是否需要重新索引
		fmt.Print(lang.T("文档已经索引，是否需要重新索引? (y/n): "))
		var answer string
		fmt.Scanln(&answer)
		if answer == "y" || answer == "Y" {
			err = ragSystem.IndexDocuments(ctx, options.DocsDir)
			if err != nil {
				log.Fatalf("文档重新索引失败: %v", err)
			}
		}
	}

	// 如果启用自动同步
	if autoSync {
		fmt.Println(lang.T("启动自动同步服务..."))
		err = ragSystem.StartAutomaticSync()
		if err != nil {
			log.Fatalf("启动自动同步服务失败: %v", err)
		}
	}

	// 启动RAG会话
	err = ragSystem.Start(ctx)
	if err != nil {
		log.Fatalf("RAG会话错误: %v", err)
	}
}

// initializeModels 初始化LLM和嵌入模型
func initializeModels(config *config.SchemaConfig) (llms.Model, embeddings.Embedder, error) {
	// 初始化LLM
	llm, err := cnllms.CreateLLM(config.MasterLLM.Type, config.MasterLLM.Params)
	if err != nil {
		return nil, nil, fmt.Errorf("初始化LLM失败: %v", err)
	}

	// 初始化嵌入模型
	// 确保返回的LLM模型实现了embeddings.Embedder接口
	embeddingLLM, err := cnllms.CreateEmbedding(config.EmbeddingLLM.Type, config.EmbeddingLLM.Params)
	if err != nil {
		return llm, nil, fmt.Errorf("初始化嵌入模型失败: %v", err)
	}

	return llm, embeddingLLM, nil
}
