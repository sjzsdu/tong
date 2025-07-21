package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sjzsdu/tong/project"
)

// CmdGitBlamer 使用命令行git blame实现的Git blame分析器
type CmdGitBlamer struct {
	ShowEmail  bool
	Project    *project.Project
	FileFilter func(path string) bool // 文件过滤器，返回 true 表示需要分析该文件
}

// NewCmdGitBlamer 创建一个新的基于命令行的Git blame分析器
func NewCmdGitBlamer(p *project.Project) (*CmdGitBlamer, error) {
	// 检查git命令是否可用
	cmd := exec.Command("git", "--version")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git命令不可用: %w", err)
	}

	return &CmdGitBlamer{
		ShowEmail: true,
		Project:   p,
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

// normalizePath 规范化文件路径为项目相对路径
func (g *CmdGitBlamer) normalizePath(filePath string) (string, *project.Node, error) {
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

// handleUncommittedFile 处理未提交文件的blame信息
func (g *CmdGitBlamer) handleUncommittedFile(filePath string) (*BlameInfo, error) {
	content, err := g.Project.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read file content: %w", err)
	}
	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// 尝试获取git配置中的用户名
	cmd := exec.Command("git", "config", "user.name")
	cmd.Dir = g.Project.GetRootPath()
	output, err := cmd.Output()
	author := "Unknown"
	if err == nil && len(output) > 0 {
		author = strings.TrimSpace(string(output))
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

// Blame 使用命令行git blame分析文件或目录
func (g *CmdGitBlamer) Blame(filePath string) (*BlameInfo, error) {
	// 检查git命令是否可用
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = g.Project.GetRootPath()
	if err := cmd.Run(); err != nil {
		// 不是Git仓库，返回空的BlameInfo
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

// BlameFile 使用命令行git blame分析单个文件
func (g *CmdGitBlamer) BlameFile(p *project.Project, filePath string) (*BlameInfo, error) {
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
	cmd := exec.Command("git", "ls-files", "--error-unmatch", relFilePath)
	cmd.Dir = rootPath
	if err := cmd.Run(); err != nil {
		// 文件不在Git仓库中，可能是未提交的文件
		return g.handleUncommittedFile(normalizedPath)
	}

	// 使用git blame命令获取blame信息
	var args []string
	if g.ShowEmail {
		args = []string{"blame", "--line-porcelain", relFilePath}
	} else {
		args = []string{"blame", "--porcelain", relFilePath}
	}

	cmd = exec.Command("git", args...)
	cmd.Dir = rootPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute git blame: %w", err)
	}

	// 解析git blame输出
	return g.parseBlameOutput(output, normalizedPath)
}

// parseBlameOutput 解析git blame命令的输出
func (g *CmdGitBlamer) parseBlameOutput(output []byte, filePath string) (*BlameInfo, error) {
	scanner := bufio.NewScanner(bytes.NewReader(output))
	blameInfo := &BlameInfo{
		Lines:    make([]LineInfo, 0),
		Authors:  make(map[string]int),
		Dates:    make(map[string]int),
		FilePath: filePath,
	}

	// 正则表达式用于解析commit行
	commitRegex := regexp.MustCompile(`^([0-9a-f]{40}) (\d+) (\d+) (\d+)$`)

	var currentLine LineInfo
	lineNum := 0
	hasContent := false

	for scanner.Scan() {
		line := scanner.Text()

		// 解析commit行
		if matches := commitRegex.FindStringSubmatch(line); matches != nil {
			// 如果已经有一个完整的行信息，添加到结果中
			if hasContent {
				blameInfo.Lines = append(blameInfo.Lines, currentLine)
				blameInfo.Authors[currentLine.Author]++
				dateStr := currentLine.CommitTime.Format("2006-01-02")
				blameInfo.Dates[dateStr]++
			}

			// 开始新的行信息
			lineNum++
			currentLine = LineInfo{
				LineNum:  lineNum,
				CommitID: matches[1][:7], // 只取前7位作为短ID
			}
			hasContent = false
			continue
		}

		// 解析作者信息
		if strings.HasPrefix(line, "author ") {
			currentLine.Author = strings.TrimPrefix(line, "author ")
			continue
		}

		// 解析邮箱信息
		if strings.HasPrefix(line, "author-mail ") {
			email := strings.TrimPrefix(line, "author-mail ")
			// 去除尖括号
			email = strings.TrimPrefix(email, "<")
			email = strings.TrimSuffix(email, ">")
			currentLine.Email = email
			continue
		}

		// 解析提交时间
		if strings.HasPrefix(line, "author-time ") {
			timeStr := strings.TrimPrefix(line, "author-time ")
			timestamp, err := strconv.ParseInt(timeStr, 10, 64)
			if err == nil {
				currentLine.CommitTime = time.Unix(timestamp, 0)
			}
			continue
		}

		// 解析内容行（不以制表符开头的行是元数据，以制表符开头的是内容）
		if strings.HasPrefix(line, "\t") {
			currentLine.Content = strings.TrimPrefix(line, "\t")
			hasContent = true
			continue
		}
	}

	// 添加最后一行
	if hasContent {
		blameInfo.Lines = append(blameInfo.Lines, currentLine)
		blameInfo.Authors[currentLine.Author]++
		dateStr := currentLine.CommitTime.Format("2006-01-02")
		blameInfo.Dates[dateStr]++
	}

	blameInfo.TotalLines = len(blameInfo.Lines)
	return blameInfo, nil
}

// VisitFile 实现NodeVisitor接口，用于文件节点
func (g *CmdGitBlamer) VisitFile(node *project.Node, path string, depth int) error {
	if node.IsDir {
		return nil
	}

	// 应用文件过滤器
	if g.FileFilter != nil && !g.FileFilter(path) {
		return nil
	}

	// 这里不需要实现具体逻辑，因为CmdGitBlamer只是为了兼容GitBlameVisitor接口
	// 实际的文件分析在BlameFile方法中进行
	return nil
}

// VisitDirectory 实现NodeVisitor接口，用于目录节点
func (g *CmdGitBlamer) VisitDirectory(node *project.Node, path string, depth int) error {
	return nil
}

// BlameDirectory 使用命令行git blame分析目录中的所有文件
func (g *CmdGitBlamer) BlameDirectory(p *project.Project, dirPath string) (map[string]*BlameInfo, error) {
	normalizedPath, node, err := g.normalizePath(dirPath)
	if err != nil {
		return nil, err
	}
	if !node.IsDir {
		return nil, fmt.Errorf("%s is not a directory", normalizedPath)
	}

	// 使用访问者模式遍历目录
	// 创建自定义的访问者，而不是使用GitBlameVisitor
	visitor := &CmdGitBlameVisitor{
		Blamer:    g,
		Project:   p,
		Results:   make(map[string]*BlameInfo),
		semaphore: make(chan struct{}, 1), // 使用单线程处理
	}
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

	// 即使有错误也返回结果
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

// BlameProject 使用命令行git blame分析整个项目
func (g *CmdGitBlamer) BlameProject(p *project.Project) (map[string]*BlameInfo, error) {
	// 分析整个项目就是分析根目录
	return g.BlameDirectory(p, "/")
}

// CmdGitBlameVisitor 用于在项目遍历过程中收集blame信息的访问者
type CmdGitBlameVisitor struct {
	Blamer    *CmdGitBlamer
	Project   *project.Project
	Results   map[string]*BlameInfo
	Errors    []error
	semaphore chan struct{}
	mutex     sync.Mutex
	wg        sync.WaitGroup
}

// VisitFile 实现NodeVisitor接口，用于访问文件节点
func (v *CmdGitBlameVisitor) VisitFile(node *project.Node, path string, depth int) error {
	if node.IsDir {
		return nil
	}

	// 应用文件过滤器
	if v.Blamer.FileFilter != nil && !v.Blamer.FileFilter(path) {
		return nil
	}

	// 使用信号量控制并发
	v.semaphore <- struct{}{}
	v.wg.Add(1)

	go func() {
		defer func() {
			<-v.semaphore
			v.wg.Done()
		}()

		// 分析文件
		blameInfo, err := v.Blamer.BlameFile(v.Project, path)
		if err != nil {
			v.mutex.Lock()
			v.Errors = append(v.Errors, fmt.Errorf("分析文件 %s 失败: %v", path, err))
			v.mutex.Unlock()
			return
		}

		// 保存结果
		v.mutex.Lock()
		v.Results[path] = blameInfo
		v.mutex.Unlock()
	}()

	return nil
}

// VisitDirectory 实现NodeVisitor接口，用于访问目录节点
func (v *CmdGitBlameVisitor) VisitDirectory(node *project.Node, path string, depth int) error {
	// 目录节点不需要特殊处理
	return nil
}
