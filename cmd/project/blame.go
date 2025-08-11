package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	projblame "github.com/sjzsdu/tong/project/blame"
	"github.com/spf13/cobra"
)

var (
	blameSince         string
	blameUntil         string
	blameGranularity   string
	blameIncludeMerges bool
	blameExtensions    []string
	blameIncludeHidden bool
	blameUseEmail      bool
	blameSubdir        string
)

var BlameCmd = &cobra.Command{
	Use:   "blame",
	Short: "统计不同时间段、不同作者的代码归属行数",
	Long: `基于 git blame --line-porcelain，并发统计 Git 仓库在指定子树范围内每行代码的归属，
按时间粒度与作者聚合，输出各周期各作者的行数占比（计数）。

默认使用作者邮箱聚合（--use-email=true），时间粒度为周（--granularity=week）。
可通过 --since/--until 指定时间范围（格式：YYYY-MM-DD）。`,
	Args: cobra.NoArgs,
	Run:  runBlame,
}

func init() {
	BlameCmd.Flags().StringVar(&blameSince, "since", "", "起始日期（含，格式：YYYY-MM-DD）")
	BlameCmd.Flags().StringVar(&blameUntil, "until", "", "结束日期（含，格式：YYYY-MM-DD）")
	BlameCmd.Flags().StringVar(&blameGranularity, "granularity", "week", "时间粒度：day|week|month")
	// 兼容旧参数，但当前基于 blame 的统计不会使用合并提交开关
	BlameCmd.Flags().BoolVar(&blameIncludeMerges, "include-merges", false, "兼容选项：保留但对 blame 统计无影响")
	BlameCmd.Flags().StringSliceVar(&blameExtensions, "ext", []string{}, "只统计指定扩展名文件，例如: go,md；为空表示不过滤")
	BlameCmd.Flags().BoolVar(&blameIncludeHidden, "hidden", false, "包含隐藏文件/目录")
	BlameCmd.Flags().BoolVar(&blameUseEmail, "use-email", true, "按作者邮箱聚合（否则按作者名聚合）")
	BlameCmd.Flags().StringVar(&blameSubdir, "subdir", ".", "限定统计的子目录（相对项目根）")
}

func runBlame(cmd *cobra.Command, args []string) {
	// 必须使用共享项目实例
	if sharedProject == nil {
		fmt.Printf("错误: 未找到共享的项目实例\n")
		os.Exit(1)
	}

	// 定位子树根节点
	var finalTargetPath string
	if filepath.IsAbs(blameSubdir) {
		finalTargetPath = blameSubdir
	} else {
		finalTargetPath = filepath.Join(sharedProject.GetRootPath(), blameSubdir)
	}
	targetNode, err := GetTargetNode(finalTargetPath)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	// 解析日期
	var sincePtr, untilPtr *time.Time
	if t, ok := parseDate(blameSince); ok {
		sincePtr = &t
	}
	if t, ok := parseDate(blameUntil); ok {
		// until 设为当天 23:59:59 以便“含当日”
		end := t.Add(24*time.Hour - time.Nanosecond)
		untilPtr = &end
	}

	// 粒度
	g := projblame.Granularity(strings.ToLower(blameGranularity))
	switch g {
	case projblame.GranularityDay, projblame.GranularityWeek, projblame.GranularityMonth:
	default:
		g = projblame.GranularityWeek
	}

	// 构建选项
	opts := projblame.DefaultOptions()
	opts.Since = sincePtr
	opts.Until = untilPtr
	opts.Granularity = g
	opts.Extensions = normalizeExts(blameExtensions)
	opts.IncludeHidden = blameIncludeHidden
	opts.UseEmail = blameUseEmail

	// 执行分析
	ctx := context.Background()
	report, err := projblame.Analyze(ctx, targetNode, opts)
	if err != nil {
		fmt.Printf("分析出错: %v\n", err)
		os.Exit(1)
	}

	// 输出结果
	if len(report.ByPeriod) == 0 {
		fmt.Println("没有匹配的代码行")
		return
	}

	periods := projblame.SortedKeys(report.ByPeriod)
	// 总体汇总（所有周期合计）
	overall := make(map[string]int)
	overallTotal := 0

	for _, p := range periods {
		m := report.ByPeriod[p]
		// 计算周期总行数
		periodTotal := 0
		for _, st := range m {
			periodTotal += st.Lines
		}
		fmt.Printf("%s (total lines=%d)\n", p, periodTotal)

		authors := projblame.SortedKeys(m)
		for _, a := range authors {
			st := m[a]
			ratio := 0.0
			if periodTotal > 0 {
				ratio = float64(st.Lines) * 100.0 / float64(periodTotal)
			}
			fmt.Printf("  - %s: lines=%d (%.1f%%)\n", a, st.Lines, ratio)

			// 汇总到整体
			overall[a] += st.Lines
			overallTotal += st.Lines
		}
	}

	// 打印整体汇总
	if overallTotal > 0 {
		fmt.Printf("\nOverall (total lines=%d)\n", overallTotal)
		authorsAll := make([]string, 0, len(overall))
		for a := range overall {
			authorsAll = append(authorsAll, a)
		}
		sort.Slice(authorsAll, func(i, j int) bool {
			if overall[authorsAll[i]] == overall[authorsAll[j]] {
				return authorsAll[i] < authorsAll[j]
			}
			return overall[authorsAll[i]] > overall[authorsAll[j]]
		})
		for _, a := range authorsAll {
			lines := overall[a]
			ratio := float64(lines) * 100.0 / float64(overallTotal)
			fmt.Printf("  - %s: lines=%d (%.1f%%)\n", a, lines, ratio)
		}
	}
}

func parseDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, true
	}
	// 兼容 RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	return time.Time{}, false
}
