package project

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/project"
	"github.com/spf13/cobra"
)

//go:embed templates/*.html
var templateFS embed.FS

var (
	markdownServer *http.Server
	serverMutex    sync.Mutex
	serverPort     int = 9595
	// 添加新的全局变量来存储传入的markdown内容
	markdownContent string
	showContentOnly bool
	markdownPattern string
)

// MarkdownCommand markdown子命令
var MarkdownCommand = &cobra.Command{
	Use:   "markdown",
	Short: "启动Markdown文档服务",
	Long:  "启动一个HTTP服务器，用于浏览项目中的Markdown文件",
	Run: func(cmd *cobra.Command, args []string) {
		runMarkdownServer()
	},
}

func init() {
	MarkdownCommand.Flags().IntVarP(&serverPort, "port", "p", 9595, "服务端口")
	// 添加新的命令行参数
	MarkdownCommand.Flags().StringVarP(&markdownContent, "content", "c", "", "直接提供Markdown内容而不是从文件加载")
	MarkdownCommand.Flags().BoolVarP(&showContentOnly, "content-only", "", false, "仅显示提供的Markdown内容，不显示其他文件列表")
	MarkdownCommand.Flags().StringVarP(&markdownPattern, "pattern", "f", "", "使用blob匹配模式筛选Markdown文件，例如: *.md, docs/*.md")
}

// runMarkdownServer 启动markdown服务器
func runMarkdownServer() {
	serverMutex.Lock()
	defer serverMutex.Unlock()

	if markdownServer != nil {
		fmt.Printf("Markdown服务已在端口 %d 运行\n", serverPort)
		return
	}

	// 设置路由
	mux := http.NewServeMux()

	// 无论是否提供了markdown内容，都注册所有路由
	if markdownContent != "" {
		// 如果提供了内容，首页直接显示该内容
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				handleMarkdownContent(w, r)
			} else if r.URL.Path == "/raw-content" {
				// 处理直接提供内容的下载
				handleRawContentDownload(w, r)
			} else {
				http.NotFound(w, r)
			}
		})
	} else {
		// 否则显示文件列表
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				handleMarkdownList(w, r)
			} else {
				http.NotFound(w, r)
			}
		})
	}

	// 专门的文件列表路由，无论是否提供了内容都显示列表
	mux.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		handleMarkdownList(w, r)
	})

	// 查看markdown文件
	mux.HandleFunc("/view/", func(w http.ResponseWriter, r *http.Request) {
		handleMarkdownView(w, r)
	})

	// 原始markdown内容
	mux.HandleFunc("/raw/", func(w http.ResponseWriter, r *http.Request) {
		handleMarkdownRaw(w, r)
	})

	// 本地图片访问
	mux.HandleFunc("/images/", func(w http.ResponseWriter, r *http.Request) {
		handleMarkdownImages(w, r)
	})

	maxPort := serverPort + 20 // 最多尝试20个端口
	var lastErr error
	for port := serverPort; port <= maxPort; port++ {
		markdownServer = &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		}

		fmt.Printf("正在启动Markdown文档服务，端口: %d\n", port)

		// 启动服务器（使用goroutine避免阻塞错误处理）
		serverErr := make(chan error, 1)
		go func() {
			err := markdownServer.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				serverErr <- err
			}
		}()

		// 给服务器一点时间启动，检查是否有端口占用错误
		time.Sleep(100 * time.Millisecond)
		select {
		case err := <-serverErr:
			if strings.Contains(err.Error(), "address already in use") {
				fmt.Printf("端口 %d 已被占用，尝试下一个端口...\n", port)
				lastErr = err
				continue
			} else {
				fmt.Printf("服务器启动失败: %v\n", err)
				markdownServer = nil
				return
			}
		default:
			// 服务器启动成功
			fmt.Printf("Markdown文档服务已启动: http://localhost:%d\n", port)
			fmt.Println("按 Ctrl+C 停止服务...")
			go openBrowser(fmt.Sprintf("http://localhost:%d", port))

			// 等待中断信号以优雅地关闭服务器
			quit := make(chan os.Signal, 1)
			signal.Notify(quit, os.Interrupt)
			<-quit
			fmt.Println("\n正在关闭Markdown文档服务...")
			return
		}
	}
	fmt.Printf("所有端口均不可用，最后错误: %v\n", lastErr)
	markdownServer = nil
}

// getCurrentProject 获取当前最新的项目树
func getCurrentProject() (*project.Project, error) {
	currentDir, _ := os.Getwd()
	return project.BuildProjectTree(currentDir, helper.WalkDirOptions{})
}

// handleMarkdownList 处理markdown文件列表页面
func handleMarkdownList(w http.ResponseWriter, r *http.Request) {
	proj, err := getCurrentProject()
	if err != nil {
		http.Error(w, fmt.Sprintf("加载项目失败: %v", err), http.StatusInternalServerError)
		return
	}
	markdownFiles, err := getMarkdownFiles(proj)
	if err != nil {
		http.Error(w, fmt.Sprintf("获取markdown文件失败: %v", err), http.StatusInternalServerError)
		return
	}

	data := struct {
		Files []MarkdownFile
		Total int
	}{
		Files: markdownFiles,
		Total: len(markdownFiles),
	}

	// 从 embed 文件系统加载模板
	tmplContent, err := templateFS.ReadFile("templates/list.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("模板文件读取失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 创建带有自定义函数的模板
	funcMap := template.FuncMap{
		"div": func(a, b interface{}) float64 {
			var af, bf float64
			switch v := a.(type) {
			case int64:
				af = float64(v)
			case float64:
				af = v
			case int:
				af = float64(v)
			}
			switch v := b.(type) {
			case int64:
				bf = float64(v)
			case float64:
				bf = v
			case int:
				bf = float64(v)
			}
			if bf != 0 {
				return af / bf
			}
			return 0
		},
		"multiply": func(a, b interface{}) int {
			var ai, bi int
			switch v := a.(type) {
			case int:
				ai = v
			case int64:
				ai = int(v)
			case float64:
				ai = int(v)
			}
			switch v := b.(type) {
			case int:
				bi = v
			case int64:
				bi = int(v)
			case float64:
				bi = int(v)
			}
			return ai * bi
		},
	}

	tmpl := template.Must(template.New("list").Funcs(funcMap).Parse(string(tmplContent)))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("模板渲染失败: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleMarkdownView 处理markdown文件查看页面
func handleMarkdownView(w http.ResponseWriter, r *http.Request) {
	// 从URL中提取文件路径
	filePath := strings.TrimPrefix(r.URL.Path, "/view")
	if filePath == "" || filePath == "/" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// 检查是否是通过--content参数提供的文档
	if markdownContent != "" {
		// 获取默认文件名
		defaultFileName := "/document.md"
		// 尝试从内容中提取标题作为文件名
		title, _ := extractTitleAndDescription(markdownContent)
		if title != "" {
			// 将标题转换为有效的文件名
			fileName := strings.ToLower(title)
			fileName = strings.ReplaceAll(fileName, " ", "-")
			// 移除特殊字符
			fileName = regexp.MustCompile(`[^a-z0-9\-]`).ReplaceAllString(fileName, "")
			if fileName != "" {
				defaultFileName = "/" + fileName + ".md"
			}
		}

		// 如果请求的是这个特殊文档
		if filePath == defaultFileName {
			// 处理 Markdown 内容，修复 Mermaid 图表中的语法问题
			processedContent := sanitizeMarkdownForMermaid(markdownContent)

			// 将本地图片引用转换为 /images/ 路径（假设图片在根目录）
			processedContent = convertLocalImagesToServerPath(processedContent, "./")

			// 获取最新项目树
			proj, err := getCurrentProject()
			if err != nil {
				http.Error(w, fmt.Sprintf("加载项目失败: %v", err), http.StatusInternalServerError)
				return
			}

			// 获取所有markdown文件列表
			markdownFiles, err := getMarkdownFiles(proj)
			if err != nil {
				http.Error(w, fmt.Sprintf("获取文件列表失败: %v", err), http.StatusInternalServerError)
				return
			}

			// 从 embed 文件系统加载模板
			tmplContent, err := templateFS.ReadFile("templates/view.html")
			if err != nil {
				http.Error(w, fmt.Sprintf("模板文件读取失败: %v", err), http.StatusInternalServerError)
				return
			}

			tmpl := template.Must(template.New("view").Parse(string(tmplContent)))
			data := struct {
				FilePath      string
				Content       template.HTML
				RawPath       string
				MarkdownFiles []MarkdownFile
			}{
				FilePath:      filePath,
				Content:       template.HTML(processedContent),
				RawPath:       "/raw" + filePath,
				MarkdownFiles: markdownFiles,
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err := tmpl.Execute(w, data); err != nil {
				http.Error(w, fmt.Sprintf("模板渲染失败: %v", err), http.StatusInternalServerError)
				return
			}
			return
		}
	}

	// 获取最新项目树
	proj, err := getCurrentProject()
	if err != nil {
		http.Error(w, fmt.Sprintf("加载项目失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 查找文件节点
	node, err := proj.FindNode(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("文件不存在: %v", err), http.StatusNotFound)
		return
	}

	// 读取文件内容（确保获取最新内容）
	content, err := node.ReadContent()
	if err != nil {
		http.Error(w, fmt.Sprintf("读取文件失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 处理 Markdown 内容，修复 Mermaid 图表中的语法问题
	processedContent := sanitizeMarkdownForMermaid(string(content))

	// 获取当前文件所在目录，用于解析相对图片路径
	currentDir := filepath.Dir(filePath)
	// 将本地图片引用转换为 /images/ 路径
	processedContent = convertLocalImagesToServerPath(processedContent, currentDir)

	// 获取所有markdown文件列表
	markdownFiles, err := getMarkdownFiles(proj)
	if err != nil {
		http.Error(w, fmt.Sprintf("获取文件列表失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 从 embed 文件系统加载模板
	tmplContent, err := templateFS.ReadFile("templates/view.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("模板文件读取失败: %v", err), http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(template.New("view").Parse(string(tmplContent)))
	data := struct {
		FilePath      string
		Content       template.HTML
		RawPath       string
		MarkdownFiles []MarkdownFile
	}{
		FilePath:      filePath,
		Content:       template.HTML(processedContent),
		RawPath:       "/raw" + filePath,
		MarkdownFiles: markdownFiles,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("模板渲染失败: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleMarkdownRaw 处理原始markdown内容
func handleMarkdownRaw(w http.ResponseWriter, r *http.Request) {
	// 从URL中提取文件路径
	filePath := strings.TrimPrefix(r.URL.Path, "/raw")
	if filePath == "" || filePath == "/" {
		http.Error(w, "文件路径不能为空", http.StatusBadRequest)
		return
	}

	// 检查是否是通过--content参数提供的文档
	if markdownContent != "" {
		// 获取默认文件名
		defaultFileName := "/document.md"
		// 尝试从内容中提取标题作为文件名
		title, _ := extractTitleAndDescription(markdownContent)
		if title != "" {
			// 将标题转换为有效的文件名
			fileName := strings.ToLower(title)
			fileName = strings.ReplaceAll(fileName, " ", "-")
			// 移除特殊字符
			fileName = regexp.MustCompile(`[^a-z0-9\-]`).ReplaceAllString(fileName, "")
			if fileName != "" {
				defaultFileName = "/" + fileName + ".md"
			}
		}

		// 如果请求的是这个特殊文档
		if filePath == defaultFileName {
			// 从文件路径中提取文件名
			fileName := filepath.Base(filePath)

			// 设置响应头，支持文件下载
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
			w.Write([]byte(markdownContent))
			return
		}
	}

	// 获取最新项目树
	proj, err := getCurrentProject()
	if err != nil {
		http.Error(w, fmt.Sprintf("加载项目失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 查找文件节点
	node, err := proj.FindNode(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("文件不存在: %v", err), http.StatusNotFound)
		return
	}

	// 读取文件内容（确保获取最新内容）
	content, err := node.ReadContent()
	if err != nil {
		http.Error(w, fmt.Sprintf("读取文件失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 从文件路径中提取文件名
	fileName := filepath.Base(filePath)

	// 确保文件名有.md扩展名
	if !strings.HasSuffix(fileName, ".md") && !strings.HasSuffix(fileName, ".markdown") {
		fileName += ".md"
	}

	// 设置响应头，支持文件下载
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	w.Write(content)
}

// MarkdownFile 表示一个markdown文件的信息
type MarkdownFile struct {
	Path         string
	Name         string
	Size         int64
	RelativePath string
	Title        string // 从 MD 文件中提取的主标题
	Description  string // 从 MD 文件中提取的描述（第一段文字）
}

// getMarkdownFiles 获取项目中所有的markdown文件
func getMarkdownFiles(proj *project.Project) ([]MarkdownFile, error) {
	var markdownFiles []MarkdownFile

	err := proj.Visit(func(path string, node *project.Node, depth int) error {
		// 检查是否是markdown文件
		if !node.IsDir && strings.HasSuffix(strings.ToLower(node.Name), ".md") {
			// 如果指定了模式，检查文件是否匹配
			if markdownPattern != "" {
				// 使用filepath.Match进行blob匹配
				// 支持多种匹配方式：
				// 1. 完整路径匹配：path（如 docs/file.md）
				// 2. 文件名匹配：node.Name（如 file.md）
				// 3. 相对目录匹配：如果模式包含/，则匹配相对路径
				// 4. 递归目录匹配：*.md 匹配所有目录下的md文件

				// 检查模式是否包含路径分隔符
				containsSlash := strings.Contains(markdownPattern, "/")

				// 处理递归目录匹配：*.md 匹配所有目录下的md文件
				wildcardPattern := markdownPattern

				// 初始化匹配结果
				match := false

				// 尝试1: 完整路径匹配（如 docs/file.md）
				match, _ = filepath.Match(wildcardPattern, path)

				// 尝试2: 文件名匹配（如 file.md）
				if !match {
					match, _ = filepath.Match(wildcardPattern, node.Name)
				}

				// 尝试3: 相对目录匹配（如果模式包含斜杠）
				if !match && containsSlash {
					// 提取当前文件的目录路径和文件名
					fileDir := filepath.Dir(path)
					fileBase := filepath.Base(path)

					// 获取模式的目录部分和文件名部分
					patternDir := filepath.Dir(wildcardPattern)
					patternBase := filepath.Base(wildcardPattern)

					// 尝试多种目录匹配方式：
					// 1. 完整路径匹配
					dirMatch, _ := filepath.Match(patternDir, fileDir)
					if dirMatch {
						match, _ = filepath.Match(patternBase, fileBase)
					}

					// 2. 相对路径匹配（去掉前导斜杠）
					if !match {
						relativeFileDir := strings.TrimPrefix(fileDir, "/")
						dirMatch, _ = filepath.Match(patternDir, relativeFileDir)
						if dirMatch {
							match, _ = filepath.Match(patternBase, fileBase)
						}
					}

					// 3. 直接匹配路径的最后一部分
					if !match {
						fileDirLastPart := filepath.Base(fileDir)
						dirMatch, _ = filepath.Match(patternDir, fileDirLastPart)
						if dirMatch {
							match, _ = filepath.Match(patternBase, fileBase)
						}
					}
				}

				// 尝试4: 递归匹配（如果模式是 *.md 或 *.markdown）
				if !match && !containsSlash {
					// 模式不包含斜杠，且是 *.md 或 *.markdown，匹配所有目录下的对应文件
					match, _ = filepath.Match(wildcardPattern, node.Name)
				}

				// 尝试5: 支持 ** 通配符（递归目录匹配）
				if !match && strings.Contains(wildcardPattern, "**") {
					// 简单处理 ** 通配符：替换为 * 并尝试匹配文件名
					simplePattern := strings.ReplaceAll(wildcardPattern, "**", "*")
					match, _ = filepath.Match(simplePattern, node.Name)
				}

				// 如果都不匹配，跳过该文件
				if !match {
					return nil
				}
			}

			file := MarkdownFile{
				Path:         node.Path,
				Name:         node.Name,
				RelativePath: path,
				Size:         0,
			}

			// 尝试获取文件大小
			if node.Info != nil {
				file.Size = node.Info.Size()
			}

			// 读取内容提取标题和描述
			if content, err := node.ReadContent(); err == nil {
				if file.Size == 0 {
					file.Size = int64(len(content))
				}

				// 提取标题和描述
				title, desc := extractTitleAndDescription(string(content))
				file.Title = title
				file.Description = desc
			}

			markdownFiles = append(markdownFiles, file)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 如果提供了markdown内容，将其添加到文件列表
	if markdownContent != "" {
		// 使用默认文件名
		defaultFileName := "document.md"
		// 尝试从内容中提取标题作为文件名
		title, _ := extractTitleAndDescription(markdownContent)
		if title != "" {
			// 将标题转换为有效的文件名
			fileName := strings.ToLower(title)
			fileName = strings.ReplaceAll(fileName, " ", "-")
			// 移除特殊字符
			fileName = regexp.MustCompile(`[^a-z0-9\-]`).ReplaceAllString(fileName, "")
			if fileName != "" {
				defaultFileName = fileName + ".md"
			}
		}

		// 添加到文件列表，确保RelativePath以斜杠开头
		file := MarkdownFile{
			Path:         "/" + defaultFileName,
			Name:         defaultFileName,
			RelativePath: "/" + defaultFileName,
			Size:         int64(len(markdownContent)),
			Title:        title,
		}

		markdownFiles = append(markdownFiles, file)
	}

	// 按路径排序
	sort.Slice(markdownFiles, func(i, j int) bool {
		return markdownFiles[i].RelativePath < markdownFiles[j].RelativePath
	})

	return markdownFiles, nil
}

// extractTitleAndDescription 从 Markdown 内容中提取标题和描述
func extractTitleAndDescription(content string) (title string, description string) {
	lines := strings.Split(content, "\n")
	foundTitle := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 跳过空行
		if trimmed == "" {
			continue
		}

		// 提取第一个 # 标题作为标题
		if !foundTitle && strings.HasPrefix(trimmed, "#") {
			// 去掉 # 符号和空格
			title = strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			foundTitle = true
			continue
		}

		// 提取第一段非空文本作为描述（跳过代码块、引用等）
		if foundTitle && description == "" {
			// 跳过代码块标记
			if strings.HasPrefix(trimmed, "```") {
				continue
			}
			// 跳过引用块
			if strings.HasPrefix(trimmed, ">") {
				trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
			}
			// 跳过列表项
			if strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "+") {
				continue
			}
			// 跳过标题
			if strings.HasPrefix(trimmed, "#") {
				continue
			}

			// 如果是普通文本，作为描述
			if len(trimmed) > 0 {
				description = trimmed
				// 限制描述长度
				if len(description) > 120 {
					description = description[:120] + "..."
				}
				break
			}
		}
	}

	// 如果没有找到标题，使用文件名
	if title == "" {
		title = "未命名文档"
	}

	// 如果没有找到描述
	if description == "" {
		description = "暂无描述"
	}

	return title, description
}

// sanitizeMarkdownForMermaid 处理 Markdown 内容，修复 Mermaid 图表中的语法问题
func sanitizeMarkdownForMermaid(content string) string {
	// 只处理 Mermaid 代码块中的内容
	lines := strings.Split(content, "\n")
	var result strings.Builder
	inMermaidBlock := false
	isClassDiagram := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 检测 Mermaid 代码块的开始
		if strings.HasPrefix(trimmed, "```mermaid") {
			inMermaidBlock = true
			isClassDiagram = false
			result.WriteString(line + "\n")
			continue
		}

		// 检测代码块的结束
		if inMermaidBlock && strings.HasPrefix(trimmed, "```") {
			inMermaidBlock = false
			isClassDiagram = false
			result.WriteString(line + "\n")
			continue
		}

		// 在 Mermaid 代码块内，进行处理
		if inMermaidBlock {
			// 检测是否是类图
			if strings.HasPrefix(trimmed, "classDiagram") {
				isClassDiagram = true
			}

			// 1. 替换 interface{} 为 any（Go 1.18+）
			line = strings.ReplaceAll(line, "interface{}", "any")
			line = strings.ReplaceAll(line, "map[string]interface{}", "map[string]any")
			line = strings.ReplaceAll(line, "[]interface{}", "[]any")
			line = strings.ReplaceAll(line, "...interface{}", "...any")
			line = strings.ReplaceAll(line, "chan interface{}", "chan any")

			// 2. 只在类图中处理类名中的特殊字符
			// Mermaid 类图中的关系符号：*-- (组合), o-- (聚合), --> (关联), ..> (依赖) 等
			// 如果类名以 * 开头（如 *agents.Executor），会与 *-- 冲突
			// 序列图、流程图等使用不同的箭头语法，不应该被处理
			if isClassDiagram {
				// 检查是否包含类图的关系符号
				if strings.Contains(line, "-->") || strings.Contains(line, "<--") ||
					strings.Contains(line, "..|>") || strings.Contains(line, "<|..") ||
					strings.Contains(line, "*--") || strings.Contains(line, "o--") ||
					strings.Contains(line, "--o") || strings.Contains(line, "--*") ||
					strings.Contains(line, "<|--") || strings.Contains(line, "--|>") {
					// 处理指针类型的类名（*ClassName）
					// 将 *Package.Class 改为 Package.Class (去掉前导 *)
					line = sanitizeMermaidClassName(line)
				}
			}
		}

		result.WriteString(line + "\n")
	}

	return result.String()
}

// sanitizeMermaidClassName 清理 Mermaid 类图中的类名特殊字符
func sanitizeMermaidClassName(line string) string {
	// 匹配关系箭头后的类名
	// 支持的关系：-->, <--, .., *--, o--, --o, --* 等

	// 先处理箭头右侧的类名（如：A --> *B）
	patterns := []string{
		"-->", "<--", "..", "*--", "o--", "--o", "--*",
		"<|--", "--|>", "<|..", "..|>",
	}

	for _, pattern := range patterns {
		if !strings.Contains(line, pattern) {
			continue
		}

		parts := strings.Split(line, pattern)
		if len(parts) != 2 {
			continue
		}

		// 处理右侧部分（可能包含类名）
		rightPart := strings.TrimSpace(parts[1])

		// 如果以 * 开头，去掉 *
		if strings.HasPrefix(rightPart, "*") {
			// 找到类名（可能后面还有 : 标签）
			tokens := strings.Fields(rightPart)
			if len(tokens) > 0 && strings.HasPrefix(tokens[0], "*") {
				// 去掉前导 *
				tokens[0] = strings.TrimPrefix(tokens[0], "*")
				rightPart = strings.Join(tokens, " ")
			}
		}

		line = parts[0] + pattern + " " + rightPart
		break // 只处理第一个匹配
	}

	return line
}

// handleMarkdownImages 处理Markdown文档中的本地图片请求
func handleMarkdownImages(w http.ResponseWriter, r *http.Request) {
	// 获取最新项目树
	proj, err := getCurrentProject()
	if err != nil {
		http.Error(w, fmt.Sprintf("加载项目失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 从URL中提取图片文件路径
	// URL格式: /images/[图片路径]
	imagePath := strings.TrimPrefix(r.URL.Path, "/images")
	if imagePath == "" || imagePath == "/" {
		http.Error(w, "图片路径不能为空", http.StatusBadRequest)
		return
	}

	// 查找图片文件节点
	node, err := proj.FindNode(imagePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("图片不存在: %v", err), http.StatusNotFound)
		return
	}

	// 检查是否为文件
	if node.IsDir {
		http.Error(w, "路径不是图片文件", http.StatusBadRequest)
		return
	}

	// 读取图片内容
	content, err := node.ReadContent()
	if err != nil {
		http.Error(w, fmt.Sprintf("读取图片失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 设置正确的Content-Type
	contentType := "application/octet-stream"
	if ext := strings.ToLower(filepath.Ext(node.Name)); ext != "" {
		if mime, ok := mimeTypes[ext[1:]]; ok {
			contentType = mime
		}
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))

	// 返回图片内容
	w.Write(content)
}

// handleMarkdownContent 处理直接提供的markdown内容
func handleMarkdownContent(w http.ResponseWriter, r *http.Request) {
	// 处理 Markdown 内容，修复 Mermaid 图表中的语法问题
	processedContent := sanitizeMarkdownForMermaid(markdownContent)

	// 将本地图片引用转换为 /images/ 路径
	processedContent = convertLocalImagesToServerPath(processedContent, "./")

	// 准备数据
	var markdownFiles []MarkdownFile
	if !showContentOnly {
		// 如果不是仅显示内容，获取所有markdown文件列表
		proj, err := getCurrentProject()
		if err == nil {
			markdownFiles, _ = getMarkdownFiles(proj)
		}
	}

	// 从 embed 文件系统加载模板
	tmplContent, err := templateFS.ReadFile("templates/view.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("模板文件读取失败: %v", err), http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(template.New("view").Parse(string(tmplContent)))
	data := struct {
		FilePath      string
		Content       template.HTML
		RawPath       string
		MarkdownFiles []MarkdownFile
	}{
		FilePath:      "直接提供的内容",
		Content:       template.HTML(processedContent),
		RawPath:       "/raw-content", // 设置一个固定路径用于下载
		MarkdownFiles: markdownFiles,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("模板渲染失败: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleRawContentDownload 处理直接提供的markdown内容的下载
func handleRawContentDownload(w http.ResponseWriter, r *http.Request) {
	// 设置响应头，支持文件下载
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="document.md"`)
	w.Write([]byte(markdownContent))
}

// 常用图片类型的MIME映射
var mimeTypes = map[string]string{
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
	"png":  "image/png",
	"gif":  "image/gif",
	"svg":  "image/svg+xml",
	"bmp":  "image/bmp",
	"webp": "image/webp",
	"ico":  "image/x-icon",
	"tif":  "image/tiff",
	"tiff": "image/tiff",
}

// convertLocalImagesToServerPath 将Markdown文档中的本地图片引用转换为服务器路径
func convertLocalImagesToServerPath(content, currentDir string) string {
	// 使用简单的字符串处理，避免复杂正则表达式
	var result strings.Builder

	// 遍历内容，寻找Markdown图片语法
	for i := 0; i < len(content); i++ {
		// 检查是否是图片开始标记：![
		if i+1 < len(content) && content[i] == '!' && content[i+1] == '[' {
			// 记录当前位置
			start := i
			i += 2 // 跳过![

			// 寻找alt text结束标记：]
			altEnd := strings.Index(content[i:], "]")
			if altEnd == -1 {
				// 不是完整的图片语法，继续
				result.WriteString(content[start:i])
				continue
			}

			altText := content[i : i+altEnd]
			i += altEnd + 1 // 跳过]和(

			// 检查是否是(，如果不是则不是完整的图片语法
			if i >= len(content) || content[i] != '(' {
				result.WriteString(content[start:i])
				continue
			}
			i++ // 跳过(

			// 寻找图片路径结束标记：)
			pathEnd := strings.Index(content[i:], ")")
			if pathEnd == -1 {
				// 不是完整的图片语法，继续
				result.WriteString(content[start:i])
				continue
			}

			imagePath := content[i : i+pathEnd]
			i += pathEnd + 1 // 跳过)

			// 检查是否是HTTP/HTTPS开头的图片，若是则不处理
			if strings.HasPrefix(strings.ToLower(imagePath), "http://") || strings.HasPrefix(strings.ToLower(imagePath), "https://") {
				// 外部图片，保持原样
				result.WriteString(fmt.Sprintf("![%s](%s)", altText, imagePath))
			} else {
				// 清理图片路径，移除可能的查询参数或锚点
				imagePath = strings.Split(imagePath, "?")[0]
				imagePath = strings.Split(imagePath, "#")[0]

				// 解析图片路径相对当前文件目录
				resolvedPath := imagePath
				if !strings.HasPrefix(imagePath, "/") {
					// 相对路径：将图片路径与当前文件目录结合
					resolvedPath = filepath.Join(currentDir, imagePath)
				}
				// 清理路径，处理 .. 和 . segments
				resolvedPath = filepath.Clean(resolvedPath)

				// 确保路径以 / 开头
				if !strings.HasPrefix(resolvedPath, "/") {
					resolvedPath = "/" + resolvedPath
				}

				// 转换为 /images/ 路径
				result.WriteString(fmt.Sprintf("![%s](/images%s)", altText, resolvedPath))
			}
		} else {
			// 不是图片语法，直接写入
			result.WriteByte(content[i])
		}
	}

	return result.String()
}

// openBrowser 在默认浏览器中打开URL
func openBrowser(url string) {
	cmd := exec.Command("open", url)
	if err := cmd.Run(); err != nil {
		fmt.Printf("无法打开浏览器: %v\n", err)
	}
}
