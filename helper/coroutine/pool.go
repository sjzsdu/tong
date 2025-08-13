package coroutine

import (
	"context"
	"runtime"
	"sync"
)

// CoroutinePool 协程池，用于控制并发执行的协程数量
type CoroutinePool[T any] struct {
	maxWorkers int
	results    []Result[T]
	mutex      sync.Mutex
}

// DefaultMaxWorkers 返回基于CPU核心数的默认最大协程数
func DefaultMaxWorkers() int {
	numCPU := runtime.NumCPU()
	// 使用CPU核心数的2倍作为默认值，这是一个常见的经验值
	// 因为IO密集型任务可以有效利用超过CPU核心数的协程
	return numCPU * 2
}

// NewCoroutinePool 创建一个新的协程池
func NewCoroutinePool[T any](maxWorkers int) *CoroutinePool[T] {
	if maxWorkers <= 0 {
		maxWorkers = DefaultMaxWorkers() // 使用基于CPU核心数的默认值
	}

	return &CoroutinePool[T]{
		maxWorkers: maxWorkers,
		results:    make([]Result[T], 0),
	}
}

// Execute 执行一组工作函数，控制并发数量，并等待所有协程完成
func (p *CoroutinePool[T]) Execute(ctx context.Context, works []WorkFunc[T]) []Result[T] {
	if len(works) == 0 {
		return []Result[T]{}
	}

	// 重置结果集
	p.mutex.Lock()
	p.results = make([]Result[T], len(works))
	p.mutex.Unlock()

	// 创建工作通道和等待组
	workChan := make(chan int, len(works))
	var wg sync.WaitGroup

	// 启动工作协程
	workerCount := p.maxWorkers
	if workerCount > len(works) {
		workerCount = len(works)
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go p.worker(ctx, &wg, workChan, works)
	}

	// 启动一个协程来监控上下文取消
	// 创建完成通道
	doneChan := make(chan struct{})

	// 启动一个协程来监控上下文取消和工作完成
	go func() {
		select {
		case <-ctx.Done():
			// 上下文被取消
			close(workChan)
		case <-doneChan:
			// 所有工作已发送完成
			close(workChan)
		}
	}()

	// 发送工作索引到通道
	for i := 0; i < len(works); i++ {
		select {
		case workChan <- i:
			// 成功发送工作
		case <-ctx.Done():
			// 上下文已取消，退出循环
			break
		}
	}

	// 通知所有工作已发送完成
	close(doneChan)

	// 等待所有工作完成
	wg.Wait()

	return p.results
}

// worker 工作协程，从通道获取工作并执行
func (p *CoroutinePool[T]) worker(ctx context.Context, wg *sync.WaitGroup, workChan <-chan int, works []WorkFunc[T]) {
	defer wg.Done()

	for {
		select {
		case index, ok := <-workChan:
			if !ok {
				// 通道已关闭，没有更多工作
				return
			}

			// 执行工作函数
			value, err := works[index]()

			// 保存结果
			p.mutex.Lock()
			p.results[index] = Result[T]{
				Value: value,
				Err:   err,
				Index: index,
			}
			p.mutex.Unlock()

		case <-ctx.Done():
			// 上下文被取消
			return
		}
	}
}