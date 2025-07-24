package project

import (
	"fmt"
	"os"
	"path/filepath"
)

// BuildIndex 为项目构建索引，确保所有文件内容被加载
// 这个方法会遍历项目中的所有文件，并加载它们的内容
// 对于并发访问和搜索操作非常有用
func (p *Project) BuildIndex() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 创建访问者函数
	visitor := VisitorFunc(func(path string, node *Node, depth int) error {
		if !node.IsDir {
			// 确保文件内容已加载
			_, err := node.ReadContent()
			if err != nil {
				// 如果文件不存在，尝试在文件系统中创建它
				absPath := filepath.Join(p.rootPath, path)
				dir := filepath.Dir(absPath)
				
				// 确保目录存在
				if err := os.MkdirAll(dir, 0755); err != nil {
					return fmt.Errorf("无法创建目录 %s: %v", dir, err)
				}
				
				// 创建空文件
				if err := os.WriteFile(absPath, []byte{}, 0644); err != nil {
					return fmt.Errorf("无法创建文件 %s: %v", absPath, err)
				}
				
				// 更新节点内容
				node.Content = []byte{}
				node.ContentLoaded = true
			}
		}
		return nil
	})

	// 遍历项目树
	traverser := NewTreeTraverser(p)
	err := traverser.TraverseTree(visitor)
	if err != nil {
		return err
	}

	return nil
}