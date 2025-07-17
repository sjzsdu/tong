package project

import (
	"os"
	"sync"
)

type Node struct {
	Name     string
	IsDir    bool
	modified bool
	Info     os.FileInfo
	Content  []byte
	Children map[string]*Node
	Parent   *Node
	mu       sync.RWMutex
}

// Project 表示整个文档树
type Project struct {
	root     *Node
	rootPath string
	mu       sync.RWMutex
}

type Item struct {
	Name    string `json:"name"`
	Feature string `json:"feature"`
}

type Response struct {
	Functions    []Item `json:"functions"`
	Classes      []Item `json:"classes"`
	Interfaces   []Item `json:"interfaces"`
	Variables    []Item `json:"variables"`
	OtherSymbols []Item `json:"other_symbols"`
}

// VisitorFunc 定义了访问节点的函数类型
type VisitorFunc func(path string, node *Node, depth int) error

// VisitFile 实现 NodeVisitor 接口
func (f VisitorFunc) VisitFile(node *Node, path string, level int) error {
	return f(path, node, level)
}

// VisitDirectory 实现 NodeVisitor 接口
func (f VisitorFunc) VisitDirectory(node *Node, path string, level int) error {
	return f(path, node, level)
}
