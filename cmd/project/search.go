package project

import (
	"context"
	"fmt"
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
	searchAny             bool
	searchSubdir          string
)

var SearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "在指定目录节点下并发搜索",
	Long: `search 子命令在指定子树中并发搜索。

查询来源有两种：
1) 位置参数 query （例如: tong project search README）
2) 显式条件 --name / --name-regex / --content / --content-regex

优先级与默认规则：
• 若提供显式条件（任意 name/content 相关 flag），位置参数仅作为补充忽略，不再自动注入。
• 若未提供显式条件：
    - 有位置参数 => 作为名称子串条件 (NameContains)
    - 无位置参数 => 报错（必须至少给一个条件）
• 同时指定名称与内容条件默认 AND，可用 --any 改为 OR。
• 默认区分大小写；使用 --ignore-case 开启不敏感匹配。

示例：
  tong project search README                       # 名称包含 README（默认名称匹配）
  tong project search --name README                # 同上（无位置参数，用 flag）
  tong project search --content TODO               # 内容包含 TODO
  tong project search '.*_test\\.go$' --name-regex  # 名称正则
  tong project search --name util --content util   # 名称 AND 内容同时匹配
  tong project search --name util --content util --any # 名称 OR 内容
  tong project search README --ext go,md           # 扩展过滤（逗号或多次传参）
  tong project search --name README --ignore-case  # 名称忽略大小写
  tong project search --name README --depth 2      # 深度限制（根为 0）
  tong project search --content license --any      # 与其他条件 OR`,
	Args: cobra.ArbitraryArgs,
	RunE: runSearch,
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
	SearchCmd.Flags().BoolVar(&searchIgnoreCase, "ignore-case", false, "大小写不敏感匹配（对子串与正则均生效）")
	SearchCmd.Flags().BoolVar(&searchAny, "any", false, "名称与内容条件采用 OR 逻辑（默认 AND）")
	SearchCmd.Flags().StringVar(&searchSubdir, "subdir", ".", "限定搜索的子目录（相对项目根）")
}

func runSearch(cmd *cobra.Command, args []string) error {
	// 可选位置参数
	var query string
	if len(args) > 0 {
		query = args[0]
	}

	// 获取共享项目实例
	if sharedProject == nil {
		fmt.Printf("错误: 未找到共享的项目实例\n")
		return fmt.Errorf("no shared project")
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
		return err
	}

	// 构建搜索选项
	opts := projsearch.DefaultSearchOptions()
	opts.NameContains = searchNameContains
	opts.NameRegex = searchNameRegex
	opts.ContentContains = searchContentContains
	opts.ContentRegex = searchContentRegex
	// 判断用户是否提供显式条件
	hasExplicit := opts.NameContains != "" || opts.NameRegex != "" || opts.ContentContains != "" || opts.ContentRegex != ""
	if !hasExplicit {
		if query == "" {
			fmt.Println("错误: 缺少查询条件。请提供位置参数或使用 --name/--content 其中之一。")
			return fmt.Errorf("missing query")
		}
		// 只有在无显式条件时才注入位置参数
		opts.NameContains = query
	}
	// 用户显式选择 OR
	if searchAny {
		opts.MatchAny = true
	}
	opts.Extensions = normalizeExts(searchExtensions)
	opts.IncludeHidden = searchIncludeHidden
	opts.IncludeDirs = searchIncludeDirs
	opts.IncludeFiles = searchIncludeFiles
	if searchDepth < 0 {
		fmt.Println("错误: depth 不能为负数")
		return fmt.Errorf("invalid depth")
	}
	opts.MaxDepth = searchDepth
	// 不设置并发度，内部默认按 CPU 核心数计算
	opts.CaseInsensitive = searchIgnoreCase

	// 防止用户同时禁用文件与目录
	if !searchIncludeFiles && !searchIncludeDirs {
		fmt.Println("错误: --files=false 与 --dirs=false 会导致无结果，请至少启用一种")
		return fmt.Errorf("empty result configuration")
	}

	// 执行搜索
	ctx := context.Background()
	matched, err := projsearch.Search(ctx, targetNode, opts)
	if err != nil {
		fmt.Printf("搜索出错: %v\n", err)
		return err
	}

	// 输出结果（使用项目相对路径）
	if len(matched) == 0 {
		fmt.Println("未找到匹配项")
		return nil
	}

	for _, n := range matched {
		typeStr := "file"
		if n.IsDir {
			typeStr = "dir"
		}
		p := strings.TrimPrefix(n.Path, "/") // 显示为相对项目根
		if p == "" {
			p = "."
		}
		fmt.Printf("[%s] %s\n", typeStr, p)
	}
	return nil
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
