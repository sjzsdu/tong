package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/share"
)

// searchWithBaiduAPI 使用百度App Builder Web Search API进行搜索
// 参考文档: https://cloud.baidu.com/doc/AppBuilder/s/amaxd2det
func searchWithBaiduAPI(ctx context.Context, query string, limit int, apiKey string) (*mcp.CallToolResult, error) {
	// 尝试百度千帆AI搜索API
	result, err := searchWithBaiduQianfanAPI(ctx, query, limit, apiKey)
	if err != nil {
		// 如果千帆API失败，可以在这里添加备用的百度搜索API
		fmt.Printf("千帆API搜索失败: %v，尝试使用备用方法\n", err)

		// 目前我们只返回失败信息，未来可以添加备用API
		return nil, err
	}
	return result, nil
}

// searchWithBaiduQianfanAPI 使用百度千帆AI搜索API
func searchWithBaiduQianfanAPI(ctx context.Context, query string, limit int, apiKey string) (*mcp.CallToolResult, error) {
	// 构建API URL
	searchURL := "https://qianfan.baidubce.com/v2/ai_search/chat/completions"

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 30 * time.Second, // 增加超时时间到30秒
	}

	// 构建请求数据 - 根据百度千帆API文档修改
	requestData := map[string]interface{}{
		"messages": []map[string]string{
			{
				"content": query,
				"role":    "user",
			},
		},
		"search_source": "baidu_search_v2",
		"resource_type_filter": []map[string]interface{}{
			{
				"type":  "web",
				"top_k": limit,
			},
		},
		// 可选参数
		"search_filter": map[string]interface{}{
			"match": map[string]interface{}{
				"site": []string{}, // 可以指定搜索特定网站
			},
		},
		// 时间过滤，默认不限制
		"search_recency_filter": "year",
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("序列化请求数据失败: %v", err)
	}

	// 发送POST请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", searchURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头 - 尝试几种不同的认证方式
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// 方式1: 使用 API Key 作为 Authorization
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	// 执行请求
	resp, err := client.Do(httpReq)
	if err != nil {
		// 更详细的错误处理
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return nil, fmt.Errorf("请求百度API超时 (30秒): %v，请检查网络连接或稍后重试", err)
		}
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应内容失败: %v", err)
	}

	if share.GetDebug() {
		helper.PrintWithLabel("Baidu API Response", string(body))
	} else {
		// 即使不在调试模式，也打印响应以便查看问题
		fmt.Println("百度API响应:", string(body))
	}

	// 解析结果
	results, err := parseBaiduAPISearchResults(body, limit)
	if err != nil {
		return nil, fmt.Errorf("解析搜索结果失败: %v", err)
	}

	// 构建结果
	result := map[string]interface{}{
		"query":   query,
		"engine":  "baidu",
		"results": results,
	}

	return mcp.NewToolResultText(helper.ToJSONString(result)), nil
}

// parseBaiduAPISearchResults 解析百度API搜索结果
func parseBaiduAPISearchResults(data []byte, limit int) ([]SearchResult, error) {
	// 先尝试通用的 JSON 解析，以便检查实际结构
	var genericResponse map[string]interface{}
	if err := json.Unmarshal(data, &genericResponse); err != nil {
		return nil, fmt.Errorf("无法解析JSON响应: %v", err)
	}

	// 打印解析后的响应结构
	fmt.Printf("百度API响应结构: %+v\n", genericResponse)

	// 检查响应中是否有references字段 - 千帆AI搜索结构
	results := []SearchResult{}

	// 检查并解析千帆API的响应格式
	if refs, ok := genericResponse["references"].([]interface{}); ok {
		for i, ref := range refs {
			if i >= limit {
				break
			}
			if refMap, ok := ref.(map[string]interface{}); ok {
				title, _ := refMap["title"].(string)
				url, _ := refMap["url"].(string)
				content, _ := refMap["content"].(string)
				results = append(results, SearchResult{
					Title:   title,
					URL:     url,
					Snippet: content,
				})
			}
		}
		return results, nil
	}

	// 尝试查找搜索结果 - 处理可能的结构
	// 方式1: 检查 search.web 路径
	if search, ok := genericResponse["search"].(map[string]interface{}); ok {
		if web, ok := search["web"].([]interface{}); ok {
			for i, item := range web {
				if i >= limit {
					break
				}
				if itemMap, ok := item.(map[string]interface{}); ok {
					title, _ := itemMap["title"].(string)
					url, _ := itemMap["url"].(string)
					snippet, _ := itemMap["snippet"].(string)
					results = append(results, SearchResult{
						Title:   title,
						URL:     url,
						Snippet: snippet,
					})
				}
			}
			if len(results) > 0 {
				return results, nil
			}
		}
	}

	// 方式2: 检查 result.result 路径
	if result, ok := genericResponse["result"].(map[string]interface{}); ok {
		if resultItems, ok := result["result"].([]interface{}); ok {
			for i, item := range resultItems {
				if i >= limit {
					break
				}
				if itemMap, ok := item.(map[string]interface{}); ok {
					title, _ := itemMap["title"].(string)
					url, _ := itemMap["url"].(string)
					snippet, _ := itemMap["snippet"].(string)
					results = append(results, SearchResult{
						Title:   title,
						URL:     url,
						Snippet: snippet,
					})
				}
			}
			if len(results) > 0 {
				return results, nil
			}
		}
	}

	// 方式3: 检查其他可能的数据结构
	if len(results) == 0 {
		// 检查错误信息
		if err, ok := genericResponse["error"].(map[string]interface{}); ok {
			message, _ := err["message"].(string)
			code, _ := err["code"].(string)
			if message != "" {
				return nil, fmt.Errorf("API错误: %s (代码: %s)", message, code)
			}
		}

		// 如果没有找到任何结果但也没有错误，返回空结果集
		fmt.Println("警告: 百度API返回了成功响应，但无法解析出任何搜索结果")
	}

	return results, nil
}
