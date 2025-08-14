package mcpserver

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sjzsdu/tong/project"
)

var toolHandlers map[string]func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)

// RegisterFileTools 将文件系统相关工具注册到 MCP 服务器
func RegisterFileTools(s *server.MCPServer, proj *project.Project) {
	if s == nil || proj == nil {
		return
	}
	toolHandlers = make(map[string]func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error))
	// fs_list
	toolList := mcp.NewTool(
		"fs_list",
		mcp.WithDescription("列出目录内容，支持最大深度、是否包含文件/目录、是否包含隐藏项"),
		mcp.WithString("dir", mcp.Required(), mcp.Description("目录路径，如 / 或 /src")),
		mcp.WithNumber("maxDepth", mcp.Description("最大深度（0 表示不限制；1 表示仅当前目录）")),
		mcp.WithBoolean("includeFiles", mcp.Description("结果是否包含文件，默认 true")),
		mcp.WithBoolean("includeDirs", mcp.Description("结果是否包含目录，默认 false")),
		mcp.WithBoolean("includeHidden", mcp.Description("是否包含隐藏文件/目录，默认 false")),
	)
	hList := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { return fsList(ctx, proj, req) }
	s.AddTool(toolList, hList)
	toolHandlers["fs_list"] = hList

	// fs_read
	toolRead := mcp.NewTool(
		"fs_read",
		mcp.WithDescription("读取文件内容（文本）"),
		mcp.WithString("path", mcp.Required(), mcp.Description("文件路径，如 /README.md")),
	)
	hRead := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { return fsRead(ctx, proj, req) }
	s.AddTool(toolRead, hRead)
	toolHandlers["fs_read"] = hRead

	// fs_write
	toolWrite := mcp.NewTool(
		"fs_write",
		mcp.WithDescription("写入文件内容；若文件不存在则创建"),
		mcp.WithString("path", mcp.Required(), mcp.Description("文件路径")),
		mcp.WithString("content", mcp.Required(), mcp.Description("要写入的文本内容")),
	)
	hWrite := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { return fsWrite(ctx, proj, req) }
	s.AddTool(toolWrite, hWrite)
	toolHandlers["fs_write"] = hWrite

	// fs_create_file
	toolCreateFile := mcp.NewTool(
		"fs_create_file",
		mcp.WithDescription("创建文件，可选初始内容"),
		mcp.WithString("path", mcp.Required(), mcp.Description("文件路径")),
		mcp.WithString("content", mcp.Description("初始文本内容，可选")),
	)
	hCreateFile := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { return fsCreateFile(ctx, proj, req) }
	s.AddTool(toolCreateFile, hCreateFile)
	toolHandlers["fs_create_file"] = hCreateFile

	// fs_create_dir
	toolCreateDir := mcp.NewTool(
		"fs_create_dir",
		mcp.WithDescription("创建目录（递归创建父目录）"),
		mcp.WithString("path", mcp.Required(), mcp.Description("目录路径")),
	)
	hCreateDir := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { return fsCreateDir(ctx, proj, req) }
	s.AddTool(toolCreateDir, hCreateDir)
	toolHandlers["fs_create_dir"] = hCreateDir

	// fs_delete
	toolDelete := mcp.NewTool(
		"fs_delete",
		mcp.WithDescription("删除文件或目录（会从项目与磁盘中移除）"),
		mcp.WithString("path", mcp.Required(), mcp.Description("要删除的路径")),
	)
	hDelete := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { return fsDelete(ctx, proj, req) }
	s.AddTool(toolDelete, hDelete)
	toolHandlers["fs_delete"] = hDelete

	// fs_tree
	toolTree := mcp.NewTool(
		"fs_tree",
		mcp.WithDescription("输出目录的树形结构（文本）"),
		mcp.WithString("path", mcp.Required(), mcp.Description("目录路径")),
		mcp.WithBoolean("showFiles", mcp.Description("是否显示文件，默认 true")),
		mcp.WithBoolean("showHidden", mcp.Description("是否显示隐藏项，默认 false")),
		mcp.WithNumber("maxDepth", mcp.Description("最大深度（0 表示不限制）")),
	)
	hTree := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { return fsTree(ctx, proj, req) }
	s.AddTool(toolTree, hTree)
	toolHandlers["fs_tree"] = hTree

	// fs_search
	toolSearch := mcp.NewTool(
		"fs_search",
		mcp.WithDescription("搜索名称与/或内容，支持正则、扩展名、深度、隐藏项等"),
		mcp.WithString("path", mcp.Required(), mcp.Description("搜索根路径")),
		mcp.WithString("nameContains", mcp.Description("名称包含的子串，可选")),
		mcp.WithString("nameRegex", mcp.Description("名称正则，可选")),
		mcp.WithString("contentContains", mcp.Description("内容包含的子串，可选")),
		mcp.WithString("contentRegex", mcp.Description("内容正则，可选")),
		mcp.WithString("extensions", mcp.Description("扩展名列表（逗号分隔，如: go,md 或 *）")),
		mcp.WithBoolean("includeHidden", mcp.Description("包含隐藏项，默认 false")),
		mcp.WithBoolean("includeDirs", mcp.Description("返回目录结果，默认 false")),
		mcp.WithBoolean("includeFiles", mcp.Description("返回文件结果，默认 true")),
		mcp.WithBoolean("caseInsensitive", mcp.Description("大小写不敏感，默认 true")),
		mcp.WithBoolean("matchAny", mcp.Description("名称或内容任一匹配即命中，默认 false(与逻辑)")),
		mcp.WithNumber("maxDepth", mcp.Description("最大深度，0 不限")),
	)
	hSearch := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { return fsSearch(ctx, proj, req) }
	s.AddTool(toolSearch, hSearch)
	toolHandlers["fs_search"] = hSearch

	// fs_stat
	toolStat := mcp.NewTool(
		"fs_stat",
		mcp.WithDescription("获取文件/目录的元信息，支持可选内容哈希"),
		mcp.WithString("path", mcp.Required(), mcp.Description("路径")),
		mcp.WithBoolean("hash", mcp.Description("是否计算文件/目录哈希，默认 false")),
	)
	hStat := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { return fsStat(ctx, proj, req) }
	s.AddTool(toolStat, hStat)
	toolHandlers["fs_stat"] = hStat

	// fs_hash
	toolHash := mcp.NewTool(
		"fs_hash",
		mcp.WithDescription("计算文件或目录的内容哈希（目录为结构哈希）"),
		mcp.WithString("path", mcp.Required(), mcp.Description("路径")),
	)
	hHash := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { return fsHash(ctx, proj, req) }
	s.AddTool(toolHash, hHash)
	toolHandlers["fs_hash"] = hHash

	// fs_save
	toolSave := mcp.NewTool(
		"fs_save",
		mcp.WithDescription("将项目的内存状态保存到磁盘（确保目录/新文件落盘）"),
	)
	hSave := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { return fsSave(ctx, proj, req) }
	s.AddTool(toolSave, hSave)
	toolHandlers["fs_save"] = hSave

	// fs_sync
	toolSync := mcp.NewTool(
		"fs_sync",
		mcp.WithDescription("从磁盘同步文件树到内存（会重建节点映射）"),
	)
	hSync := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { return fsSync(ctx, proj, req) }
	s.AddTool(toolSync, hSync)
	toolHandlers["fs_sync"] = hSync
}
