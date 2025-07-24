package project

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
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
	// 加读锁保护 Children 访问
	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.Content == nil {
		return "", nil
	}
	hash := sha256.Sum256(node.Content)
	return hex.EncodeToString(hash[:]), nil
}

// calculateDirHash 计算目录的哈希值
func (node *Node) calculateDirHash() (string, error) {
	// 加读锁保护 Children 访问
	node.mu.RLock()
	defer node.mu.RUnlock()

	// 处理空目录情况
	if len(node.Children) == 0 {
		// 为空目录返回特定哈希值
		emptyHash := sha256.Sum256([]byte("empty_dir:" + node.Name))
		return hex.EncodeToString(emptyHash[:]), nil
	}

	// 更高效地创建和填充切片
	sortedChildren := make([]*Node, 0, len(node.Children))
	for _, child := range node.Children {
		sortedChildren = append(sortedChildren, child)
	}

	// 排序保持不变
	sort.Slice(sortedChildren, func(i, j int) bool {
		return sortedChildren[i].Name < sortedChildren[j].Name
	})

	// 计算子节点哈希
	hashes := make([]string, 0, len(sortedChildren))
	for _, child := range sortedChildren {
		hash, err := child.CalculateHash()
		if err != nil {
			return "", err
		}
		hashes = append(hashes, hash)
	}

	// 合并哈希值并计算最终哈希
	combined := []byte(strings.Join(hashes, ""))
	hash := sha256.Sum256(combined)
	return hex.EncodeToString(hash[:]), nil
}

func (n *Node) CountNodes() int {
	if n == nil {
		return 0
	}

	n.mu.RLock()
	defer n.mu.RUnlock()

	// 如果是根节点且没有子节点，则只计算根节点本身
	if len(n.Children) == 0 {
		return 1
	}

	count := 1 // 当前节点
	for _, child := range n.Children {
		count += child.CountNodes()
	}
	return count
}

func (n *Node) GetFiles(basePath string) []string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var files []string
	currentPath := filepath.Join(basePath, n.Name)

	if !n.IsDir {
		return []string{currentPath}
	}

	for _, child := range n.Children {
		childFiles := child.GetFiles(currentPath)
		files = append(files, childFiles...)
	}

	return files
}

// ReadContent 读取节点内容，支持延迟加载
func (n *Node) ReadContent() ([]byte, error) {
	n.mu.RLock()

	if n.IsDir {
		n.mu.RUnlock()
		return nil, errors.New("cannot read directory")
	}

	// 如果内容已加载，直接返回
	if n.ContentLoaded && n.Content != nil {
		content := n.Content
		n.mu.RUnlock()
		return content, nil
	}

	// 内容未加载，从磁盘读取
	// 释放读锁，获取写锁
	n.mu.RUnlock()
	n.mu.Lock()
	defer n.mu.Unlock()
	
	// 双重检查，防止在锁切换期间其他协程已加载内容
	if n.ContentLoaded && n.Content != nil {
		return n.Content, nil
	}
	
	// 检查Path是否有效
	if n.Path == "" {
		return nil, errors.New("node path is empty")
	}
	
	// 使用 Path 字段读取文件内容
	content, err := os.ReadFile(n.Path)
	if err != nil {
		return nil, err
	}

	// 更新节点状态
	n.Content = content
	n.ContentLoaded = true
	
	return n.Content, nil
}

func (n *Node) WriteContent(content []byte) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.IsDir {
		return errors.New("cannot write to directory")
	}

	n.Content = content
	n.MarkModified()
	return nil
}

// AddChild 添加子节点
func (n *Node) AddChild(child *Node) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.IsDir {
		return errors.New("cannot add child to a file node")
	}

	if _, exists := n.Children[child.Name]; exists {
		return errors.New("child with name already exists")
	}

	child.Parent = n
	n.Children[child.Name] = child
	return nil
}

// GetChild 获取子节点
func (n *Node) GetChild(name string) (*Node, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	child, exists := n.Children[name]
	return child, exists
}

func (n *Node) MarkModified() {
	n.modified = true
	if n.Parent != nil {
		n.Parent.MarkModified()
	}
}

func (n *Node) IsModified() bool {
	return n.modified
}

func (n *Node) ClearModified() {
	n.modified = false
	for _, child := range n.Children {
		child.ClearModified()
	}
}

// ListFiles 返回节点子树中所有文件的名称（不包含路径）
func (n *Node) ListFiles() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var fileNames []string

	if !n.IsDir {
		return []string{n.Name}
	}

	for _, child := range n.Children {
		if !child.IsDir {
			fileNames = append(fileNames, child.Name)
		} else {
			childFiles := child.ListFiles()
			fileNames = append(fileNames, childFiles...)
		}
	}

	return fileNames
}
