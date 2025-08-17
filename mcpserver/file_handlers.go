package mcpserver

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/project"
	prjsearch "github.com/sjzsdu/tong/project/search"
	prjtree "github.com/sjzsdu/tong/project/tree"
)

// 时间格式常量，用于时间格式化
const timeLayout = "2006-01-02 15:04:05"

// ensureParentDirs 确保文件路径的父目录在项目与磁盘中就绪（逐级创建缺失目录）
func ensureParentDirs(proj *project.Project, path string) error {
	p := proj.NormalizePath(path)
	// 提取父目录路径
	dir := "/"
	if idx := strings.LastIndex(p, "/"); idx > 0 {
		dir = p[:idx]
	}
	if dir == "/" {
		return nil
	}
	parts := strings.Split(strings.TrimPrefix(dir, "/"), "/")
	cur := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		cur += "/" + part
		if _, err := proj.FindNode(cur); err != nil {
			if err2 := proj.CreateDir(cur); err2 != nil {
				return err2
			}
		}
	}
	return nil
}

func fsList(ctx context.Context, proj *project.Project, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dir, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError("missing or invalid path parameter: required argument \"path\" not found"), nil
	}

	maxDepth := req.GetInt("maxDepth", 1)
	includeFiles := req.GetBool("includeFiles", true)
	includeDirs := req.GetBool("includeDirs", false)
	includeHidden := req.GetBool("includeHidden", false)

	dir = proj.NormalizePath(dir)
	n, err := proj.FindNode(dir)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("目录不存在: %s", dir)), nil
	}
	if !n.IsDir {
		return mcp.NewToolResultError("指定路径不是目录"), nil
	}

	opts := prjsearch.DefaultSearchOptions()
	opts.IncludeFiles = includeFiles
	opts.IncludeDirs = includeDirs
	opts.IncludeHidden = includeHidden
	opts.MaxDepth = maxDepth

	matched, err := prjsearch.Search(ctx, n, opts)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	sort.Slice(matched, func(i, j int) bool { return matched[i].Path < matched[j].Path })
	type itemT struct {
		Name    string `json:"name"`
		Path    string `json:"path"`
		IsDir   bool   `json:"isDir"`
		Size    int64  `json:"size,omitempty"`
		ModTime string `json:"modTime,omitempty"`
	}
	items := make([]itemT, 0, len(matched))
	for _, m := range matched {
		it := itemT{Name: m.Name, Path: m.Path, IsDir: m.IsDir}
		if m.Info != nil {
			it.Size = m.Info.Size()
			it.ModTime = m.Info.ModTime().Format(timeLayout)
		}
		items = append(items, it)
	}
	return mcp.NewToolResultText(helper.ToJSON(map[string]any{"dir": dir, "items": items})), nil
}

func fsRead(ctx context.Context, proj *project.Project, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError("missing or invalid path parameter: required argument \"path\" not found"), nil
	}
	p = proj.NormalizePath(p)
	n, err := proj.FindNode(p)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("文件不存在: %s", p)), nil
	}
	if n.IsDir {
		return mcp.NewToolResultError("不能读取目录"), nil
	}
	b, err := n.ReadContent()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

func fsWrite(ctx context.Context, proj *project.Project, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	content, err := req.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	p = proj.NormalizePath(p)
	// 确保父目录存在
	if err := ensureParentDirs(proj, p); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := proj.WriteFile(p, []byte(content)); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	n, _ := proj.FindNode(p)
	res := map[string]any{"path": p}
	if n != nil && n.Info != nil {
		res["size"] = n.Info.Size()
		res["modTime"] = n.Info.ModTime().Format(helper.TimeLayout)
	}
	return mcp.NewToolResultText(helper.ToJSON(res)), nil
}

func fsCreateFile(ctx context.Context, proj *project.Project, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	p = proj.NormalizePath(p)
	// 先确保父目录存在
	if err := ensureParentDirs(proj, p); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	content := req.GetString("content", "")
	var werr error
	if content == "" {
		werr = proj.CreateFileNode(p)
	} else {
		werr = proj.CreateFileWithContent(p, []byte(content))
	}
	if werr != nil {
		return mcp.NewToolResultError(werr.Error()), nil
	}
	return mcp.NewToolResultText(helper.ToJSON(map[string]any{"path": p, "created": true})), nil
}

func fsCreateDir(ctx context.Context, proj *project.Project, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	p = proj.NormalizePath(p)
	if err := proj.CreateDir(p); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(helper.ToJSON(map[string]any{"path": p, "created": true})), nil
}

func fsDelete(ctx context.Context, proj *project.Project, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	p = proj.NormalizePath(p)
	if err := proj.DeleteNode(p); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	abs := proj.GetAbsolutePath(strings.TrimPrefix(p, "/"))
	if stat, statErr := os.Stat(abs); statErr == nil {
		if stat.IsDir() {
			_ = os.RemoveAll(abs)
		} else {
			_ = os.Remove(abs)
		}
	}
	return mcp.NewToolResultText(helper.ToJSON(map[string]any{"path": p, "deleted": true})), nil
}

func fsTree(ctx context.Context, proj *project.Project, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	p = proj.NormalizePath(p)
	n, err := proj.FindNode(p)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("路径不存在: %s", p)), nil
	}
	if !n.IsDir {
		return mcp.NewToolResultError("fs_tree 仅支持目录"), nil
	}
	showFiles := req.GetBool("showFiles", true)
	showHidden := req.GetBool("showHidden", false)
	maxDepth := req.GetInt("maxDepth", 0)
	txt := prjtree.TreeWithOptions(n, showFiles, showHidden, maxDepth)
	return mcp.NewToolResultText(txt), nil
}

func fsSearch(ctx context.Context, proj *project.Project, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	p = proj.NormalizePath(p)
	n, err := proj.FindNode(p)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("路径不存在: %s", p)), nil
	}

	opts := prjsearch.DefaultSearchOptions()
	opts.NameContains = req.GetString("nameContains", "")
	opts.NameRegex = req.GetString("nameRegex", "")
	opts.ContentContains = req.GetString("contentContains", "")
	opts.ContentRegex = req.GetString("contentRegex", "")
	extStr := req.GetString("extensions", "")
	if strings.TrimSpace(extStr) != "" {
		parts := strings.Split(extStr, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		opts.Extensions = parts
	}
	opts.IncludeHidden = req.GetBool("includeHidden", false)
	opts.IncludeDirs = req.GetBool("includeDirs", false)
	opts.IncludeFiles = req.GetBool("includeFiles", true)
	opts.CaseInsensitive = req.GetBool("caseInsensitive", true)
	opts.MatchAny = req.GetBool("matchAny", false)
	opts.MaxDepth = req.GetInt("maxDepth", 0)

	matched, err := prjsearch.Search(ctx, n, opts)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	paths := make([]string, 0, len(matched))
	for _, m := range matched {
		paths = append(paths, m.Path)
	}
	return mcp.NewToolResultText(helper.ToJSON(map[string]any{"count": len(paths), "paths": paths})), nil
}

func fsStat(ctx context.Context, proj *project.Project, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	p = proj.NormalizePath(p)
	n, err := proj.FindNode(p)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("路径不存在: %s", p)), nil
	}
	info := map[string]any{"name": n.Name, "path": n.Path, "isDir": n.IsDir}
	if n.Info != nil {
		info["size"] = n.Info.Size()
		info["mode"] = n.Info.Mode().String()
		info["modTime"] = n.Info.ModTime().Format(helper.TimeLayout)
	} else {
		abs := proj.GetAbsolutePath(strings.TrimPrefix(p, "/"))
		if st, e := os.Stat(abs); e == nil {
			info["size"] = st.Size()
			info["mode"] = st.Mode().String()
			info["modTime"] = st.ModTime().Format(helper.TimeLayout)
		}
	}
	if req.GetBool("hash", false) {
		h, herr := n.CalculateHash()
		if herr == nil {
			info["hash"] = h
		}
	}
	return mcp.NewToolResultText(helper.ToJSON(info)), nil
}

func fsHash(ctx context.Context, proj *project.Project, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	p = proj.NormalizePath(p)
	n, err := proj.FindNode(p)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("路径不存在: %s", p)), nil
	}
	h, err := n.CalculateHash()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(helper.ToJSON(map[string]any{"path": p, "hash": h})), nil
}

func fsSave(_ context.Context, proj *project.Project, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := proj.SaveToFS(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText("saved"), nil
}

func fsSync(_ context.Context, proj *project.Project, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := proj.SyncFromFS(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText("synced"), nil
}
