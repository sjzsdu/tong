package project

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
