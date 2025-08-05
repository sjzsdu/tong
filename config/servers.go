package config

// PopularMCPServers 流行 MCP 服务器配置的映射，键为功能名称，值为 MCPServerConfig
var PopularMCPServers = map[string]MCPServerConfig{
	// 文件系统访问
	"filesystem": {
		Disabled:      false,
		Timeout:       300,
		Command:       "npx",
		Args:          []string{"-y", "@modelcontextprotocol/server-filesystem"},
		Env:           []string{},
		TransportType: "stdio",
		Url:           "",
		AutoApprove:   []string{},
	},
	// 版本控制
	"git": {
		Disabled:      false,
		Timeout:       300,
		Command:       "uvx",
		Args:          []string{"mcp-server-git"},
		Env:           []string{},
		TransportType: "stdio",
		Url:           "",
		AutoApprove:   []string{},
	},
	// 代码托管平台
	"github": {
		Disabled:      false,
		Timeout:       300,
		Command:       "npx",
		Args:          []string{"-y", "@modelcontextprotocol/server-github"},
		Env:           []string{"GITHUB_PERSONAL_ACCESS_TOKEN=your_token"},
		TransportType: "stdio",
		Url:           "",
		AutoApprove:   []string{},
	},
	// 数据库
	"postgresql": {
		Disabled:      false,
		Timeout:       300,
		Command:       "npx",
		Args:          []string{"-y", "@modelcontextprotocol/server-postgresql"},
		Env:           []string{"DATABASE_URL=your_db_url"},
		TransportType: "stdio",
		Url:           "",
		AutoApprove:   []string{},
	},
	// 社交平台
	"slack": {
		Disabled:      false,
		Timeout:       300,
		Command:       "npx",
		Args:          []string{"-y", "@modelcontextprotocol/server-slack"},
		Env:           []string{"SLACK_BOT_TOKEN=your_token"},
		TransportType: "stdio",
		Url:           "",
		AutoApprove:   []string{},
	},
	// 浏览器自动化
	"playwright": {
		Disabled:      false,
		Timeout:       300,
		Command:       "npx",
		Args:          []string{"-y", "@executeautomation/playwright-mcp-server"},
		Env:           []string{},
		TransportType: "stdio",
		Url:           "",
		AutoApprove:   []string{},
	},
	// 网页搜索
	"web-search": {
		Disabled:      false,
		Timeout:       300,
		Command:       "npx",
		Args:          []string{"-y", "@pskill9/web-search"},
		Env:           []string{},
		TransportType: "stdio",
		Url:           "",
		AutoApprove:   []string{},
	},
	// YouTube 视频处理
	"youtube": {
		Disabled:      false,
		Timeout:       300,
		Command:       "npx",
		Args:          []string{"-y", "@kimtaeyoon83/mcp-server-youtube-transcript"},
		Env:           []string{},
		TransportType: "stdio",
		Url:           "",
		AutoApprove:   []string{},
	},
	// 聚合器 - 将多个 MCP 服务器聚合为一个
	"aggregator": {
		Disabled:      false,
		Timeout:       300,
		Command:       "npx",
		Args:          []string{"-y", "1mcp/agent"},
		Env:           []string{},
		TransportType: "stdio",
		Url:           "",
		AutoApprove:   []string{},
	},
	// B站内容搜索
	"bilibili": {
		Disabled:      false,
		Timeout:       300,
		Command:       "npx",
		Args:          []string{"-y", "@34892002/bilibili-mcp-js"},
		Env:           []string{},
		TransportType: "stdio",
		Url:           "",
		AutoApprove:   []string{},
	},
	// 网站数据提取
	"web-extract": {
		Disabled:      false,
		Timeout:       300,
		Command:       "npx",
		Args:          []string{"-y", "@getrupt/ashra-mcp"},
		Env:           []string{},
		TransportType: "stdio",
		Url:           "",
		AutoApprove:   []string{},
	},
	// Apple 提醒事项集成
	"apple-reminders": {
		Disabled:      false,
		Timeout:       300,
		Command:       "npx",
		Args:          []string{"-y", "@fradser/mcp-server-apple-reminders"},
		Env:           []string{},
		TransportType: "stdio",
		Url:           "",
		AutoApprove:   []string{},
	},
	// 八字排盘
	"bazi": {
		Disabled:      false,
		Timeout:       300,
		Command:       "npx",
		Args:          []string{"-y", "cantian-ai/bazi-mcp"},
		Env:           []string{},
		TransportType: "stdio",
		Url:           "",
		AutoApprove:   []string{},
	},
}
