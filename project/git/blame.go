package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
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

// GitBlamer interface for Git blame analyzer
type GitBlamer interface {
	Blame(filePath string) (*BlameInfo, error)
}

// DefaultGitBlamer default implementation of Git blame analyzer using git command line
type DefaultGitBlamer struct {
	ShowEmail bool
	Project   *project.Project
}

// GoGitBlamer implementation of Git blame analyzer using go-git library
type GoGitBlamer struct {
	ShowEmail bool
	Project   *project.Project
	repo      *git.Repository
}

// NewDefaultGitBlamer creates a new default Git blame analyzer using git command line
func NewDefaultGitBlamer(p *project.Project) *DefaultGitBlamer {
	return &DefaultGitBlamer{
		ShowEmail: true,
		Project:   p,
	}
}

// NewGoGitBlamer creates a new Git blame analyzer using go-git library
func NewGoGitBlamer(p *project.Project) (*GoGitBlamer, error) {
	rootPath := p.GetRootPath()
	repo, err := git.PlainOpen(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open git repository: %w", err)
	}
	return &GoGitBlamer{
		ShowEmail: true,
		Project:   p,
		repo:      repo,
	}, nil
}

// BlamerType defines the type of Git blamer to use
type BlamerType string

const (
	// CommandLineBlamer uses git command line for blame operations
	CommandLineBlamer BlamerType = "command-line"
	// GoGitLibraryBlamer uses go-git library for blame operations
	GoGitLibraryBlamer BlamerType = "go-git"
)

// NewGitBlamer creates a new Git blame analyzer based on the specified type
func NewGitBlamer(p *project.Project, blamerType BlamerType) (GitBlamer, error) {
	switch blamerType {
	case CommandLineBlamer:
		return NewDefaultGitBlamer(p), nil
	case GoGitLibraryBlamer:
		return NewGoGitBlamer(p)
	default:
		// 默认使用命令行实现，保持向后兼容性
		return NewDefaultGitBlamer(p), nil
	}
}

// normalizePath normalizes the file path to project relative path
func (g *DefaultGitBlamer) normalizePath(filePath string) (string, *project.Node, error) {
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

// normalizePath normalizes the file path to project relative path
func (g *GoGitBlamer) normalizePath(filePath string) (string, *project.Node, error) {
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
func (g *DefaultGitBlamer) handleUncommittedFile(filePath string) (*BlameInfo, error) {
	rootPath := g.Project.GetRootPath()
	content, err := g.Project.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read file content: %w", err)
	}
	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	authorCmd := exec.Command("git", "-C", rootPath, "config", "user.name")
	authorOutput, _ := authorCmd.CombinedOutput()
	author := strings.TrimSpace(string(authorOutput))
	if author == "" {
		author = "Unknown"
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

// handleUncommittedFile handles blame for uncommitted files
func (g *GoGitBlamer) handleUncommittedFile(filePath string) (*BlameInfo, error) {
	content, err := g.Project.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read file content: %w", err)
	}
	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	
	// 获取用户配置
	cfg, err := g.repo.Config()
	if err != nil {
		// 如果无法获取配置，使用默认值
		cfg = nil
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

// Blame performs git blame analysis based on file path
func (g *DefaultGitBlamer) Blame(filePath string) (*BlameInfo, error) {
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
		Authors:    make(map[string]int),
		Dates:      make(map[string]int),
		FilePath:   normalizedPath,
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

// Blame performs git blame analysis based on file path using go-git
func (g *GoGitBlamer) Blame(filePath string) (*BlameInfo, error) {
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
		Authors:    make(map[string]int),
		Dates:      make(map[string]int),
		FilePath:   normalizedPath,
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

// BlameFile analyzes blame information for a single file
func (g *DefaultGitBlamer) BlameFile(p *project.Project, filePath string) (*BlameInfo, error) {
	rootPath := p.GetRootPath()
	normalizedPath, node, err := g.normalizePath(filePath)
	if err != nil {
		return nil, err
	}
	if node.IsDir {
		return nil, fmt.Errorf("%s is a directory, not a file", normalizedPath)
	}
	relFilePath := strings.TrimPrefix(normalizedPath, rootPath)
	relFilePath = strings.TrimPrefix(relFilePath, "/")
	checkCmd := exec.Command("git", "-C", rootPath, "ls-files", "--error-unmatch", relFilePath)
	if checkErr := checkCmd.Run(); checkErr != nil {
		return g.handleUncommittedFile(normalizedPath)
	}
	cmd := exec.Command("git", "blame", "--line-porcelain", "-w", relFilePath)
	cmd.Dir = rootPath
	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git blame command failed: %w", err)
	}
	blameInfo := &BlameInfo{
		Lines:   make([]LineInfo, 0),
		Authors: make(map[string]int),
		Dates:   make(map[string]int),
		FilePath: normalizedPath,
	}
	reader := bytes.NewReader(cmdOutput)
	scanner := bufio.NewScanner(reader)
	var currentLine LineInfo
	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) >= 40 && !strings.Contains(line, " ") || strings.HasPrefix(line, "^") {
			if currentLine.CommitID != "" && currentLine.Author != "" {
				blameInfo.Lines = append(blameInfo.Lines, currentLine)
				blameInfo.Authors[currentLine.Author]++
				if !currentLine.CommitTime.IsZero() {
					dateStr := currentLine.CommitTime.Format("2006-01-02")
					blameInfo.Dates[dateStr]++
				}
			}
			lineNum++
			currentLine = LineInfo{
				LineNum:  lineNum,
				CommitID: strings.TrimPrefix(line, "^"),
			}
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		switch parts[0] {
		case "author":
			currentLine.Author = parts[1]
		case "author-mail":
			email := strings.Trim(parts[1], "<>")
			currentLine.Email = email
		case "author-time":
			unixTime, err := parseInt64(parts[1])
			if err == nil {
				currentLine.CommitTime = time.Unix(unixTime, 0)
			}
		case "\t":
			currentLine.Content = strings.TrimPrefix(line, "\t")
		}
	}
	if currentLine.CommitID != "" && currentLine.Author != "" {
		blameInfo.Lines = append(blameInfo.Lines, currentLine)
		blameInfo.Authors[currentLine.Author]++
		if !currentLine.CommitTime.IsZero() {
			dateStr := currentLine.CommitTime.Format("2006-01-02")
			blameInfo.Dates[dateStr]++
		}
	}
	blameInfo.TotalLines = len(blameInfo.Lines)
	return blameInfo, nil
}

// BlameFile analyzes blame information for a single file using go-git
func (g *GoGitBlamer) BlameFile(p *project.Project, filePath string) (*BlameInfo, error) {
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

// parseInt64 helper function to convert string to int64
func parseInt64(s string) (int64, error) {
	var result int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid number: %s", s)
		}
		result = result*10 + int64(c-'0')
	}
	return result, nil
}

// GitBlameVisitor for collecting blame information during project traversal
type GitBlameVisitor struct {
	Blamer       *DefaultGitBlamer
	Project      *project.Project
	Results      map[string]*BlameInfo
	Errors       []error
	Concurrency  int
	semaphore    chan struct{}
	mutex        sync.Mutex
	wg           sync.WaitGroup
}

// NewGitBlameVisitor creates a new Git blame visitor
func NewGitBlameVisitor(blamer *DefaultGitBlamer, p *project.Project, concurrency int) *GitBlameVisitor {
	if concurrency <= 0 {
		concurrency = 10
	}
	return &GitBlameVisitor{
		Blamer:      blamer,
		Project:     p,
		Results:     make(map[string]*BlameInfo),
		semaphore:   make(chan struct{}, concurrency),
	}
}

// VisitFile implements NodeVisitor for file nodes
func (v *GitBlameVisitor) VisitFile(node *project.Node, path string, depth int) error {
	if node.IsDir {
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

// BlameDirectory analyzes blame information for all files in the specified directory
func (g *DefaultGitBlamer) BlameDirectory(p *project.Project, dirPath string) (map[string]*BlameInfo, error) {
	normalizedPath, node, err := g.normalizePath(dirPath)
	if err != nil {
		return nil, err
	}
	if !node.IsDir {
		return nil, fmt.Errorf("%s is not a directory", normalizedPath)
	}
	visitor := NewGitBlameVisitor(g, p, 10)
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

// GoGitBlameVisitor for collecting blame information during project traversal using go-git
type GoGitBlameVisitor struct {
	Blamer       *GoGitBlamer
	Project      *project.Project
	Results      map[string]*BlameInfo
	Errors       []error
	Concurrency  int
	semaphore    chan struct{}
	mutex        sync.Mutex
	wg           sync.WaitGroup
}

// NewGoGitBlameVisitor creates a new Git blame visitor using go-git
func NewGoGitBlameVisitor(blamer *GoGitBlamer, p *project.Project, concurrency int) *GoGitBlameVisitor {
	if concurrency <= 0 {
		concurrency = 10
	}
	return &GoGitBlameVisitor{
		Blamer:      blamer,
		Project:     p,
		Results:     make(map[string]*BlameInfo),
		semaphore:   make(chan struct{}, concurrency),
	}
}

// VisitFile implements NodeVisitor for file nodes
func (v *GoGitBlameVisitor) VisitFile(node *project.Node, path string, depth int) error {
	if node.IsDir {
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
func (v *GoGitBlameVisitor) VisitDirectory(node *project.Node, path string, depth int) error {
	return nil
}

// BlameDirectory analyzes blame information for all files in the specified directory using go-git
func (g *GoGitBlamer) BlameDirectory(p *project.Project, dirPath string) (map[string]*BlameInfo, error) {
	normalizedPath, node, err := g.normalizePath(dirPath)
	if err != nil {
		return nil, err
	}
	if !node.IsDir {
		return nil, fmt.Errorf("%s is not a directory", normalizedPath)
	}
	visitor := NewGoGitBlameVisitor(g, p, 10)
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
