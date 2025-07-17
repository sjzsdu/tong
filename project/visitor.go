package project

// NodeVisitor 定义了节点访问器的接口
type NodeVisitor interface {
	// VisitDirectory 访问目录节点
	VisitDirectory(node *Node, path string, depth int) error
	// VisitFile 访问文件节点
	VisitFile(node *Node, path string, depth int) error
}

type FilteredVisitor struct {
	Visitor    NodeVisitor                        // 实际的访问器
	FileFilter func(node *Node, path string) bool // 文件过滤函数
	DirFilter  func(node *Node, path string) bool // 目录过滤函数
}

// VisitDirectory 实现 NodeVisitor 接口
func (fv *FilteredVisitor) VisitDirectory(node *Node, path string, depth int) error {
	if fv.DirFilter != nil && !fv.DirFilter(node, path) {
		return nil // 跳过此目录
	}
	return fv.Visitor.VisitDirectory(node, path, depth)
}

// VisitFile 实现 NodeVisitor 接口
func (fv *FilteredVisitor) VisitFile(node *Node, path string, depth int) error {
	if fv.FileFilter != nil && !fv.FileFilter(node, path) {
		return nil // 跳过此文件
	}
	return fv.Visitor.VisitFile(node, path, depth)
}
