package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	projsearch "github.com/sjzsdu/tong/project/search"
	"github.com/spf13/cobra"
)

var (
	searchNameContains    string
	searchNameRegex       string
	searchContentContains string
	searchContentRegex    string
	searchExtensions      []string
	searchIncludeHidden   bool
	searchIncludeDirs     bool
	searchIncludeFiles    bool
	searchDepth           int
	searchIgnoreCase      bool
	searchSubdir          string
)

var SearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "在指定目录节点下并发搜索",
	Long: `search 子命令对给定查询词进行并发搜索，默认同时按名称与内容进行匹配（任一匹配即命中）。也可通过参数指定仅按名称或内容、或使用正则。

示例：
  tong project search README                        # 默认：名称或内容包含 "README"
  tong project search '.*_test\\.go$' --name-regex   # 按名称正则
  tong project search TODO --content                # 仅按内容子串
  tong project search '(?i)license' --content-regex # 按内容正则（忽略大小写）
  tong project search README --ext go,md            # 仅搜索 go、md 文件
  tong project search README --hidden               # 包含隐藏文件/目录
  tong project search src --dirs --subdir src       # 仅返回目录，并限定在 src 子目录
  tong project search util --files=false            # 不返回文件
  tong project search read --depth 2                # 限制相对根的深度为 2（根为 0）`,
	Args: cobra.ExactArgs(1),
	Run:  runSearch,
}

func init() {
	SearchCmd.Flags().StringVar(&searchNameContains, "name", "", "名称包含（子串匹配）")
	SearchCmd.Flags().StringVar(&searchNameRegex, "name-regex", "", "名称正则（优先于 name）")
	SearchCmd.Flags().StringVar(&searchContentContains, "content", "", "内容包含（子串匹配）")
	SearchCmd.Flags().StringVar(&searchContentRegex, "content-regex", "", "内容正则（优先于 content）")
	// 不使用短选项，避免与上层命令冲突
	SearchCmd.Flags().StringSliceVar(&searchExtensions, "ext", []string{}, "文件扩展名过滤，例如: go,md；为空表示不过滤")
	SearchCmd.Flags().BoolVar(&searchIncludeHidden, "hidden", false, "包含隐藏文件/目录")
	SearchCmd.Flags().BoolVar(&searchIncludeDirs, "dirs", false, "结果中包含目录")
	SearchCmd.Flags().BoolVar(&searchIncludeFiles, "files", true, "结果中包含文件")
	SearchCmd.Flags().IntVar(&searchDepth, "depth", 0, "限制搜索深度（根为0，0表示不限制）")
	SearchCmd.Flags().BoolVar(&searchIgnoreCase, "ignore-case", true, "大小写不敏感匹配（对子串与正则均生效）")
	SearchCmd.Flags().StringVar(&searchSubdir, "subdir", ".", "限定搜索的子目录（相对项目根）")
}

func runSearch(cmd *cobra.Command, args []string) {
	// 查询词
	query := args[0]

	// 获取共享项目实例
	if sharedProject == nil {
		fmt.Printf("错误: 未找到共享的项目实例\n")
		os.Exit(1)
	}

	// 基于项目根路径来处理子目录
	var finalTargetPath string
	if filepath.IsAbs(searchSubdir) {
		finalTargetPath = searchSubdir
	} else {
		finalTargetPath = filepath.Join(sharedProject.GetRootPath(), searchSubdir)
	}

	// 使用通用函数获取目标节点
	targetNode, err := GetTargetNode(finalTargetPath)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	// 构建搜索选项
	opts := projsearch.DefaultSearchOptions()
	opts.NameContains = searchNameContains
	opts.NameRegex = searchNameRegex
	opts.ContentContains = searchContentContains
	opts.ContentRegex = searchContentRegex
	// 若未显式指定 name/content 相关选项，则默认同时按名称与内容匹配（OR 逻辑）
	if opts.NameContains == "" && opts.NameRegex == "" && opts.ContentContains == "" && opts.ContentRegex == "" {
		opts.NameContains = query
		opts.ContentContains = query
		opts.MatchAny = true
	}
	opts.Extensions = normalizeExts(searchExtensions)
	opts.IncludeHidden = searchIncludeHidden
	opts.IncludeDirs = searchIncludeDirs
	opts.IncludeFiles = searchIncludeFiles
	if searchDepth < 0 {
		opts.MaxDepth = 0 // 不限制
	} else {
		opts.MaxDepth = searchDepth
	}
	// 不设置并发度，内部默认按 CPU 核心数计算
	opts.CaseInsensitive = searchIgnoreCase

	// 执行搜索
	ctx := context.Background()
	matched, err := projsearch.Search(ctx, targetNode, opts)
	if err != nil {
		fmt.Printf("搜索出错: %v\n", err)
		os.Exit(1)
	}

	// 输出结果（使用项目相对路径）
	if len(matched) == 0 {
		fmt.Println("未找到匹配项")
		return
	}

	for _, n := range matched {
		typeStr := "file"
		if n.IsDir {
			typeStr = "dir"
		}
		fmt.Printf("[%s] %s\n", typeStr, n.Path)
	}
}

func normalizeExts(exts []string) []string {
	if len(exts) == 0 {
		return exts
	}
	out := make([]string, 0, len(exts))
	for _, e := range exts {
		if e == "" {
			continue
		}
		// 支持逗号拼接输入（兼容用户传法）
		parts := strings.Split(e, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			p = strings.TrimPrefix(p, ".")
			if p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}
