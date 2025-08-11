package blame

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sjzsdu/tong/project"
)

// 粒度定义
type Granularity string

const (
	GranularityDay   Granularity = "day"
	GranularityWeek  Granularity = "week"
	GranularityMonth Granularity = "month"
)

// Options 配置
// - 以 root 节点为统计子树根
// - Since/Until 时间范围可选（按每行的 author-time 过滤）
// - 并发处理文件与 blame
// - 仅统计文件（目录忽略）
// - 过滤扩展名与隐藏文件
// - Author 用 email 聚合（无邮箱则退化为 name）
type Options struct {
	Since         *time.Time
	Until         *time.Time
	Granularity   Granularity
	MaxWorkers    int
	Extensions    []string
	IncludeHidden bool
	UseEmail      bool // true: 按邮箱聚合；false: 按作者名聚合
}

// DefaultOptions 默认配置
func DefaultOptions() *Options {
	return &Options{
		Granularity:   GranularityWeek,
		MaxWorkers:    0,
		Extensions:    nil,
		IncludeHidden: false,
		UseEmail:      true,
	}
}

// Stat 聚合后的指标（基于 blame 行归属）
// Lines: 归属于该作者且落入该时间粒度周期内的代码行数
type Stat struct {
	Lines int
}

// Report 统计结果
// period(YYYY-MM / YYYY-Www / YYYY-MM-DD) -> author -> Stat
type Report struct {
	Granularity Granularity
	ByPeriod    map[string]map[string]*Stat
}

// Analyze 使用系统 git blame 对子树下文件进行统计（并发）
func Analyze(ctx context.Context, root *project.Node, opts *Options) (*Report, error) {
	if root == nil {
		return &Report{Granularity: GranularityWeek, ByPeriod: map[string]map[string]*Stat{}}, nil
	}
	if opts == nil {
		opts = DefaultOptions()
	}

	// 项目根路径与子树前缀
	proj := project.GetProjectByRoot(findRoot(root))
	if proj == nil {
		return &Report{Granularity: opts.Granularity, ByPeriod: map[string]map[string]*Stat{}}, nil
	}
	repoPath := proj.GetRootPath()
	subtreePrefix := strings.TrimPrefix(root.Path, "/")

	// 收集需要统计的文件节点
	results := project.ProcessConcurrentBFSTyped(ctx, root, opts.MaxWorkers, func(n *project.Node) (*project.Node, error) {
		if n.IsDir {
			return nil, nil
		}
		// 过滤隐藏路径
		if !opts.IncludeHidden && isHiddenPath(strings.TrimPrefix(n.Path, "/")) {
			return nil, nil
		}
		// 过滤扩展名
		if !allowByExt(n.Name, opts.Extensions) {
			return nil, nil
		}
		// 限定在子树（root 下自然成立，这里冗余校验）
		if subtreePrefix != "" && !isUnder(strings.TrimPrefix(n.Path, "/"), subtreePrefix) {
			return nil, nil
		}
		return n, nil
	})

	files := make([]*project.Node, 0, len(results))
	for _, r := range results {
		if r.Err != nil || r.Value == nil {
			continue
		}
		files = append(files, r.Value)
	}
	if len(files) == 0 {
		return &Report{Granularity: opts.Granularity, ByPeriod: map[string]map[string]*Stat{}}, nil
	}

	workers := opts.MaxWorkers
	if workers <= 0 {
		workers = runtime.NumCPU()
		if workers < 1 {
			workers = 1
		}
	}

	report := &Report{Granularity: opts.Granularity, ByPeriod: make(map[string]map[string]*Stat)}
	var mu sync.Mutex

	fileCh := make(chan *project.Node, workers*2)
	var wg sync.WaitGroup
	workerFn := func() {
		defer wg.Done()
		for n := range fileCh {
			select {
			case <-ctx.Done():
				return
			default:
			}
			rel := strings.TrimPrefix(n.Path, "/")
			lines, err := blameFile(ctx, repoPath, rel)
			if err != nil {
				continue
			}
			for _, ln := range lines {
				// 时间过滤
				if opts.Since != nil && ln.When.Before(*opts.Since) {
					continue
				}
				if opts.Until != nil && ln.When.After(*opts.Until) {
					continue
				}
				period := formatPeriod(ln.When, opts.Granularity)
				author := ln.Author
				if opts.UseEmail && ln.Email != "" {
					author = strings.ToLower(ln.Email)
				}
				mu.Lock()
				m, ok := report.ByPeriod[period]
				if !ok {
					m = make(map[string]*Stat)
					report.ByPeriod[period] = m
				}
				st, ok := m[author]
				if !ok {
					st = &Stat{}
					m[author] = st
				}
				st.Lines += 1
				mu.Unlock()
			}
		}
	}

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go workerFn()
	}
	for _, f := range files {
		fileCh <- f
	}
	close(fileCh)
	wg.Wait()

	return report, nil
}

// blameLine 保存 blame 中每一行的作者与时间
type blameLine struct {
	Author string
	Email  string
	When   time.Time
}

// 使用 git blame --line-porcelain 解析每行作者信息
func blameFile(ctx context.Context, repoPath, relPath string) ([]blameLine, error) {
	cmd := exec.CommandContext(ctx, "git", "blame", "--line-porcelain", "--", relPath)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parsePorcelain(out)
}

func parsePorcelain(out []byte) ([]blameLine, error) {
	s := bufio.NewScanner(bytes.NewReader(out))
	// 提高扫描缓冲以支持长行
	buf := make([]byte, 0, 64*1024)
	s.Buffer(buf, 10*1024*1024)

	var lines []blameLine
	cur := blameLine{}
	for s.Scan() {
		line := s.Text()
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "author ") {
			cur.Author = strings.TrimSpace(line[len("author "):])
			continue
		}
		if strings.HasPrefix(line, "author-mail ") {
			em := strings.TrimSpace(line[len("author-mail "):])
			em = strings.Trim(em, "<>")
			cur.Email = em
			continue
		}
		if strings.HasPrefix(line, "author-time ") {
			secStr := strings.TrimSpace(line[len("author-time "):])
			if sec, err := strconv.ParseInt(secStr, 10, 64); err == nil {
				cur.When = time.Unix(sec, 0)
			}
			continue
		}
		if strings.HasPrefix(line, "\t") {
			// 一行代码内容，对应当前作者/时间
			lines = append(lines, cur)
			continue
		}
		// 其他字段忽略
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

// findRoot 找到树根节点
func findRoot(n *project.Node) *project.Node {
	root := n
	for root.Parent != nil {
		root = root.Parent
	}
	return root
}

func formatPeriod(t time.Time, g Granularity) string {
	y, m, _ := t.Date()
	switch g {
	case GranularityDay:
		return t.Format("2006-01-02")
	case GranularityWeek:
		yw, ww := t.ISOWeek()
		return sprintf("%04d-W%02d", yw, ww)
	case GranularityMonth:
		return sprintf("%04d-%02d", y, m)
	default:
		return sprintf("%04d-%02d", y, m)
	}
}

func isUnder(path, prefix string) bool {
	if prefix == "" {
		return true
	}
	p := strings.TrimPrefix(path, "/")
	pre := strings.TrimSuffix(strings.TrimPrefix(prefix, "/"), "/")
	if p == pre {
		return true
	}
	return strings.HasPrefix(p, pre+"/")
}

func isHiddenPath(path string) bool {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for _, p := range parts {
		if strings.HasPrefix(p, ".") {
			return true
		}
	}
	return false
}

func allowByExt(name string, exts []string) bool {
	if len(exts) == 0 {
		return true
	}
	for _, e := range exts {
		if e == "*" || e == "" {
			return true
		}
	}
	i := strings.LastIndexByte(name, '.')
	if i < 0 || i == len(name)-1 {
		return false
	}
	ext := name[i+1:]
	for _, e := range exts {
		if strings.EqualFold(ext, e) {
			return true
		}
	}
	return false
}

// 排序帮助：按字符串键排序（适用于 map[string]V）
func SortedKeys[K ~string, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

// 为了避免引入 fmt 依赖在热路径，定义一个简单的 Sprintf 包装（编译器会内联）
func sprintf(format string, a ...any) string { return fmt.Sprintf(format, a...) }
