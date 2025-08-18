package config

// PopularMCPServers 流行 MCP 服务器配置的映射，键为功能名称，值为 MCPServerConfig
var PopularMCPServers = map[string]MCPServerConfig{
	// 版本控制
	"git": {
		Timeout:       30,
		Command:       "uvx",
		Args:          []string{"mcp-server-git"},
		TransportType: "stdio",
	},
	// 代码托管平台
	"github": {
		Timeout:       30,
		Command:       "npx",
		Args:          []string{"-y", "@modelcontextprotocol/server-github"},
		Env:           []string{"GITHUB_PERSONAL_ACCESS_TOKEN=your_token"},
		TransportType: "stdio",
	},
	// 数据库
	"postgresql": {
		Timeout:       30,
		Command:       "npx",
		Args:          []string{"-y", "@modelcontextprotocol/server-postgresql"},
		Env:           []string{"DATABASE_URL=your_db_url"},
		TransportType: "stdio",
	},
	// 浏览器自动化
	"playwright": {
		Timeout:       30,
		Command:       "npx",
		Args:          []string{"-y", "@executeautomation/playwright-mcp-server"},
		TransportType: "stdio",
	},
}
