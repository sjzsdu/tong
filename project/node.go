package project

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// CalculateHash 计算节点的哈希值
func (node *Node) CalculateHash() (string, error) {
	if node.IsDir {
		return node.calculateDirHash()
	}
	return node.calculateFileHash()
}

// calculateFileHash 计算文件内容的哈希值
func (node *Node) calculateFileHash() (string, error) {
	if node.Content == nil {
		return "", nil
	}
	hash := sha256.Sum256(node.Content)
	return hex.EncodeToString(hash[:]), nil
}

// calculateDirHash 计算目录的哈希值
func (node *Node) calculateDirHash() (string, error) {
	var hashes []string
	// 先对 Children 按名称排序
	sortedChildren := make([]*Node, len(node.Children))
	i := 0
	for _, child := range node.Children {
		sortedChildren[i] = child
		i++
	}
	sort.Slice(sortedChildren, func(i, j int) bool {
		return sortedChildren[i].Name < sortedChildren[j].Name
	})

	// 使用排序后的切片计算哈希
	for _, child := range sortedChildren {
		hash, err := child.CalculateHash()
		if err != nil {
			return "", err
		}
		hashes = append(hashes, hash)
	}

	combined := []byte(strings.Join(hashes, ""))
	hash := sha256.Sum256(combined)
	return hex.EncodeToString(hash[:]), nil
}

func countNodes(node *Node) int {
	if node == nil || node.Name == "." {
		return 0
	}

	// 检查是否是特殊目录
	if node.Info != nil && node.Info.IsDir() && node.Info.Name() == "." {
		return 0
	}

	count := 1 // 当前节点
	for _, child := range node.Children {
		count += countNodes(child)
	}
	return count
}
