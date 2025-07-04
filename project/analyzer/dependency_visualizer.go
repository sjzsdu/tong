package analyzer

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/sjzsdu/tong/helper"
)

// DependencyVisualizer 依赖关系可视化器
type DependencyVisualizer struct {
	graph *DependencyGraph
}

// NewDependencyVisualizer 创建一个新的依赖关系可视化器
func NewDependencyVisualizer(graph *DependencyGraph) *DependencyVisualizer {
	return &DependencyVisualizer{
		graph: graph,
	}
}

// PrintDependencies 打印依赖分析结果
func (dv *DependencyVisualizer) PrintDependencies() {
	if dv.graph == nil {
		fmt.Println("依赖图为空")
		return
	}

	// 输出分析结果
	helper.PrintColorText("依赖分析结果:", helper.ColorBlueBold)
	fmt.Printf("总依赖数: %d\n", len(dv.graph.Nodes))

	// 按类型分组依赖
	typeGroups := make(map[string][]*DependencyNode)
	for name, node := range dv.graph.Nodes {
		node.Name = name // 确保名称正确设置
		typeGroups[node.Type] = append(typeGroups[node.Type], node)
	}

	// 排序依赖类型
	var types []string
	for t := range typeGroups {
		types = append(types, t)
	}
	sort.Strings(types)

	// 按类型分组输出
	helper.PrintColorText("\n依赖列表 (按类型分组):", helper.ColorGreenBold)
	for _, t := range types {
		nodes := typeGroups[t]

		// 排序依赖名称
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].Name < nodes[j].Name
		})

		// 输出类型标题
		fmt.Printf("\n%s (%d):\n", helper.ColorText(strings.ToUpper(t), helper.ColorYellowBold), len(nodes))

		// 输出依赖
		for _, node := range nodes {
			if node.Version != "" {
				fmt.Printf("  - %s: %s\n",
					helper.ColorText(node.Name, helper.ColorCyan),
					helper.ColorText(node.Version, helper.ColorPurple))
			} else {
				fmt.Printf("  - %s\n", helper.ColorText(node.Name, helper.ColorCyan))
			}
		}
	}

	// 构建依赖树
	depTrees := dv.buildDependencyTrees()

	// 输出依赖树
	helper.PrintColorText("\n依赖关系树:", helper.ColorGreenBold)
	for root, children := range depTrees {
		dv.printDependencyTree(root, children, "", true)
	}

	// 提示可以使用DOT文件输出
	fmt.Println("\n提示: 使用 -o output.dot 参数可以生成可视化依赖图")
}

// GenerateDotFile 生成依赖关系DOT文件 (用于Graphviz可视化)
func (dv *DependencyVisualizer) GenerateDotFile(outputFile string) error {
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("创建DOT文件失败: %v", err)
	}
	defer file.Close()

	// 写入DOT文件头
	file.WriteString("digraph DependencyGraph {\n")
	file.WriteString("  rankdir=LR;\n")
	file.WriteString("  node [shape=box, style=filled, fillcolor=lightblue];\n\n")

	// 定义节点
	for name, node := range dv.graph.Nodes {
		label := name
		if node.Version != "" {
			label += "\\n" + node.Version
		}
		file.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\", fillcolor=%s];\n",
			name, label, dv.getColorForType(node.Type)))
	}

	file.WriteString("\n")

	// 定义边
	for src, dsts := range dv.graph.Edges {
		for _, dst := range dsts {
			file.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\";\n", src, dst))
		}
	}

	// 写入DOT文件尾
	file.WriteString("}\n")

	fmt.Printf("依赖图已生成: %s\n", outputFile)
	fmt.Println("可以使用以下命令生成图片:")
	fmt.Printf("  dot -Tpng %s -o dependency.png\n", outputFile)
	fmt.Printf("  或: dot -Tsvg %s -o dependency.svg\n", outputFile)

	return nil
}

// 根据依赖类型获取颜色
func (dv *DependencyVisualizer) getColorForType(depType string) string {
	switch depType {
	case "direct":
		return "\"#c2e0c6\"" // 浅绿色
	case "dev":
		return "\"#ffdfb9\"" // 浅橙色
	case "import":
		return "\"#d1eefd\"" // 浅蓝色
	default:
		return "\"#e0e0e0\"" // 浅灰色
	}
}

// 构建依赖树
func (dv *DependencyVisualizer) buildDependencyTrees() map[string]map[string]bool {
	// 创建依赖树的根节点
	roots := make(map[string]bool)
	for src := range dv.graph.Edges {
		roots[src] = true
	}

	// 移除作为目标的节点（它们不是根节点）
	for _, dsts := range dv.graph.Edges {
		for _, dst := range dsts {
			delete(roots, dst)
		}
	}

	// 构建树结构
	trees := make(map[string]map[string]bool)
	for root := range roots {
		trees[root] = make(map[string]bool)
		dv.buildDependencyTreeRecursive(root, trees[root], make(map[string]bool))
	}

	// 处理孤立的节点
	for name := range dv.graph.Nodes {
		_, hasOutgoing := dv.graph.Edges[name]
		isTarget := false

		for _, dsts := range dv.graph.Edges {
			for _, dst := range dsts {
				if dst == name {
					isTarget = true
					break
				}
			}
			if isTarget {
				break
			}
		}

		if !hasOutgoing && !isTarget {
			trees[name] = make(map[string]bool)
		}
	}

	return trees
}

// 递归构建依赖树
func (dv *DependencyVisualizer) buildDependencyTreeRecursive(node string, visited map[string]bool, path map[string]bool) {
	// 检查是否已访问或形成循环
	if visited[node] || path[node] {
		return
	}

	// 标记为已访问
	path[node] = true

	// 处理子节点
	for _, child := range dv.graph.Edges[node] {
		visited[child] = true
		dv.buildDependencyTreeRecursive(child, visited, path)
	}

	// 移除路径标记
	delete(path, node)
}

// 打印依赖树
func (dv *DependencyVisualizer) printDependencyTree(node string, visited map[string]bool, prefix string, isLast bool) {
	// 构建当前行的前缀
	fmt.Print(prefix)

	if isLast {
		fmt.Print(helper.ColorText("└── ", helper.ColorGray))
		prefix += "    "
	} else {
		fmt.Print(helper.ColorText("├── ", helper.ColorGray))
		prefix += helper.ColorText("│   ", helper.ColorGray)
	}

	// 打印节点
	nodeColor := helper.ColorCyan
	if n, ok := dv.graph.Nodes[node]; ok && n.Type == "direct" {
		nodeColor = helper.ColorGreen
	} else if n, ok := dv.graph.Nodes[node]; ok && n.Type == "dev" {
		nodeColor = helper.ColorYellow
	}

	fmt.Println(helper.ColorText(node, nodeColor))

	// 获取并排序子节点
	var children []string
	for child := range visited {
		children = append(children, child)
	}
	sort.Strings(children)

	// 递归打印子节点
	for i, child := range children {
		isLastChild := i == len(children)-1
		dv.printDependencyTree(child, make(map[string]bool), prefix, isLastChild)
	}
}
