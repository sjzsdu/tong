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

// MapDict 并行执行字典的map操作，将输入字典中的每个键值对应用函数并返回结果
func MapDict[K comparable, V, R any](ctx context.Context, maxWorkers int, dict map[K]V, mapFunc func(K, V) (R, error)) map[K]Result[R] {
	if maxWorkers <= 0 {
		maxWorkers = DefaultMaxWorkers()
	}

	// 创建工作函数切片
	works := make([]WorkFunc[R], 0, len(dict))
	// 保存键的映射，用于后续关联结果
	keyMap := make(map[int]K)
	
	i := 0
	for k, v := range dict {
		// 捕获循环变量
		capturedKey := k
		capturedValue := v
		works = append(works, func() (R, error) {
			return mapFunc(capturedKey, capturedValue)
		})
		keyMap[i] = capturedKey
		i++
	}

	// 创建协程池并执行
	pool := NewCoroutinePool[R](maxWorkers)
	results := pool.Execute(ctx, works)

	// 将结果与原始键关联
	resultMap := make(map[K]Result[R], len(results))
	for _, result := range results {
		key := keyMap[result.Index]
		resultMap[key] = result
	}

	return resultMap
}

// EachDict 并行执行字典的forEach操作，对输入字典中的每个键值对应用函数
func EachDict[K comparable, V any](ctx context.Context, maxWorkers int, dict map[K]V, eachFunc func(K, V) error) map[K]error {
	if maxWorkers <= 0 {
		maxWorkers = DefaultMaxWorkers()
	}

	// 使用MapDict实现，但忽略返回值
	results := MapDict(ctx, maxWorkers, dict, func(k K, v V) (struct{}, error) {
		err := eachFunc(k, v)
		return struct{}{}, err
	})

	// 提取错误信息
	errorMap := make(map[K]error, len(results))
	for k, result := range results {
		errorMap[k] = result.Err
	}

	return errorMap
}