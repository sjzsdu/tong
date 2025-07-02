package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// ReadFile 读取文件内容
func (d *Project) ReadFile(path string) ([]byte, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	node, err := d.findNode(path)
	if err != nil {
		return nil, err
	}

	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.IsDir {
		return nil, errors.New("cannot read directory")
	}

	return node.Content, nil
}

// WriteFile 写入文件内容
func (d *Project) WriteFile(path string, content []byte) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	node, err := d.findNode(path)
	if err != nil {
		return err
	}

	node.mu.Lock()
	defer node.mu.Unlock()

	if node.IsDir {
		return errors.New("cannot write to directory")
	}

	node.Content = content
	return nil
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
	return countNodes(p.root)
}

func (p *Project) GetLLMResponse() string {
	if p.root == nil {
		return ""
	}
	return p.root.GetLLMResponse()
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

func (p *Project) GetName() string {
	if p.rootPath == "" {
		return "root"
	}
	return filepath.Base(p.rootPath)
}
