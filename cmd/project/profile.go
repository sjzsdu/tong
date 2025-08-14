package project

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sjzsdu/tong/lang"
	proj "github.com/sjzsdu/tong/project"
	projtree "github.com/sjzsdu/tong/project/tree"
	"github.com/spf13/cobra"
)

var ProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: lang.T("项目画像与规模分级"),
	Long:  lang.T("扫描当前项目，输出文件/目录统计、扩展名分布与规模分级"),
	Run:   runProfile,
}

func init() {}

func runProfile(cmd *cobra.Command, args []string) {
	if sharedProject == nil {
		fmt.Println("错误: 未找到共享的项目实例")
		os.Exit(1)
	}

	rootNode, err := GetTargetNode(sharedProject.GetRootPath())
	if err != nil || rootNode == nil {
		fmt.Printf("获取根节点失败: %v\n", err)
		os.Exit(1)
	}

	stats := projtree.Stats(rootNode)

	extDist := make(map[string]int)
	totalFiles := 0
	collectExtDist(rootNode, extDist, &totalFiles)

	tier := classifySize(stats.FileCount)

	fmt.Printf("Project: %s\n", sharedProject.GetName())
	fmt.Printf("Root: %s\n\n", sharedProject.GetRootPath())

	fmt.Println("Overview:")
	fmt.Printf("- Directories: %d\n", stats.DirectoryCount)
	fmt.Printf("- Files: %d\n", stats.FileCount)
	fmt.Printf("- Summary: %s\n", stats.String())
	fmt.Printf("- Tier: %s\n\n", tier)

	fmt.Println("Top extensions:")
	for _, kv := range topNExt(extDist, 10) {
		fmt.Printf("- %s: %d\n", kv.key, kv.value)
	}

	fmt.Println()
	fmt.Println("Ecosystem:")
	ec := detectEcosystem(rootNode)
	for _, line := range ec {
		fmt.Printf("- %s\n", line)
	}
}

type kv struct {
	key   string
	value int
}

func topNExt(m map[string]int, n int) []kv {
	arr := make([]kv, 0, len(m))
	for k, v := range m {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].value > arr[j].value })
	if len(arr) > n {
		return arr[:n]
	}
	return arr
}

func classifySize(files int) string {
	if files < 500 {
		return "Small"
	}
	if files < 5000 {
		return "Medium"
	}
	return "Large"
}

func collectExtDist(n *proj.Node, dist map[string]int, total *int) {
	if n == nil {
		return
	}
	if !n.IsDir {
		ext := strings.ToLower(filepath.Ext(n.Name))
		if ext == "" {
			ext = "(noext)"
		}
		dist[ext]++
		*total++
		return
	}
	for _, c := range n.Children {
		collectExtDist(c, dist, total)
	}
}

func detectEcosystem(n *proj.Node) []string {
	var out []string
	seen := map[string]bool{}
	// simple stack-based walk
	var stack []*proj.Node
	stack = append(stack, n)
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if !cur.IsDir {
			name := cur.Name
			switch name {
			case "go.mod":
				seen["Go (modules)"] = true
			case "package.json":
				seen["Node.js (npm)"] = true
			case "requirements.txt", "pyproject.toml":
				seen["Python"] = true
			case "pom.xml":
				seen["Java (Maven)"] = true
			case "build.gradle", "settings.gradle", "build.gradle.kts":
				seen["Java/Kotlin (Gradle)"] = true
			}
			continue
		}
		for _, c := range cur.Children {
			stack = append(stack, c)
		}
	}
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	if len(out) == 0 {
		out = append(out, "Unknown")
	}
	return out
}
