// traverseError 封装遍历过程中的错误信息
package project

import "fmt"

type traverseError struct {
	Path     string
	NodeName string
	Err      error
}

func (e *traverseError) Error() string {
	return fmt.Sprintf("遍历错误 [%s] 在节点 '%s': %v", e.Path, e.NodeName, e.Err)
}
