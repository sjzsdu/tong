package rag

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sjzsdu/tong/config"
	"github.com/sjzsdu/tong/lang"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
)

// VectorStoreAddDocuments 向量存储添加文档的接口
type VectorStoreAddDocuments interface {
	AddDocuments(ctx context.Context, docs []schema.Document, opts ...vectorstores.Option) ([]string, error)
}

// DocumentSyncManager 文档同步管理器
type DocumentSyncManager struct {
	// 向量存储
	VectorStore     VectorStoreAddDocuments
	DocsDirectory   string
	Metadata        map[string]DocumentMetadata
	StorageOptions  StorageOptions
	SplitterOptions SplitterOptions
}

// NewDocumentSyncManager 创建新的文档同步管理器
func NewDocumentSyncManager(vectorStore VectorStoreAddDocuments,
	storageOptions StorageOptions, splitterOptions SplitterOptions, docsDir string) *DocumentSyncManager {
	return &DocumentSyncManager{
		VectorStore:     vectorStore,
		DocsDirectory:   docsDir,
		Metadata:        make(map[string]DocumentMetadata),
		StorageOptions:  storageOptions,
		SplitterOptions: splitterOptions,
	}
}

// SyncDocuments 同步文档
func (m *DocumentSyncManager) SyncDocuments(ctx context.Context) error {
	fmt.Println(lang.T("开始同步文档..."))

	// 1. 扫描文档目录
	currentDocs, err := m.scanDocumentDirectory()
	if err != nil {
		return &RagError{
			Code:    "scan_docs_failed",
			Message: "扫描文档目录失败",
			Cause:   err,
		}
	}

	// 2. 加载已有的元数据
	err = m.loadMetadata()
	if err != nil {
		// 如果无法加载元数据，可能是首次运行，继续处理
		fmt.Println(lang.T("无法加载元数据，可能是首次运行"))
	}

	// 3. 识别需要添加、更新和删除的文档
	toAdd, toUpdate, toDelete := m.identifyChanges(currentDocs)

	// 打印同步计划
	fmt.Printf(lang.T("同步计划: 添加 %d 个文档, 更新 %d 个文档, 删除 %d 个文档\n"),
		len(toAdd), len(toUpdate), len(toDelete))

	// 4. 删除不再存在的文档
	if len(toDelete) > 0 {
		err = m.deleteDocuments(ctx, toDelete)
		if err != nil {
			return err
		}
	}

	// 5. 更新已修改的文档
	if len(toUpdate) > 0 {
		err = m.updateDocuments(ctx, toUpdate)
		if err != nil {
			return err
		}
	}

	// 6. 添加新文档
	if len(toAdd) > 0 {
		err = m.addNewDocuments(ctx, toAdd)
		if err != nil {
			return err
		}
	}

	// 7. 保存更新后的元数据
	return m.saveMetadata()
}

// scanDocumentDirectory 扫描文档目录，返回文件路径到文件信息的映射
func (m *DocumentSyncManager) scanDocumentDirectory() (map[string]os.FileInfo, error) {
	result := make(map[string]os.FileInfo)

	err := filepath.Walk(m.DocsDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 检查是否是支持的文件类型
		ext := filepath.Ext(path)
		if supported, known := supportedExtensions[ext]; known && supported {
			result[path] = info
		}

		return nil
	})

	return result, err
}

// loadMetadata 从配置中加载元数据
func (m *DocumentSyncManager) loadMetadata() error {
	metadataJSON := config.GetConfig("RAG_DOCS_METADATA_" + m.StorageOptions.CollectionName)
	if metadataJSON == "" {
		return &RagError{
			Code:    "no_metadata",
			Message: "未找到元数据",
		}
	}

	var metadata map[string]DocumentMetadata
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		return &RagError{
			Code:    "invalid_metadata",
			Message: "元数据格式无效",
			Cause:   err,
		}
	}

	m.Metadata = metadata
	return nil
}

// saveMetadata 保存元数据到配置
func (m *DocumentSyncManager) saveMetadata() error {
	metadataJSON, err := json.Marshal(m.Metadata)
	if err != nil {
		return &RagError{
			Code:    "marshal_metadata_failed",
			Message: "序列化元数据失败",
			Cause:   err,
		}
	}

	config.SetConfig("RAG_DOCS_METADATA_"+m.StorageOptions.CollectionName, string(metadataJSON))
	return config.SaveConfig()
}

// identifyChanges 识别文档变化
func (m *DocumentSyncManager) identifyChanges(currentDocs map[string]os.FileInfo) ([]string, []string, []string) {
	var toAdd, toUpdate, toDelete []string

	// 找出需要添加和更新的文档
	for path, info := range currentDocs {
		meta, exists := m.Metadata[path]
		if !exists {
			// 新文档
			toAdd = append(toAdd, path)
		} else if info.ModTime().After(meta.LastModified) {
			// 检查文件哈希是否变化
			newHash, err := calculateFileHash(path)
			if err != nil || newHash != meta.Hash {
				toUpdate = append(toUpdate, path)
			}
		}
	}

	// 找出需要删除的文档
	for path := range m.Metadata {
		if _, exists := currentDocs[path]; !exists {
			toDelete = append(toDelete, path)
		}
	}

	return toAdd, toUpdate, toDelete
}

// calculateFileHash 计算文件的MD5哈希值
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// deleteDocuments 从向量存储中删除文档
func (m *DocumentSyncManager) deleteDocuments(ctx context.Context, paths []string) error {
	fmt.Println(lang.T("删除过期文档..."))

	// 收集要删除的向量ID
	var vectorIDs []string
	for _, path := range paths {
		if meta, exists := m.Metadata[path]; exists {
			vectorIDs = append(vectorIDs, meta.VectorIDs...)
			// 从元数据中删除
			delete(m.Metadata, path)
		}
	}

	// TODO: 实现向量存储的删除功能
	// 目前大多数向量存储都支持按ID删除，但不同实现可能有差异
	// 这里需要根据实际使用的向量存储来实现
	fmt.Printf(lang.T("需要删除 %d 个向量\n"), len(vectorIDs))

	return nil
}

// updateDocuments 更新已修改的文档
func (m *DocumentSyncManager) updateDocuments(ctx context.Context, paths []string) error {
	fmt.Println(lang.T("更新已修改的文档..."))

	// 对于更新，先删除旧文档，再添加新文档
	err := m.deleteDocuments(ctx, paths)
	if err != nil {
		return err
	}

	// 添加更新后的文档
	return m.addNewDocuments(ctx, paths)
}

// addNewDocuments 添加新文档
func (m *DocumentSyncManager) addNewDocuments(ctx context.Context, paths []string) error {
	fmt.Println(lang.T("添加新文档..."))

	for _, path := range paths {
		// 加载文档
		docs, err := LoadDocument(ctx, path)
		if err != nil {
			fmt.Printf(lang.T("警告: 加载文档 %s 失败: %v, 跳过\n"), path, err)
			continue
		}

		// 分割文档
		splitDocs, err := SplitDocuments(docs, m.SplitterOptions)
		if err != nil {
			fmt.Printf(lang.T("警告: 分割文档 %s 失败: %v, 跳过\n"), path, err)
			continue
		}

		// 存储文档并获取向量ID
		vectorIDs, err := m.VectorStore.AddDocuments(ctx, splitDocs)
		if err != nil {
			fmt.Printf(lang.T("警告: 存储文档 %s 失败: %v, 跳过\n"), path, err)
			continue
		}

		// 计算文件哈希
		hash, err := calculateFileHash(path)
		if err != nil {
			hash = "" // 如果计算失败，使用空字符串
		}

		// 更新元数据
		fileInfo, _ := os.Stat(path)
		m.Metadata[path] = DocumentMetadata{
			Path:         path,
			LastModified: fileInfo.ModTime(),
			Hash:         hash,
			VectorIDs:    vectorIDs,
		}

		fmt.Printf(lang.T("文档 %s 已添加/更新, 生成了 %d 个向量\n"), path, len(vectorIDs))
	}

	return nil
}

// SyncDocumentsForRAG 为RAG系统同步文档
func SyncDocumentsForRAG(rag *RAG) error {
	// 确保同步管理器已初始化
	if rag.SyncManager == nil {
		rag.SyncManager = NewDocumentSyncManager(rag.VectorStore, rag.Options.Storage, rag.Options.Splitter, rag.Options.DocsDir)
	}

	// 执行同步
	return rag.SyncManager.SyncDocuments(context.Background())
}

// StartAutomaticSync 启动自动同步服务
func StartAutomaticSync(rag *RAG) error {
	// 检查同步选项
	if rag.Options.Sync.SyncInterval <= 0 {
		return &RagError{
			Code:    "invalid_sync_interval",
			Message: "同步间隔必须大于0",
		}
	}

	// 确保同步管理器已初始化
	if rag.SyncManager == nil {
		rag.SyncManager = NewDocumentSyncManager(rag.VectorStore, rag.Options.Storage, rag.Options.Splitter, rag.Options.DocsDir)
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 启动定时同步
	go func() {
		ticker := time.NewTicker(rag.Options.Sync.SyncInterval)
		defer ticker.Stop()
		defer cancel()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fmt.Println(lang.T("执行自动文档同步..."))
				err := rag.SyncManager.SyncDocuments(ctx)
				if err != nil {
					fmt.Printf(lang.T("自动同步失败: %v\n"), err)
				} else {
					fmt.Println(lang.T("自动同步完成"))
				}
			}
		}
	}()

	fmt.Printf(lang.T("自动同步服务已启动，间隔: %v\n"), rag.Options.Sync.SyncInterval)
	return nil
}
