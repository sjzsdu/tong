package project

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/sjzsdu/tong/helper"
)

// 节点查找和遍历相关方法

// FindNode 根据路径查找节点
func (p *Project) FindNode(path string) (*Node, error) {
	// 处理根路径
	if path == "/" {
		return p.root, nil
	}

	// 标准化路径
	cleanPath := helper.StandardizePath(path)

	// 从缓存中查找
	p.mu.RLock()
	node, exists := p.nodes[cleanPath]
	p.mu.RUnlock()

	if exists {
		return node, nil
	}

	// 如果缓存中不存在，则遍历查找
	return p.findNodeDirect(cleanPath)
}

// findNodeDirect 直接查找节点
func (p *Project) findNodeDirect(path string) (*Node, error) {
	// 处理根路径
	if path == "/" {
		return p.root, nil
	}

	// 移除开头的 /
	cleanPath := strings.TrimPrefix(path, "/")

	// 分割路径
	parts := strings.Split(cleanPath, "/")

	// 从根节点开始查找
	current := p.root
	for _, part := range parts {
		if part == "" {
			continue
		}

		current.mu.RLock()
		child, exists := current.Children[part]
		current.mu.RUnlock()

		if !exists {
			return nil, errors.New("node not found: " + part)
		}

		current = child
	}

	return current, nil
}

// ListFiles 列出指定目录下的所有文件和目录
func (p *Project) ListFiles(dirPath string) ([]string, error) {
	// 查找目录节点
	node, err := p.FindNode(dirPath)
	if err != nil {
		return nil, err
	}

	if !node.IsDir {
		return nil, errors.New("not a directory")
	}

	// 获取目录下的所有文件和目录
	var files []string

	node.mu.RLock()
	for name := range node.Children {
		files = append(files, name)
	}
	node.mu.RUnlock()

	return files, nil
}

// GetAllFiles 返回项目中所有文件的相对路径
func (p *Project) GetAllFiles() ([]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.root == nil {
		return nil, errors.New("project is empty")
	}

	files := make([]string, 0)

	// 使用访问者模式遍历所有节点
	err := p.Visit(func(path string, node *Node, depth int) error {
		if !node.IsDir {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// Visit 使用访问者模式遍历项目
func (p *Project) Visit(visitor VisitorFunc) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.root == nil {
		return errors.New("project is empty")
	}

	return p.visitNode(p.root, "/", 0, visitor)
}

// visitNode 递归遍历节点
func (p *Project) visitNode(node *Node, path string, depth int, visitor VisitorFunc) error {
	if node == nil {
		return nil
	}

	// 访问当前节点
	err := visitor(path, node, depth)
	if err != nil {
		return err
	}

	// 如果是目录，则递归访问子节点
	if node.IsDir {
		node.mu.RLock()
		children := make([]*Node, 0, len(node.Children))
		childPaths := make([]string, 0, len(node.Children))

		for name, child := range node.Children {
			children = append(children, child)
			childPath := filepath.Join(path, name)
			childPaths = append(childPaths, childPath)
		}
		node.mu.RUnlock()

		// 递归访问子节点
		for i, child := range children {
			err := p.visitNode(child, childPaths[i], depth+1, visitor)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
