package coroutine

import (
	"context"
)

// ExecuteWithoutResult 执行一组不需要返回结果的工作函数
func ExecuteWithoutResult(ctx context.Context, maxWorkers int, works []func() error) []error {
	if len(works) == 0 {
		return []error{}
	}

	if maxWorkers <= 0 {
		maxWorkers = DefaultMaxWorkers()
	}

	// 将无返回值的函数转换为有返回值的函数
	typedWorks := make([]WorkFunc[struct{}], len(works))
	for i, work := range works {
		typedWorks[i] = func() (struct{}, error) {
			err := work()
			return struct{}{}, err
		}
	}

	// 创建协程池并执行
	pool := NewCoroutinePool[struct{}](maxWorkers)
	results := pool.Execute(ctx, typedWorks)

	// 提取错误信息
	errors := make([]error, len(results))
	for i, result := range results {
		errors[i] = result.Err
	}

	return errors
}

// Map 并行执行map操作，将输入切片中的每个元素应用函数并返回结果
func Map[T, R any](ctx context.Context, maxWorkers int, items []T, mapFunc func(T) (R, error)) []Result[R] {
	if maxWorkers <= 0 {
		maxWorkers = DefaultMaxWorkers()
	}

	works := make([]WorkFunc[R], len(items))
	for i, item := range items {
		// 捕获循环变量
		capturedItem := item
		works[i] = func() (R, error) {
			return mapFunc(capturedItem)
		}
	}

	pool := NewCoroutinePool[R](maxWorkers)
	return pool.Execute(ctx, works)
}

// Each 并行执行forEach操作，对输入切片中的每个元素应用函数
func Each[T any](ctx context.Context, maxWorkers int, items []T, eachFunc func(T) error) []error {
	if maxWorkers <= 0 {
		maxWorkers = DefaultMaxWorkers()
	}

	works := make([]func() error, len(items))
	for i, item := range items {
		// 捕获循环变量
		capturedItem := item
		works[i] = func() error {
			return eachFunc(capturedItem)
		}
	}

	return ExecuteWithoutResult(ctx, maxWorkers, works)
}