package project

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (p *Project) GetRootPath() string {
	return p.rootPath
}

// CreateDir 创建一个新目录
func (p *Project) CreateDir(path string, info os.FileInfo) error {
	if path == "." {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	parent, name, err := p.resolvePath(path)
	if err != nil {
		return err
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	if _, exists := parent.Children[name]; exists {
		return errors.New("directory already exists")
	}

	// 确保路径以 / 开头
	cleanPath := path
	if len(cleanPath) > 0 && cleanPath[0] != '/' {
		cleanPath = "/" + cleanPath
	}

	// 构建完整路径
	nodePath := filepath.Join(p.rootPath, cleanPath)

	node := &Node{
		Name:     name,
		Path:     nodePath, // 设置完整路径
		IsDir:    true,
		Info:     info,
		Children: make(map[string]*Node),
		Parent:   parent,
	}

	parent.Children[name] = node

	// 添加到 nodes 映射中
	if p.nodes == nil {
		p.nodes = make(map[string]*Node)
	}
	p.nodes[cleanPath] = node

	return nil
}

// CreateFile 创建一个新文件
func (p *Project) CreateFile(path string, content []byte, info os.FileInfo) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	parent, name, err := p.resolvePath(path)
	if err != nil {
		return err
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	if _, exists := parent.Children[name]; exists {
		return errors.New("file already exists")
	}

	// 确保路径以 / 开头
	cleanPath := path
	if len(cleanPath) > 0 && cleanPath[0] != '/' {
		cleanPath = "/" + cleanPath
	}

	// 构建完整路径
	nodePath := filepath.Join(p.rootPath, cleanPath)

	node := &Node{
		Name:          name,
		Path:          nodePath, // 设置完整路径
		IsDir:         false,
		Info:          info,
		Content:       content,
		ContentLoaded: true, // 设置内容已加载标志
		Parent:        parent,
		Children:      make(map[string]*Node),
	}

	parent.Children[name] = node

	// 添加到 nodes 映射中
	if p.nodes == nil {
		p.nodes = make(map[string]*Node)
	}
	p.nodes[cleanPath] = node

	return nil
}

// 修改 Project 中的方法
func (d *Project) ReadFile(path string) ([]byte, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	node, err := d.findNode(path)
	if err != nil {
		return nil, err
	}

	return node.ReadContent()
}

func (d *Project) WriteFile(path string, content []byte) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	node, err := d.findNode(path)
	if err != nil {
		return err
	}

	return node.WriteContent(content)
}

// 辅助函数，用于解析路径
func (d *Project) resolvePath(path string) (*Node, string, error) {
	// 处理根路径
	if path == "/" || path == "" {
		return d.root, "", nil
	}

	// 清理路径
	path = filepath.Clean(path)
	// 移除开头的 /
	if path[0] == '/' {
		path = path[1:]
	}

	// 分割路径组件
	components := strings.Split(path, string(filepath.Separator))
	parent := d.root

	// 遍历到倒数第二个组件
	for i := 0; i < len(components)-1; i++ {
		comp := components[i]
		if comp == "" {
			continue
		}

		parent.mu.RLock()
		child, ok := parent.Children[comp]
		parent.mu.RUnlock()

		if !ok {
			return parent, components[len(components)-1], nil
		}
		if !child.IsDir {
			return nil, "", errors.New("path component is not a directory")
		}
		parent = child
	}

	return parent, components[len(components)-1], nil
}

// 辅助函数，用于查找节点
func (d *Project) findNode(path string) (*Node, error) {
	// 处理根路径
	if path == "/" || path == "" {
		return d.root, nil
	}

	// 清理路径
	path = filepath.Clean(path)
	// 移除开头的 /
	if path[0] == '/' {
		path = path[1:]
	}

	// 分割路径组件
	components := strings.Split(path, string(filepath.Separator))
	current := d.root

	// 遍历所有组件
	for _, comp := range components {
		if comp == "" {
			continue
		}

		current.mu.RLock()
		child, ok := current.Children[comp]
		current.mu.RUnlock()

		if !ok {
			return nil, errors.New("path not found")
		}
		current = child
	}

	return current, nil
}

// IsEmpty 检查项目是否为空
func (d *Project) IsEmpty() bool {
	if d == nil || d.root == nil {
		return true
	}

	// 检查根节点是否有子节点
	d.root.mu.RLock()
	defer d.root.mu.RUnlock()

	// 空目录项目只有根节点，没有子节点
	return len(d.root.Children) == 0
}

func (p *Project) GetAbsolutePath(path string) string {
	return filepath.Join(p.rootPath, path)
}

// GetTotalNodes 计算项目中的总节点数（文件+目录）
func (p *Project) GetTotalNodes() int {
	if p.root == nil {
		return 0
	}
	return p.root.CountNodes()
}

// GetAllFiles 返回项目中所有文件的相对路径
func (p *Project) GetAllFiles() ([]string, error) {
	if p.root == nil {
		return nil, fmt.Errorf("project root is nil")
	}

	var files []string
	traverser := NewTreeTraverser(p)
	visitor := VisitorFunc(func(path string, node *Node, depth int) error {
		if node.IsDir {
			return nil
		}
		files = append(files, path)
		return nil
	})
	err := traverser.TraverseTree(visitor)

	if err != nil {
		return nil, err
	}
	return files, nil
}

// ListFiles 返回项目中所有文件的名称（不包含路径）
func (p *Project) ListFiles() ([]string, error) {
	if p.root == nil {
		return nil, fmt.Errorf("project root is nil")
	}

	return p.root.ListFiles(), nil
}

func (p *Project) GetName() string {
	if p.rootPath == "" {
		return "root"
	}
	return filepath.Base(p.rootPath)
}

// FindNode 查找指定路径的节点（公开方法）
func (p *Project) FindNode(path string) (*Node, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.findNode(path)
}

func (p *Project) SaveToFS() error {
	if p.root == nil {
		return nil
	}

	return p.saveNodeToFS(p.root, p.rootPath)
}

func (p *Project) saveNodeToFS(node *Node, path string) error {
	if node == nil {
		return nil
	}

	nodePath := filepath.Join(path, node.Name)

	// 如果是目录，确保目录存在
	if node.IsDir {
		if err := os.MkdirAll(nodePath, 0755); err != nil {
			return err
		}

		// 递归处理所有子节点
		for _, child := range node.Children {
			if err := p.saveNodeToFS(child, nodePath); err != nil {
				return err
			}
		}
		return nil
	}

	// 如果是文件且被修改，写入磁盘
	if !node.IsDir && node.IsModified() {
		if err := os.WriteFile(nodePath, node.Content, 0644); err != nil {
			return err
		}
		node.ClearModified()
	}

	return nil
}

func (p *Project) AutoSave(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			p.SaveToFS()
		}
	}()
}

// DeleteNode 从项目中删除指定节点（文件或目录）
func (p *Project) DeleteNode(node *Node) error {
	if node == nil {
		return errors.New("cannot delete nil node")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// 如果节点是根节点，不允许删除
	if node == p.root {
		return errors.New("cannot delete root node")
	}

	parent := node.Parent
	if parent == nil {
		return errors.New("node has no parent")
	}

	// 获取父节点写锁
	parent.mu.Lock()
	defer parent.mu.Unlock()

	// 检查节点是否存在于父节点的子节点映射中
	if _, exists := parent.Children[node.Name]; !exists {
		return errors.New("node not found in parent's children")
	}

	// 从父节点的子节点映射中删除节点
	delete(parent.Children, node.Name)

	// 标记父节点为已修改
	parent.MarkModified()

	// 如果文件系统也要删除，这里可以增加实际文件系统的删除操作
	if p.rootPath != "" {
		nodePath := filepath.Join(p.rootPath, p.GetNodePath(node))
		if node.IsDir {
			// 删除目录
			err := os.RemoveAll(nodePath)
			if err != nil {
				// 即使文件系统操作失败，内存中的节点已经被删除
				log.Printf("Warning: Failed to remove directory from filesystem: %v", err)
			}
		} else {
			// 删除文件
			err := os.Remove(nodePath)
			if err != nil {
				// 即使文件系统操作失败，内存中的节点已经被删除
				log.Printf("Warning: Failed to remove file from filesystem: %v", err)
			}
		}
	}

	return nil
}

// GetPath 获取节点在项目中的相对路径
func (p *Project) GetNodePath(node *Node) string {
	if node == nil || node == p.root {
		return "/"
	}

	var path []string
	current := node

	// 从当前节点向上遍历到根节点，收集路径组件
	for current != nil && current != p.root {
		path = append([]string{current.Name}, path...)
		current = current.Parent
	}

	return "/" + filepath.Join(path...)
}

// Traverse 便捷方法，使用前序遍历访问项目中的所有节点
func (p *Project) Traverse(fn func(node *Node) error) error {
	if p.root == nil {
		return nil
	}

	traverser := NewTreeTraverser(p)
	visitor := VisitorFunc(func(path string, node *Node, depth int) error {
		return fn(node)
	})

	return traverser.TraverseTree(visitor)
}

// CreateFileNode 创建一个新文件节点，但不加载内容
func (p *Project) CreateFileNode(path string, info os.FileInfo) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	parent, name, err := p.resolvePath(path)
	if err != nil {
		return err
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	if _, exists := parent.Children[name]; exists {
		return errors.New("file already exists")
	}

	// 构建完整路径
	// 确保路径以 / 开头
	cleanPath := path
	if len(cleanPath) > 0 && cleanPath[0] != '/' {
		cleanPath = "/" + cleanPath
	}
	absPath := filepath.Join(p.rootPath, cleanPath)

	node := &Node{
		Name:          name,
		Path:          absPath,
		IsDir:         false,
		Info:          info,
		ContentLoaded: false,
		Parent:        parent,
		Children:      make(map[string]*Node),
	}

	parent.Children[name] = node

	// 添加到 nodes 映射中
	if p.nodes == nil {
		p.nodes = make(map[string]*Node)
	}
	// 确保在nodes映射中使用标准化的路径
	p.nodes[cleanPath] = node

	return nil
}

// CreateFileWithContent 创建一个新文件节点并加载内容
func (p *Project) CreateFileWithContent(path string, content []byte, info os.FileInfo) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	parent, name, err := p.resolvePath(path)
	if err != nil {
		return err
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	if _, exists := parent.Children[name]; exists {
		return errors.New("file already exists")
	}

	// 确保路径以 / 开头
	cleanPath := path
	if len(cleanPath) > 0 && cleanPath[0] != '/' {
		cleanPath = "/" + cleanPath
	}

	// 构建完整路径
	nodePath := filepath.Join(p.rootPath, cleanPath)

	node := &Node{
		Name:          name,
		Path:          nodePath,
		IsDir:         false,
		Info:          info,
		Content:       content,
		ContentLoaded: true,
		Parent:        parent,
		Children:      make(map[string]*Node),
	}

	parent.Children[name] = node

	// 添加到 nodes 映射中
	if p.nodes == nil {
		p.nodes = make(map[string]*Node)
	}
	p.nodes[cleanPath] = node

	return nil
}
