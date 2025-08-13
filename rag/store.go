package rag

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/share"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

// CreateVectorStore 创建向量存储
func CreateVectorStore(ctx context.Context, embeddingModel embeddings.Embedder, options StorageOptions) (qdrant.Store, error) {
	qdrantURL, err := url.Parse(options.URL)
	if err != nil {
		return qdrant.Store{}, &RagError{
			Code:    "parse_url_failed",
			Message: "解析向量数据库URL失败",
			Cause:   err,
		}
	}

	if share.GetDebug() {
		fmt.Printf("[DEBUG] qdrant url=%s collection=%s\n", qdrantURL.String(), options.CollectionName)
	}

	// 获取嵌入维度以用于集合创建
	if share.GetDebug() {
		fmt.Println("[DEBUG] probing embedding dimension...")
	}
	vec, err := embeddingModel.EmbedQuery(ctx, "ping")
	if err != nil {
		return qdrant.Store{}, &RagError{
			Code:    "embedding_dim_failed",
			Message: "获取嵌入维度失败",
			Cause:   err,
		}
	}
	dim := len(vec)
	if share.GetDebug() {
		fmt.Printf("[DEBUG] embedding dim=%d\n", dim)
	}

	// 可选 API Key（优先环境变量）
	apiKey := os.Getenv("QDRANT_API_KEY")

	// 确认服务可用且集合存在，不存在则自动创建
	if err := ensureQdrantCollection(ctx, options.URL, options.CollectionName, dim, apiKey); err != nil {
		return qdrant.Store{}, &RagError{
			Code:    "ensure_collection_failed",
			Message: "检查/创建Qdrant集合失败",
			Cause:   err,
		}
	}
	if share.GetDebug() {
		fmt.Println("[DEBUG] qdrant collection is ready")
	}

	// 创建向量存储
	var vopts []qdrant.Option
	vopts = append(vopts,
		qdrant.WithURL(*qdrantURL),
		qdrant.WithCollectionName(options.CollectionName),
		qdrant.WithEmbedder(embeddingModel),
	)
	if apiKey != "" {
		vopts = append(vopts, qdrant.WithAPIKey(apiKey))
	}

	vectorStore, err := qdrant.New(vopts...)
	if err != nil {
		return qdrant.Store{}, &RagError{
			Code:    "create_vector_store_failed",
			Message: "创建向量存储失败",
			Cause:   err,
		}
	}
	if share.GetDebug() {
		fmt.Println("[DEBUG] vector store created")
	}

	return vectorStore, nil
}

// SplitDocuments 将文档分割成更小的块
func SplitDocuments(docs []schema.Document, options SplitterOptions) ([]schema.Document, error) {
	fmt.Println(lang.T("开始分割文档..."))

	// 创建递归字符分割器
	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(options.ChunkSize),
		textsplitter.WithChunkOverlap(options.ChunkOverlap),
	)

	// 使用分割器拆分文档
	splitDocs, err := textsplitter.SplitDocuments(splitter, docs)
	if err != nil {
		return nil, &RagError{
			Code:    "split_documents_failed",
			Message: "分割文档失败",
			Cause:   err,
		}
	}

	fmt.Printf(lang.T("文档分割完成，共 %d 个文档块\n"), len(splitDocs))
	return splitDocs, nil
}

// StoreDocuments 将文档添加到向量存储
func StoreDocuments(ctx context.Context, vectorStore qdrant.Store, docs []schema.Document) error {
	if len(docs) == 0 {
		fmt.Println(lang.T("警告：没有文档需要存储"))
		return nil
	}

	fmt.Println(lang.T("开始向量化文档并存储..."))

	batchSize := getMaxEmbedBatchSize()
	if share.GetDebug() {
		fmt.Printf("[DEBUG] total docs=%d batchSize=%d\n", len(docs), batchSize)
	}
	for i := 0; i < len(docs); i += batchSize {
		end := i + batchSize
		if end > len(docs) {
			end = len(docs)
		}
		if share.GetDebug() {
			fmt.Printf("[DEBUG] add documents batch: %d-%d\n", i, end)
		}
		_, err := vectorStore.AddDocuments(ctx, docs[i:end])
		if err != nil {
			return &RagError{
				Code:    "store_documents_failed",
				Message: "添加文档到向量存储失败",
				Cause:   err,
			}
		}
	}

	fmt.Println(lang.T("文档索引完成"))
	return nil
}

// IndexDocuments 完整的文档索引流程
func IndexDocuments(ctx context.Context, vectorStore qdrant.Store, docsDir string, options RAGOptions) error {
	// 加载文档
	fmt.Println(lang.T("开始加载文档..."))
	docs, err := LoadDocumentsFromDir(ctx, docsDir)
	if err != nil {
		return &RagError{
			Code:    "load_documents_failed",
			Message: "加载文档失败",
			Cause:   err,
		}
	}
	fmt.Printf(lang.T("成功加载 %d 个文档\n"), len(docs))

	// 分割文档
	splitDocs, err := SplitDocuments(docs, options.Splitter)
	if err != nil {
		return err
	}

	// 添加文档到向量存储
	if err := StoreDocuments(ctx, vectorStore, splitDocs); err != nil {
		return err
	}

	// 更新索引状态
	if err := UpdateIndexStatus(options.Storage.CollectionName, len(splitDocs)); err != nil {
		fmt.Printf("警告：保存索引状态失败: %v\n", err)
	}

	return nil
}

// getMaxEmbedBatchSize returns the max batch size for embedding requests.
// Default 25 to comply with providers like OpenAI; can be overridden by env EMBED_BATCH_SIZE.
func getMaxEmbedBatchSize() int {
	if v := os.Getenv("EMBED_BATCH_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 25
}
