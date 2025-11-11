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
		Content:       template.HTML(content),
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

			// 如果大小为0，尝试读取内容获取大小
			if file.Size == 0 {
				if content, err := node.ReadContent(); err == nil {
					file.Size = int64(len(content))
				}
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

// openBrowser 在默认浏览器中打开URL
func openBrowser(url string) {
	cmd := exec.Command("open", url)
	if err := cmd.Run(); err != nil {
		fmt.Printf("无法打开浏览器: %v\n", err)
	}
}
