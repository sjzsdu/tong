package project

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"

	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/project"
	"github.com/spf13/cobra"
)

//go:embed templates/*.html
var templateFS embed.FS

var (
	markdownServer *http.Server
	serverMutex    sync.Mutex
	serverPort     int = 8080
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
	MarkdownCommand.Flags().IntVarP(&serverPort, "port", "p", 8080, "服务端口")
}

// runMarkdownServer 启动markdown服务器
func runMarkdownServer() {
	serverMutex.Lock()
	defer serverMutex.Unlock()

	if markdownServer != nil {
		fmt.Printf("Markdown服务已在端口 %d 运行\n", serverPort)
		return
	}

	// 获取当前项目
	currentDir, _ := os.Getwd()
	proj, err := project.BuildProjectTree(currentDir, helper.WalkDirOptions{})
	if err != nil {
		fmt.Printf("加载项目失败: %v\n", err)
		return
	}

	// 设置路由
	mux := http.NewServeMux()

	// 首页 - 文件列表
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			handleMarkdownList(w, r, proj)
		} else {
			http.NotFound(w, r)
		}
	})

	// 查看markdown文件
	mux.HandleFunc("/view/", func(w http.ResponseWriter, r *http.Request) {
		handleMarkdownView(w, r, proj)
	})

	// 原始markdown内容
	mux.HandleFunc("/raw/", func(w http.ResponseWriter, r *http.Request) {
		handleMarkdownRaw(w, r, proj)
	})

	// 启动服务器
	markdownServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", serverPort),
		Handler: mux,
	}

	fmt.Printf("正在启动Markdown文档服务，端口: %d\n", serverPort)
	fmt.Printf("Markdown文档服务已启动: http://localhost:%d\n", serverPort)
	fmt.Println("按 Ctrl+C 停止服务...")

	// 自动打开浏览器
	go openBrowser(fmt.Sprintf("http://localhost:%d", serverPort))

	// 启动服务器
	if err := markdownServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("服务器启动失败: %v\n", err)
		markdownServer = nil
	}
}

// handleMarkdownList 处理markdown文件列表页面
func handleMarkdownList(w http.ResponseWriter, r *http.Request, proj *project.Project) {
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
func handleMarkdownView(w http.ResponseWriter, r *http.Request, proj *project.Project) {
	// 从URL中提取文件路径
	filePath := strings.TrimPrefix(r.URL.Path, "/view")
	if filePath == "" || filePath == "/" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// 查找文件节点
	node, err := proj.FindNode(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("文件不存在: %v", err), http.StatusNotFound)
		return
	}

	// 读取文件内容
	content, err := node.ReadContent()
	if err != nil {
		http.Error(w, fmt.Sprintf("读取文件失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 处理 Markdown 内容，修复 Mermaid 图表中的语法问题
	processedContent := sanitizeMarkdownForMermaid(string(content))

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
func handleMarkdownRaw(w http.ResponseWriter, r *http.Request, proj *project.Project) {
	// 从URL中提取文件路径
	filePath := strings.TrimPrefix(r.URL.Path, "/raw")
	if filePath == "" || filePath == "/" {
		http.Error(w, "文件路径不能为空", http.StatusBadRequest)
		return
	}

	// 查找文件节点
	node, err := proj.FindNode(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("文件不存在: %v", err), http.StatusNotFound)
		return
	}

	// 读取文件内容
	content, err := node.ReadContent()
	if err != nil {
		http.Error(w, fmt.Sprintf("读取文件失败: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
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
		if !node.IsDir && strings.HasSuffix(strings.ToLower(node.Name), ".md") {
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

// openBrowser 在默认浏览器中打开URL
func openBrowser(url string) {
	cmd := exec.Command("open", url)
	if err := cmd.Run(); err != nil {
		fmt.Printf("无法打开浏览器: %v\n", err)
	}
}
