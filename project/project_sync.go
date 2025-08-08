package project

import (
	"errors"
	"log"
	"os"
	"path/filepath"
)

// 持久化和同步相关方法

// SaveToFS 将项目保存到文件系统
func (p *Project) SaveToFS() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.root == nil {
		return errors.New("project is empty")
	}

	if p.rootPath == "" {
		return errors.New("project has no root path")
	}

	// 确保根目录存在
	err := os.MkdirAll(p.rootPath, 0755)
	if err != nil {
		return err
	}

	// 使用根节点的SaveToFS方法保存所有节点
	return p.root.SaveToFS()
}

// SyncFromFS 从文件系统同步项目
func (p *Project) SyncFromFS() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.rootPath == "" {
		return errors.New("project has no root path")
	}
	// 清空当前项目
	p.root.Children = make(map[string]*Node)
	p.nodes = make(map[string]*Node)
	p.nodes["/"] = p.root

	// 递归加载文件系统
	return filepath.Walk(p.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Warning: Error accessing path %s: %v", path, err)
			return nil // 跳过无法访问的文件或目录
		}

		// 跳过根目录
		if path == p.rootPath {
			return nil
		}

		// 计算相对路径
		relPath, err := filepath.Rel(p.rootPath, path)
		if err != nil {
			log.Printf("Warning: Cannot get relative path for %s: %v", path, err)
			return nil
		}

		// 转换为项目路径格式
		projPath := "/" + filepath.ToSlash(relPath)

		// 创建节点（使用内部方法，避免重复加锁）
		if info.IsDir() {
			err = p.createDirInternal(projPath, info)
		} else {
			// 对于文件，只创建节点，不加载内容
			err = p.createFileNodeInternal(projPath, info)
		}

		if err != nil {
			log.Printf("Warning: Error creating node for %s: %v", path, err)
		}

		return nil
	})
}

// LoadFileContent 加载文件内容
func (p *Project) LoadFileContent(path string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 查找节点
	node, err := p.FindNode(path)
	if err != nil {
		return err
	}

	if node.IsDir {
		return errors.New("cannot load content for directory")
	}

	// 使用Node的ReadContent方法加载内容
	_, err = node.ReadContent()
	return err
}

// UnloadFileContent 卸载文件内容以节省内存
func (p *Project) UnloadFileContent(path string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 查找节点
	node, err := p.FindNode(path)
	if err != nil {
		return err
	}

	// 使用Node的UnloadContent方法卸载内容
	return node.UnloadContent()
}

// createDirInternal 内部创建目录方法（不加锁）
func (p *Project) createDirInternal(path string, info os.FileInfo) error {
	// 标准化路径
	cleanPath := p.NormalizePath(path)

	// 检查节点是否已存在
	_, exists := p.nodes[cleanPath]
	if exists {
		return nil // 节点已存在，跳过
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
		Info:     info,
	}

	// 添加节点到项目
	p.addNodeToProject(node, parent, name, cleanPath)

	return nil
}

// createFileNodeInternal 内部创建文件节点方法（不加锁）
func (p *Project) createFileNodeInternal(path string, info os.FileInfo) error {
	// 标准化路径
	cleanPath := p.NormalizePath(path)

	// 检查节点是否已存在
	_, exists := p.nodes[cleanPath]
	if exists {
		return nil // 节点已存在，跳过
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
		Info:          info,
	}

	// 添加节点到项目
	p.addNodeToProject(node, parent, name, cleanPath)

	return nil
}
