package project

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	cnllms "github.com/sjzsdu/langchaingo-cn/llms"
	"github.com/sjzsdu/tong/config"
	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/rag"
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
	// 添加streamMode标志（本地变量，避免依赖外部包变量）
	RagCmd.Flags().BoolVarP(&ragStreamMode, "stream", "s", true, lang.T("启用流式输出模式"))
	// 添加Qdrant URL标志
	RagCmd.Flags().StringVarP(&qdrantURL, "qdrant", "q", "http://localhost:6333", lang.T("Qdrant服务URL"))
	// 添加集合名称标志（默认留空，运行时按项目名回填）
	RagCmd.Flags().StringVarP(&collectionName, "collection", "c", "", lang.T("Qdrant集合名称（默认使用项目名）"))
	// 添加文本分块大小标志
	RagCmd.Flags().IntVarP(&chunkSize, "chunk-size", "", 1000, lang.T("文本分块大小"))
	// 添加文本分块重叠标志
	RagCmd.Flags().IntVarP(&chunkOverlap, "chunk-overlap", "", 200, lang.T("文本分块重叠大小"))
	// 添加文档目录标志（去掉短选项，避免与上层 -d 冲突）
	RagCmd.Flags().StringVar(&docsDir, "docs-dir", ".", lang.T("文档目录路径（相对项目根或绝对路径）"))
	// 限定子目录（相对项目根），与 search 命令保持一致
	RagCmd.Flags().StringVar(&ragSubdir, "subdir", ".", lang.T("限定索引的子目录（相对项目根）"))

	// 添加同步相关标志
	RagCmd.Flags().BoolVarP(&syncOnly, "sync", "", false, lang.T("仅同步文档，不启动交互会话"))
	RagCmd.Flags().BoolVarP(&autoSync, "auto-sync", "", false, lang.T("启用自动同步"))
	RagCmd.Flags().IntVarP(&syncInterval, "sync-interval", "", 300, lang.T("自动同步间隔（秒）"))
	RagCmd.Flags().BoolVarP(&forceReindex, "force-reindex", "", false, lang.T("强制重新索引所有文档"))
}

func runRag(cmd *cobra.Command, args []string) {
	// 基于项目根和 subdir/docs-dir 计算最终目录，并校验对应的 Node 存在
	if sharedProject == nil {
		log.Fatalf("错误: 未找到共享的项目实例")
	}
	projectRoot := sharedProject.GetRootPath()

	finalTargetPath := ""
	// 优先使用 docs-dir；若为默认值或空，则回退到 subdir
	if docsDir != "" && docsDir != "." {
		if filepath.IsAbs(docsDir) {
			finalTargetPath = docsDir
		} else {
			finalTargetPath = filepath.Join(projectRoot, docsDir)
		}
	} else {
		if filepath.IsAbs(ragSubdir) {
			finalTargetPath = ragSubdir
		} else {
			finalTargetPath = filepath.Join(projectRoot, ragSubdir)
		}
	}

	// 使用通用函数获取目标节点，确保路径在项目树中
	if _, err := GetTargetNode(finalTargetPath); err != nil {
		log.Fatalf("目标路径无效: %v", err)
	}

	// 回填默认集合名称
	if collectionName == "" {
		collectionName = filepath.Base(projectRoot)
	}

	// 加载配置（相对项目根），避免依赖外部 cmd 包函数
	cfg, err := config.LoadMCPConfig(projectRoot, "")
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
			Stream: ragStreamMode,
		},
		DocsDir: finalTargetPath,
		Sync: rag.SyncOptions{
			ForceReindex: forceReindex,
			SyncInterval: time.Duration(syncInterval) * time.Second,
		},
	}

	if share.GetDebug() {
		helper.PrintWithLabel("RAG Options", options)
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
