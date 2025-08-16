package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sjzsdu/tong/helper"
)

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
