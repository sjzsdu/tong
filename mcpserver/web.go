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

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sjzsdu/tong/helper"
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
	// 解析参数
	urlStr, err := req.RequireString("url")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("missing or invalid url parameter: %v", err)), nil
	}

	// 解析timeout参数，默认为10秒
	args := helper.GetArgs(req)
	timeout := helper.GetFloatDefault(args, "timeout", 10.0)

	// 验证URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid URL: %v", err)), nil
	}

	// 确保URL包含协议
	if parsedURL.Scheme == "" {
		urlStr = "https://" + urlStr
	}

	// 创建具有超时的HTTP客户端
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// 发送GET请求
	httpReq, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create request: %v", err)), nil
	}

	// 设置用户代理
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36")

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

	// 构建结果
	result := map[string]interface{}{
		"url":         urlStr,
		"status_code": resp.StatusCode,
		"content":     string(body),
		"headers":     resp.Header,
	}

	return mcp.NewToolResultText(helper.ToJSONString(result)), nil
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
		if strings.HasPrefix(url, "http://www.baidu.com/link?url=") {
			// 实际情况中可能需要进一步跟踪获取真实URL
			url = strings.TrimPrefix(url, "http://www.baidu.com/link?url=")
		}

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
