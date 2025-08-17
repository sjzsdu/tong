package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	mcppkg "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	pkgmcp "github.com/sjzsdu/tong/mcp"
	"github.com/sjzsdu/tong/project"
)

// getToolHandler uses reflection to extract a tool handler from server.MCPServer for testing
func getToolHandler(t *testing.T, s *server.MCPServer, name string) func(context.Context, mcppkg.CallToolRequest) (*mcppkg.CallToolResult, error) {
	// Prefer package-level handlers when available
	if toolHandlers != nil {
		if h, ok := toolHandlers[name]; ok {
			return func(ctx context.Context, req mcppkg.CallToolRequest) (*mcppkg.CallToolResult, error) {
				return h(ctx, req)
			}
		}
	}

	t.Helper()
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		t.Fatalf("unexpected server value kind: %v", v.Kind())
	}
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		ft := f.Type()
		if f.Kind() == reflect.Map && ft.Key().Kind() == reflect.String && ft.Elem().Kind() == reflect.Func {
			// try find the func by name and signature
			key := reflect.ValueOf(name)
			val := f.MapIndex(key)
			if !val.IsValid() {
				continue
			}
			fnType := val.Type()
			// expect func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
			if fnType.NumIn() == 2 && fnType.In(0).String() == "context.Context" && strings.HasSuffix(fnType.In(1).String(), ".CallToolRequest") && fnType.NumOut() == 2 {
				return func(ctx context.Context, req mcppkg.CallToolRequest) (*mcppkg.CallToolResult, error) {
					outs := val.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(req)})
					res, _ := outs[0].Interface().(*mcppkg.CallToolResult)
					var err error
					if !outs[1].IsNil() {
						err, _ = outs[1].Interface().(error)
					}
					return res, err
				}
			}
		}
	}
	t.Fatalf("handler for tool %s not found", name)
	return nil
}

func textFromResult(t *testing.T, r *mcppkg.CallToolResult) string {
	t.Helper()
	if r == nil {
		return ""
	}
	for _, c := range r.Content {
		if tc, ok := c.(mcppkg.TextContent); ok {
			return tc.Text
		}
	}
	return fmt.Sprintf("%v", r.Result)
}

func newTestServerAndProject(t *testing.T) (*server.MCPServer, *project.Project) {
	t.Helper()
	root := t.TempDir()
	proj := project.NewProject(root)
	if err := proj.SyncFromFS(); err != nil {
		t.Fatalf("sync project: %v", err)
	}
	s, err := NewTongMCPServer(proj)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	return s, proj
}

func TestFSCreateReadWriteDelete(t *testing.T) {
	ctx := context.Background()
	s, proj := newTestServerAndProject(t)

	// create dir
	hCreateDir := getToolHandler(t, s, "fs_create_dir")
	res, err := hCreateDir(ctx, pkgmcp.NewToolCallRequest("fs_create_dir", map[string]interface{}{"path": "/a"}))
	if err != nil {
		t.Fatalf("create dir err: %v", err)
	}
	_ = textFromResult(t, res)
	if _, err := proj.FindNode("/a"); err != nil {
		t.Fatalf("dir not created: %v", err)
	}

	// create file with content
	hCreateFile := getToolHandler(t, s, "fs_create_file")
	res, err = hCreateFile(ctx, pkgmcp.NewToolCallRequest("fs_create_file", map[string]interface{}{"path": "/a/hello.txt", "content": "hi"}))
	if err != nil {
		t.Fatalf("create file err: %v", err)
	}
	if _, err := proj.FindNode("/a/hello.txt"); err != nil {
		t.Fatalf("file not created: %v", err)
	}

	// read
	hRead := getToolHandler(t, s, "fs_read")
	res, err = hRead(ctx, pkgmcp.NewToolCallRequest("fs_read", map[string]interface{}{"path": "/a/hello.txt"}))
	if err != nil {
		t.Fatalf("read err: %v", err)
	}
	if got := textFromResult(t, res); got != "hi" {
		t.Fatalf("read content mismatch: %q", got)
	}

	// write
	hWrite := getToolHandler(t, s, "fs_write")
	res, err = hWrite(ctx, pkgmcp.NewToolCallRequest("fs_write", map[string]interface{}{"path": "/a/hello.txt", "content": "world"}))
	if err != nil {
		t.Fatalf("write err: %v", err)
	}
	res, err = hRead(ctx, pkgmcp.NewToolCallRequest("fs_read", map[string]interface{}{"path": "/a/hello.txt"}))
	if err != nil {
		t.Fatalf("read after write err: %v", err)
	}
	if got := textFromResult(t, res); got != "world" {
		t.Fatalf("write content mismatch: %q", got)
	}

	// delete
	hDelete := getToolHandler(t, s, "fs_delete")
	_, err = hDelete(ctx, pkgmcp.NewToolCallRequest("fs_delete", map[string]interface{}{"path": "/a/hello.txt"}))
	if err != nil {
		t.Fatalf("delete err: %v", err)
	}
	if _, err := proj.FindNode("/a/hello.txt"); err == nil {
		t.Fatalf("file still exists after delete")
	}
}

func TestFSListTreeSearchStatHash(t *testing.T) {
	ctx := context.Background()
	s, _ := newTestServerAndProject(t)

	// prepare files
	hCreateDir := getToolHandler(t, s, "fs_create_dir")
	_, _ = hCreateDir(ctx, pkgmcp.NewToolCallRequest("fs_create_dir", map[string]interface{}{"path": "/b"}))
	hCreateFile := getToolHandler(t, s, "fs_create_file")
	_, _ = hCreateFile(ctx, pkgmcp.NewToolCallRequest("fs_create_file", map[string]interface{}{"path": "/b/readme.md", "content": "hello mcp"}))
	_, _ = hCreateFile(ctx, pkgmcp.NewToolCallRequest("fs_create_file", map[string]interface{}{"path": "/b/main.go", "content": "package main"}))

	// list
	hList := getToolHandler(t, s, "fs_list")
	res, err := hList(ctx, pkgmcp.NewToolCallRequest("fs_list", map[string]interface{}{"path": "/b", "maxDepth": 1, "includeFiles": true, "includeDirs": false}))
	if err != nil {
		t.Fatalf("list err: %v", err)
	}
	var listObj map[string]interface{}
	_ = json.Unmarshal([]byte(textFromResult(t, res)), &listObj)
	if listObj["dir"].(string) != "/b" {
		t.Fatalf("list dir mismatch")
	}

	// tree
	hTree := getToolHandler(t, s, "fs_tree")
	res, err = hTree(ctx, pkgmcp.NewToolCallRequest("fs_tree", map[string]interface{}{"path": "/b", "showFiles": true}))
	if err != nil {
		t.Fatalf("tree err: %v", err)
	}
	treeTxt := textFromResult(t, res)
	if !strings.Contains(treeTxt, "readme.md") || !strings.Contains(treeTxt, "main.go") {
		t.Fatalf("tree missing entries: %s", treeTxt)
	}

	// search by nameContains
	hSearch := getToolHandler(t, s, "fs_search")
	res, err = hSearch(ctx, pkgmcp.NewToolCallRequest("fs_search", map[string]interface{}{"path": "/b", "nameContains": "readme", "includeFiles": true}))
	if err != nil {
		t.Fatalf("search err: %v", err)
	}
	var searchObj map[string]interface{}
	_ = json.Unmarshal([]byte(textFromResult(t, res)), &searchObj)
	if int(searchObj["count"].(float64)) < 1 {
		t.Fatalf("expected search count >= 1")
	}

	// stat + hash
	hStat := getToolHandler(t, s, "fs_stat")
	res, err = hStat(ctx, pkgmcp.NewToolCallRequest("fs_stat", map[string]interface{}{"path": "/b/readme.md", "hash": true}))
	if err != nil {
		t.Fatalf("stat err: %v", err)
	}
	var statObj map[string]interface{}
	_ = json.Unmarshal([]byte(textFromResult(t, res)), &statObj)
	if statObj["isDir"].(bool) {
		t.Fatalf("stat expected file, got dir")
	}
	if _, ok := statObj["hash"].(string); !ok {
		t.Fatalf("expected hash in stat")
	}
}

func TestFSSaveAndSync(t *testing.T) {
	ctx := context.Background()
	s, proj := newTestServerAndProject(t)

	// create and save
	hCreateFile := getToolHandler(t, s, "fs_create_file")
	_, _ = hCreateFile(ctx, pkgmcp.NewToolCallRequest("fs_create_file", map[string]interface{}{"path": "/c/x.txt", "content": "x"}))
	hSave := getToolHandler(t, s, "fs_save")
	if _, err := hSave(ctx, pkgmcp.NewToolCallRequest("fs_save", map[string]interface{}{})); err != nil {
		t.Fatalf("save err: %v", err)
	}

	// ensure file exists on disk
	abs := filepath.Join(proj.GetRootPath(), "/c/x.txt")
	abs = strings.ReplaceAll(abs, "//", "/")
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("file not on disk after save: %v", err)
	}

	// modify using write then save
	hWrite := getToolHandler(t, s, "fs_write")
	_, _ = hWrite(ctx, pkgmcp.NewToolCallRequest("fs_write", map[string]interface{}{"path": "/c/x.txt", "content": "y"}))
	_, _ = hSave(ctx, pkgmcp.NewToolCallRequest("fs_save", map[string]interface{}{}))

	// sync from fs to ensure consistent state
	hSync := getToolHandler(t, s, "fs_sync")
	if _, err := hSync(ctx, pkgmcp.NewToolCallRequest("fs_sync", map[string]interface{}{})); err != nil {
		t.Fatalf("sync err: %v", err)
	}
}
