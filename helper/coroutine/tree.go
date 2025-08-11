package coroutine

import (
	"context"
)

// ProcessTree 并行处理树形结构
func ProcessTree[T any](ctx context.Context, maxWorkers int, root TreeNode, processFunc func(TreeNode) (T, error)) map[string]TreeResult[T] {
	if maxWorkers <= 0 {
		maxWorkers = DefaultMaxWorkers()
	}

	// 使用BFS遍历树并收集所有节点
	nodes := []TreeNode{root}
	var queue []TreeNode
	queue = append(queue, root)

	for len(queue) > 0 {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return make(map[string]TreeResult[T])
		default:
		}

		node := queue[0]
		queue = queue[1:]

		children := node.GetChildren()
		for _, child := range children {
			nodes = append(nodes, child)
			queue = append(queue, child)
		}
	}

	// 创建工作函数
	works := make([]WorkFunc[struct {
		ID    string
		Value T
		Err   error
	}], len(nodes))
	for i, node := range nodes {
		// 捕获循环变量
		capturedNode := node
		works[i] = func() (struct {
			ID    string
			Value T
			Err   error
		}, error) {
			// 检查上下文是否已取消
			select {
			case <-ctx.Done():
				var zero T
				return struct {
					ID    string
					Value T
					Err   error
				}{
					ID:    capturedNode.GetID(),
					Value: zero,
					Err:   ctx.Err(),
				}, nil
			default:
			}

			value, err := processFunc(capturedNode)
			return struct {
				ID    string
				Value T
				Err   error
			}{
				ID:    capturedNode.GetID(),
				Value: value,
				Err:   err,
			}, nil
		}
	}

	// 执行并收集结果
	pool := NewCoroutinePool[struct {
		ID    string
		Value T
		Err   error
	}](maxWorkers)
	results := pool.Execute(ctx, works)

	// 转换为map结果
	resultMap := make(map[string]TreeResult[T], len(results))
	for _, result := range results {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return make(map[string]TreeResult[T])
		default:
		}

		if result.Err != nil {
			// 如果执行工作函数时出错，记录错误
			var zero T
			resultMap["error"] = TreeResult[T]{
				Value:  zero,
				Err:    result.Err,
				NodeID: "unknown",
			}
			continue
		}

		// 记录节点处理结果
		resultMap[result.Value.ID] = TreeResult[T]{
			Value:  result.Value.Value,
			Err:    result.Value.Err,
			NodeID: result.Value.ID,
		}
	}

	return resultMap
}

// ProcessTreeBFS 使用BFS策略并行处理树形结构，按层处理
func ProcessTreeBFS[T any](ctx context.Context, maxWorkers int, root TreeNode, processFunc func(TreeNode) (T, error)) map[string]TreeResult[T] {
	if maxWorkers <= 0 {
		maxWorkers = DefaultMaxWorkers()
	}

	resultMap := make(map[string]TreeResult[T])
	if root == nil {
		return resultMap
	}

	// 创建协程池（只创建一次）
	pool := NewCoroutinePool[struct {
		ID       string
		Value    T
		Err      error
		Children []TreeNode
	}](maxWorkers)

	// 使用BFS按层处理
	currentLayer := []TreeNode{root}

	for len(currentLayer) > 0 {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return resultMap
		default:
		}

		// 为当前层创建工作函数
		works := make([]WorkFunc[struct {
			ID       string
			Value    T
			Err      error
			Children []TreeNode
		}], len(currentLayer))
		for i, node := range currentLayer {
			// 捕获循环变量
			capturedNode := node
			works[i] = func() (struct {
				ID       string
				Value    T
				Err      error
				Children []TreeNode
			}, error) {
				value, err := processFunc(capturedNode)
				return struct {
					ID       string
					Value    T
					Err      error
					Children []TreeNode
				}{
					ID:       capturedNode.GetID(),
					Value:    value,
					Err:      err,
					Children: capturedNode.GetChildren(),
				}, nil
			}
		}

		// 执行当前层的处理
		results := pool.Execute(ctx, works)

		// 收集结果并准备下一层
		var nextLayer []TreeNode

		for _, result := range results {
			if result.Err != nil {
				// 如果执行工作函数时出错，记录错误
				var zero T
				resultMap["error"] = TreeResult[T]{
					Value:  zero,
					Err:    result.Err,
					NodeID: "unknown",
				}
				continue
			}

			// 记录节点处理结果
			resultMap[result.Value.ID] = TreeResult[T]{
				Value:  result.Value.Value,
				Err:    result.Value.Err,
				NodeID: result.Value.ID,
			}

			// 直接添加子节点到下一层，避免二次查找
			nextLayer = append(nextLayer, result.Value.Children...)
		}

		// 更新当前层为下一层
		currentLayer = nextLayer
	}

	return resultMap
}
