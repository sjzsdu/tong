package mcpserver

import (
	"context"
	"fmt"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sjzsdu/tong/config"
	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/share"
)

// SearchResult 表示搜索结果
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

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
		mcp.WithNumber("limit", mcp.Description("返回结果数量")),
	)
	hSearch := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return webSearch(ctx, req)
	}
	s.AddTool(toolSearch, hSearch)
	toolHandlers["web_search"] = hSearch
}

// webFetch 获取网页内容
func webFetch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		urlStr  string
		timeout float64 = 10.0
	)

	// 使用辅助函数获取url参数
	url, found := helper.GetStringFromRequest(req, "url", "")
	if !found {
		return mcp.NewToolResultError("missing or invalid url parameter: required argument \"url\" not found"), nil
	}
	urlStr = strings.TrimSpace(url)

	// 使用辅助函数获取timeout参数
	timeoutVal, found := helper.GetFloatFromRequest(req, "timeout", timeout)
	if found {
		timeout = timeoutVal
	}

	// normalize URL
	if parsed, perr := neturl.Parse(urlStr); perr == nil && parsed.Scheme == "" {
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
	if share.GetDebug() {
		helper.PrintWithLabel("web_search request", req)
	}

	// 使用辅助函数获取query参数
	query, found := helper.GetStringFromRequest(req, "query", "")
	if !found {
		return mcp.NewToolResultError("missing or invalid query parameter: required argument \"query\" not found"), nil
	}

	// 使用辅助函数获取limit参数，默认为5
	limit, _ := helper.GetIntFromRequest(req, "limit", 5)
	
	// 使用辅助函数获取engine参数
	userEngine, _ := helper.GetStringFromRequest(req, "engine", "")

	// 从配置中获取API密钥
	googleApiKey := config.GetConfig(config.KeyGoogleAPIKey)
	googleSearchEngineId := config.GetConfig(config.KeyGoogleSearchEngineID)
	bingApiKey := config.GetConfig(config.KeyBingAPIKey)

	// 确定可用的搜索引擎列表
	availableEngines := make(map[string]bool)

	// 百度API可用性检查
	baiduApiKey := config.GetConfig(config.KeyBaiduAPIKey)
	if baiduApiKey != "" {
		availableEngines["baidu"] = true
	}

	// Google需要API密钥和搜索引擎ID
	if googleApiKey != "" && googleSearchEngineId != "" {
		availableEngines["google"] = true
	}

	// Bing需要API密钥
	if bingApiKey != "" {
		availableEngines["bing"] = true
	}

	// 从配置中获取搜索引擎优先级
	enginePriority := strings.Split(config.GetConfigWithDefault(config.KeySearchEngines, "google,baidu,bing"), ",")

	// 如果用户明确指定了搜索引擎，且该引擎可用，则使用该引擎
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

		// 如果没有找到可用的搜索引擎，返回错误
		if engine == "" {
			return mcp.NewToolResultError("没有配置可用的搜索引擎API密钥"), nil
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
			return mcp.NewToolResultError("Google搜索需要API密钥和搜索引擎ID"), nil
		}
	case "bing":
		if bingApiKey != "" {
			// 使用Bing API
			result, searchErr = searchWithBingAPI(ctx, query, limit, bingApiKey)
		} else {
			return mcp.NewToolResultError("Bing搜索需要API密钥"), nil
		}
	case "baidu":
		// 使用百度API
		if baiduApiKey != "" {
			result, searchErr = searchWithBaiduAPI(ctx, query, limit, baiduApiKey)
		} else {
			return mcp.NewToolResultError("百度搜索需要API密钥和Secret Key"), nil
		}
	default:
		return mcp.NewToolResultError(fmt.Sprintf("不支持的搜索引擎: %s", engine)), nil
	}

	// 处理错误
	if searchErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("搜索失败: %v", searchErr)), nil
	}

	return result, nil
}
