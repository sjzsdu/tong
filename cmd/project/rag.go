package project

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	cnllms "github.com/sjzsdu/langchaingo-cn/llms"
	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/rag"
	"github.com/sjzsdu/tong/schema"
	"github.com/sjzsdu/tong/share"
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
	ragSubdir      string
	ragStreamMode  bool
	syncOnly       bool
	autoSync       bool
	syncInterval   int
	forceReindex   bool
)

var RagCmd = &cobra.Command{
	Use:   "rag",
	Short: lang.T("Rag application"),
	Long:  lang.T("Rag application"),
	Run:   runRag,
}

func init() {
	// 不设置实际默认（置空/置零），避免覆盖 tong.json；最终按优先级合并
	RagCmd.Flags().BoolVarP(&ragStreamMode, "stream", "s", false, lang.T("启用流式输出模式"))
	RagCmd.Flags().StringVarP(&qdrantURL, "qdrant", "q", "", lang.T("Qdrant服务URL"))
	RagCmd.Flags().StringVarP(&collectionName, "collection", "c", "", lang.T("Qdrant集合名称（默认使用项目名）"))
	RagCmd.Flags().IntVarP(&chunkSize, "chunk-size", "", 0, lang.T("文本分块大小"))
	RagCmd.Flags().IntVarP(&chunkOverlap, "chunk-overlap", "", 0, lang.T("文本分块重叠大小"))
	RagCmd.Flags().StringVar(&docsDir, "docs-dir", "", lang.T("文档目录路径（相对项目根或绝对路径）"))
	RagCmd.Flags().StringVar(&ragSubdir, "subdir", "", lang.T("限定索引的子目录（相对项目根）"))

	RagCmd.Flags().BoolVarP(&syncOnly, "sync", "", false, lang.T("仅同步文档，不启动交互会话"))
	RagCmd.Flags().BoolVarP(&autoSync, "auto-sync", "", false, lang.T("启用自动同步"))
	RagCmd.Flags().IntVarP(&syncInterval, "sync-interval", "", 0, lang.T("自动同步间隔（秒）"))
	RagCmd.Flags().BoolVarP(&forceReindex, "force-reindex", "", false, lang.T("强制重新索引所有文档"))
}

func runRag(cmd *cobra.Command, args []string) {
	if sharedProject == nil {
		log.Fatalf("错误: 未找到共享的项目实例")
	}
	projectRoot := sharedProject.GetRootPath()
	cfg, err := schema.LoadMCPConfig(projectRoot, "")
	if err != nil {
		log.Fatalf("获取配置失败: %v", err)
	}

	llmModel, embeddingModel, _ := initializeModels(cfg)
	options := resolveOptions(cmd, cfg, projectRoot)

	if share.GetDebug() {
		helper.PrintWithLabel("RAG Options", options)
	}

	ctx := context.Background()
	ragSystem, err := rag.InitializeFromConfig(ctx, llmModel, embeddingModel, options)
	if err != nil {
		log.Fatalf("初始化RAG系统失败: %v", err)
	}

	if syncOnly {
		fmt.Println(lang.T("执行文档同步..."))
		if err := ragSystem.SyncDocuments(ctx); err != nil {
			log.Fatalf("文档同步失败: %v", err)
		}
		fmt.Println(lang.T("文档同步完成"))
		return
	}

	indexed, err := rag.IsIndexed(options.Storage.CollectionName)
	if err != nil {
		log.Fatalf("检查索引状态失败: %v", err)
	}

	if !indexed || forceReindex {
		log.Println(lang.T("开始执行文档索引..."))
		if err := ragSystem.IndexDocuments(ctx, options.DocsDir); err != nil {
			log.Fatalf("文档索引失败: %v", err)
		}
	}
	if autoSync {
		fmt.Println(lang.T("启动自动同步服务..."))
		if err := ragSystem.StartAutomaticSync(); err != nil {
			log.Fatalf("启动自动同步服务失败: %v", err)
		}
	}

	if err := ragSystem.Start(ctx); err != nil {
		log.Fatalf("RAG会话错误: %v", err)
	}
}

func resolveOptions(cmd *cobra.Command, cfg *schema.SchemaConfig, projectRoot string) rag.RAGOptions {
	// 标记哪些参数由命令行显式提供
	flags := cmd.Flags()
	streamChanged := flags.Changed("stream")
	qdrantChanged := flags.Changed("qdrant")
	collectionChanged := flags.Changed("collection")
	chunkSizeChanged := flags.Changed("chunk-size")
	chunkOverlapChanged := flags.Changed("chunk-overlap")
	docsDirChanged := flags.Changed("docs-dir")
	subdirChanged := flags.Changed("subdir")
	syncIntervalChanged := flags.Changed("sync-interval")
	forceChanged := flags.Changed("force-reindex")

	// 先应用 tong.json，再由命令行覆盖（若显式设置）
	if !qdrantChanged && cfg.Rag.Storage.URL != "" {
		qdrantURL = cfg.Rag.Storage.URL
	}
	if !collectionChanged && cfg.Rag.Storage.Collection != "" {
		collectionName = cfg.Rag.Storage.Collection
	}
	if !chunkSizeChanged && cfg.Rag.Splitter.ChunkSize > 0 {
		chunkSize = cfg.Rag.Splitter.ChunkSize
	}
	if !chunkOverlapChanged && cfg.Rag.Splitter.ChunkOverlap > 0 {
		chunkOverlap = cfg.Rag.Splitter.ChunkOverlap
	}
	// retriever
	topK := 4
	if cfg.Rag.Retriever.TopK > 0 {
		topK = cfg.Rag.Retriever.TopK
	}
	// session stream
	if !streamChanged && cfg.Rag.Session.Stream != nil {
		ragStreamMode = *cfg.Rag.Session.Stream
	}
	// docs dir
	if !docsDirChanged && cfg.Rag.DocsDir != "" {
		docsDir = cfg.Rag.DocsDir
	}
	// sync
	if !syncIntervalChanged && cfg.Rag.Sync.SyncIntervalSeconds > 0 {
		syncInterval = cfg.Rag.Sync.SyncIntervalSeconds
	}
	if !forceChanged && cfg.Rag.Sync.ForceReindex {
		forceReindex = true
	}

	// 代码级默认值（在未通过 CLI 或 tong.json 指定时）
	if qdrantURL == "" {
		qdrantURL = "http://localhost:6333"
	}
	if chunkSize == 0 {
		chunkSize = 1000
	}
	if chunkOverlap == 0 {
		chunkOverlap = 200
	}
	if syncInterval == 0 {
		syncInterval = 300
	}

	// 计算最终索引目录
	finalTargetPath := ""
	if docsDir != "" {
		if filepath.IsAbs(docsDir) {
			finalTargetPath = docsDir
		} else {
			finalTargetPath = filepath.Join(projectRoot, docsDir)
		}
	} else if subdirChanged && ragSubdir != "" { // 仅当用户显式提供 subdir 时使用
		if filepath.IsAbs(ragSubdir) {
			finalTargetPath = ragSubdir
		} else {
			finalTargetPath = filepath.Join(projectRoot, ragSubdir)
		}
	} else {
		finalTargetPath = projectRoot
	}

	if _, err := GetTargetNode(finalTargetPath); err != nil {
		log.Fatalf("目标路径无效: %v", err)
	}

	if collectionName == "" {
		collectionName = filepath.Base(projectRoot)
	}

	options := rag.RAGOptions{
		Storage: rag.StorageOptions{
			URL:            qdrantURL,
			CollectionName: collectionName,
		},
		Splitter: rag.SplitterOptions{
			ChunkSize:    orDefault(chunkSize, 1000),
			ChunkOverlap: orDefault(chunkOverlap, 200),
		},
		Retriever: rag.RetrieverOptions{
			TopK: topK,
		},
		Session: rag.SessionOptions{
			Stream: ragStreamMode,
		},
		DocsDir: finalTargetPath,
		Sync: rag.SyncOptions{
			ForceReindex: forceReindex,
			SyncInterval: time.Duration(syncInterval) * time.Second,
		},
	}
	return options
}

func orDefault(v, d int) int {
	if v > 0 {
		return v
	}
	return d
}

// initializeModels 初始化LLM和嵌入模型
func initializeModels(config *schema.SchemaConfig) (llms.Model, embeddings.Embedder, error) {
	llm, err := cnllms.CreateLLM(config.MasterLLM.Type, config.MasterLLM.Params)
	if err != nil {
		return nil, nil, fmt.Errorf("初始化LLM失败: %v", err)
	}

	embeddingLLM, err := cnllms.CreateEmbedding(config.EmbeddingLLM.Type, config.EmbeddingLLM.Params)
	if err != nil {
		return llm, nil, fmt.Errorf("初始化嵌入模型失败: %v", err)
	}

	return llm, embeddingLLM, nil
}
