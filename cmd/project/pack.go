package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sjzsdu/tong/project/pack"
	"github.com/spf13/cobra"
)

var (
	outputFile    string
	includeHidden bool
	excludeExts   []string
	showProgress  bool
)

var PackCmd = &cobra.Command{
	Use:   "pack [path]",
	Short: "打包项目文件",
	Long: `pack 命令用于将项目文件打包成指定格式的输出文件。

支持的功能：
- 打包指定目录或文件
- 从输出文件扩展名自动推断格式
- 包含或排除隐藏文件
- 排除指定扩展名的文件
- 显示打包进度

示例：
  tong project pack                    # 打包当前目录到./packed.md
  tong project pack /path/to/dir       # 打包指定目录到./packed.md
  tong project pack --file ./output.md # 指定输出文件路径(Markdown格式)
  tong project pack --file ./output.txt # 指定输出文件路径(文本格式)
  tong project pack --hidden           # 包含隐藏文件
  tong project pack --exclude-exts .js,.css  # 排除指定扩展名的文件
  tong project pack --progress         # 显示打包进度`,
	Args: cobra.MaximumNArgs(1),
	Run:  runPack,
}

func init() {
	PackCmd.Flags().StringVarP(&outputFile, "file", "f", "./packed.md", "指定输出文件路径 (从扩展名推断格式: .md 为markdown, .txt 为text)")
	PackCmd.Flags().BoolVarP(&includeHidden, "hidden", "a", false, "包含隐藏文件")
	PackCmd.Flags().StringSliceVarP(&excludeExts, "exclude-exts", "m", []string{}, "排除的文件扩展名，用逗号分隔")
	PackCmd.Flags().BoolVarP(&showProgress, "progress", "p", false, "显示打包进度")
}

func runPack(cmd *cobra.Command, args []string) {
	// 确定目标路径
	targetPath := "."
	if len(args) > 0 {
		targetPath = args[0]
	}

	// 获取共享项目实例
	if sharedProject == nil {
		fmt.Printf("错误: 未找到共享的项目实例\n")
		os.Exit(1)
	}

	// 基于项目根路径来处理目标路径
	var finalTargetPath string
	if filepath.IsAbs(targetPath) {
		finalTargetPath = targetPath
	} else {
		finalTargetPath = filepath.Join(sharedProject.GetRootPath(), targetPath)
	}

	// 使用通用函数获取目标节点
	targetNode, err := GetTargetNode(finalTargetPath)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	// 从输出文件路径提取格式
	format := "markdown" // 默认格式
	ext := strings.ToLower(filepath.Ext(outputFile))
	if ext == ".txt" {
		format = "text"
	} else if ext == ".md" {
		format = "markdown"
	}

	// 准备打包选项
	options := pack.DefaultOptions()
	options.ExcludeExts = excludeExts
	options.IncludeHidden = includeHidden

	// 获取格式化器
	formatter := pack.GetFormatter(format)
	if formatter == nil {
		fmt.Printf("错误: 不支持的格式 '%s'\n", format)
		os.Exit(1)
	}
	options.Formatter = formatter

	// 确保输出目录存在
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("创建输出目录失败: %v\n", err)
		os.Exit(1)
	}

	// 执行打包
	err = pack.PackNode(targetNode, outputFile, options)
	if err != nil {
		fmt.Printf("打包失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("打包成功! 文件已保存到: %s\n", outputFile)
}
