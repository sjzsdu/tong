package project

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
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

func (n *Node) CountNodes() int {
	if n == nil {
		return 0
	}

	n.mu.RLock()
	defer n.mu.RUnlock()

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

// Node 中添加的方法
func (n *Node) ReadContent() ([]byte, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.IsDir {
		return nil, errors.New("cannot read directory")
	}

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
