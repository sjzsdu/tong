package tree

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/sjzsdu/tong/helper/coroutine"
	"github.com/sjzsdu/tong/project"
)

// NodeInfo 存储节点信息，用于并行处理
type NodeInfo struct {
	Name     string
	IsDir    bool
	IsLast   bool
	IsRoot   bool
	Prefix   string
	Children []*NodeInfo
	Size     int64
}

// Tree 生成树状结构的字符串表示，类似于 Unix tree 命令
func Tree(node *project.Node) string {
	if node == nil {
		return ""
	}

	// 并行收集节点信息
	ctx := context.Background()
	maxWorkers := 10
	rootInfo := collectNodeInfoParallel(ctx, node, "", true, true, maxWorkers)

	// 构建树状结构
	var result strings.Builder
	buildTreeFromInfo(rootInfo, &result)
	return result.String()
}

// TreeWithOptions 生成带选项的树状结构
func TreeWithOptions(node *project.Node, showFiles bool, showHidden bool, maxDepth int) string {
	if node == nil {
		return ""
	}

	// 并行收集节点信息
	ctx := context.Background()
	maxWorkers := 10
	// 将初始深度设为0（根为0层），当 maxDepth=2 时，仅包含第0层和第1层（根的直接子节点）
	rootInfo := collectNodeInfoParallelWithOptions(ctx, node, "", true, true, showFiles, showHidden, 0, maxDepth, maxWorkers)

	// 构建树状结构
	var result strings.Builder
	buildTreeFromInfo(rootInfo, &result)
	return result.String()
}

// collectNodeInfoParallel 并行收集节点信息
func collectNodeInfoParallel(ctx context.Context, node *project.Node, prefix string, isLast bool, isRoot bool, maxWorkers int) *NodeInfo {
	// 创建当前节点信息
	info := &NodeInfo{
		Name:   node.Name,
		IsDir:  node.IsDir,
		IsLast: isLast,
		IsRoot: isRoot,
		Prefix: prefix,
	}

	// 如果节点有大小信息，保存它
	if node.Info != nil {
		info.Size = node.Info.Size()
	}

	// 如果是目录且有子节点，处理子节点
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

		if isRoot {
			// 并发处理根目录的直接子节点（仅一层）
			works := make([]coroutine.WorkFunc[*NodeInfo], len(children))
			for i, child := range children {
				captured := child
				isChildLast := (i == len(children)-1)
				works[i] = func() (*NodeInfo, error) {
					return collectNodeInfoSequential(ctx, captured, newPrefix, isChildLast, false), nil
				}
			}
			pool := coroutine.NewCoroutinePool[*NodeInfo](maxWorkers)
			results := pool.Execute(ctx, works)

			info.Children = make([]*NodeInfo, 0, len(children))
			for _, r := range results {
				if r.Err == nil && r.Value != nil {
					info.Children = append(info.Children, r.Value)
				}
			}
		} else {
			// 非根目录顺序处理
			info.Children = make([]*NodeInfo, 0, len(children))
			for i, child := range children {
				isChildLast := (i == len(children)-1)
				childInfo := collectNodeInfoSequential(ctx, child, newPrefix, isChildLast, false)
				if childInfo != nil {
					info.Children = append(info.Children, childInfo)
				}
			}
		}

		// 确保子节点顺序正确
		sort.Slice(info.Children, func(i, j int) bool {
			return info.Children[i].Name < info.Children[j].Name
		})
	}

	return info
}

// collectNodeInfoSequential 顺序收集节点信息（非并行）
func collectNodeInfoSequential(ctx context.Context, node *project.Node, prefix string, isLast bool, isRoot bool) *NodeInfo {
	// 创建当前节点信息
	info := &NodeInfo{
		Name:   node.Name,
		IsDir:  node.IsDir,
		IsLast: isLast,
		IsRoot: isRoot,
		Prefix: prefix,
	}

	// 如果节点有大小信息，保存它
	if node.Info != nil {
		info.Size = node.Info.Size()
	}

	// 如果是目录且有子节点，处理子节点
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

		// 顺序处理所有子节点
		info.Children = make([]*NodeInfo, 0, len(children))
		for i, child := range children {
			isChildLast := (i == len(children)-1)
			childInfo := collectNodeInfoSequential(ctx, child, newPrefix, isChildLast, false)
			if childInfo != nil {
				info.Children = append(info.Children, childInfo)
			}
		}

		// 确保子节点顺序正确
		sort.Slice(info.Children, func(i, j int) bool {
			return info.Children[i].Name < info.Children[j].Name
		})
	}

	return info
}

// collectNodeInfoParallelWithOptions 带选项的并行收集节点信息
func collectNodeInfoParallelWithOptions(ctx context.Context, node *project.Node, prefix string, isLast bool, isRoot bool, showFiles bool, showHidden bool, currentDepth int, maxDepth int, maxWorkers int) *NodeInfo {
	// 检查深度限制
	// 注意：currentDepth 表示当前节点的深度，根节点为 0
	if maxDepth > 0 && currentDepth >= maxDepth {
		return nil
	}

	// 检查是否显示隐藏文件
	if !showHidden && strings.HasPrefix(node.Name, ".") && !isRoot {
		return nil
	}

	// 检查是否显示文件
	if !showFiles && !node.IsDir && !isRoot {
		return nil
	}

	// 创建当前节点信息
	info := &NodeInfo{
		Name:   node.Name,
		IsDir:  node.IsDir,
		IsLast: isLast,
		IsRoot: isRoot,
		Prefix: prefix,
	}

	// 如果节点有大小信息，保存它
	if node.Info != nil {
		info.Size = node.Info.Size()
	}

	// 如果是目录且有子节点，处理子节点
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

		// 对于根目录的直接子节点，使用并行处理（仅一层）
		if isRoot {
			works := make([]coroutine.WorkFunc[*NodeInfo], len(children))
			for i, child := range children {
				captured := child
				isChildLast := (i == len(children)-1)
				works[i] = func() (*NodeInfo, error) {
					return collectNodeInfoSequentialWithOptions(
						ctx,
						captured,
						newPrefix,
						isChildLast,
						false,
						showFiles,
						showHidden,
						currentDepth+1,
						maxDepth,
					), nil
				}
			}
			pool := coroutine.NewCoroutinePool[*NodeInfo](maxWorkers)
			results := pool.Execute(ctx, works)

			info.Children = make([]*NodeInfo, 0, len(children))
			for _, r := range results {
				if r.Err == nil && r.Value != nil {
					info.Children = append(info.Children, r.Value)
				}
			}
		} else {
			// 对于非根目录，使用顺序处理避免过多协程
			info.Children = make([]*NodeInfo, 0)
			for i, child := range children {
				isChildLast := (i == len(children)-1)
				childInfo := collectNodeInfoSequentialWithOptions(
					ctx,
					child,
					newPrefix,
					isChildLast,
					false,
					showFiles,
					showHidden,
					currentDepth+1,
					maxDepth,
				)
				if childInfo != nil {
					info.Children = append(info.Children, childInfo)
				}
			}
		}

		// 确保子节点顺序正确
		sort.Slice(info.Children, func(i, j int) bool {
			return info.Children[i].Name < info.Children[j].Name
		})
	}

	return info
}

// collectNodeInfoSequentialWithOptions 带选项的顺序收集节点信息（非并行）
func collectNodeInfoSequentialWithOptions(ctx context.Context, node *project.Node, prefix string, isLast bool, isRoot bool, showFiles bool, showHidden bool, currentDepth int, maxDepth int) *NodeInfo {
	// 检查深度限制
	// 注意：currentDepth 表示当前节点的深度，根节点为 0
	if maxDepth > 0 && currentDepth >= maxDepth {
		return nil
	}

	// 检查是否显示隐藏文件
	if !showHidden && strings.HasPrefix(node.Name, ".") && !isRoot {
		return nil
	}

	// 检查是否显示文件
	if !showFiles && !node.IsDir && !isRoot {
		return nil
	}

	// 创建当前节点信息
	info := &NodeInfo{
		Name:   node.Name,
		IsDir:  node.IsDir,
		IsLast: isLast,
		IsRoot: isRoot,
		Prefix: prefix,
	}

	// 如果节点有大小信息，保存它
	if node.Info != nil {
		info.Size = node.Info.Size()
	}

	// 如果是目录且有子节点，处理子节点
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

		// 顺序处理所有子节点
		info.Children = make([]*NodeInfo, 0)
		for i, child := range children {
			isChildLast := (i == len(children)-1)
			childInfo := collectNodeInfoSequentialWithOptions(
				ctx,
				child,
				newPrefix,
				isChildLast,
				false,
				showFiles,
				showHidden,
				currentDepth+1,
				maxDepth,
			)
			if childInfo != nil {
				info.Children = append(info.Children, childInfo)
			}
		}

		// 确保子节点顺序正确
		sort.Slice(info.Children, func(i, j int) bool {
			return info.Children[i].Name < info.Children[j].Name
		})
	}

	return info
}

// buildTreeFromInfo 根据节点信息构建树状结构
func buildTreeFromInfo(info *NodeInfo, result *strings.Builder) {
	if info == nil {
		return
	}

	// 构建当前节点的显示
	if !info.IsRoot {
		if info.IsLast {
			result.WriteString(info.Prefix + "└── ")
		} else {
			result.WriteString(info.Prefix + "├── ")
		}
	}

	// 添加节点名称和类型标识
	if info.IsDir {
		if info.IsRoot && info.Name == "/" {
			result.WriteString(".") // 根目录显示为 "."
		} else {
			result.WriteString(info.Name + "/")
		}
	} else {
		result.WriteString(info.Name)
		// 添加文件大小信息
		if info.Size > 0 {
			result.WriteString(fmt.Sprintf(" (%d bytes)", info.Size))
		}
	}
	result.WriteString("\n")

	// 如果是目录，处理子节点
	if info.IsDir && len(info.Children) > 0 {
		// 递归处理每个子节点
		for _, child := range info.Children {
			buildTreeFromInfo(child, result)
		}
	}
}
