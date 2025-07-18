package project

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// TraverseOrder 定义遍历顺序
type TraverseOrder int

const (
	PreOrder  TraverseOrder = iota // 前序遍历
	PostOrder                      // 后序遍历
	InOrder                        // 中序遍历
)

// ProgressCallback 进度回调函数类型
type ProgressCallback func(current int, filePath string)

// TraverseOption 定义遍历选项
type TraverseOption struct {
	ContinueOnError  bool             // 遇到错误时是否继续
	Errors           []error          // 记录所有错误
	ProgressCallback ProgressCallback // 进度回调函数
	ProcessedFiles   int              // 已处理文件数
}

// TreeTraverser 提供了树遍历的基本功能
type TreeTraverser struct {
	project *Project
	order   TraverseOrder
	option  *TraverseOption
	wg      sync.WaitGroup // 添加等待组
}

// SetOption 设置遍历选项
func (t *TreeTraverser) SetOption(option *TraverseOption) *TreeTraverser {
	t.option = option
	return t
}

// WithProgressCallback 设置进度回调函数
func (t *TreeTraverser) WithProgressCallback(callback ProgressCallback) *TreeTraverser {
	if t.option == nil {
		t.option = &TraverseOption{
			Errors: make([]error, 0),
		}
	}
	t.option.ProgressCallback = callback
	t.option.ProcessedFiles = 0
	return t
}

// WithContinueOnError 设置遇到错误时是否继续
func (t *TreeTraverser) WithContinueOnError(continueOnError bool) *TreeTraverser {
	if t.option == nil {
		t.option = &TraverseOption{
			Errors: make([]error, 0),
		}
	}
	t.option.ContinueOnError = continueOnError
	return t
}

// GetErrors 获取遍历过程中收集的错误
func (t *TreeTraverser) GetErrors() []error {
	if t.option == nil {
		return nil
	}
	return t.option.Errors
}

// HasErrors 检查遍历过程中是否有错误
func (t *TreeTraverser) HasErrors() bool {
	return t.option != nil && len(t.option.Errors) > 0
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

// TraverseNode 遍历指定路径的节点
func (t *TreeTraverser) TraverseNode(visitor NodeVisitor, filePath string) error {
	node, err := t.project.FindNode(filePath)
	if err != nil {
		return fmt.Errorf("文件路径 %s 不存在", filePath)
	}
	return t.Traverse(node, filePath, 0, visitor)
}

// TraverseTree 遍历整个项目树
func (t *TreeTraverser) TraverseTree(visitor NodeVisitor) error {
	if t.project.root == nil {
		return nil
	}
	return t.Traverse(t.project.root, "/", 0, visitor)
}

// TraverseTreeParallel 并行遍历整个项目树
// 此方法适用于大型项目，可以显著提高遍历速度
func (t *TreeTraverser) TraverseTreeParallel(visitor NodeVisitor) error {
	if t.project.root == nil {
		return nil
	}

	// 设置默认选项
	if t.option == nil {
		t.option = &TraverseOption{
			ContinueOnError: false,
			Errors:          make([]error, 0),
		}
	}

	// 创建一个带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// 创建错误通道
	errChan := make(chan error, 100)

	// 启动根节点的遍历
	go func() {
		defer close(errChan)
		t.traverseParallel(ctx, t.project.root, "/", 0, visitor, errChan)
	}()

	// 收集错误
	var errs []error
	for err := range errChan {
		if err != nil {
			if !t.option.ContinueOnError {
				return err
			}
			errs = append(errs, err)
		}
	}

	// 处理收集到的错误
	if len(errs) > 0 {
		t.option.Errors = append(t.option.Errors, errs...)
		if !t.option.ContinueOnError {
			return fmt.Errorf("遍历过程中发生 %d 个错误", len(errs))
		}
	}

	return nil
}

// traverseParallel 并行遍历节点
func (t *TreeTraverser) traverseParallel(ctx context.Context, node *Node, path string, depth int, visitor NodeVisitor, errChan chan<- error) {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		errChan <- fmt.Errorf("遍历被取消: %v", ctx.Err())
		return
	default:
		// 继续执行
	}

	// 处理空节点
	if node == nil {
		return
	}

	// 处理文件节点
	if !node.IsDir {
		if err := visitor.VisitFile(node, path, depth); err != nil {
			errChan <- &traverseError{
				Path:     path,
				NodeName: node.Name,
				Err:      err,
			}
		}

		// 更新进度（需要线程安全）
		if t.option.ProgressCallback != nil {
			// 使用原子操作更新计数
			t.option.ProcessedFiles++
			t.option.ProgressCallback(t.option.ProcessedFiles, path)
		}

		return
	}

	// 跳过特殊目录
	if node.Name == "." {
		return
	}

	// 根据遍历顺序决定是否先访问当前目录
	if t.order == PreOrder {
		if err := visitor.VisitDirectory(node, path, depth); err != nil {
			errChan <- &traverseError{
				Path:     path,
				NodeName: node.Name,
				Err:      err,
			}
			if !t.option.ContinueOnError {
				return
			}
		}
	}

	// 获取并排序子节点
	node.mu.RLock()
	children := make([]*Node, 0, len(node.Children))
	for _, child := range node.Children {
		children = append(children, child)
	}
	node.mu.RUnlock()

	sort.Slice(children, func(i, j int) bool {
		return children[i].Name < children[j].Name
	})

	// 使用信号量限制并发
	sem := make(chan struct{}, maxConcurrentTraversals)
	var wg sync.WaitGroup

	// 并行处理子节点
	for _, child := range children {
		childPath := filepath.Join(path, child.Name)
		wg.Add(1)

		go func(c *Node, p string) {
			defer wg.Done()
			// 获取信号量
			sem <- struct{}{}
			defer func() { <-sem }()

			// 处理 panic
			defer func() {
				if r := recover(); r != nil {
					errChan <- &traverseError{
						Path:     p,
						NodeName: c.Name,
						Err:      fmt.Errorf("panic in traversal: %v", r),
					}
				}
			}()

			// 递归遍历子节点
			t.traverseParallel(ctx, c, p, depth+1, visitor, errChan)
		}(child, childPath)
	}

	// 等待所有子节点处理完成
	wg.Wait()

	// 根据遍历顺序决定是否后访问当前目录
	if t.order == PostOrder || t.order == InOrder {
		if err := visitor.VisitDirectory(node, path, depth); err != nil {
			errChan <- &traverseError{
				Path:     path,
				NodeName: node.Name,
				Err:      err,
			}
		}
	}
}

// traversePreOrder 处理前序遍历
func (t *TreeTraverser) traversePreOrder(node *Node, children []*Node, path string, depth int, visitor NodeVisitor) error {
	// 初始化选项
	if t.option == nil {
		t.option = &TraverseOption{
			ContinueOnError: false,
			Errors:          make([]error, 0),
		}
	}

	// 先访问当前目录
	if err := visitor.VisitDirectory(node, path, depth); err != nil {
		if !t.option.ContinueOnError {
			return &traverseError{
				Path:     path,
				NodeName: node.Name,
				Err:      err,
			}
		}
		t.option.Errors = append(t.option.Errors, &traverseError{
			Path:     path,
			NodeName: node.Name,
			Err:      err,
		})
	}

	// 然后访问子节点
	for _, child := range children {
		childPath := filepath.Join(path, child.Name)
		if err := t.Traverse(child, childPath, depth+1, visitor); err != nil {
			if !t.option.ContinueOnError {
				return err
			}
			t.option.Errors = append(t.option.Errors, err)
		}
	}
	return nil
}

// traversePostOrder 处理后序遍历
func (t *TreeTraverser) traversePostOrder(node *Node, children []*Node, path string, depth int, visitor NodeVisitor) error {
	// 初始化选项
	if t.option == nil {
		t.option = &TraverseOption{
			ContinueOnError: false,
			Errors:          make([]error, 0),
		}
	}

	// 使用 WaitGroup 和 errChan 来管理并发和错误收集
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

			if err := t.Traverse(c, p, depth+1, visitor); err != nil {
				errChan <- &traverseError{
					Path:     p,
					NodeName: c.Name,
					Err:      err,
				}
			}
		}(child, childPath)
	}

	// 创建一个 context 用于超时控制
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 等待所有子节点完成并收集错误
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// 收集所有错误，设置超时
	var errs []error
	errCollection := make(chan struct{})

	go func() {
		for err := range errChan {
			if err != nil {
				if t.option.ContinueOnError {
					errs = append(errs, err)
				} else {
					// 如果不继续执行，则取消操作
					cancel()
					return
				}
			}
		}
		close(errCollection)
	}()

	// 等待完成或超时
	select {
	case <-done:
		// 关闭错误通道，等待错误收集完成
		close(errChan)
		<-errCollection
	case <-ctx.Done():
		return fmt.Errorf("遍历超时: 路径 '%s'", path)
	}

	// 如果有错误且设置了继续执行
	if len(errs) > 0 {
		t.option.Errors = append(t.option.Errors, errs...)
		if !t.option.ContinueOnError {
			return fmt.Errorf("遍历过程中发生 %d 个错误", len(errs))
		}
	}

	// 所有子节点处理完成后，处理当前目录
	if err := visitor.VisitDirectory(node, path, depth); err != nil {
		return &traverseError{
			Path:     path,
			NodeName: node.Name,
			Err:      err,
		}
	}

	return nil
}

// traverseInOrder 处理中序遍历
func (t *TreeTraverser) traverseInOrder(node *Node, children []*Node, path string, depth int, visitor NodeVisitor) error {
	// 初始化选项
	if t.option == nil {
		t.option = &TraverseOption{
			ContinueOnError: false,
			Errors:          make([]error, 0),
		}
	}

	mid := len(children) / 2

	// 前半部分
	for i := 0; i < mid; i++ {
		childPath := filepath.Join(path, children[i].Name)
		if err := t.Traverse(children[i], childPath, depth+1, visitor); err != nil {
			if !t.option.ContinueOnError {
				return err
			}
			t.option.Errors = append(t.option.Errors, err)
		}
	}

	// 当前节点
	if err := visitor.VisitDirectory(node, path, depth); err != nil {
		if !t.option.ContinueOnError {
			return &traverseError{
				Path:     path,
				NodeName: node.Name,
				Err:      err,
			}
		}
		t.option.Errors = append(t.option.Errors, &traverseError{
			Path:     path,
			NodeName: node.Name,
			Err:      err,
		})
	}

	// 后半部分
	for i := mid; i < len(children); i++ {
		childPath := filepath.Join(path, children[i].Name)
		if err := t.Traverse(children[i], childPath, depth+1, visitor); err != nil {
			if !t.option.ContinueOnError {
				return err
			}
			t.option.Errors = append(t.option.Errors, err)
		}
	}
	return nil
}

// 添加一个用于限制并发的常量
const maxConcurrentTraversals = 10

// Traverse 遍历节点的通用方法
func (t *TreeTraverser) Traverse(node *Node, path string, depth int, visitor NodeVisitor) error {
	// 初始化选项（如果尚未初始化）
	if t.option == nil {
		t.option = &TraverseOption{
			ContinueOnError: false,
			Errors:          make([]error, 0),
		}
	}

	// 处理空节点
	if node == nil {
		return nil
	}

	// 处理文件节点
	if !node.IsDir {
		if err := visitor.VisitFile(node, path, depth); err != nil {
			// 记录错误信息
			fileErr := &traverseError{
				Path:     path,
				NodeName: node.Name,
				Err:      err,
			}

			if t.option.ContinueOnError {
				t.option.Errors = append(t.option.Errors, fileErr)
			} else {
				return fileErr
			}
		}

		// 更新进度
		if t.option.ProgressCallback != nil {
			t.option.ProcessedFiles++
			t.option.ProgressCallback(t.option.ProcessedFiles, path)
		}

		return nil
	}

	// 跳过特殊目录
	if node.Name == "." {
		return nil
	}

	// 对子节点进行排序，确保遍历顺序一致
	node.mu.RLock()
	children := make([]*Node, 0, len(node.Children))
	for _, child := range node.Children {
		children = append(children, child)
	}
	node.mu.RUnlock()

	sort.Slice(children, func(i, j int) bool {
		return children[i].Name < children[j].Name
	})

	// 根据遍历顺序选择相应的处理方法
	switch t.order {
	case PreOrder:
		return t.traversePreOrder(node, children, path, depth, visitor)
	case PostOrder:
		return t.traversePostOrder(node, children, path, depth, visitor)
	case InOrder:
		return t.traverseInOrder(node, children, path, depth, visitor)
	default:
		// 默认使用前序遍历
		return t.traversePreOrder(node, children, path, depth, visitor)
	}
}
