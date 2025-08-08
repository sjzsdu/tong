package project

import (
	"fmt"
	"os"

	"github.com/sjzsdu/tong/project/tree"
	"github.com/spf13/cobra"
)

var (
	depth      int
	showFiles  bool
	showHidden bool
	noFiles    bool
	showStats  bool
)

var TreeCmd = &cobra.Command{
	Use:   "tree [path]",
	Short: "显示目录的树状结构",
	Long: `tree 命令以树状结构显示指定目录的内容。

支持的功能：
- 显示目录和文件的层次结构
- 控制显示深度
- 选择性显示文件或目录
- 显示隐藏文件
- 统计文件和目录数量

示例：
  tong project tree                    # 显示当前目录的树状结构
  tong project tree /path/to/dir       # 显示指定目录的树状结构
  tong project tree --depth 2          # 限制显示深度为2层
  tong project tree --no-files         # 只显示目录，不显示文件
  tong project tree --hidden           # 显示隐藏文件
  tong project tree --stats            # 显示统计信息`,
	Args: cobra.MaximumNArgs(1),
	Run:  runTree,
}

func init() {
	TreeCmd.Flags().IntVarP(&depth, "depth", "", -1, "限制显示深度 (-1 表示无限制)")
	TreeCmd.Flags().BoolVarP(&showFiles, "files", "f", true, "显示文件")
	TreeCmd.Flags().BoolVarP(&showHidden, "hidden", "a", false, "显示隐藏文件")
	TreeCmd.Flags().BoolVarP(&noFiles, "no-files", "", false, "不显示文件，只显示目录")
	TreeCmd.Flags().BoolVarP(&showStats, "stats", "s", false, "显示统计信息")
}

func runTree(cmd *cobra.Command, args []string) {
	// 确定目标路径
	targetPath := "."
	if len(args) > 0 {
		targetPath = args[0]
	}

	// 使用通用函数获取目标节点
	targetNode, err := GetTargetNode(targetPath)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	// 处理 --no-files 标志
	if noFiles {
		showFiles = false
	}

	// 使用新的 tree 包生成树状结构
	output := tree.TreeWithOptions(targetNode, showFiles, showHidden, depth)
	fmt.Print(output)

	// 显示统计信息
	if showStats {
		stats := tree.Stats(targetNode)
		fmt.Printf("\n%s\n", stats.String())
	}
}
