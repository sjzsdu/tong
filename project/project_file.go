package project

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/sjzsdu/tong/helper"
)

// 文件和目录操作相关方法

// createNodeInFS 在文件系统中创建节点并获取文件信息
func (p *Project) createNodeInFS(fsPath string, isDir bool, content []byte) (os.FileInfo, error) {
	if isDir {
		// 创建目录
		err := os.MkdirAll(fsPath, 0755)
		if err != nil {
			return nil, err
		}
	} else {
		// 确保父目录存在
		parentDir := filepath.Dir(fsPath)
		err := os.MkdirAll(parentDir, 0755)
		if err != nil {
			return nil, err
		}

		// 写入文件内容
		err = os.WriteFile(fsPath, content, 0644)
		if err != nil {
			return nil, err
		}
	}

	// 获取文件信息
	fileInfo, err := os.Stat(fsPath)
	if err != nil {
		return nil, err
	}

	return fileInfo, nil
}

// addNodeToProject 将节点添加到项目中
func (p *Project) addNodeToProject(node *Node, parent *Node, name string, cleanPath string) {
	// 添加到父节点
	parent.mu.Lock()
	parent.Children[name] = node
	parent.mu.Unlock()

	// 添加到节点映射
	p.nodes[cleanPath] = node
}

// CreateDir 创建目录
func (p *Project) CreateDir(path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 标准化路径
	cleanPath := helper.StandardizePath(path)

	// 检查节点是否已存在
	_, exists := p.nodes[cleanPath]
	if exists {
		return errors.New("node already exists: " + cleanPath)
	}

	// 解析路径，获取父节点和目录名
	parent, name, err := p.resolvePath(cleanPath)
	if err != nil {
		return err
	}

	// 创建目录节点
	node := &Node{
		Name:     name,
		Path:     cleanPath,
		IsDir:    true,
		Children: make(map[string]*Node),
		Parent:   parent,
	}

	// 创建实际的文件系统目录并获取文件信息
	fsPath := filepath.Join(p.rootPath, cleanPath[1:])
	fileInfo, err := p.createNodeInFS(fsPath, true, nil)
	if err != nil {
		return err
	}

	// 使用文件系统中的实际信息
	node.Info = fileInfo

	// 添加节点到项目
	p.addNodeToProject(node, parent, name, cleanPath)

	return nil
}

// CreateFile 创建文件
func (p *Project) CreateFile(path string, content []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 标准化路径
	cleanPath := helper.StandardizePath(path)

	// 检查节点是否已存在
	if _, exists := p.nodes[cleanPath]; exists {
		return errors.New("node already exists: " + cleanPath)
	}

	// 解析路径，获取父节点和文件名
	parent, name, err := p.resolvePath(cleanPath)
	if err != nil {
		return err
	}

	// 创建文件节点
	node := &Node{
		Name:          name,
		Path:          cleanPath,
		IsDir:         false,
		Content:       content,
		ContentLoaded: true,
		Children:      make(map[string]*Node),
		Parent:        parent,
	}

	// 创建实际的文件系统文件并获取文件信息
	fsPath := filepath.Join(p.rootPath, cleanPath[1:])
	fileInfo, err := p.createNodeInFS(fsPath, false, content)
	if err != nil {
		return err
	}

	// 使用文件系统中的实际信息
	node.Info = fileInfo

	// 添加节点到项目
	p.addNodeToProject(node, parent, name, cleanPath)

	return nil
}

// CreateFileNode 创建文件节点（不加载内容）
func (p *Project) CreateFileNode(path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 标准化路径
	cleanPath := helper.StandardizePath(path)

	// 检查节点是否已存在
	if _, exists := p.nodes[cleanPath]; exists {
		return errors.New("node already exists: " + cleanPath)
	}

	// 解析路径，获取父节点和文件名
	parent, name, err := p.resolvePath(cleanPath)
	if err != nil {
		return err
	}

	// 创建文件节点
	node := &Node{
		Name:          name,
		Path:          cleanPath,
		IsDir:         false,
		ContentLoaded: false,
		Children:      make(map[string]*Node),
		Parent:        parent,
	}

	// 获取文件系统路径
	fsPath := filepath.Join(p.rootPath, cleanPath[1:])
	
	// 检查文件是否已存在
	fileInfo, err := os.Stat(fsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，创建空文件
			fileInfo, err = p.createNodeInFS(fsPath, false, []byte{})
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// 使用文件系统中的实际信息
	node.Info = fileInfo

	// 添加节点到项目
	p.addNodeToProject(node, parent, name, cleanPath)

	return nil
}

// CreateFileWithContent 创建文件并设置内容
func (p *Project) CreateFileWithContent(path string, content []byte) error {
	return p.CreateFile(path, content)
}

// ReadFile 读取文件内容
func (p *Project) ReadFile(path string) ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 查找节点
	node, err := p.FindNode(path)
	if err != nil {
		return nil, err
	}

	if node.IsDir {
		return nil, errors.New("cannot read directory")
	}

	// 使用Node的ReadContent方法读取内容
	return node.ReadContent()
}

// WriteFile 写入文件内容
func (p *Project) WriteFile(path string, content []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 标准化路径
	cleanPath := helper.StandardizePath(path)

	// 查找节点
	node, exists := p.nodes[cleanPath]
	if !exists {
		// 如果节点不存在，则创建新文件
		// 解析路径，获取父节点和文件名
		parent, name, err := p.resolvePath(cleanPath)
		if err != nil {
			return err
		}

		// 创建文件节点
		node = &Node{
			Name:          name,
			Path:          cleanPath,
			IsDir:         false,
			Content:       content,
			ContentLoaded: true,
			Children:      make(map[string]*Node),
			Parent:        parent,
		}

		// 创建实际的文件系统文件并获取文件信息
		fsPath := filepath.Join(p.rootPath, cleanPath[1:])
		fileInfo, err := p.createNodeInFS(fsPath, false, content)
		if err != nil {
			return err
		}

		// 使用文件系统中的实际信息
		node.Info = fileInfo

		// 添加节点到项目
		p.addNodeToProject(node, parent, name, cleanPath)
		return nil
	}

	// 使用Node的WriteContent方法写入内容
	return node.WriteContent(content)
}

// DeleteNode 删除节点
func (p *Project) DeleteNode(path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 标准化路径
	cleanPath := p.NormalizePath(path)

	// 不允许删除根节点
	if cleanPath == "/" {
		return errors.New("cannot delete root node")
	}

	// 查找节点
	node, exists := p.nodes[cleanPath]
	if !exists {
		return errors.New("node not found: " + cleanPath)
	}

	// 获取父节点
	parent := node.Parent
	if parent == nil {
		return errors.New("node has no parent")
	}

	// 从父节点中移除
	parent.mu.Lock()
	delete(parent.Children, node.Name)
	parent.mu.Unlock()

	// 从节点映射中移除当前节点及其所有子节点
	if node.IsDir {
		// 递归删除子节点
		for _, child := range node.Children {
			childPath := p.GetNodePath(child)
			delete(p.nodes, childPath)
		}
	}

	// 从节点映射中移除当前节点
	delete(p.nodes, cleanPath)

	return nil
}
