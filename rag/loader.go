package rag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sjzsdu/tong/lang"
	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/schema"
)

// 支持的文件类型
var supportedExtensions = map[string]bool{
	".txt":  true,
	".md":   true,
	".pdf":  true,
	".csv":  true,
	".html": false, // 暂不支持，但已识别
	".htm":  false, // 暂不支持，但已识别
	".json": false, // 暂不支持，但已识别
}

// LoadDocument 加载单个文档
func LoadDocument(ctx context.Context, path string) ([]schema.Document, error) {
	// 打开文件
	file, err := os.Open(path)
	if err != nil {
		return nil, &RagError{
			Code:    "open_file_failed",
			Message: "打开文件失败",
			Cause:   err,
		}
	}
	defer file.Close()

	// 创建加载器
	loader, err := CreateLoader(file, path)
	if err != nil {
		return nil, err
	}

	// 加载文档
	docs, err := loader.Load(ctx)
	if err != nil {
		return nil, &RagError{
			Code:    "load_document_failed",
			Message: fmt.Sprintf("加载文件 %s 失败", path),
			Cause:   err,
		}
	}

	// 为文档添加源信息
	for i := range docs {
		if docs[i].Metadata == nil {
			docs[i].Metadata = make(map[string]any)
		}
		docs[i].Metadata["source"] = path
		docs[i].Metadata["filename"] = filepath.Base(path)
	}

	return docs, nil
}

// LoadDocumentsFromDir 从目录加载所有文档
func LoadDocumentsFromDir(ctx context.Context, dir string) ([]schema.Document, error) {
	var allDocs []schema.Document
	var supportedCount, unsupportedCount int

	// 确保目录存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, &RagError{
			Code:    "directory_not_exist",
			Message: fmt.Sprintf("目录 %s 不存在", dir),
			Cause:   err,
		}
	}

	// 遍历目录
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 检查文件类型
		ext := strings.ToLower(filepath.Ext(path))
		supported, known := supportedExtensions[ext]

		if !known {
			// 未知文件类型，跳过
			unsupportedCount++
			return nil
		}

		if !supported {
			// 已知但不支持的文件类型，跳过
			unsupportedCount++
			return nil
		}

		// 加载文档
		docs, err := LoadDocument(ctx, path)
		if err != nil {
			// 记录错误但继续处理其他文件
			fmt.Printf("警告: 加载文件 %s 失败: %v\n", path, err)
			return nil
		}

		// 为文档添加额外的相对路径元数据
		for i := range docs {
			relPath, _ := filepath.Rel(dir, path)
			docs[i].Metadata["rel_path"] = relPath
		}

		allDocs = append(allDocs, docs...)
		supportedCount++
		return nil
	})

	if err != nil {
		return nil, &RagError{
			Code:    "walk_directory_failed",
			Message: fmt.Sprintf("遍历目录 %s 失败", dir),
			Cause:   err,
		}
	}

	// 输出加载统计信息
	fmt.Printf(lang.T("文档加载完成: 成功加载 %d 个文件, 跳过 %d 个不支持的文件\n"),
		supportedCount, unsupportedCount)

	return allDocs, nil
}

// CreateLoader 根据文件路径创建合适的文档加载器
func CreateLoader(file *os.File, path string) (documentloaders.Loader, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".txt", ".md":
		return documentloaders.NewText(file), nil
	case ".pdf":
		// 获取文件大小
		fileInfo, err := file.Stat()
		if err != nil {
			return nil, &RagError{
				Code:    "get_file_info_failed",
				Message: "获取PDF文件信息失败",
				Cause:   err,
			}
		}
		return documentloaders.NewPDF(file, fileInfo.Size()), nil
	case ".csv":
		return documentloaders.NewCSV(file), nil
	case ".html", ".htm":
		return nil, &RagError{
			Code:    "loader_not_implemented",
			Message: "HTML加载器尚未实现",
		}
	case ".json":
		return nil, &RagError{
			Code:    "loader_not_implemented",
			Message: "JSON加载器尚未实现",
		}
	default:
		return nil, &RagError{
			Code:    "unsupported_file_type",
			Message: fmt.Sprintf("不支持的文件类型: %s", ext),
		}
	}
}
