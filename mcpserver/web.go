package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sjzsdu/tong/config"
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
		mcp.WithDescription("通过搜索引擎获取信息，优先级由SEARCH_ENGINES配置决定"),
		mcp.WithString("query", mcp.Required(), mcp.Description("搜索关键词")),
		mcp.WithString("engine", mcp.Description("可选：指定搜索引擎，支持：google, bing, baidu, duckduckgo。若未指定则按SEARCH_ENGINES配置的优先级自动选择")),
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

	// 解析参数
	args := helper.GetArgs(req)

	// 解析结果数量参数，默认为5
	limit := helper.GetIntDefault(args, "limit", 5)

	// 从配置中获取API密钥
	googleApiKey := config.GetConfig("GOOGLE_API_KEY")
	googleSearchEngineId := config.GetConfig("GOOGLE_SEARCH_ENGINE_ID")
	bingApiKey := config.GetConfig("BING_API_KEY")

	// 确定可用的搜索引擎列表
	availableEngines := make(map[string]bool)

	// 百度始终可用（通过网页抓取）
	availableEngines["baidu"] = true

	// Google需要API密钥和搜索引擎ID
	if googleApiKey != "" && googleSearchEngineId != "" {
		availableEngines["google"] = true
	}

	// Bing需要API密钥
	if bingApiKey != "" {
		availableEngines["bing"] = true
	}

	// DuckDuckGo始终可用（通过网页抓取）
	availableEngines["duckduckgo"] = true

	// 从配置中获取搜索引擎优先级
	enginePriority := strings.Split(config.GetConfigWithDefault("SEARCH_ENGINES", "google,baidu,bing,duckduckgo"), ",")

	// 如果用户明确指定了搜索引擎，且该引擎可用，则使用该引擎
	userEngine := helper.GetStringDefault(args, "engine", "")
	var engine string

	if userEngine != "" && availableEngines[userEngine] {
		engine = userEngine
	} else {
		// 根据优先级选择第一个可用的搜索引擎
		for _, e := range enginePriority {
			e = strings.TrimSpace(e)
			if availableEngines[e] {
				engine = e
				break
			}
		}

		// 如果没有找到可用的搜索引擎，默认使用百度（通过网页抓取）
		if engine == "" {
			engine = "baidu"
		}
	}

	if share.GetDebug() {
		helper.PrintWithLabel("selected_engine", engine)
	}

	// 根据选择的搜索引擎调用相应的处理函数
	var result *mcp.CallToolResult
	var searchErr error

	switch engine {
	case "google":
		if googleApiKey != "" && googleSearchEngineId != "" {
			// 使用Google API
			result, searchErr = searchWithGoogleAPI(ctx, query, limit, googleApiKey, googleSearchEngineId)
		} else {
			// 使用网页抓取
			result, searchErr = searchWithWebFetch(ctx, "google", query, limit)
		}
	case "bing":
		if bingApiKey != "" {
			// 使用Bing API
			result, searchErr = searchWithBingAPI(ctx, query, limit, bingApiKey)
		} else {
			// 使用网页抓取
			result, searchErr = searchWithWebFetch(ctx, "bing", query, limit)
		}
	case "baidu":
		// 百度始终使用网页抓取
		result, searchErr = searchWithWebFetch(ctx, "baidu", query, limit)
	case "duckduckgo":
		// DuckDuckGo始终使用网页抓取
		result, searchErr = searchWithWebFetch(ctx, "duckduckgo", query, limit)
	default:
		return mcp.NewToolResultError(fmt.Sprintf("不支持的搜索引擎: %s", engine)), nil
	}

	// 处理错误
	if searchErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("搜索失败: %v", searchErr)), nil
	}

	return result, nil
}

// searchWithWebFetch 使用webFetch工具抓取搜索引擎结果页面
func searchWithWebFetch(ctx context.Context, engine string, query string, limit int) (*mcp.CallToolResult, error) {
	// 构建搜索URL
	var searchURL string
	switch engine {
	case "google":
		searchURL = fmt.Sprintf("https://www.google.com/search?q=%s&num=%d",
			url.QueryEscape(query), limit)
	case "bing":
		searchURL = fmt.Sprintf("https://www.bing.com/search?q=%s&count=%d",
			url.QueryEscape(query), limit)
	case "baidu":
		searchURL = fmt.Sprintf("https://www.baidu.com/s?wd=%s&rn=%d",
			url.QueryEscape(query), limit)
	case "duckduckgo":
		searchURL = fmt.Sprintf("https://duckduckgo.com/?q=%s",
			url.QueryEscape(query))
	}

	// 使用现有的webFetch工具获取搜索结果页面
	mockReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"url": searchURL,
			},
		},
	}

	// 调用webFetch
	return webFetch(ctx, mockReq)
}

// searchWithGoogleAPI 使用Google Custom Search API进行搜索
func searchWithGoogleAPI(ctx context.Context, query string, limit int, apiKey string, searchEngineId string) (*mcp.CallToolResult, error) {
	// 构建API URL
	searchURL := fmt.Sprintf("https://www.googleapis.com/customsearch/v1?key=%s&cx=%s&q=%s&num=%d",
		apiKey, searchEngineId, url.QueryEscape(query), limit)

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	// 发送GET请求
	httpReq, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36")

	// 执行请求
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP请求失败，状态码: %d", resp.StatusCode)
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应内容失败: %v", err)
	}

	// 解析结果
	results, err := parseGoogleSearchResults(body, limit)
	if err != nil {
		return nil, fmt.Errorf("解析搜索结果失败: %v", err)
	}

	// 构建结果
	result := map[string]interface{}{
		"query":   query,
		"engine":  "google",
		"results": results,
	}

	return mcp.NewToolResultText(helper.ToJSONString(result)), nil
}

// searchWithBingAPI 使用Bing Search API进行搜索
func searchWithBingAPI(ctx context.Context, query string, limit int, apiKey string) (*mcp.CallToolResult, error) {
	// 构建API URL
	searchURL := fmt.Sprintf("https://api.bing.microsoft.com/v7.0/search?q=%s&count=%d",
		url.QueryEscape(query), limit)

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	// 发送GET请求
	httpReq, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	httpReq.Header.Set("Ocp-Apim-Subscription-Key", apiKey)

	// 执行请求
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP请求失败，状态码: %d", resp.StatusCode)
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应内容失败: %v", err)
	}

	// 解析结果
	results, err := parseBingSearchResults(body, limit)
	if err != nil {
		return nil, fmt.Errorf("解析搜索结果失败: %v", err)
	}

	// 构建结果
	result := map[string]interface{}{
		"query":   query,
		"engine":  "bing",
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
// 注意：目前通过webFetch直接获取并展示百度搜索页面，此函数保留以备将来使用
func parseBaiduSearchResults(data []byte, limit int) ([]SearchResult, error) {
	// 使用goquery解析HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0)
	// 百度搜索结果容器
	doc.Find(".result, .c-container").Each(func(i int, s *goquery.Selection) {
		if i >= limit {
			return
		}

		// 提取标题和URL
		titleEl := s.Find("h3 a")
		title := strings.TrimSpace(titleEl.Text())
		url, _ := titleEl.Attr("href")

		// 提取摘要
		snippetEl := s.Find(".c-abstract, .content-right_8Zs40")
		if snippetEl.Length() == 0 {
			snippetEl = s.Find(".c-span-last")
		}
		snippet := strings.TrimSpace(snippetEl.Text())

		// 添加结果
		results = append(results, SearchResult{
			Title:   title,
			URL:     url,
			Snippet: snippet,
		})
	})

	// 如果没有找到结果，尝试其他选择器
	if len(results) == 0 {
		doc.Find("div[id^='content_']").Each(func(i int, s *goquery.Selection) {
			if i >= limit {
				return
			}

			// 提取标题
			title := strings.TrimSpace(s.Find(".t").Text())

			// 提取URL
			url := ""
			s.Find("a.c-showurl").Each(func(_ int, a *goquery.Selection) {
				href, exists := a.Attr("href")
				if exists {
					url = href
				}
			})
			if url == "" {
				s.Find("a").Each(func(_ int, a *goquery.Selection) {
					href, exists := a.Attr("href")
					if exists && strings.Contains(href, "http") {
						url = href
					}
				})
			}

			// 提取摘要
			snippet := strings.TrimSpace(s.Find(".c-abstract").Text())

			// 添加结果
			if title != "" {
				results = append(results, SearchResult{
					Title:   title,
					URL:     url,
					Snippet: snippet,
				})
			}
		})
	}

	return results, nil
}

// parseDuckDuckGoSearchResults 解析DuckDuckGo搜索结果
// 注意：目前通过webFetch直接获取并展示DuckDuckGo搜索页面，此函数保留以备将来使用
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
