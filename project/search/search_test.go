package search

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sjzsdu/tong/project"
)

// 准备测试用的临时项目结构
func setupTestProject(t *testing.T) (*project.Project, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "search_test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	cleanup := func() { os.RemoveAll(dir) }

	files := map[string]string{
		"README.md":           "# Readme\n",
		"src/main.go":         "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n",
		"src/utils/helper.go": "package utils\n\nfunc Helper() string { return \"helper\" }\n",
		"docs/api.md":         "# API\n",
		"docs/guide.md":       "# Guide\n",
		".gitignore":          "node_modules\n",
		"config.json":         "{}\n",
	}
	for p, c := range files {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(c), 0644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	proj := project.NewProject(dir)
	if err := proj.SyncFromFS(); err != nil {
		t.Fatalf("sync project: %v", err)
	}
	return proj, cleanup
}

func getRootNode(t *testing.T, p *project.Project) *project.Node {
	t.Helper()
	r, err := p.FindNode("/")
	if err != nil || r == nil {
		t.Fatalf("find root: %v", err)
	}
	return r
}

func collectPaths(nodes []*project.Node) []string {
	out := make([]string, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, n.Path)
	}
	return out
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func TestSearch_ByNameContains(t *testing.T) {
	proj, cleanup := setupTestProject(t)
	defer cleanup()
	root := getRootNode(t, proj)

	opts := DefaultSearchOptions()
	opts.NameContains = "readme" // 忽略大小写
	matched, err := Search(context.Background(), root, opts)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	paths := collectPaths(matched)
	if !contains(paths, "/README.md") {
		t.Fatalf("expect /README.md in results, got %v", paths)
	}
}

func TestSearch_ByContentContains(t *testing.T) {
	proj, cleanup := setupTestProject(t)
	defer cleanup()
	root := getRootNode(t, proj)

	opts := DefaultSearchOptions()
	opts.ContentContains = "Hello, World!"
	matched, err := Search(context.Background(), root, opts)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	paths := collectPaths(matched)
	if !contains(paths, "/src/main.go") {
		t.Fatalf("expect /src/main.go in results, got %v", paths)
	}
}

func TestSearch_ExtensionsFilter(t *testing.T) {
	proj, cleanup := setupTestProject(t)
	defer cleanup()
	root := getRootNode(t, proj)

	opts := DefaultSearchOptions()
	opts.Extensions = []string{"md"}
	matched, err := Search(context.Background(), root, opts)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	paths := collectPaths(matched)
	if !contains(paths, "/README.md") || !contains(paths, "/docs/api.md") || !contains(paths, "/docs/guide.md") {
		t.Fatalf("expect md files in results, got %v", paths)
	}
	// 不应包含 .go
	if contains(paths, "/src/main.go") || contains(paths, "/src/utils/helper.go") {
		t.Fatalf("unexpected go files in results: %v", paths)
	}
}

func TestSearch_HiddenFiles(t *testing.T) {
	proj, cleanup := setupTestProject(t)
	defer cleanup()
	root := getRootNode(t, proj)

	opts := DefaultSearchOptions()
	opts.NameRegex = ".*gitignore$"
	// 默认不包含隐藏
	matched, err := Search(context.Background(), root, opts)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(matched) != 0 {
		t.Fatalf("expected 0 results without hidden, got %d", len(matched))
	}
	// 包含隐藏
	opts.IncludeHidden = true
	matched, err = Search(context.Background(), root, opts)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	paths := collectPaths(matched)
	if !contains(paths, "/.gitignore") {
		t.Fatalf("expect /.gitignore, got %v", paths)
	}
}

func TestSearch_IncludeDirsOnly(t *testing.T) {
	proj, cleanup := setupTestProject(t)
	defer cleanup()
	root := getRootNode(t, proj)

	opts := DefaultSearchOptions()
	opts.IncludeFiles = false
	opts.IncludeDirs = true
	opts.NameContains = "src"
	matched, err := Search(context.Background(), root, opts)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	paths := collectPaths(matched)
	if !contains(paths, "/src") {
		t.Fatalf("expect /src directory, got %v", paths)
	}
}

func TestSearch_DepthLimit(t *testing.T) {
	proj, cleanup := setupTestProject(t)
	defer cleanup()
	root := getRootNode(t, proj)

	opts := DefaultSearchOptions()
	opts.NameRegex = ".*\\.go$"
	opts.MaxDepth = 1
	matched, err := Search(context.Background(), root, opts)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(matched) != 0 {
		t.Fatalf("expect no .go files within depth=1, got %d", len(matched))
	}

	// depth=3 应包含两处 go 文件（/src/main.go 深度2，/src/utils/helper.go 深度3）
	opts.MaxDepth = 3
	matched, err = Search(context.Background(), root, opts)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	paths := collectPaths(matched)
	if !(contains(paths, "/src/main.go") && contains(paths, "/src/utils/helper.go")) {
		t.Fatalf("expect go files at depth=3, got %v", paths)
	}
}

func TestSearch_RegexAndCase(t *testing.T) {
	proj, cleanup := setupTestProject(t)
	defer cleanup()
	root := getRootNode(t, proj)

	opts := DefaultSearchOptions()
	opts.NameRegex = ".*README\\.MD$" // 用正则 + (?i) 由实现自动添加
	opts.CaseInsensitive = true
	matched, err := Search(context.Background(), root, opts)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	paths := collectPaths(matched)
	if !contains(paths, "/README.md") {
		t.Fatalf("expect /README.md matched by case-insensitive regex, got %v", paths)
	}
}
