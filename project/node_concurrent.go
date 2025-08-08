package project

import (
	"context"

	"github.com/sjzsdu/tong/helper/coroutine"
)

// ProcessConcurrent 并行处理节点树
func (n *Node) ProcessConcurrent(ctx context.Context, maxWorkers int, processFunc func(*Node) (interface{}, error)) map[string]coroutine.TreeResult[interface{}] {
	if n == nil {
		return make(map[string]coroutine.TreeResult[interface{}])
	}

	// 创建处理函数的适配器
	adapterProcessFunc := func(treeNode coroutine.TreeNode) (interface{}, error) {
		if node, ok := treeNode.(*Node); ok {
			return processFunc(node)
		}
		return nil, nil
	}

	// 直接使用 Node 作为 TreeNode，因为 Node 现在实现了 TreeNode 接口
	return coroutine.ProcessTree(ctx, maxWorkers, n, adapterProcessFunc)
}

// ProcessConcurrentBFS 使用BFS策略并行处理节点树
func (n *Node) ProcessConcurrentBFS(ctx context.Context, maxWorkers int, processFunc func(*Node) (interface{}, error)) map[string]coroutine.TreeResult[interface{}] {
	if n == nil {
		return make(map[string]coroutine.TreeResult[interface{}])
	}

	// 创建处理函数的适配器
	adapterProcessFunc := func(treeNode coroutine.TreeNode) (interface{}, error) {
		if node, ok := treeNode.(*Node); ok {
			return processFunc(node)
		}
		return nil, nil
	}

	// 直接使用 Node 作为 TreeNode
	return coroutine.ProcessTreeBFS(ctx, maxWorkers, n, adapterProcessFunc)
}

// ProcessConcurrentTyped 泛型版本的并行处理节点树
func ProcessConcurrentTyped[T any](ctx context.Context, node *Node, maxWorkers int, processFunc func(*Node) (T, error)) map[string]coroutine.TreeResult[T] {
	if node == nil {
		return make(map[string]coroutine.TreeResult[T])
	}

	// 创建处理函数的适配器
	adapterProcessFunc := func(treeNode coroutine.TreeNode) (T, error) {
		if n, ok := treeNode.(*Node); ok {
			return processFunc(n)
		}
		var zero T
		return zero, nil
	}

	// 直接使用 Node 作为 TreeNode
	return coroutine.ProcessTree(ctx, maxWorkers, node, adapterProcessFunc)
}

// ProcessConcurrentBFSTyped 泛型版本的BFS并行处理节点树
func ProcessConcurrentBFSTyped[T any](ctx context.Context, node *Node, maxWorkers int, processFunc func(*Node) (T, error)) map[string]coroutine.TreeResult[T] {
	if node == nil {
		return make(map[string]coroutine.TreeResult[T])
	}

	// 创建处理函数的适配器
	adapterProcessFunc := func(treeNode coroutine.TreeNode) (T, error) {
		if n, ok := treeNode.(*Node); ok {
			return processFunc(n)
		}
		var zero T
		return zero, nil
	}

	// 直接使用 Node 作为 TreeNode
	return coroutine.ProcessTreeBFS(ctx, maxWorkers, node, adapterProcessFunc)
}

// Project 的多协程遍历方法

// ProcessConcurrent 并行处理项目中的所有节点
func (p *Project) ProcessConcurrent(ctx context.Context, maxWorkers int, processFunc func(*Node) (interface{}, error)) map[string]coroutine.TreeResult[interface{}] {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.root == nil {
		return make(map[string]coroutine.TreeResult[interface{}])
	}

	return p.root.ProcessConcurrent(ctx, maxWorkers, processFunc)
}

// ProcessConcurrentBFS 使用BFS策略并行处理项目中的所有节点
func (p *Project) ProcessConcurrentBFS(ctx context.Context, maxWorkers int, processFunc func(*Node) (interface{}, error)) map[string]coroutine.TreeResult[interface{}] {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.root == nil {
		return make(map[string]coroutine.TreeResult[interface{}])
	}

	return p.root.ProcessConcurrentBFS(ctx, maxWorkers, processFunc)
}

// ProcessProjectConcurrentTyped 泛型版本的并行处理项目中的所有节点
func ProcessProjectConcurrentTyped[T any](ctx context.Context, p *Project, maxWorkers int, processFunc func(*Node) (T, error)) map[string]coroutine.TreeResult[T] {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.root == nil {
		return make(map[string]coroutine.TreeResult[T])
	}

	return ProcessConcurrentTyped(ctx, p.root, maxWorkers, processFunc)
}

// ProcessProjectConcurrentBFSTyped 泛型版本的BFS并行处理项目中的所有节点
func ProcessProjectConcurrentBFSTyped[T any](ctx context.Context, p *Project, maxWorkers int, processFunc func(*Node) (T, error)) map[string]coroutine.TreeResult[T] {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.root == nil {
		return make(map[string]coroutine.TreeResult[T])
	}

	return ProcessConcurrentBFSTyped(ctx, p.root, maxWorkers, processFunc)
}