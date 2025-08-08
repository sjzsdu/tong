package project

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/sjzsdu/tong/helper"
)

// 路径处理相关方法

// resolvePath 解析路径，返回父节点和名称
func (p *Project) resolvePath(path string) (*Node, string, error) {
	// 处理根路径
	if path == "/" {
		return nil, "", errors.New("cannot resolve root path")
	}

	// 标准化路径
	path = p.NormalizePath(path)

	// 分割路径获取目录和文件名
	dir, name := filepath.Split(path)

	// 处理目录为根目录的情况
	if dir == "/" {
		return p.root, name, nil
	}

	// 移除开头和结尾的 /
	dir = strings.TrimPrefix(dir, "/")
	dir = strings.TrimSuffix(dir, "/")

	// 查找父目录节点
	parent := p.root
	if dir != "" {
		parts := strings.Split(dir, "/")
		for _, part := range parts {
			if part == "" {
				continue
			}

			// 在访问 Children 前加锁
			parent.mu.Lock()
			child, exists := parent.Children[part]
			parent.mu.Unlock()

			if !exists {
				return nil, "", errors.New("parent directory does not exist: " + part)
			}

			if !child.IsDir {
				return nil, "", errors.New("path component is not a directory: " + part)
			}

			parent = child
		}
	}

	return parent, name, nil
}

// GetNodePath 获取节点在项目中的相对路径
func (p *Project) GetNodePath(node *Node) string {
	if node == nil {
		return "/"
	}

	// 如果节点已经有 Path，直接返回
	if node.Path != "" {
		return node.Path
	}

	// 对于根节点，返回 /
	if node == p.root {
		return "/"
	}

	// 如果节点没有 Path，则计算路径
	var path []string
	current := node

	// 从当前节点向上遍历到根节点，收集路径组件
	for current != nil && current != p.root {
		path = append([]string{current.Name}, path...)
		current = current.Parent
	}

	// 构建路径并保存到节点的 Path 字段
	node.Path = "/" + filepath.Join(path...)
	return node.Path
}

// NormalizePath 标准化路径，确保以 / 开头，并处理相对路径组件
func (p *Project) NormalizePath(path string) string {
	// 首先使用 helper.StandardizePath 进行基本标准化
	cleanPath := helper.StandardizePath(path)
	
	// 处理空路径
	if cleanPath == "" {
		return "/"
	}
	
	// 使用 filepath.Clean 处理 .. 和 . 组件
	cleanPath = filepath.Clean(cleanPath)
	
	// 确保路径以 / 开头
	if len(cleanPath) > 0 && cleanPath[0] != '/' {
		cleanPath = "/" + cleanPath
	}
	
	// 处理根路径的特殊情况
	if cleanPath == "." || cleanPath == "" {
		return "/"
	}
	
	return cleanPath
}
