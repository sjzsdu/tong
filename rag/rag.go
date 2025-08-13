package rag

import (
	"context"
	"fmt"
	"os"

	"github.com/sjzsdu/tong/config"
	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/share"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

// InitializeFromConfig 从配置初始化RAG系统
func InitializeFromConfig(ctx context.Context, masterLLM llms.Model, embeddingModel embeddings.Embedder, options RAGOptions) (*RAG, error) {
	if share.GetDebug() {
		fmt.Printf("[DEBUG] init RAG: qdrant=%s collection=%s docsDir=%s topK=%d\n",
			options.Storage.URL, options.Storage.CollectionName, options.DocsDir, options.Retriever.TopK)
	}
	// 创建向量存储
	vectorStore, err := CreateVectorStore(ctx, embeddingModel, options.Storage)
	if err != nil {
		return nil, err
	}
	if share.GetDebug() {
		fmt.Println("[DEBUG] vector store ready")
	}

	// 创建检索器
	retriever := CreateRetriever(vectorStore, options.Retriever)
	if share.GetDebug() {
		fmt.Println("[DEBUG] retriever ready")
	}

	// 创建RAG实例
	rag := &RAG{
		LLM:            masterLLM,
		EmbeddingModel: embeddingModel,
		VectorStore:    vectorStore,
		Retriever:      retriever,
		Options:        options,
	}

	return rag, nil
}

// Initialize 初始化RAG系统
func Initialize(ctx context.Context, cfg *config.SchemaConfig, options RAGOptions) (*RAG, error) {
	if cfg == nil {
		return nil, &RagError{
			Code:    "config_required",
			Message: "配置对象不能为空",
		}
	}

	// 1. 初始化大语言模型
	llmInstance, embeddingModel, err := InitializeModels(cfg)
	if err != nil {
		return nil, err
	}

	// 2. 使用模型初始化RAG系统
	return InitializeFromConfig(ctx, llmInstance, embeddingModel, options)
}

// Run 运行RAG系统
func Run(ctx context.Context, cfg *config.SchemaConfig, options RAGOptions) error {
	// 初始化RAG系统
	rag, err := Initialize(ctx, cfg, options)
	if err != nil {
		return err
	}

	// 检查是否需要重新索引
	shouldReindex, err := ShouldReindex(options.Storage.CollectionName, false)
	if err != nil {
		return &RagError{
			Code:    "check_index_failed",
			Message: "检查索引状态失败",
			Cause:   err,
		}
	}

	// 如果需要重新索引，询问用户
	if !shouldReindex {
		fmt.Print(lang.T("文档已经索引，是否需要重新索引? (y/n): "))
		var answer string
		fmt.Scanln(&answer)
		shouldReindex = answer == "y" || answer == "Y"
	}

	// 执行索引过程
	if shouldReindex {
		// 确保文档目录存在
		if _, err := os.Stat(options.DocsDir); os.IsNotExist(err) {
			return &RagError{
				Code:    "docs_dir_not_exist",
				Message: fmt.Sprintf("文档目录 %s 不存在", options.DocsDir),
				Cause:   err,
			}
		}

		// 执行索引
		if err := IndexDocuments(ctx, rag.VectorStore, options.DocsDir, options); err != nil {
			return err
		}
	}

	// 创建会话
	session := NewSession(rag.LLM, rag.Retriever, options.Session)

	// 启动会话
	return session.Start(ctx)
}

// InitializeModels 初始化LLM和嵌入模型
func InitializeModels(cfg *config.SchemaConfig) (llms.Model, embeddings.Embedder, error) {
	// TODO: 替换为您的实际模型初始化逻辑
	return nil, nil, &RagError{
		Code:    "not_implemented",
		Message: "InitializeModels需要实现",
	}
}

// Query 使用RAG系统执行单次查询
func (r *RAG) Query(ctx context.Context, query string) (string, error) {
	// 创建会话
	session := NewSession(r.LLM, r.Retriever, r.Options.Session)

	// 执行查询
	return session.Query(ctx, query)
}

// Start 启动RAG交互式会话
func (r *RAG) Start(ctx context.Context) error {
	// 创建会话
	session := NewSession(r.LLM, r.Retriever, r.Options.Session)

	// 启动会话
	return session.Start(ctx)
}

// AddDocuments 添加文档到RAG系统
func (r *RAG) AddDocuments(ctx context.Context, docs []schema.Document) error {
	// 分割文档
	splitDocs, err := SplitDocuments(docs, r.Options.Splitter)
	if err != nil {
		return err
	}

	// 存储文档
	if err := StoreDocuments(ctx, r.VectorStore, splitDocs); err != nil {
		return err
	}

	// 更新索引状态
	return UpdateIndexStatus(r.Options.Storage.CollectionName, len(splitDocs))
}

// IsIndexed 检查是否已索引
func (r *RAG) IsIndexed(collectionName string) (bool, error) {
	return IsIndexed(collectionName)
}

// IndexDocuments 索引文档
func (r *RAG) IndexDocuments(ctx context.Context, docsDir string) error {
	// 先使用传统方式索引文档
	err := IndexDocuments(ctx, r.VectorStore, docsDir, r.Options)
	if err != nil {
		return err
	}

	// 如果同步管理器已初始化，则更新同步管理器的元数据
	if r.SyncManager != nil {
		// 直接调用同步，这会更新文档的元数据
		return r.SyncManager.SyncDocuments(ctx)
	}

	return nil
}

// SyncDocuments 同步文档，但不执行完整的索引过程
func (r *RAG) SyncDocuments(ctx context.Context) error {
	return SyncDocumentsForRAG(r)
}

// StartAutomaticSync 启动自动同步服务
func (r *RAG) StartAutomaticSync() error {
	return StartAutomaticSync(r)
}
