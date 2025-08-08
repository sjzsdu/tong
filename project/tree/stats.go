package tree

import (
	"fmt"

	"github.com/sjzsdu/tong/project"
)

// Statistics 树的统计信息
type Statistics struct {
	TotalNodes     int   // 总节点数
	DirectoryCount int   // 目录数量
	FileCount      int   // 文件数量
	TotalSize      int64 // 总大小（字节）
	MaxDepth       int   // 最大深度
}

// Stats 返回树的统计信息
func Stats(node *project.Node) Statistics {
	if node == nil {
		return Statistics{}
	}
	
	stats := Statistics{}
	collectStats(node, &stats)
	return stats
}

// collectStats 递归收集统计信息
func collectStats(node *project.Node, stats *Statistics) {
	stats.TotalNodes++
	
	if node.IsDir {
		stats.DirectoryCount++
		// 递归处理子节点
		for _, child := range node.Children {
			collectStats(child, stats)
		}
	} else {
		stats.FileCount++
		if node.Info != nil {
			stats.TotalSize += node.Info.Size()
		}
	}
}

// String 返回统计信息的字符串表示
func (s Statistics) String() string {
	var sizeStr string
	if s.TotalSize < 1024 {
		sizeStr = fmt.Sprintf("%d bytes", s.TotalSize)
	} else if s.TotalSize < 1024*1024 {
		sizeStr = fmt.Sprintf("%.1f KB", float64(s.TotalSize)/1024)
	} else if s.TotalSize < 1024*1024*1024 {
		sizeStr = fmt.Sprintf("%.1f MB", float64(s.TotalSize)/(1024*1024))
	} else {
		sizeStr = fmt.Sprintf("%.1f GB", float64(s.TotalSize)/(1024*1024*1024))
	}
	
	return fmt.Sprintf("%d directories, %d files, %s total", 
		s.DirectoryCount, s.FileCount, sizeStr)
}