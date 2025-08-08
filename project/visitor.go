package project

// NodeVisitor 定义了节点访问器的接口
type NodeVisitor interface {
	// VisitNode 访问任意节点（文件或目录）
	VisitNode(node *Node, path string, depth int) error
}

type FilteredVisitor struct {
	Visitor    NodeVisitor                        // 实际的访问器
	NodeFilter func(node *Node, path string) bool // 节点过滤函数
}

// VisitNode 实现 NodeVisitor 接口
func (fv *FilteredVisitor) VisitNode(node *Node, path string, depth int) error {
	if fv.NodeFilter != nil && !fv.NodeFilter(node, path) {
		return nil // 跳过此节点
	}
	return fv.Visitor.VisitNode(node, path, depth)
}
