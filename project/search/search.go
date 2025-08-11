package search

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/sjzsdu/tong/project"
)

// SearchOptions 定义搜索选项
type SearchOptions struct {
	// 名称包含（子串匹配）
	NameContains string
	// 名称正则（优先于 NameContains）
	NameRegex string
	// 内容包含（子串匹配）
	ContentContains string
	// 内容正则（优先于 ContentContains）
	ContentRegex string
	// 仅对文件生效的扩展名过滤，如 []string{"go","md"}；为空或包含 "*" 表示不过滤
	Extensions []string
	// 是否包含隐藏文件/目录（以 . 开头）
	IncludeHidden bool
	// 是否在结果中包含目录
	IncludeDirs bool
	// 是否在结果中包含文件
	IncludeFiles bool
	// 限制搜索深度（相对 root），0 表示不限制；root 深度为 0
	MaxDepth int
	// 并发 worker 数，<=0 使用默认
	MaxWorkers int
	// 名称与内容的大小写不敏感匹配（对子串与正则均生效）
	CaseInsensitive bool
	// MatchAny 为 true 时，名称或内容任一匹配即算命中；为 false 时采用“与”逻辑
	MatchAny bool
}

// DefaultSearchOptions 返回默认搜索选项
func DefaultSearchOptions() *SearchOptions {
	return &SearchOptions{
		IncludeHidden:   false,
		IncludeDirs:     false,
		IncludeFiles:    true,
		MaxDepth:        0,
		MaxWorkers:      0,
		CaseInsensitive: true,
		MatchAny:        false,
	}
}

// Search 在指定的 node 子树下并发搜索，返回匹配的节点（按 Path 排序）
func Search(ctx context.Context, root *project.Node, opts *SearchOptions) ([]*project.Node, error) {
	if root == nil {
		return nil, nil
	}
	if opts == nil {
		opts = DefaultSearchOptions()
	}

	// 预编译正则（带大小写选项）
	var nameRe, contentRe *regexp.Regexp
	var err error
	if opts.NameRegex != "" {
		pattern := opts.NameRegex
		if opts.CaseInsensitive && !strings.HasPrefix(pattern, "(?i)") {
			pattern = "(?i)" + pattern
		}
		nameRe, err = regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
	}
	if opts.ContentRegex != "" {
		pattern := opts.ContentRegex
		if opts.CaseInsensitive && !strings.HasPrefix(pattern, "(?i)") {
			pattern = "(?i)" + pattern
		}
		contentRe, err = regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
	}

	applyName := opts.NameRegex != "" || opts.NameContains != ""
	applyContent := opts.ContentRegex != "" || opts.ContentContains != ""

	rootSeg := pathSegments(root.Path)

	// 使用并发 BFS 遍历整个子树
	results := project.ProcessConcurrentBFSTyped(ctx, root, opts.MaxWorkers, func(n *project.Node) ([]*project.Node, error) {
		// 深度限制
		relDepth := pathSegments(n.Path) - rootSeg
		if opts.MaxDepth > 0 && relDepth > opts.MaxDepth {
			return nil, nil
		}

		// 隐藏项过滤（根除外）
		if !opts.IncludeHidden && n != root && strings.HasPrefix(n.Name, ".") {
			return nil, nil
		}

		// 类型过滤（是否纳入结果，不影响遍历）
		if n.IsDir && !opts.IncludeDirs {
			// 不返回目录，但继续遍历
			return nil, nil
		}
		if !n.IsDir && !opts.IncludeFiles {
			return nil, nil
		}

		// 扩展名过滤（仅对文件）
		if !n.IsDir && !allowByExt(n.Name, opts.Extensions) {
			return nil, nil
		}

		nameOK := matchNodeName(n, opts, nameRe)
		contentOK := matchNodeContent(n, opts, contentRe)

		if opts.MatchAny {
			matched := false
			if applyName && nameOK {
				matched = true
			}
			if applyContent && contentOK {
				matched = true
			}
			// 若两者均未指定，默认按名称逻辑
			if !applyName && !applyContent {
				matched = nameOK
			}
			if matched {
				return []*project.Node{n}, nil
			}
		} else {
			// AND 逻辑：仅对启用的条件进行判断
			if (!applyName || nameOK) && (!applyContent || contentOK) {
				return []*project.Node{n}, nil
			}
		}
		return nil, nil
	})

	matched := make([]*project.Node, 0)
	for _, r := range results {
		if r.Err != nil || r.Value == nil {
			continue
		}
		if len(r.Value) > 0 && r.Value[0] != nil {
			matched = append(matched, r.Value[0])
		}
	}

	// 稳定排序
	sort.Slice(matched, func(i, j int) bool { return matched[i].Path < matched[j].Path })
	return matched, nil
}

func matchNodeName(n *project.Node, opts *SearchOptions, re *regexp.Regexp) bool {
	name := n.Name
	if re != nil {
		return re.MatchString(name)
	}
	if opts.NameContains == "" {
		return true
	}
	if opts.CaseInsensitive {
		return strings.Contains(strings.ToLower(name), strings.ToLower(opts.NameContains))
	}
	return strings.Contains(name, opts.NameContains)
}

func matchNodeContent(n *project.Node, opts *SearchOptions, re *regexp.Regexp) bool {
	// 目录节点在启用内容过滤时不应匹配；未启用内容过滤时返回 true 以不影响其他条件
	if n.IsDir {
		if opts.ContentRegex != "" || opts.ContentContains != "" {
			return false
		}
		return true
	}
	if re == nil && opts.ContentContains == "" {
		return true
	}

	data, err := n.ReadContent()
	if err != nil || data == nil {
		return false
	}
	text := string(data)
	if re != nil {
		return re.MatchString(text)
	}
	if opts.CaseInsensitive {
		return strings.Contains(strings.ToLower(text), strings.ToLower(opts.ContentContains))
	}
	return strings.Contains(text, opts.ContentContains)
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
	// 提取扩展名（不含点）
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

func pathSegments(p string) int {
	if p == "/" || p == "" {
		return 0
	}
	sp := strings.TrimPrefix(p, "/")
	if sp == "" {
		return 0
	}
	return len(strings.Split(sp, "/"))
}
