package tree

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sjzsdu/tong/project"
)

// Tree 生成树状结构的字符串表示，类似于 Unix tree 命令
func Tree(node *project.Node) string {
	if node == nil {
		return ""
	}
	
	var result strings.Builder
	buildTree(node, &result, "", true, true)
	return result.String()
}

// TreeWithOptions 生成带选项的树状结构
func TreeWithOptions(node *project.Node, showFiles bool, showHidden bool, maxDepth int) string {
	if node == nil {
		return ""
	}
	
	var result strings.Builder
	buildTreeWithOptions(node, &result, "", true, true, showFiles, showHidden, 0, maxDepth)
	return result.String()
}

// buildTree 递归构建树状结构
func buildTree(node *project.Node, result *strings.Builder, prefix string, isLast bool, isRoot bool) {
	// 构建当前节点的显示
	if !isRoot {
		if isLast {
			result.WriteString(prefix + "└── ")
		} else {
			result.WriteString(prefix + "├── ")
		}
	}
	
	// 添加节点名称和类型标识
	if node.IsDir {
		if isRoot && node.Name == "/" {
			result.WriteString(".")  // 根目录显示为 "."
		} else {
			result.WriteString(node.Name + "/")
		}
	} else {
		result.WriteString(node.Name)
		// 可以添加文件大小信息
		if node.Info != nil {
			result.WriteString(fmt.Sprintf(" (%d bytes)", node.Info.Size()))
		}
	}
	result.WriteString("\n")
	
	// 如果是目录，递归处理子节点
	if node.IsDir && len(node.Children) > 0 {
		// 将子节点按名称排序
		children := make([]*project.Node, 0, len(node.Children))
		for _, child := range node.Children {
			children = append(children, child)
		}
		sort.Slice(children, func(i, j int) bool {
			// 目录优先，然后按名称排序
			if children[i].IsDir != children[j].IsDir {
				return children[i].IsDir
			}
			return children[i].Name < children[j].Name
		})
		
		// 构建新的前缀
		var newPrefix string
		if isRoot {
			newPrefix = ""
		} else if isLast {
			newPrefix = prefix + "    "
		} else {
			newPrefix = prefix + "│   "
		}
		
		// 递归处理每个子节点
		for i, child := range children {
			isChildLast := (i == len(children)-1)
			buildTree(child, result, newPrefix, isChildLast, false)
		}
	}
}

// buildTreeWithOptions 递归构建带选项的树状结构
func buildTreeWithOptions(node *project.Node, result *strings.Builder, prefix string, isLast bool, isRoot bool, 
	showFiles bool, showHidden bool, currentDepth int, maxDepth int) {
	
	// 检查深度限制
	if maxDepth > 0 && currentDepth >= maxDepth {
		return
	}
	
	// 检查是否显示隐藏文件
	if !showHidden && strings.HasPrefix(node.Name, ".") && !isRoot {
		return
	}
	
	// 检查是否显示文件
	if !showFiles && !node.IsDir && !isRoot {
		return
	}
	
	// 构建当前节点的显示
	if !isRoot {
		if isLast {
			result.WriteString(prefix + "└── ")
		} else {
			result.WriteString(prefix + "├── ")
		}
	}
	
	// 添加节点名称和详细信息
	if node.IsDir {
		if isRoot && node.Name == "/" {
			result.WriteString(".")  // 根目录显示为 "."
		} else {
			result.WriteString(node.Name + "/")
		}
		// 添加子项数量
		if len(node.Children) > 0 {
			visibleChildren := 0
			for _, child := range node.Children {
				if showHidden || !strings.HasPrefix(child.Name, ".") {
					if showFiles || child.IsDir {
						visibleChildren++
					}
				}
			}
			result.WriteString(fmt.Sprintf(" [%d items]", visibleChildren))
		}
	} else {
		result.WriteString(node.Name)
		// 添加文件详细信息
		if node.Info != nil {
			size := node.Info.Size()
			var sizeStr string
			if size < 1024 {
				sizeStr = fmt.Sprintf("%d B", size)
			} else if size < 1024*1024 {
				sizeStr = fmt.Sprintf("%.1f KB", float64(size)/1024)
			} else if size < 1024*1024*1024 {
				sizeStr = fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
			} else {
				sizeStr = fmt.Sprintf("%.1f GB", float64(size)/(1024*1024*1024))
			}
			result.WriteString(fmt.Sprintf(" (%s)", sizeStr))
		}
	}
	result.WriteString("\n")
	
	// 如果是目录，递归处理子节点
	if node.IsDir && len(node.Children) > 0 {
		// 过滤和排序子节点
		children := make([]*project.Node, 0, len(node.Children))
		for _, child := range node.Children {
			// 应用过滤条件
			if !showHidden && strings.HasPrefix(child.Name, ".") {
				continue
			}
			if !showFiles && !child.IsDir {
				continue
			}
			children = append(children, child)
		}
		
		// 排序：目录优先，然后按名称排序
		sort.Slice(children, func(i, j int) bool {
			if children[i].IsDir != children[j].IsDir {
				return children[i].IsDir
			}
			return children[i].Name < children[j].Name
		})
		
		// 构建新的前缀
		var newPrefix string
		if isRoot {
			newPrefix = ""
		} else if isLast {
			newPrefix = prefix + "    "
		} else {
			newPrefix = prefix + "│   "
		}
		
		// 递归处理每个子节点
		for i, child := range children {
			isChildLast := (i == len(children)-1)
			buildTreeWithOptions(child, result, newPrefix, isChildLast, false, 
				showFiles, showHidden, currentDepth+1, maxDepth)
		}
	}
}