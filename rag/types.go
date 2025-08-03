package rag

import (
	"time"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

// 错误类型定义
var (
	// ErrNotImplemented 表示功能尚未实现
	ErrNotImplemented = NewError("not implemented", "功能尚未实现")
	// ErrConfigNotFound 表示配置未找到
	ErrConfigNotFound = NewError("config not found", "配置未找到")
	// ErrDocumentLoadFailed 表示文档加载失败
	ErrDocumentLoadFailed = NewError("document load failed", "文档加载失败")
)

// RagError 表示RAG系统中的错误
type RagError struct {
	Code    string // 错误代码
	Message string // 错误消息
	Cause   error  // 原始错误
}

// Error 实现error接口
func (e *RagError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// NewError 创建一个新的RagError
func NewError(code, message string) *RagError {
	return &RagError{
		Code:    code,
		Message: message,
	}
}

// WithCause 为错误添加原因
func (e *RagError) WithCause(cause error) *RagError {
	return &RagError{
		Code:    e.Code,
		Message: e.Message,
		Cause:   cause,
	}
}

// IndexStatus 表示文档索引的状态
type IndexStatus struct {
	LastIndexedTime time.Time `json:"last_indexed_time"`
	DocumentCount   int       `json:"document_count"`
	CollectionName  string    `json:"collection_name"`
}

// StorageOptions 表示存储选项
type StorageOptions struct {
	URL            string
	CollectionName string
}

// SplitterOptions 表示文本分割选项
type SplitterOptions struct {
	ChunkSize    int
	ChunkOverlap int
}

// RetrieverOptions 表示检索选项
type RetrieverOptions struct {
	TopK           int
	ScoreThreshold float32
}

// SessionOptions 表示会话选项
type SessionOptions struct {
	Stream     bool
	MaxHistory int
}

// DocumentMetadata 文档元数据
type DocumentMetadata struct {
	Path         string    `json:"path"`
	LastModified time.Time `json:"last_modified"`
	Hash         string    `json:"hash"`
	VectorIDs    []string  `json:"vector_ids"`
}

// SyncOptions 同步选项
type SyncOptions struct {
	// 是否执行完整重新索引
	ForceReindex bool
	// 自动检测更改的间隔时间（0表示禁用自动同步）
	SyncInterval time.Duration
}

// RAGOptions 表示RAG系统的配置选项
type RAGOptions struct {
	Storage   StorageOptions
	Splitter  SplitterOptions
	Retriever RetrieverOptions
	Session   SessionOptions
	DocsDir   string
	Sync      SyncOptions
}

// RAG 表示一个完整的RAG系统
type RAG struct {
	LLM            llms.Model
	EmbeddingModel embeddings.Embedder
	VectorStore    qdrant.Store
	Chain          chains.Chain
	Retriever      schema.Retriever
	Options        RAGOptions
	SyncManager    *DocumentSyncManager
}
