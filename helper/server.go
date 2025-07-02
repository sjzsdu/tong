package helper

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"os/exec"
	"sync"
)

var (
	content   string
	contentMu sync.RWMutex
	server    *http.Server
	etag      string // Add etag variable
)

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Blog Preview</title>
    <link rel="stylesheet" href="https://cdn.bootcdn.net/ajax/libs/github-markdown-css/5.2.0/github-markdown.min.css">
<script src="https://cdn.bootcdn.net/ajax/libs/markdown-it/12.0.4/markdown-it.min.js"></script>
<script src="https://cdn.bootcdn.net/ajax/libs/markdown-it-emoji/2.0.0/markdown-it-emoji.min.js"></script>
<script src="https://cdn.bootcdn.net/ajax/libs/markdown-it-footnote/3.0.3/markdown-it-footnote.min.js"></script>
<script src="https://cdn.bootcdn.net/ajax/libs/markdown-it-task-lists/2.1.1/markdown-it-task-lists.min.js"></script>
<script src="https://cdn.bootcdn.net/ajax/libs/markdown-it-anchor/8.4.1/markdown-it-anchor.min.js"></script>
<script src="https://cdn.bootcdn.net/ajax/libs/markdown-it-toc-done-right/4.1.0/markdown-it-toc-done-right.min.js"></script>
<script src="https://cdn.bootcdn.net/ajax/libs/mermaid/10.2.0/mermaid.min.js"></script>
<script src="https://cdn.bootcdn.net/ajax/libs/KaTeX/0.16.4/katex.min.js"></script>
<script src="https://cdn.bootcdn.net/ajax/libs/markdown-it-texmath/0.9.0/texmath.min.js"></script>
<link rel="stylesheet" href="https://cdn.bootcdn.net/ajax/libs/KaTeX/0.16.4/katex.min.css">
    <style>
        body {
            box-sizing: border-box;
            min-width: 200px;
            max-width: 980px;
            margin: 0 auto;
            padding: 45px;
            background-color: #f6f8fa;
        }
        .markdown-body {
            background-color: white;
            border-radius: 6px;
            padding: 20px;
            margin: 20px 0;
            box-shadow: 0 1px 3px rgba(0,0,0,0.12);
        }
        pre {
            background-color: #f6f8fa;
            border-radius: 6px;
            padding: 16px;
        }
    </style>
</head>
<body>
    <div class="markdown-body" id="content"></div>
    <script>
        // 初始化 markdown-it
        const md = window.markdownit({
            html: true,
            linkify: true,
            typographer: true,
            breaks: true,
            highlight: function (str, lang) {
                if (lang && lang === 'mermaid') {
                    return '<div class="mermaid">' + str + '</div>';
                }
                return '';
            }
        });

        // 确保插件已经加载后再使用
        if (window.markdownitEmoji) {
            md.use(window.markdownitEmoji);
        }
        if (window.markdownitFootnote) {
            md.use(window.markdownitFootnote);
        }
        if (window.markdownitTaskLists) {
            md.use(window.markdownitTaskLists);
        }
        if (window.markdownitAnchor) {
            md.use(window.markdownitAnchor);
        }
        if (window.markdownitTocDoneRight) {
            md.use(window.markdownitTocDoneRight);
        }
        if (window.texmath && window.katex) {
            md.use(window.texmath, { engine: window.katex });
        }

        // 初始化 Mermaid
        mermaid.initialize({
            startOnLoad: true,
            theme: 'default',
            securityLevel: 'loose',
            flowchart: {
                useMaxWidth: true,
                htmlLabels: true,
                curve: 'basis'
            }
        });

        // 渲染内容
        function renderContent(markdown) {
            const contentDiv = document.getElementById('content');
            
            // 渲染 Markdown
            contentDiv.innerHTML = md.render(markdown);
            
            // 重新初始化 Mermaid
            mermaid.init(undefined, document.querySelectorAll('.mermaid'));
        }

        let lastEtag = '';

        // 自动刷新内容
        function refreshContent() {
            fetch('/content', {
                headers: {
                    'If-None-Match': lastEtag
                }
            })
            .then(response => {
                if (response.status === 304) {
                    return null;
                }
                lastEtag = response.headers.get('ETag') || '';
                return response.text();
            })
            .then(newContent => {
                if (newContent) {
                    renderContent(newContent);
                }
            })
            .catch(error => {
                console.error('Failed to fetch content:', error);
            });
        }

        // 初始化内容
        refreshContent();

        // 每秒刷新一次
        setInterval(refreshContent, 1000);
    </script>
</body>
</html>
`

func StartPreviewServer(port int) string {
	tmpl := template.Must(template.New("preview").Parse(htmlTemplate))

	// 处理基础 HTML 页面请求
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, nil)
	})

	// 处理内容请求
	http.HandleFunc("/content", func(w http.ResponseWriter, r *http.Request) {
		contentMu.RLock()
		currentContent := content
		currentEtag := etag
		contentMu.RUnlock()

		// 检查内容是否有变化
		if r.Header.Get("If-None-Match") == currentEtag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("ETag", currentEtag)
		w.Write([]byte(currentContent))
	})

	server = &http.Server{
		Addr: fmt.Sprintf(":%d", port),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	url := fmt.Sprintf("http://localhost:%d", port)
	// 在默认浏览器中打开 URL
	go func() {
		cmd := exec.Command("open", url)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Failed to open browser: %v\n", err)
		}
	}()
	// OSC 8 格式：\033]8;;URL\007TEXT\033]8;;\007
	return fmt.Sprintf("\033]8;;%s\007%s\033]8;;\007", url, url)
}

func UpdatePreviewContent(newContent string) {
	contentMu.Lock()
	content = newContent
	// Generate MD5 hash as ETag
	hash := md5.Sum([]byte(newContent))
	etag = hex.EncodeToString(hash[:])
	contentMu.Unlock()
}

// 添加关闭服务器的函数
func StopPreviewServer() error {
	if server != nil {
		return server.Close()
	}
	return nil
}
