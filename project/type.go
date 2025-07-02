package project

import (
	"os"
	"sync"
)

type Node struct {
	Name        string
	IsDir       bool
	Info        os.FileInfo
	Content     []byte
	LLMResponse *LLMResponse    // 改为指针类型
	Children    map[string]*Node
	Parent      *Node
	mu          sync.RWMutex
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
