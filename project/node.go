package project

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

func (node *Node) SetLLMResponse(response string) *Node {
	llmResp, err := NewLLMResponse(response)
	if err != nil {
		// 如果解析失败，可以记录日志或者其他处理
		return node
	}
	node.LLMResponse = llmResp
	return node
}

func (node *Node) GetLLMResponse() string {
	if node.LLMResponse == nil {
		return ""
	}
	jsonStr, err := node.LLMResponse.ToJSON()
	if err != nil {
		return ""
	}
	return jsonStr
}

// GetLLMResponseContent 返回所有符号的名称和特性
func (node *Node) GetLLMResponseContent() interface{} {
	if node.LLMResponse == nil {
		return nil
	}

	var result []Item
	// 从 Functions 添加
	for _, f := range node.LLMResponse.Functions {
		result = append(result, Item{Name: f.Name, Feature: f.Feature})
	}
	// 从 Classes 添加
	for _, c := range node.LLMResponse.Classes {
		result = append(result, Item{Name: c.Name, Feature: c.Feature})
	}
	// 从 Interfaces 添加
	for _, i := range node.LLMResponse.Interfaces {
		result = append(result, Item{Name: i.Name, Feature: i.Feature})
	}
	// 从 Variables 添加
	for _, v := range node.LLMResponse.Variables {
		result = append(result, Item{Name: v.Name, Feature: v.Feature})
	}
	// 从 OtherSymbols 添加
	for _, s := range node.LLMResponse.OtherSymbols {
		result = append(result, Item{Name: s.Name, Feature: s.Feature})
	}

	return result
}

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

// GetChildrenResponses 获取直接子节点的 LLMResponse 内容
func (node *Node) GetChildrenResponses() (string, error) {
	if !node.IsDir || len(node.Children) == 0 {
		return "", nil
	}

	var responses []string
	hasValidContent := false

	// 对子节点进行排序以保证顺序一致
	children := make([]*Node, 0, len(node.Children))
	for _, child := range node.Children {
		children = append(children, child)
	}
	sort.Slice(children, func(i, j int) bool {
		return children[i].Name < children[j].Name
	})

	for _, child := range children {
		// 跳过非程序文件的响应
		if child.LLMResponse != nil && child.LLMResponse.IsNotProgramResponse() {
			continue
		}

		if child.LLMResponse != nil {
			if !child.IsDir && len(child.Content) == 0 {
				return "", fmt.Errorf("empty file content: %s", child.Name)
			}

			// 使用简化版的响应内容
			if items, ok := child.GetLLMResponseContent().([]Item); ok {
				var itemStrs []string
				for _, item := range items {
					itemStrs = append(itemStrs, fmt.Sprintf("%s: %s", item.Name, item.Feature))
				}
				responses = append(responses, fmt.Sprintf("%s:\n%s", child.Name, strings.Join(itemStrs, "\n")))
				hasValidContent = true
			}
		}
	}

	if !hasValidContent || len(responses) == 0 {
		return "", fmt.Errorf("no valid content found in directory")
	}

	return strings.Join(responses, "\n\n"), nil
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
