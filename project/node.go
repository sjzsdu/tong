package project

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sjzsdu/tong/helper/coroutine"
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
	// 加读锁保护访问
	node.mu.RLock()
	defer node.mu.RUnlock()

	// 如果内容未加载，先加载内容
	if !node.ContentLoaded {
		node.mu.RUnlock()
		_, err := node.ReadContent()
		if err != nil {
			return "", err
		}
		node.mu.RLock()
	}

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

// CountNodes 计算节点及其子节点的总数
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

// GetFiles 获取节点及其子节点的所有文件路径
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

	// 获取项目根路径
	var rootPath string
	if n.Parent != nil {
		// 向上查找到根节点
		root := n
		for root.Parent != nil {
			root = root.Parent
		}

		// 获取项目实例
		project := GetProjectByRoot(root)
		if project == nil {
			return nil, errors.New("cannot find project for node: project not registered")
		}
		rootPath = project.GetRootPath()
	} else {
		// 如果是根节点，直接使用Path作为绝对路径
		return nil, errors.New("root node should not have content")
	}

	// 构建文件系统路径
	fsPath := filepath.Join(rootPath, n.Path[1:])

	// 使用文件系统路径读取文件内容
	content, err := os.ReadFile(fsPath)
	if err != nil {
		return nil, err
	}

	// 更新节点状态
	n.Content = content
	n.ContentLoaded = true

	return n.Content, nil
}

// WriteContent 写入节点内容并标记为已修改
func (n *Node) WriteContent(content []byte) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.IsDir {
		return errors.New("cannot write to directory")
	}

	// 更新内存中的内容
	n.Content = content
	n.ContentLoaded = true
	n.MarkModified()

	// 获取项目根路径并写入文件系统
	var rootPath string
	if n.Parent != nil {
		// 向上查找到根节点
		root := n
		for root.Parent != nil {
			root = root.Parent
		}

		// 获取项目实例
		project := GetProjectByRoot(root)
		if project == nil {
			return errors.New("cannot find project for node: project not registered")
		}
		rootPath = project.GetRootPath()
	} else {
		// 如果是根节点，直接使用Path作为绝对路径
		return errors.New("root node should not have content")
	}

	// 构建文件系统路径
	fsPath := filepath.Join(rootPath, n.Path[1:])

	// 写入文件
	if err := os.WriteFile(fsPath, content, 0644); err != nil {
		return err
	}

	// 更新文件信息
	fileInfo, err := os.Stat(fsPath)
	if err != nil {
		return err
	}
	n.Info = fileInfo

	return nil
}

// UnloadContent 卸载节点内容以节省内存
func (n *Node) UnloadContent() error {
	if n.IsDir {
		return errors.New("cannot unload content for directory")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	// 如果内容已修改，先保存到文件系统
	if n.modified {
		// 获取项目根路径
		var rootPath string
		if n.Parent != nil {
			// 向上查找到根节点
			root := n
			for root.Parent != nil {
				root = root.Parent
			}

			// 获取项目实例
			project := GetProjectByRoot(root)
			if project == nil {
				return errors.New("cannot find project for node: project not registered")
			}
			rootPath = project.GetRootPath()
		} else {
			// 如果是根节点，直接使用Path作为绝对路径
			return errors.New("root node should not have content")
		}

		// 构建文件系统路径
		fsPath := filepath.Join(rootPath, n.Path[1:])

		// 写入文件
		if err := os.WriteFile(fsPath, n.Content, 0644); err != nil {
			return err
		}
		n.modified = false
	}

	// 卸载内容
	n.Content = nil
	n.ContentLoaded = false

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
	if n.Children == nil {
		n.Children = make(map[string]*Node)
	}
	n.Children[child.Name] = child
	n.MarkModified()
	return nil
}

// RemoveChild 移除子节点
func (n *Node) RemoveChild(name string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.IsDir {
		return errors.New("cannot remove child from a file node")
	}

	if _, exists := n.Children[name]; !exists {
		return errors.New("child does not exist")
	}

	delete(n.Children, name)
	n.MarkModified()
	return nil
}

// GetChild 获取子节点
func (n *Node) GetChild(name string) (*Node, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	child, exists := n.Children[name]
	return child, exists
}

// MarkModified 标记节点为已修改，并递归标记父节点
func (n *Node) MarkModified() {
	n.modified = true
	if n.Parent != nil {
		n.Parent.MarkModified()
	}
}

// IsModified 检查节点是否被修改
func (n *Node) IsModified() bool {
	return n.modified
}

// ClearModified 清除节点及其子节点的修改标记
func (n *Node) ClearModified() {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.modified = false
	for _, child := range n.Children {
		child.ClearModified()
	}
}

// SaveToDisk 将节点内容保存到磁盘
func (n *Node) SaveToDisk() error {
	if n.IsDir {
		// 对于目录，递归保存所有子节点
		n.mu.RLock()
		children := make([]*Node, 0, len(n.Children))
		for _, child := range n.Children {
			children = append(children, child)
		}
		n.mu.RUnlock()

		for _, child := range children {
			if err := child.SaveToDisk(); err != nil {
				return err
			}
		}
		return nil
	}

	// 对于文件，检查是否已修改
	n.mu.RLock()
	if !n.modified || !n.ContentLoaded {
		n.mu.RUnlock()
		return nil
	}

	// 获取内容的副本
	content := make([]byte, len(n.Content))
	copy(content, n.Content)
	n.mu.RUnlock()

	// 获取项目根路径
	var rootPath string
	if n.Parent != nil {
		// 向上查找到根节点
		root := n
		for root.Parent != nil {
			root = root.Parent
		}

		// 获取项目实例
		project := GetProjectByRoot(root)
		if project == nil {
			return errors.New("cannot find project for node")
		}
		rootPath = project.GetRootPath()
	} else {
		// 如果是根节点，直接使用Path作为绝对路径
		return errors.New("root node should not have content")
	}

	// 构建文件系统路径
	fsPath := filepath.Join(rootPath, n.Path[1:])

	// 写入文件
	if err := os.WriteFile(fsPath, content, 0644); err != nil {
		return err
	}

	// 清除修改标记
	n.mu.Lock()
	n.modified = false
	n.mu.Unlock()

	return nil
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

// GetChildren 获取所有子节点（实现TreeNode接口）
func (n *Node) GetChildren() []coroutine.TreeNode {
	n.mu.RLock()
	defer n.mu.RUnlock()

	children := make([]coroutine.TreeNode, 0, len(n.Children))
	for _, child := range n.Children {
		children = append(children, child)
	}
	return children
}

// GetChildrenNodes 获取所有子节点（返回具体类型）
func (n *Node) GetChildrenNodes() []*Node {
	n.mu.RLock()
	defer n.mu.RUnlock()

	children := make([]*Node, 0, len(n.Children))
	for _, child := range n.Children {
		children = append(children, child)
	}
	return children
}

// GetID 获取节点ID
func (n *Node) GetID() string {
	return n.Path
}

// Clone 创建节点的深度克隆
func (n *Node) Clone() *Node {
	n.mu.RLock()
	defer n.mu.RUnlock()

	clone := &Node{
		Name:          n.Name,
		Path:          n.Path,
		IsDir:         n.IsDir,
		modified:      n.modified,
		Info:          n.Info,
		ContentLoaded: n.ContentLoaded,
	}

	// 复制内容
	if n.Content != nil {
		clone.Content = make([]byte, len(n.Content))
		copy(clone.Content, n.Content)
	}

	// 递归克隆子节点
	if n.IsDir && len(n.Children) > 0 {
		clone.Children = make(map[string]*Node, len(n.Children))
		for name, child := range n.Children {
			childClone := child.Clone()
			childClone.Parent = clone
			clone.Children[name] = childClone
		}
	}

	return clone
}

// EnsureContentLoaded 确保节点内容已加载
func (n *Node) EnsureContentLoaded() error {
	if n.IsDir {
		return nil // 目录节点不需要加载内容
	}

	n.mu.RLock()
	if n.ContentLoaded {
		n.mu.RUnlock()
		return nil
	}
	n.mu.RUnlock()

	_, err := n.ReadContent()
	return err
}

// SaveToFS 将节点及其子节点保存到文件系统
// SaveToFS 将节点及其子节点保存到文件系统
func (n *Node) SaveToFS() error {
	if n == nil {
		return nil
	}

	// 获取项目实例
	project := n.GetProject()
	if project == nil {
		return errors.New("cannot find project for node")
	}

	// 获取项目根路径
	rootPath := project.GetRootPath()

	n.mu.RLock()
	defer n.mu.RUnlock()

	// 构建当前节点的完整路径
	var nodePath string
	if n.Path == "/" {
		// 根节点
		nodePath = rootPath
	} else {
		// 非根节点
		nodePath = filepath.Join(rootPath, n.Path[1:])
	}

	// 如果是目录，创建目录并递归保存子节点
	if n.IsDir {
		// 创建目录
		err := os.MkdirAll(nodePath, 0755)
		if err != nil {
			return err
		}

		// 递归保存子节点
		for _, child := range n.Children {
			err := child.SaveToFS()
			if err != nil {
				return err
			}
		}
	} else {
		// 如果是文件，写入文件内容
		if n.ContentLoaded {
			// 如果内容已加载，直接写入
			err := os.WriteFile(nodePath, n.Content, 0644)
			if err != nil {
				return err
			}
		} else {
			// 如果内容未加载，且文件不存在，则创建空文件
			if _, err := os.Stat(nodePath); os.IsNotExist(err) {
				err := os.WriteFile(nodePath, []byte{}, 0644)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// GetProject 获取节点所属的项目实例
func (n *Node) GetProject() *Project {
	if n == nil {
		return nil
	}

	// 向上查找到根节点
	root := n
	for root.Parent != nil {
		root = root.Parent
	}

	// 使用根节点查找项目实例
	return GetProjectByRoot(root)
}
