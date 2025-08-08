package project

import (
	"os"
	"sync"
)

type Node struct {
	Name          string
	Path          string
	IsDir         bool
	modified      bool
	Info          os.FileInfo
	Content       []byte // 文件内容
	ContentLoaded bool   // 标记内容是否已加载
	Children      map[string]*Node
	Parent        *Node
	mu            sync.RWMutex
}

// Project 表示整个文档树
type Project struct {
	root     *Node
	rootPath string
	inGit    bool
	nodes    map[string]*Node
	mu       sync.RWMutex
}

// VisitorFunc 定义了访问节点的函数类型
type VisitorFunc func(path string, node *Node, depth int) error

// VisitNode 实现 NodeVisitor 接口
func (f VisitorFunc) VisitNode(node *Node, path string, level int) error {
	return f(path, node, level)
}
