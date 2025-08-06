package coroutine

// WorkFunc 定义了协程中执行的工作函数类型
type WorkFunc[T any] func() (T, error)

// Result 定义了协程执行的结果
type Result[T any] struct {
	Value T
	Err   error
	Index int
}

// TreeNode 定义了树形结构的节点接口
type TreeNode interface {
	// Children 返回当前节点的所有子节点
	Children() []TreeNode
	// ID 返回节点的唯一标识符，用于在结果中标识节点
	ID() string
}

// TreeResult 定义了树形结构处理的结果
type TreeResult[T any] struct {
	Value  T
	Err    error
	NodeID string
}