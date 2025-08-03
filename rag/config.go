package rag

import (
	"fmt"
	"time"

	"github.com/sjzsdu/tong/helper/json"
)

const (
	// RAGConfigDir 存储RAG配置的子目录
	RAGConfigDir = "rag"
	// IndexStatusFile 存储索引状态的文件名
	IndexStatusFile = "index_status"
)

// 配置管理器实例
var configStore *json.JSONStore

// 初始化配置管理器
func init() {
	store, err := json.NewJSONStore(RAGConfigDir)
	if err != nil {
		fmt.Printf("初始化RAG配置存储失败: %v\n", err)
		return
	}
	configStore = store
}

// 默认配置
var DefaultOptions = RAGOptions{
	Storage: StorageOptions{
		URL:            "http://localhost:6333",
		CollectionName: "tong_docs",
	},
	Splitter: SplitterOptions{
		ChunkSize:    1000,
		ChunkOverlap: 200,
	},
	Retriever: RetrieverOptions{
		TopK:           4,
		ScoreThreshold: 0.0,
	},
	Session: SessionOptions{
		Stream:     true,
		MaxHistory: 10,
	},
	DocsDir: ".",
}

// GetDefaultOptions 获取默认配置选项
func GetDefaultOptions() RAGOptions {
	return DefaultOptions
}

// LoadIndexStatus 从配置中加载索引状态
func LoadIndexStatus(collectionName string) (*IndexStatus, error) {
	// 使用集合名称作为key的一部分，允许多个集合有各自的状态
	statusKey := fmt.Sprintf("%s_%s", IndexStatusFile, collectionName)

	var status IndexStatus
	_, err := configStore.Get(statusKey, &status)
	if err != nil {
		// 如果文件不存在，返回空状态
		if fmt.Sprintf("%v", err) == fmt.Sprintf("文件不存在: %s.json", statusKey) {
			return &IndexStatus{
				CollectionName: collectionName,
			}, nil
		}
		return nil, fmt.Errorf("加载索引状态失败: %w", err)
	}

	return &status, nil
}

// SaveIndexStatus 保存索引状态到配置
func SaveIndexStatus(status *IndexStatus) error {
	statusKey := fmt.Sprintf("%s_%s", IndexStatusFile, status.CollectionName)
	return configStore.Set(statusKey, status)
}

// UpdateIndexStatus 更新索引状态
func UpdateIndexStatus(collectionName string, docCount int) error {
	status := &IndexStatus{
		LastIndexedTime: time.Now(),
		DocumentCount:   docCount,
		CollectionName:  collectionName,
	}
	return SaveIndexStatus(status)
}

// IsIndexed 检查指定集合是否已经索引
func IsIndexed(collectionName string) (bool, error) {
	status, err := LoadIndexStatus(collectionName)
	if err != nil {
		return false, fmt.Errorf("检查索引状态失败: %w", err)
	}

	// 如果索引时间不为零值，则认为已索引
	return !status.LastIndexedTime.IsZero(), nil
}

// ShouldReindex 判断是否需要重新索引
func ShouldReindex(collectionName string, force bool) (bool, error) {
	if force {
		return true, nil
	}

	indexed, err := IsIndexed(collectionName)
	if err != nil {
		return false, err
	}

	return !indexed, nil
}

// SaveRAGOptions 保存RAG选项到配置
func SaveRAGOptions(options RAGOptions, name string) error {
	return configStore.Set(name, options)
}

// LoadRAGOptions 从配置加载RAG选项
func LoadRAGOptions(name string) (RAGOptions, error) {
	var options RAGOptions
	_, err := configStore.Get(name, &options)
	if err != nil {
		// 如果文件不存在，返回默认选项
		if fmt.Sprintf("%v", err) == fmt.Sprintf("文件不存在: %s.json", name) {
			return DefaultOptions, nil
		}
		return DefaultOptions, fmt.Errorf("加载RAG配置失败: %w", err)
	}

	return options, nil
}
