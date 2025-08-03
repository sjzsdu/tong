package rag

import (
	"context"

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
	// 使用Qdrant的相似度搜索
	docs, err := r.Store.SimilaritySearch(ctx, query, r.TopK)
	if err != nil {
		return nil, &RagError{
			Code:    "similarity_search_failed",
			Message: "执行相似度搜索失败",
			Cause:   err,
		}
	}

	// 如果启用了得分阈值过滤，则进行过滤
	// 注意：目前Qdrant的SimilaritySearch不返回得分，所以这部分代码暂时不会生效
	// 当langchaingo支持返回得分时，可以取消注释
	/*
		if r.ScoreThreshold > 0 && len(docs) > 0 {
			// 假设文档的元数据中包含了相似度得分
			filteredDocs := make([]schema.Document, 0)
			for _, doc := range docs {
				if score, ok := doc.Metadata["score"].(float32); ok && score >= r.ScoreThreshold {
					filteredDocs = append(filteredDocs, doc)
				}
			}
			return filteredDocs, nil
		}
	*/

	return docs, nil
}

// CreateRetriever 创建检索器
func CreateRetriever(vectorStore qdrant.Store, options RetrieverOptions) schema.Retriever {
	return NewQdrantRetriever(vectorStore, options)
}
