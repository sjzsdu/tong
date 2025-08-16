package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/share"
)

// RegisterWebTools 将Web相关工具注册到MCP服务器
func RegisterWebTools(s *server.MCPServer) {
	if s == nil {
		return
	}

	// web_fetch 工具 - 获取网页内容
	toolFetch := mcp.NewTool(
		"web_fetch",
		mcp.WithDescription("获取指定URL的网页内容"),
		mcp.WithString("url", mcp.Required(), mcp.Description("要获取内容的网页URL")),
		mcp.WithNumber("timeout", mcp.Description("请求超时时间（秒），默认为10秒")),
	)
	hFetch := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return webFetch(ctx, req)
	}
	s.AddTool(toolFetch, hFetch)
	toolHandlers["web_fetch"] = hFetch

	// web_search 工具 - 搜索获取信息
	toolSearch := mcp.NewTool(
		"web_search",
		mcp.WithDescription("通过搜索引擎获取信息"),
		mcp.WithString("query", mcp.Required(), mcp.Description("搜索关键词")),
		mcp.WithString("engine", mcp.Description("搜索引擎，支持：google, bing, baidu, duckduckgo，默认为google")),
		mcp.WithNumber("limit", mcp.Description("返回结果数量，默认为5")),
	)
	hSearch := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return webSearch(ctx, req)
	}
	s.AddTool(toolSearch, hSearch)
	toolHandlers["web_search"] = hSearch
}

// SearchResult 表示搜索结果
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// webFetch 获取网页内容
func webFetch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// parse args (support top-level and nested args, and JSON-string arguments)
	var (
		urlStr  string
		timeout float64 = 10.0
	)

	// try direct first
	if u, err := req.RequireString("url"); err == nil && strings.TrimSpace(u) != "" {
		urlStr = strings.TrimSpace(u)
	}

	// extract argument map
	var argTop map[string]any = helper.GetArgs(req)
	if share.GetDebug() {
		helper.PrintWithLabel("web_fetch", argTop)
	}
	// if Arguments is a JSON string, try unmarshal
	if argTop == nil {
		switch raw := req.Params.Arguments.(type) {
		case string:
			var tmp map[string]any
			if json.Unmarshal([]byte(raw), &tmp) == nil {
				argTop = tmp
			}
		}
	}
	// flatten nested { args: { ... } }
	argUse := argTop
	if argTop != nil {
		if nested, ok := argTop["args"].(map[string]any); ok && nested != nil {
			argUse = nested
		}
	}

	if urlStr == "" && argUse != nil {
		if u, ok := argUse["url"].(string); ok {
			urlStr = strings.TrimSpace(u)
		}
	}
	// timeout from args (string/number tolerant)
	if argUse != nil {
		timeout = helper.GetFloatDefault(argUse, "timeout", timeout)
	}
	if urlStr == "" {
		return mcp.NewToolResultError("missing or invalid url parameter: expected {\"url\": \"https://...\"} or {\"args\": {\"url\": \"...\"}}"), nil
	}

	// normalize URL
	if parsed, perr := url.Parse(urlStr); perr == nil && parsed.Scheme == "" {
		urlStr = "https://" + urlStr
	}

	if share.GetDebug() {
		helper.PrintWithLabel("urlStr", urlStr)
	}

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create request: %v", err)), nil
	}
	// realistic headers
	reqHTTP.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36")
	reqHTTP.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	reqHTTP.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	// 注意：不主动声明 br，避免 Brotli 需要额外处理
	reqHTTP.Header.Set("Accept-Encoding", "gzip, deflate")

	resp, err := client.Do(reqHTTP)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("request failed: %v", err)), nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mcp.NewToolResultError(fmt.Sprintf("HTTP status %d", resp.StatusCode)), nil
	}

	// 使用helper函数读取和解码响应体
	html, err := helper.ReadDecodedBody(resp)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read response body: %v", err)), nil
	}

	// 使用goquery解析HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse HTML: %v", err)), nil
	}

	title := strings.TrimSpace(doc.Find("title").First().Text())

	// 直接使用已解析的doc生成Markdown，避免重复解析HTML
	md := helper.DocumentToMarkdown(doc)

	if share.GetDebug() {
		helper.PrintWithLabel("markdown", md)
	}

	// shorten overly long output
	const maxLen = 12000
	if len(md) > maxLen {
		md = md[:maxLen] + "\n\n...[truncated]"
	}

	out := map[string]interface{}{
		"url":         urlStr,
		"status_code": resp.StatusCode,
		"title":       title,
		"markdown":    strings.TrimSpace(md),
		"headers": map[string]string{
			"content-type":   resp.Header.Get("Content-Type"),
			"content-length": resp.Header.Get("Content-Length"),
		},
	}
	return mcp.NewToolResultText(helper.ToJSONString(out)), nil
}

// webSearch 通过搜索引擎获取信息
func webSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 解析参数
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("missing or invalid query parameter: %v", err)), nil
	}

	// 解析搜索引擎参数，默认为google
	args := helper.GetArgs(req)
	engine := helper.GetStringDefault(args, "engine", "google")

	// 解析结果数量参数，默认为5
	limit := helper.GetIntDefault(args, "limit", 5)

	// 根据不同搜索引擎构建搜索URL
	var searchURL string
	switch engine {
	case "google":
		searchURL = fmt.Sprintf("https://www.googleapis.com/customsearch/v1?key=%s&cx=%s&q=%s&num=%d",
			"API_KEY", "SEARCH_ENGINE_ID", url.QueryEscape(query), limit)
	case "bing":
		searchURL = fmt.Sprintf("https://api.bing.microsoft.com/v7.0/search?q=%s&count=%d",
			url.QueryEscape(query), limit)
	case "baidu":
		searchURL = fmt.Sprintf("https://www.baidu.com/s?wd=%s&rn=%d",
			url.QueryEscape(query), limit)
	case "duckduckgo":
		// DuckDuckGo不提供官方API，这里使用模拟方式
		searchURL = fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json",
			url.QueryEscape(query))
	default:
		return mcp.NewToolResultError(fmt.Sprintf("unsupported search engine: %s", engine)), nil
	}

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	// 发送GET请求
	httpReq, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create request: %v", err)), nil
	}

	// 根据不同搜索引擎设置请求头
	switch engine {
	case "bing":
		httpReq.Header.Set("Ocp-Apim-Subscription-Key", "API_KEY")
	case "baidu", "google", "duckduckgo":
		httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36")
	}

	// 执行请求
	resp, err := client.Do(httpReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("request failed: %v", err)), nil
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("HTTP request failed with status code: %d", resp.StatusCode)), nil
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read response body: %v", err)), nil
	}

	// 解析响应内容并提取搜索结果
	var results []SearchResult
	var parseErr error

	// 根据不同搜索引擎解析响应
	switch engine {
	case "google":
		results, parseErr = parseGoogleSearchResults(body, limit)
	case "bing":
		results, parseErr = parseBingSearchResults(body, limit)
	case "baidu":
		results, parseErr = parseBaiduSearchResults(body, limit)
	case "duckduckgo":
		results, parseErr = parseDuckDuckGoSearchResults(body, limit)
	}

	if parseErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse search results: %v", parseErr)), nil
	}

	// 构建结果
	result := map[string]interface{}{
		"query":   query,
		"engine":  engine,
		"results": results,
	}

	return mcp.NewToolResultText(helper.ToJSONString(result)), nil
}

// parseGoogleSearchResults 解析Google搜索结果
func parseGoogleSearchResults(data []byte, limit int) ([]SearchResult, error) {
	var response struct {
		Items []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"items"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(response.Items))
	for i, item := range response.Items {
		if i >= limit {
			break
		}
		results = append(results, SearchResult{
			Title:   item.Title,
			URL:     item.Link,
			Snippet: item.Snippet,
		})
	}

	return results, nil
}

// parseBingSearchResults 解析Bing搜索结果
func parseBingSearchResults(data []byte, limit int) ([]SearchResult, error) {
	var response struct {
		WebPages struct {
			Value []struct {
				Name    string `json:"name"`
				URL     string `json:"url"`
				Snippet string `json:"snippet"`
			} `json:"value"`
		} `json:"webPages"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(response.WebPages.Value))
	for i, item := range response.WebPages.Value {
		if i >= limit {
			break
		}
		results = append(results, SearchResult{
			Title:   item.Name,
			URL:     item.URL,
			Snippet: item.Snippet,
		})
	}

	return results, nil
}

// parseBaiduSearchResults 解析百度搜索结果
func parseBaiduSearchResults(data []byte, limit int) ([]SearchResult, error) {
	// 百度返回的是HTML，需要解析HTML提取结果
	content := string(data)

	// 定义正则表达式匹配搜索结果
	titleRegex := regexp.MustCompile(`<h3 class="[^"]*"><a[^>]*href="([^"]*)"[^>]*>([^<]*)</a></h3>`)
	snippetRegex := regexp.MustCompile(`<div class="c-abstract">([^<]*)</div>`)

	// 提取标题和链接
	titleMatches := titleRegex.FindAllStringSubmatch(content, -1)
	snippetMatches := snippetRegex.FindAllStringSubmatch(content, -1)

	results := make([]SearchResult, 0)
	for i, match := range titleMatches {
		if i >= limit {
			break
		}

		url := match[1]
		title := match[2]

		// 对于百度搜索结果，URL可能需要进一步处理
		// 实际情况中可能需要进一步跟踪获取真实URL
		url = strings.TrimPrefix(url, "http://www.baidu.com/link?url=")

		// 提取摘要
		snippet := ""
		if i < len(snippetMatches) {
			snippet = snippetMatches[i][1]
		}

		results = append(results, SearchResult{
			Title:   title,
			URL:     url,
			Snippet: snippet,
		})
	}

	return results, nil
}

// parseDuckDuckGoSearchResults 解析DuckDuckGo搜索结果
func parseDuckDuckGoSearchResults(data []byte, limit int) ([]SearchResult, error) {
	var response struct {
		AbstractText  string `json:"AbstractText"`
		AbstractURL   string `json:"AbstractURL"`
		RelatedTopics []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"RelatedTopics"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0)

	// 添加摘要结果
	if response.AbstractText != "" && response.AbstractURL != "" {
		results = append(results, SearchResult{
			Title:   "Abstract",
			URL:     response.AbstractURL,
			Snippet: response.AbstractText,
		})
	}

	// 添加相关主题结果
	for i, topic := range response.RelatedTopics {
		if i >= limit-len(results) {
			break
		}
		results = append(results, SearchResult{
			Title:   topic.Text,
			URL:     topic.FirstURL,
			Snippet: topic.Text,
		})
	}

	return results, nil
}
