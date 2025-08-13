package rag

import (
	"context"
	"fmt"

	"github.com/sjzsdu/tong/share"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

// QdrantRetriever 是Qdrant向量存储的检索器实现
type QdrantRetriever struct {
	Store          qdrant.Store
	TopK           int     // 返回的最大文档数量
	ScoreThreshold float32 // 文档相似度阈值，低于此值的文档将被过滤
}

// NewQdrantRetriever 创建新的Qdrant检索器
func NewQdrantRetriever(store qdrant.Store, options RetrieverOptions) *QdrantRetriever {
	return &QdrantRetriever{
		Store:          store,
		TopK:           options.TopK,
		ScoreThreshold: options.ScoreThreshold,
	}
}

// GetRelevantDocuments 实现schema.Retriever接口，获取与查询相关的文档
func (r *QdrantRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	if share.GetDebug() {
		fmt.Printf("[DEBUG] retrieve start: topK=%d query=%q\n", r.TopK, query)
	}

	docs, err := r.Store.SimilaritySearch(ctx, query, r.TopK)
	if err != nil {
		return nil, &RagError{
			Code:    "similarity_search_failed",
			Message: "执行相似度搜索失败",
			Cause:   err,
		}
	}

	if share.GetDebug() {
		fmt.Printf("[DEBUG] retrieve end: %d docs\n", len(docs))
		for i, d := range docs {
			src, _ := d.Metadata["source"].(string)
			fname, _ := d.Metadata["filename"].(string)
			rel, _ := d.Metadata["rel_path"].(string)
			fmt.Printf("[DEBUG] doc[%d]: source=%s file=%s rel=%s len=%d\n", i, src, fname, rel, len(d.PageContent))
		}
	}

	// Note: Score filtering is disabled as scores aren't returned by current API
	return docs, nil
}

// CreateRetriever 创建检索器
func CreateRetriever(vectorStore qdrant.Store, options RetrieverOptions) schema.Retriever {
	return NewQdrantRetriever(vectorStore, options)
}
