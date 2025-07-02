package project

import (
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// NodeVisitor 定义了节点访问器的接口
type NodeVisitor interface {
	// VisitDirectory 访问目录节点
	VisitDirectory(node *Node, path string, level int) error
	// VisitFile 访问文件节点
	VisitFile(node *Node, path string, level int) error
}

// TraverseOrder 定义遍历顺序
type TraverseOrder int

const (
	PreOrder  TraverseOrder = iota // 前序遍历
	PostOrder                      // 后序遍历
	InOrder                        // 中序遍历
)

// TraverseOption 定义遍历选项
type TraverseOption struct {
	ContinueOnError bool    // 遇到错误时是否继续
	Errors          []error // 记录所有错误
}

// TreeTraverser 提供了树遍历的基本功能
type TreeTraverser struct {
	project *Project
	order   TraverseOrder
	option  *TraverseOption
	wg      sync.WaitGroup // 添加等待组
}

// SetOption 设置遍历选项
func (t *TreeTraverser) SetOption(option *TraverseOption) {
	t.option = option
}

// NewTreeTraverser 创建一个树遍历器，默认使用前序遍历
func NewTreeTraverser(p *Project) *TreeTraverser {
	return &TreeTraverser{
		project: p,
		order:   PreOrder,
		option:  nil,
	}
}

// SetTraverseOrder 设置遍历顺序
func (t *TreeTraverser) SetTraverseOrder(order TraverseOrder) *TreeTraverser {
	t.order = order
	return t
}

// TraverseTree 遍历整个项目树
func (t *TreeTraverser) TraverseTree(visitor NodeVisitor) error {
	if t.project.root == nil {
		return nil
	}
	return t.Traverse(t.project.root, "/", 0, visitor)
}

// traversePreOrder 处理前序遍历
func (t *TreeTraverser) traversePreOrder(node *Node, children []*Node, path string, level int, visitor NodeVisitor) error {
	if err := visitor.VisitDirectory(node, path, level); err != nil {
		return err
	}
	for _, child := range children {
		childPath := filepath.Join(path, child.Name)
		if err := t.Traverse(child, childPath, level+1, visitor); err != nil {
			return err
		}
	}
	return nil
}

// traverseError 封装遍历过程中的错误信息
type traverseError struct {
	Path     string
	NodeName string
	Err      error
}

func (e *traverseError) Error() string {
	return fmt.Sprintf("遍历错误 [%s] 在节点 '%s': %v", e.Path, e.NodeName, e.Err)
}

// 添加一个用于限制并发的常量
const maxConcurrentTraversals = 10

// traversePostOrder 处理后序遍历
func (t *TreeTraverser) traversePostOrder(node *Node, children []*Node, path string, level int, visitor NodeVisitor) error {
	// 初始化选项
	if t.option == nil {
		t.option = &TraverseOption{
			ContinueOnError: false,
			Errors:          make([]error, 0),
		}
	}

	var wg sync.WaitGroup
	errChan := make(chan *traverseError, len(children))

	// 使用信号量限制并发
	sem := make(chan struct{}, maxConcurrentTraversals)

	// 处理子节点
	for _, child := range children {
		childPath := filepath.Join(path, child.Name)
		wg.Add(1)
		go func(c *Node, p string) {
			// 获取信号量
			sem <- struct{}{}
			defer func() {
				<-sem // 释放信号量
				if r := recover(); r != nil {
					errChan <- &traverseError{
						Path:     p,
						NodeName: c.Name,
						Err:      fmt.Errorf("panic in traversal: %v", r),
					}
				}
				wg.Done()
			}()

			if err := t.Traverse(c, p, level+1, visitor); err != nil {
				errChan <- &traverseError{
					Path:     p,
					NodeName: c.Name,
					Err:      err,
				}
			}
		}(child, childPath)
	}

	// 等待所有子节点完成并收集错误
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
		close(errChan)
	}()

	// 收集所有错误，设置超时
	var errs []error
	timeout := time.After(5 * time.Minute) // 设置合理的超时时间

	for {
		select {
		case err, ok := <-errChan:
			if !ok {
				goto PROCESS_DIRECTORY
			}
			if err != nil {
				if t.option.ContinueOnError {
					errs = append(errs, err)
				} else {
					return err
				}
			}
		case <-timeout:
			return fmt.Errorf("遍历超时: 路径 '%s'", path)
		case <-done:
			goto PROCESS_DIRECTORY
		}
	}

PROCESS_DIRECTORY:
	// 如果有错误且设置了继续执行
	if len(errs) > 0 {
		t.option.Errors = append(t.option.Errors, errs...)
		if !t.option.ContinueOnError {
			return fmt.Errorf("遍历过程中发生 %d 个错误", len(errs))
		}
	}

	// 所有子节点处理完成后，处理当前目录
	if err := visitor.VisitDirectory(node, path, level); err != nil {
		return &traverseError{
			Path:     path,
			NodeName: node.Name,
			Err:      err,
		}
	}

	return nil
}

// traverseInOrder 处理中序遍历
func (t *TreeTraverser) traverseInOrder(node *Node, children []*Node, path string, level int, visitor NodeVisitor) error {
	mid := len(children) / 2

	// 前半部分
	for i := 0; i < mid; i++ {
		childPath := filepath.Join(path, children[i].Name)
		if err := t.Traverse(children[i], childPath, level+1, visitor); err != nil {
			return err
		}
	}

	// 当前节点
	if err := visitor.VisitDirectory(node, path, level); err != nil {
		return err
	}

	// 后半部分
	for i := mid; i < len(children); i++ {
		childPath := filepath.Join(path, children[i].Name)
		if err := t.Traverse(children[i], childPath, level+1, visitor); err != nil {
			return err
		}
	}
	return nil
}

// Traverse 遍历节点的通用方法
func (t *TreeTraverser) Traverse(node *Node, path string, level int, visitor NodeVisitor) error {
	if node == nil {
		return nil
	}

	if !node.IsDir {
		if err := visitor.VisitFile(node, path, level); err != nil {
			fmt.Println("visit file error:", err)
			return err
		}
		return nil
	}

	if node.Name == "." {
		return nil
	}

	// 对子节点进行排序
	children := make([]*Node, 0, len(node.Children))
	for _, child := range node.Children {
		children = append(children, child)
	}
	sort.Slice(children, func(i, j int) bool {
		return children[i].Name < children[j].Name
	})

	// 根据遍历顺序选择相应的处理方法
	switch t.order {
	case PreOrder:
		return t.traversePreOrder(node, children, path, level, visitor)
	case PostOrder:
		return t.traversePostOrder(node, children, path, level, visitor)
	case InOrder:
		return t.traverseInOrder(node, children, path, level, visitor)
	}

	return nil
}
