package git

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/sjzsdu/tong/project"
)

// BlameInfo stores blame information for each line of a file
type BlameInfo struct {
	Lines      []LineInfo     // Detailed information for each line
	Authors    map[string]int // Line count statistics by author
	Dates      map[string]int // Modification line count by date
	TotalLines int            // Total number of lines
	FilePath   string         // File path
}

// LineInfo stores blame information for a single line
type LineInfo struct {
	LineNum    int       // Line number
	Author     string    // Author
	Email      string    // Email
	CommitID   string    // Commit ID
	CommitTime time.Time // Commit time
	Content    string    // Line content
}

// GitBlamer 使用go-git库实现的Git blame分析器
type GitBlamer struct {
	ShowEmail  bool
	Project    *project.Project
	repo       *git.Repository
	FileFilter func(path string) bool // 文件过滤器，返回 true 表示需要分析该文件
}

// NewGitBlamer creates a new Git blame analyzer using go-git library
func NewGitBlamer(p *project.Project) (*GitBlamer, error) {
	rootPath := p.GetRootPath()
	// 使用 PlainOpenWithOptions 并启用 DetectDotGit 选项，以支持在子文件夹中查找 Git 仓库
	repo, err := git.PlainOpenWithOptions(rootPath, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	// 如果找不到 Git 仓库，返回一个空的 GitBlamer 实例而不是错误
	if err != nil {
		// 返回一个空的 GitBlamer 实例，repo 为 nil
		return &GitBlamer{
			ShowEmail: true,
			Project:   p,
			repo:      nil,
			FileFilter: func(path string) bool {
				// 当 repo 为 nil 时，所有文件都不分析
				return false
			},
		}, nil
	}
	return &GitBlamer{
		ShowEmail: true,
		Project:   p,
		repo:      repo,
		FileFilter: func(path string) bool {
			// 默认过滤器：排除常见的二进制文件和临时文件
			ext := filepath.Ext(path)
			
			// 排除常见二进制文件扩展名
			binaryExts := map[string]bool{
				".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
				".pdf": true, ".zip": true, ".tar": true, ".gz": true,
				".exe": true, ".dll": true, ".so": true, ".dylib": true,
				".class": true, ".jar": true, ".war": true,
			}

			// 排除常见临时文件和隐藏文件
			tempPatterns := []string{
				".git", ".svn", ".DS_Store", "Thumbs.db",
				"node_modules", "vendor", "dist", "build",
			}

			// 检查扩展名
			if _, found := binaryExts[ext]; found {
				return false
			}

			// 检查文件名模式
			for _, pattern := range tempPatterns {
				if strings.Contains(path, pattern) {
					return false
				}
			}

			return true
		},
	}, nil
}

// normalizePath normalizes the file path to project relative path
func (g *GitBlamer) normalizePath(filePath string) (string, *project.Node, error) {
	rootPath := g.Project.GetRootPath()
	node, err := g.Project.FindNode(filePath)
	if err != nil {
		absPath := filepath.Join(rootPath, filePath)
		node, err = g.Project.FindNode(absPath)
		if err != nil {
			relPath := strings.TrimPrefix(filePath, rootPath)
			relPath = strings.TrimPrefix(relPath, "/")
			node, err = g.Project.FindNode(relPath)
			if err != nil {
				return "", nil, fmt.Errorf("cannot find file or directory: %w, tried paths: %s, %s, %s", err, filePath, absPath, relPath)
			}
			return relPath, node, nil
		}
		return absPath, node, nil
	}
	return filePath, node, nil
}

// handleUncommittedFile handles blame for uncommitted files
func (g *GitBlamer) handleUncommittedFile(filePath string) (*BlameInfo, error) {
	content, err := g.Project.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read file content: %w", err)
	}
	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// 获取用户配置
	var cfg *config.Config
	if g.repo != nil {
		cfg, err = g.repo.Config()
		if err != nil {
			// 如果无法获取配置，使用默认值
			cfg = nil
		}
	}

	author := "Unknown"
	if cfg != nil && cfg.User.Name != "" {
		author = cfg.User.Name
	}

	now := time.Now()
	dateStr := now.Format("2006-01-02")
	blameInfo := &BlameInfo{
		Lines:      make([]LineInfo, 0, len(lines)),
		Authors:    map[string]int{author: len(lines)},
		Dates:      map[string]int{dateStr: len(lines)},
		TotalLines: len(lines),
		FilePath:   filePath,
	}
	for i, line := range lines {
		blameInfo.Lines = append(blameInfo.Lines, LineInfo{
			LineNum:    i + 1,
			Author:     author,
			CommitID:   "未提交",
			CommitTime: now,
			Content:    line,
		})
	}
	return blameInfo, nil
}

// Blame performs git blame analysis based on file path using go-git
func (g *GitBlamer) Blame(filePath string) (*BlameInfo, error) {
	// 如果 repo 为 nil，表示不是 Git 仓库或无法访问，返回空的 BlameInfo
	if g.repo == nil {
		return &BlameInfo{
			Lines:      []LineInfo{},
			Authors:    make(map[string]int),
			Dates:      make(map[string]int),
			TotalLines: 0,
			FilePath:   filePath,
		}, nil
	}

	if filePath == "" {
		filePath = "/"
	}
	normalizedPath, node, err := g.normalizePath(filePath)
	if err != nil {
		return nil, err
	}
	if !node.IsDir {
		return g.BlameFile(g.Project, normalizedPath)
	}
	results, err := g.BlameDirectory(g.Project, normalizedPath)
	if err != nil {
		return nil, err
	}
	mergedInfo := &BlameInfo{
		Authors:  make(map[string]int),
		Dates:    make(map[string]int),
		FilePath: normalizedPath,
	}
	for _, info := range results {
		mergedInfo.TotalLines += info.TotalLines
		for author, count := range info.Authors {
			mergedInfo.Authors[author] += count
		}
		for date, count := range info.Dates {
			mergedInfo.Dates[date] += count
		}
	}
	return mergedInfo, nil
}

// BlameFile analyzes blame information for a single file using go-git
func (g *GitBlamer) BlameFile(p *project.Project, filePath string) (*BlameInfo, error) {
	// 如果 repo 为 nil，表示不是 Git 仓库或无法访问，返回空的 BlameInfo
	if g.repo == nil {
		return &BlameInfo{
			Lines:      []LineInfo{},
			Authors:    make(map[string]int),
			Dates:      make(map[string]int),
			TotalLines: 0,
			FilePath:   filePath,
		}, nil
	}

	rootPath := p.GetRootPath()
	normalizedPath, node, err := g.normalizePath(filePath)
	if err != nil {
		return nil, err
	}
	if node.IsDir {
		return nil, fmt.Errorf("%s is a directory, not a file", normalizedPath)
	}

	// 获取相对于仓库根目录的路径
	relFilePath := strings.TrimPrefix(normalizedPath, rootPath)
	relFilePath = strings.TrimPrefix(relFilePath, "/")

	// 检查文件是否在Git仓库中
	worktree, err := g.repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	// 检查文件是否存在于工作区
	_, err = worktree.Filesystem.Stat(relFilePath)
	if err != nil {
		// 文件不在Git仓库中，可能是未提交的文件
		return g.handleUncommittedFile(normalizedPath)
	}

	// 获取HEAD引用
	ref, err := g.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	// 获取当前提交
	commit, err := g.repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit object: %w", err)
	}

	// 使用go-git的Blame功能
	blameResult, err := git.Blame(commit, relFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get blame information: %w", err)
	}

	// 构建BlameInfo
	blameInfo := &BlameInfo{
		Lines:    make([]LineInfo, 0, len(blameResult.Lines)),
		Authors:  make(map[string]int),
		Dates:    make(map[string]int),
		FilePath: normalizedPath,
	}

	// 处理每一行的blame信息
	for i, line := range blameResult.Lines {
		// 获取提交信息
		commit, err := g.repo.CommitObject(line.Hash)
		if err != nil {
			// 如果无法获取提交，使用默认值
			blameInfo.Lines = append(blameInfo.Lines, LineInfo{
				LineNum:  i + 1,
				Author:   "Unknown",
				CommitID: line.Hash.String()[:7],
				Content:  line.Text,
			})
			continue
		}

		// 获取作者信息
		author := commit.Author.Name
		email := commit.Author.Email
		commitTime := commit.Author.When

		// 添加行信息
		blameInfo.Lines = append(blameInfo.Lines, LineInfo{
			LineNum:    i + 1,
			Author:     author,
			Email:      email,
			CommitID:   line.Hash.String()[:7],
			CommitTime: commitTime,
			Content:    line.Text,
		})

		// 更新统计信息
		blameInfo.Authors[author]++
		dateStr := commitTime.Format("2006-01-02")
		blameInfo.Dates[dateStr]++
	}

	blameInfo.TotalLines = len(blameInfo.Lines)
	return blameInfo, nil
}

// GitBlameVisitor for collecting blame information during project traversal using go-git
type GitBlameVisitor struct {
	Blamer      *GitBlamer
	Project     *project.Project
	Results     map[string]*BlameInfo
	Errors      []error
	Concurrency int
	semaphore   chan struct{}
	mutex       sync.Mutex
	wg          sync.WaitGroup
}

// NewGitBlameVisitor creates a new Git blame visitor using go-git
func NewGitBlameVisitor(blamer *GitBlamer, p *project.Project, concurrency int) *GitBlameVisitor {
	if concurrency <= 0 {
		// 将默认并发数从10降低到1，以避免并发访问问题
		concurrency = 1
	}
	return &GitBlameVisitor{
		Blamer:    blamer,
		Project:   p,
		Results:   make(map[string]*BlameInfo),
		semaphore: make(chan struct{}, concurrency),
	}
}

// VisitFile implements NodeVisitor for file nodes
func (v *GitBlameVisitor) VisitFile(node *project.Node, path string, depth int) error {
	if node.IsDir {
		return nil
	}

	// 应用文件过滤器
	if v.Blamer.FileFilter != nil && !v.Blamer.FileFilter(path) {
		return nil
	}

	v.semaphore <- struct{}{}
	v.wg.Add(1)
	go func() {
		defer v.wg.Done()
		defer func() { <-v.semaphore }()
		blameInfo, err := v.Blamer.BlameFile(v.Project, path)
		if err != nil {
			v.mutex.Lock()
			v.Errors = append(v.Errors, fmt.Errorf("failed to analyze file %s: %w", path, err))
			v.mutex.Unlock()
			return
		}
		if blameInfo != nil {
			v.mutex.Lock()
			v.Results[path] = blameInfo
			v.mutex.Unlock()
		}
	}()
	return nil
}

// VisitDirectory implements NodeVisitor for directory nodes
func (v *GitBlameVisitor) VisitDirectory(node *project.Node, path string, depth int) error {
	return nil
}

// BlameDirectory analyzes blame information for all files in the specified directory using go-git
func (g *GitBlamer) BlameDirectory(p *project.Project, dirPath string) (map[string]*BlameInfo, error) {
	// 如果 repo 为 nil，表示不是 Git 仓库或无法访问，返回空的结果
	if g.repo == nil {
		return make(map[string]*BlameInfo), nil
	}

	normalizedPath, node, err := g.normalizePath(dirPath)
	if err != nil {
		return nil, err
	}
	if !node.IsDir {
		return nil, fmt.Errorf("%s is not a directory", normalizedPath)
	}
	// 使用并发数1来避免并发访问问题
	visitor := NewGitBlameVisitor(g, p, 1)
	traverser := project.NewTreeTraverser(p)
	traverser.SetTraverseOrder(project.PreOrder)
	traverser.SetOption(&project.TraverseOption{
		ContinueOnError: true,
	})
	if normalizedPath == "/" || normalizedPath == "" {
		err = traverser.TraverseTree(visitor)
	} else {
		err = traverser.Traverse(node, normalizedPath, 0, visitor)
	}
	visitor.wg.Wait()
	if err != nil {
		// Log warning but return results
	}
	if len(visitor.Results) == 0 && len(node.Children) > 0 {
		for _, child := range node.Children {
			if !child.IsDir {
				childPath := p.GetNodePath(child)
				blameInfo, err := g.BlameFile(p, childPath)
				if err == nil && blameInfo != nil {
					visitor.Results[childPath] = blameInfo
				}
			}
		}
	}
	return visitor.Results, nil
}

// BlameProject analyzes blame information for all files in the project using go-git
func (g *GitBlamer) BlameProject(p *project.Project) (map[string]*BlameInfo, error) {
	// 分析整个项目就是分析根目录
	return g.BlameDirectory(p, "/")
}
