package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// NewProject 创建一个新的文档树
func NewProject(rootPath string) *Project {
	return &Project{
		root: &Node{
			Name:     "/",
			IsDir:    true,
			Children: make(map[string]*Node),
		},
		rootPath: rootPath,
	}
}

func (d *Project) GetRootPath() string {
	return d.rootPath
}

// CreateDir 创建一个新目录
func (d *Project) CreateDir(path string, info os.FileInfo) error {
	if path == "." {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	parent, name, err := d.resolvePath(path)
	if err != nil {
		return err
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	if _, exists := parent.Children[name]; exists {
		return errors.New("directory already exists")
	}

	parent.Children[name] = &Node{
		Name:     name,
		IsDir:    true,
		Info:     info,
		Children: make(map[string]*Node),
		Parent:   parent,
	}

	return nil
}

// CreateFile 创建一个新文件
func (d *Project) CreateFile(path string, content []byte, info os.FileInfo) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	parent, name, err := d.resolvePath(path)
	if err != nil {
		return err
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	if _, exists := parent.Children[name]; exists {
		return errors.New("file already exists")
	}

	parent.Children[name] = &Node{
		Name:     name,
		IsDir:    false,
		Info:     info,
		Content:  content,
		Parent:   parent,
		Children: make(map[string]*Node),
	}

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

	d.root.mu.RLock()
	defer d.root.mu.RUnlock()

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
