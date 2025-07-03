- [.github](#.github)

## .github

  - [workflows](#workflows)

### workflows

    - [build-linux.yml](#build-linux.yml)

#### build-linux.yml

name: Build and Release Linux

on:
  push:
    tags:
      - 'v*'
  workflow_dispatch:

permissions:
  contents: write  # 这行很重要，明确授予写入内容的权限

jobs:
  build-linux:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.24.0'
    
    - name: Build
      run: go build -v -o tong-linux .
    
    - name: Release
      uses: softprops/action-gh-release@v1
      with:
        files: ./tong-linux
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    - [build-mac.yml](#build-mac.yml)

#### build-mac.yml

name: Build and Release Mac

on:
  push:
    tags:
      - 'v*'
  workflow_dispatch:

permissions:
  contents: write  # 这行很重要，明确授予写入内容的权限

jobs:
  build-mac:
    runs-on: macos-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.24.0'
    
    - name: Verify Go installation
      run: |
        go version
        echo $GOROOT
        echo $PATH
    
    - name: Set Go environment
      run: |
        echo "GOROOT=$(go env GOROOT)" >> $GITHUB_ENV
        echo "$(go env GOROOT)/bin" >> $GITHUB_PATH
    
    - name: Build
      run: |
        echo $GOROOT
        echo $PATH
        go build -v -o tong-mac .
    
    - name: Release
      uses: softprops/action-gh-release@v1
      with:
        files: ./tong-mac
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
- [.gitignore](#.gitignore)

## .gitignore

# 编译后的二进制文件
*.exe
*.exe~
*.dll
*.so
*.dylib

# 测试二进制文件
*.test

# 覆盖率工具生成的输出文件
*.out

# 依赖目录
/vendor/

# Go工作区文件
go.work

# 编译输出目录
/bin/
/pkg/

# 日志文件
*.log

# 操作系统生成的文件
.DS_Store
Thumbs.db

# IDE 和编辑器生成的文件和目录
.idea/
.vscode/
*.swp
*.swo
*~

# 环境变量文件
.env

# 如果使用 air 进行热重载，可能需要忽略以下文件
tmp/

*.pdf

__debug**
tong
# docs  <- remove this line
- [README.md](#readme.md)

## README.md


- [cmd](#cmd)

## cmd

  - [config.go](#config.go)

### config.go

package cmd

import (
	"fmt"

	"github.com/sjzsdu/tong/config"
	"github.com/sjzsdu/tong/lang"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: lang.T("Set config"),
	Long:  lang.T("Set global configuration"),
	Run:   handleConfigCommand,
}

var (
	configOptions = map[string]string{
		"lang":        "Set language",
		"renderer":    "Set llm response render type",
	}
	showAllConfigs bool
)

func init() {
	if config.GetConfig("lang") == "" {
		config.SetConfig("lang", "en")
	}

	rootCmd.AddCommand(configCmd)
	configCmd.Flags().BoolVarP(&showAllConfigs, "list", "l", false, lang.T("List all configurations"))

	// 通过遍历 configOptions 自动添加所有配置项
	for key, desc := range configOptions {
		configCmd.Flags().String(key, config.GetConfig(key), lang.T(desc))
	}
	
	// 添加 provider 配置项
	configCmd.Flags().StringP("provider", "p", config.GetConfig("default_provider"), lang.T("Set default LLM provider"))
}

func handleConfigCommand(cmd *cobra.Command, args []string) {
	if err := config.LoadConfig(); err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	if showAllConfigs {
		fmt.Println(lang.T("Current configurations:"))
		for key := range configOptions {
			value := config.GetConfig(key)
			if value != "" {
				fmt.Printf("%s=%s\n", config.GetEnvKey(key), value)
			}
		}
		return
	}

	configChanged := false
	// 处理 configOptions 中的标准配置项
	for key := range configOptions {
		flag := cmd.Flag(key)
		if flag != nil && flag.Changed {
			value, _ := cmd.Flags().GetString(key)
			config.SetConfig(key, value)
			configChanged = true
		}
	}

	// 特殊处理 provider 标志，将其映射到 llm 配置项
	providerFlag := cmd.Flag("provider")
	if providerFlag != nil && providerFlag.Changed {
		value, err := cmd.Flags().GetString("provider")
		if err == nil {
			envKey := config.GetEnvKey("default_provider")
			if envKey != "" {
				config.SetConfig(envKey, value)
				configChanged = true
			}
		}
	}

	if configChanged {
		if err := config.SaveConfig(); err != nil {
			fmt.Println("Error saving config:", err)
			return
		}
	}
}

  - [pack.go](#pack.go)

### pack.go

package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/project"
	"github.com/sjzsdu/tong/project/output"
	"github.com/spf13/cobra"
)

var packCmd = &cobra.Command{
	Use:   "pack",
	Short: lang.T("Pack files"),
	Long:  lang.T("Pack files with specified extensions into a single output file"),
	Run:   runPack,
}

func init() {
	rootCmd.AddCommand(packCmd)
}

func runPack(cmd *cobra.Command, args []string) {
	if outputFile == "" {
		fmt.Printf("Output is required")
		return
	}
	targetPath, err := helper.GetTargetPath(workDir, repoURL)
	if err != nil {
		fmt.Printf("failed to get target path: %v\n", err)
		return
	}

	options := helper.WalkDirOptions{
		DisableGitIgnore: skipGitIgnore,
		Extensions:       extensions,
		Excludes:         excludePatterns,
	}

	// 构建项目树
	doc, err := project.BuildProjectTree(targetPath, options)
	if err != nil {
		fmt.Printf("failed to build project tree: %v\n", err)
		return
	}

	// 检查项目树是否为空
	if doc.IsEmpty() {
		fmt.Printf("No files to pack\n")
		return
	}

	// 根据输出文件扩展名选择导出格式
	switch filepath.Ext(outputFile) {
	case ".md":
		exporter := output.NewMarkdownExporter(doc)
		err = exporter.Export(outputFile)
	case ".pdf":
		exporter, err := output.NewPDFExporter(doc)
		if err != nil {
			fmt.Printf("Error creating PDF exporter: %v\n", err)
			return
		}
		exporter.Export(outputFile)
	case ".xml":
		exporter := output.NewXMLExporter(doc)
		err = exporter.Export(outputFile)
	default:
		fmt.Printf("Unsupported output format: %s\n", filepath.Ext(outputFile))
		return
	}

	if err != nil {
		fmt.Printf("Error packing files: %v\n", err)
		return
	}

	fmt.Printf("Successfully packed files into %s\n", outputFile)
}

  - [project.go](#project.go)

### project.go

package cmd

import (
	"fmt"

	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/project"
	"github.com/sjzsdu/tong/project/analyzer"
	"github.com/sjzsdu/tong/project/output"
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: lang.T("Project files"),
	Long:  lang.T("Project files with specified extensions into a single output file"),
	Run:   runproject,
}

func init() {
	rootCmd.AddCommand(projectCmd)
}

func runproject(cmd *cobra.Command, args []string) {
	targetPath, err := helper.GetTargetPath(workDir, repoURL)
	if err != nil {
		fmt.Printf("failed to get target path: %v\n", err)
		return
	}

	options := helper.WalkDirOptions{
		DisableGitIgnore: skipGitIgnore,
		Extensions:       extensions,
		Excludes:         excludePatterns,
	}

	// 构建项目树
	doc, err := project.BuildProjectTree(targetPath, options)
	if err != nil {
		fmt.Printf("failed to build project tree: %v\n", err)
		return
	}

	// 检查参数是否存在
	if len(args) == 0 {
		fmt.Println("请指定操作类型: pack, analyze-code, analyze-deps")
		return
	}

	switch args[0] {
	case "pack":
		if outputFile == "" {
			fmt.Printf("Output is required\n")
			return
		}
		if err := output.Output(doc, outputFile); err != nil {
			fmt.Printf("导出失败: %v\n", err)
		} else {
			fmt.Printf("成功导出到: %s\n", outputFile)
		}
	
	case "analyze-code":
		// 使用代码分析器
		codeAnalyzer := analyzer.NewDefaultCodeAnalyzer()
		stats, err := codeAnalyzer.Analyze(doc)
		if err != nil {
			fmt.Printf("代码分析失败: %v\n", err)
			return
		}
		
		// 输出分析结果
		fmt.Println("代码分析结果:")
		fmt.Printf("总文件数: %d\n", stats.TotalFiles)
		fmt.Printf("总目录数: %d\n", stats.TotalDirs)
		fmt.Printf("总代码行数: %d\n", stats.TotalLines)
		fmt.Printf("总大小: %.2f KB\n", float64(stats.TotalSize)/1024)
		
		fmt.Println("\n语言统计:")
		for lang, lines := range stats.LanguageStats {
			fmt.Printf("%s: %d 行\n", lang, lines)
		}
		
		fmt.Println("\n文件类型统计:")
		for ext, count := range stats.FileTypeStats {
			fmt.Printf("%s: %d 个文件\n", ext, count)
		}
	
	case "analyze-deps":
		// 使用依赖分析器
		depsAnalyzer := analyzer.NewDefaultDependencyAnalyzer()
		graph, err := depsAnalyzer.AnalyzeDependencies(doc)
		if err != nil {
			fmt.Printf("依赖分析失败: %v\n", err)
			return
		}
		
		// 输出分析结果
		fmt.Println("依赖分析结果:")
		fmt.Printf("总依赖数: %d\n", len(graph.Nodes))
		
		fmt.Println("\n依赖列表:")
		for name, node := range graph.Nodes {
			if node.Version != "" {
				fmt.Printf("%s: %s (%s)\n", name, node.Version, node.Type)
			} else {
				fmt.Printf("%s (%s)\n", name, node.Type)
			}
		}
		
		fmt.Println("\n依赖关系:")
		for src, dsts := range graph.Edges {
			for _, dst := range dsts {
				fmt.Printf("%s -> %s\n", src, dst)
			}
		}
	
	default:
		fmt.Printf("未知的操作类型: %s\n", args[0])
		fmt.Println("支持的操作: pack, analyze-code, analyze-deps")
	}
}

  - [root.go](#root.go)

### root.go

package cmd

import (
	"fmt"
	"os"

	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/share"
	"github.com/spf13/cobra"
)

var (
	workDir         string
	extensions      []string
	outputFile      string
	excludePatterns []string
	repoURL         string
	skipGitIgnore   bool
	debugMode       bool
)

var RootCmd = rootCmd

var rootCmd = &cobra.Command{
	Use:   share.BUILDNAME,
	Short: lang.T("Tong command line tool"),
	Long:  lang.T("A versatile command line tool for development"),
	// 移除 Args 限制，允许无参数调用
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Run: func(cmd *cobra.Command, args []string) {
		// 如果没有参数，显示帮助信息
		if len(args) == 0 {
			cmd.Help()
			return
		}
		fmt.Fprintln(os.Stderr, lang.T("Invalid arguments")+": ", args)
		os.Exit(1)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// 确保在初始化时已经加载了语言包
	rootCmd.PersistentFlags().StringVarP(&workDir, "directory", "d", ".", lang.T("Work directory path"))
	rootCmd.PersistentFlags().StringSliceVarP(&extensions, "extensions", "e", []string{"*"}, lang.T("File extensions to include"))
	rootCmd.PersistentFlags().StringVarP(&outputFile, "out", "o", "", lang.T("Output file name"))
	rootCmd.PersistentFlags().StringSliceVarP(&excludePatterns, "exclude", "x", []string{}, lang.T("Glob patterns to exclude"))
	rootCmd.PersistentFlags().StringVarP(&repoURL, "repository", "r", "", lang.T("Git repository URL to clone and pack"))
	rootCmd.PersistentFlags().BoolVarP(&skipGitIgnore, "no-gitignore", "n", false, lang.T("Disable .gitignore rules"))
	rootCmd.PersistentFlags().BoolVarP(&debugMode, "debug", "v", false, lang.T("Debug mode"))
	// 设置全局 debug 模式
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		share.SetDebug(debugMode)
	}
}

  - [version.go](#version.go)

### version.go

package cmd

import (
	"fmt"

	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/share"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: lang.T("Print version information"),
	Long:  lang.T("Print detailed version information of tong"),
	Run: func(cmd *cobra.Command, args []string) {
		// 使用简单的字符串拼接替代模板
		fmt.Printf("%s: %s\n", lang.T("tong version"), share.VERSION)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

- [config](#config)

## config

  - [config.go](#config.go)

### config.go

package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/share"
)

var configMap map[string]string

func init() {
	configMap = make(map[string]string)
	if err := LoadConfig(); err == nil {
		for key, value := range configMap {
			os.Setenv(key, value)
		}
	}
}

func GetConfig(key string) string {
	envKey := key
	if !strings.HasPrefix(key, share.PREFIX) {
		envKey = GetEnvKey(key)
	}
	return os.Getenv(envKey)
}

func LoadConfig() error {
	configFile := helper.GetPath("config")
	file, err := os.Open(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	// 清空现有配置
	configMap = make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			configMap[parts[0]] = parts[1]
			os.Setenv(parts[0], parts[1])
		}
	}
	return scanner.Err()
}

func SaveConfig() error {
	configDir := helper.GetPath("")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configFile := filepath.Join(configDir, "config")
	file, err := os.Create(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// 确保写入所有配置项
	for key, value := range configMap {
		if _, err := fmt.Fprintf(file, "%s=%s\n", key, value); err != nil {
			return err
		}
	}
	return file.Sync() // 确保数据写入磁盘
}

func GetEnvKey(flagKey string) string {
	return share.PREFIX + strings.ToUpper(flagKey)
}

// SetConfig 设置配置值并更新环境变量
func SetConfig(key, value string) {
	envKey := key
	if !strings.HasPrefix(key, share.PREFIX) {
		envKey = GetEnvKey(key)
	}
	configMap[envKey] = value
	os.Setenv(envKey, value)
}

// ClearConfig 清除指定配置
func ClearConfig(key string) {
	envKey := key
	if !strings.HasPrefix(key, share.PREFIX) {
		envKey = GetEnvKey(key)
	}
	delete(configMap, envKey)
	os.Unsetenv(envKey)
}

// ClearAllConfig 清除所有配置
func ClearAllConfig() {
	for key := range configMap {
		os.Unsetenv(key)
	}
	configMap = make(map[string]string)
}

// ClearAllConfig 清除所有配置
func GetConfigMap() map[string]string {
	return configMap
}

  - [config_test.go](#config_test.go)

### config_test.go

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sjzsdu/tong/config"
)

func TestGetConfig(t *testing.T) {
	// 设置临时测试目录
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// 设置测试环境变量
	os.Setenv("TONG_LANG", "zh")
	os.Setenv("TONG_DEEPSEEK_APIKEY", "test-key")

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "使用简短键获取语言",
			key:      "lang",
			expected: "zh",
		},
		{
			name:     "使用环境变量键获取语言",
			key:      "TONG_LANG",
			expected: "zh",
		},
		{
			name:     "使用简短键获取API密钥",
			key:      "deepseek_apikey",
			expected: "test-key",
		},
		{
			name:     "使用环境变量键获取API密钥",
			key:      "TONG_DEEPSEEK_APIKEY",
			expected: "test-key",
		},
		{
			name:     "获取不存在的配置",
			key:      "nonexistent",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := config.GetConfig(tt.key); got != tt.expected {
				t.Errorf("GetConfig() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfigOperations(t *testing.T) {
	// 设置临时测试目录
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// 每次测试前清除所有配置
	config.ClearAllConfig()

	tests := []struct {
		name     string
		configs  map[string]string
		wantErr  bool
		validate func(t *testing.T)
	}{
		{
			name: "基本配置保存和加载",
			configs: map[string]string{
				"TONG_LANG":            "zh",
				"TONG_DEEPSEEK_APIKEY": "test-key",
			},
			wantErr: false,
			validate: func(t *testing.T) {
				// 重新加载配置
				if err := config.LoadConfig(); err != nil {
					t.Fatalf("加载配置失败: %v", err)
				}

				if v := config.GetConfig("lang"); v != "zh" {
					t.Errorf("lang 期望为 zh，实际为 %s", v)
				}
				if v := config.GetConfig("deepseek_apikey"); v != "test-key" {
					t.Errorf("deepseek_apikey 期望为 test-key，实际为 %s", v)
				}
			},
		},
		{
			name:    "空配置",
			configs: map[string]string{},
			wantErr: false,
			validate: func(t *testing.T) {
				if v := config.GetConfig("lang"); v != "" {
					t.Errorf("期望配置为空，实际为 %s", v)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清除所有配置
			config.ClearAllConfig()

			// 设置测试配置
			for k, v := range tt.configs {
				config.SetConfig(k, v)
			}

			// 测试保存
			if err := config.SaveConfig(); (err != nil) != tt.wantErr {
				t.Errorf("SaveConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			// 清除所有配置
			config.ClearAllConfig()

			// 测试加载
			if err := config.LoadConfig(); (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			// 验证结果
			tt.validate(t)

			// 验证文件内容
			if !tt.wantErr {
				content, err := os.ReadFile(filepath.Join(tmpDir, ".tong", "config"))
				if err != nil {
					t.Errorf("读取配置文件失败: %v", err)
				}
				if len(content) == 0 && len(tt.configs) > 0 {
					t.Error("配置文件不应为空")
				}
			}
		})
	}
}

- [go.mod](#go.mod)

## go.mod

module github.com/sjzsdu/tong

go 1.23.0

toolchain go1.24.0

require (
	github.com/BurntSushi/toml v1.4.0
	github.com/c-bata/go-prompt v0.2.6
	github.com/charmbracelet/glamour v0.9.1
	github.com/fsnotify/fsnotify v1.9.0
	github.com/go-git/go-git/v5 v5.14.0
	github.com/jung-kurt/gofpdf v1.16.2
	github.com/nicksnyder/go-i18n/v2 v2.5.1
	github.com/spf13/cobra v1.9.1
	github.com/stretchr/testify v1.10.0
	golang.org/x/text v0.23.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	dario.cat/mergo v1.0.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/ProtonMail/go-crypto v1.1.5 // indirect
	github.com/alecthomas/chroma/v2 v2.14.0 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/charmbracelet/colorprofile v0.2.3-0.20250311203215-f60798e515dc // indirect
	github.com/charmbracelet/lipgloss v1.1.0 // indirect
	github.com/charmbracelet/x/ansi v0.8.0 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.13-0.20250311204145-2c3ea96c31dd // indirect
	github.com/charmbracelet/x/term v0.2.1 // indirect
	github.com/cloudflare/circl v1.6.0 // indirect
	github.com/cyphar/filepath-securejoin v0.4.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.6.2 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mattn/go-tty v0.0.3 // indirect
	github.com/microcosm-cc/bluemonday v1.0.27 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/pjbgf/sha1cd v0.3.2 // indirect
	github.com/pkg/term v1.2.0-beta.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
	github.com/skeema/knownhosts v1.3.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/yuin/goldmark v1.7.8 // indirect
	github.com/yuin/goldmark-emoji v1.0.5 // indirect
	golang.org/x/crypto v0.35.0 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/term v0.30.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
)

- [go.sum](#go.sum)

## go.sum

dario.cat/mergo v1.0.0 h1:AGCNq9Evsj31mOgNPcLyXc+4PNABt905YmuqPYYpBWk=
dario.cat/mergo v1.0.0/go.mod h1:uNxQE+84aUszobStD9th8a29P2fMDhsBdgRYvZOxGmk=
github.com/BurntSushi/toml v1.4.0 h1:kuoIxZQy2WRRk1pttg9asf+WVv6tWQuBNVmK8+nqPr0=
github.com/BurntSushi/toml v1.4.0/go.mod h1:ukJfTF/6rtPPRCnwkur4qwRxa8vTRFBF0uk2lLoLwho=
github.com/Microsoft/go-winio v0.5.2/go.mod h1:WpS1mjBmmwHBEWmogvA2mj8546UReBk4v8QkMxJ6pZY=
github.com/Microsoft/go-winio v0.6.2 h1:F2VQgta7ecxGYO8k3ZZz3RS8fVIXVxONVUPlNERoyfY=
github.com/Microsoft/go-winio v0.6.2/go.mod h1:yd8OoFMLzJbo9gZq8j5qaps8bJ9aShtEA8Ipt1oGCvU=
github.com/ProtonMail/go-crypto v1.1.5 h1:eoAQfK2dwL+tFSFpr7TbOaPNUbPiJj4fLYwwGE1FQO4=
github.com/ProtonMail/go-crypto v1.1.5/go.mod h1:rA3QumHc/FZ8pAHreoekgiAbzpNsfQAosU5td4SnOrE=
github.com/alecthomas/assert/v2 v2.7.0 h1:QtqSACNS3tF7oasA8CU6A6sXZSBDqnm7RfpLl9bZqbE=
github.com/alecthomas/assert/v2 v2.7.0/go.mod h1:Bze95FyfUr7x34QZrjL+XP+0qgp/zg8yS+TtBj1WA3k=
github.com/alecthomas/chroma/v2 v2.14.0 h1:R3+wzpnUArGcQz7fCETQBzO5n9IMNi13iIs46aU4V9E=
github.com/alecthomas/chroma/v2 v2.14.0/go.mod h1:QolEbTfmUHIMVpBqxeDnNBj2uoeI4EbYP4i6n68SG4I=
github.com/alecthomas/repr v0.4.0 h1:GhI2A8MACjfegCPVq9f1FLvIBS+DrQ2KQBFZP1iFzXc=
github.com/alecthomas/repr v0.4.0/go.mod h1:Fr0507jx4eOXV7AlPV6AVZLYrLIuIeSOWtW57eE/O/4=
github.com/anmitsu/go-shlex v0.0.0-20200514113438-38f4b401e2be h1:9AeTilPcZAjCFIImctFaOjnTIavg87rW78vTPkQqLI8=
github.com/anmitsu/go-shlex v0.0.0-20200514113438-38f4b401e2be/go.mod h1:ySMOLuWl6zY27l47sB3qLNK6tF2fkHG55UZxx8oIVo4=
github.com/armon/go-socks5 v0.0.0-20160902184237-e75332964ef5 h1:0CwZNZbxp69SHPdPJAN/hZIm0C4OItdklCFmMRWYpio=
github.com/armon/go-socks5 v0.0.0-20160902184237-e75332964ef5/go.mod h1:wHh0iHkYZB8zMSxRWpUBQtwG5a7fFgvEO+odwuTv2gs=
github.com/aymanbagabas/go-osc52/v2 v2.0.1 h1:HwpRHbFMcZLEVr42D4p7XBqjyuxQH5SMiErDT4WkJ2k=
github.com/aymanbagabas/go-osc52/v2 v2.0.1/go.mod h1:uYgXzlJ7ZpABp8OJ+exZzJJhRNQ2ASbcXHWsFqH8hp8=
github.com/aymanbagabas/go-udiff v0.2.0 h1:TK0fH4MteXUDspT88n8CKzvK0X9O2xu9yQjWpi6yML8=
github.com/aymanbagabas/go-udiff v0.2.0/go.mod h1:RE4Ex0qsGkTAJoQdQQCA0uG+nAzJO/pI/QwceO5fgrA=
github.com/aymerick/douceur v0.2.0 h1:Mv+mAeH1Q+n9Fr+oyamOlAkUNPWPlA8PPGR0QAaYuPk=
github.com/aymerick/douceur v0.2.0/go.mod h1:wlT5vV2O3h55X9m7iVYN0TBM0NH/MmbLnd30/FjWUq4=
github.com/boombuler/barcode v1.0.0/go.mod h1:paBWMcWSl3LHKBqUq+rly7CNSldXjb2rDl3JlRe0mD8=
github.com/c-bata/go-prompt v0.2.6 h1:POP+nrHE+DfLYx370bedwNhsqmpCUynWPxuHi0C5vZI=
github.com/c-bata/go-prompt v0.2.6/go.mod h1:/LMAke8wD2FsNu9EXNdHxNLbd9MedkPnCdfpU9wwHfY=
github.com/charmbracelet/colorprofile v0.2.3-0.20250311203215-f60798e515dc h1:4pZI35227imm7yK2bGPcfpFEmuY1gc2YSTShr4iJBfs=
github.com/charmbracelet/colorprofile v0.2.3-0.20250311203215-f60798e515dc/go.mod h1:X4/0JoqgTIPSFcRA/P6INZzIuyqdFY5rm8tb41s9okk=
github.com/charmbracelet/glamour v0.9.1 h1:11dEfiGP8q1BEqvGoIjivuc2rBk+5qEXdPtaQ2WoiCM=
github.com/charmbracelet/glamour v0.9.1/go.mod h1:+SHvIS8qnwhgTpVMiXwn7OfGomSqff1cHBCI8jLOetk=
github.com/charmbracelet/lipgloss v1.1.0 h1:vYXsiLHVkK7fp74RkV7b2kq9+zDLoEU4MZoFqR/noCY=
github.com/charmbracelet/lipgloss v1.1.0/go.mod h1:/6Q8FR2o+kj8rz4Dq0zQc3vYf7X+B0binUUBwA0aL30=
github.com/charmbracelet/x/ansi v0.8.0 h1:9GTq3xq9caJW8ZrBTe0LIe2fvfLR/bYXKTx2llXn7xE=
github.com/charmbracelet/x/ansi v0.8.0/go.mod h1:wdYl/ONOLHLIVmQaxbIYEC/cRKOQyjTkowiI4blgS9Q=
github.com/charmbracelet/x/cellbuf v0.0.13-0.20250311204145-2c3ea96c31dd h1:vy0GVL4jeHEwG5YOXDmi86oYw2yuYUGqz6a8sLwg0X8=
github.com/charmbracelet/x/cellbuf v0.0.13-0.20250311204145-2c3ea96c31dd/go.mod h1:xe0nKWGd3eJgtqZRaN9RjMtK7xUYchjzPr7q6kcvCCs=
github.com/charmbracelet/x/exp/golden v0.0.0-20240806155701-69247e0abc2a h1:G99klV19u0QnhiizODirwVksQB91TJKV/UaTnACcG30=
github.com/charmbracelet/x/exp/golden v0.0.0-20240806155701-69247e0abc2a/go.mod h1:wDlXFlCrmJ8J+swcL/MnGUuYnqgQdW9rhSD61oNMb6U=
github.com/charmbracelet/x/term v0.2.1 h1:AQeHeLZ1OqSXhrAWpYUtZyX1T3zVxfpZuEQMIQaGIAQ=
github.com/charmbracelet/x/term v0.2.1/go.mod h1:oQ4enTYFV7QN4m0i9mzHrViD7TQKvNEEkHUMCmsxdUg=
github.com/cloudflare/circl v1.6.0 h1:cr5JKic4HI+LkINy2lg3W2jF8sHCVTBncJr5gIIq7qk=
github.com/cloudflare/circl v1.6.0/go.mod h1:uddAzsPgqdMAYatqJ0lsjX1oECcQLIlRpzZh3pJrofs=
github.com/cpuguy83/go-md2man/v2 v2.0.6/go.mod h1:oOW0eioCTA6cOiMLiUPZOpcVxMig6NIQQ7OS05n1F4g=
github.com/cyphar/filepath-securejoin v0.4.1 h1:JyxxyPEaktOD+GAnqIqTf9A8tHyAG22rowi7HkoSU1s=
github.com/cyphar/filepath-securejoin v0.4.1/go.mod h1:Sdj7gXlvMcPZsbhwhQ33GguGLDGQL7h7bg04C/+u9jI=
github.com/davecgh/go-spew v1.1.0/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
github.com/davecgh/go-spew v1.1.1 h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=
github.com/davecgh/go-spew v1.1.1/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
github.com/dlclark/regexp2 v1.11.5 h1:Q/sSnsKerHeCkc/jSTNq1oCm7KiVgUMZRDUoRu0JQZQ=
github.com/dlclark/regexp2 v1.11.5/go.mod h1:DHkYz0B9wPfa6wondMfaivmHpzrQ3v9q8cnmRbL6yW8=
github.com/elazarl/goproxy v1.7.2 h1:Y2o6urb7Eule09PjlhQRGNsqRfPmYI3KKQLFpCAV3+o=
github.com/elazarl/goproxy v1.7.2/go.mod h1:82vkLNir0ALaW14Rc399OTTjyNREgmdL2cVoIbS6XaE=
github.com/emirpasic/gods v1.18.1 h1:FXtiHYKDGKCW2KzwZKx0iC0PQmdlorYgdFG9jPXJ1Bc=
github.com/emirpasic/gods v1.18.1/go.mod h1:8tpGGwCnJ5H4r6BWwaV6OrWmMoPhUl5jm/FMNAnJvWQ=
github.com/fsnotify/fsnotify v1.9.0 h1:2Ml+OJNzbYCTzsxtv8vKSFD9PbJjmhYF14k/jKC7S9k=
github.com/fsnotify/fsnotify v1.9.0/go.mod h1:8jBTzvmWwFyi3Pb8djgCCO5IBqzKJ/Jwo8TRcHyHii0=
github.com/gliderlabs/ssh v0.3.8 h1:a4YXD1V7xMF9g5nTkdfnja3Sxy1PVDCj1Zg4Wb8vY6c=
github.com/gliderlabs/ssh v0.3.8/go.mod h1:xYoytBv1sV0aL3CavoDuJIQNURXkkfPA/wxQ1pL1fAU=
github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 h1:+zs/tPmkDkHx3U66DAb0lQFJrpS6731Oaa12ikc+DiI=
github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376/go.mod h1:an3vInlBmSxCcxctByoQdvwPiA7DTK7jaaFDBTtu0ic=
github.com/go-git/go-billy/v5 v5.6.2 h1:6Q86EsPXMa7c3YZ3aLAQsMA0VlWmy43r6FHqa/UNbRM=
github.com/go-git/go-billy/v5 v5.6.2/go.mod h1:rcFC2rAsp/erv7CMz9GczHcuD0D32fWzH+MJAU+jaUU=
github.com/go-git/go-git-fixtures/v4 v4.3.2-0.20231010084843-55a94097c399 h1:eMje31YglSBqCdIqdhKBW8lokaMrL3uTkpGYlE2OOT4=
github.com/go-git/go-git-fixtures/v4 v4.3.2-0.20231010084843-55a94097c399/go.mod h1:1OCfN199q1Jm3HZlxleg+Dw/mwps2Wbk9frAWm+4FII=
github.com/go-git/go-git/v5 v5.14.0 h1:/MD3lCrGjCen5WfEAzKg00MJJffKhC8gzS80ycmCi60=
github.com/go-git/go-git/v5 v5.14.0/go.mod h1:Z5Xhoia5PcWA3NF8vRLURn9E5FRhSl7dGj9ItW3Wk5k=
github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 h1:f+oWsMOmNPc8JmEHVZIycC7hBoQxHH9pNKQORJNozsQ=
github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8/go.mod h1:wcDNUvekVysuuOpQKo3191zZyTpiI6se1N1ULghS0sw=
github.com/google/go-cmp v0.7.0 h1:wk8382ETsv4JYUZwIsn6YpYiWiBsYLSJiTsyBybVuN8=
github.com/google/go-cmp v0.7.0/go.mod h1:pXiqmnSA92OHEEa9HXL2W4E7lf9JzCmGVUdgjX3N/iU=
github.com/gorilla/css v1.0.1 h1:ntNaBIghp6JmvWnxbZKANoLyuXTPZ4cAMlo6RyhlbO8=
github.com/gorilla/css v1.0.1/go.mod h1:BvnYkspnSzMmwRK+b8/xgNPLiIuNZr6vbZBTPQ2A3b0=
github.com/hexops/gotextdiff v1.0.3 h1:gitA9+qJrrTCsiCl7+kh75nPqQt1cx4ZkudSTLoUqJM=
github.com/hexops/gotextdiff v1.0.3/go.mod h1:pSWU5MAI3yDq+fZBTazCSJysOMbxWL1BSow5/V2vxeg=
github.com/inconshreveable/mousetrap v1.1.0 h1:wN+x4NVGpMsO7ErUn/mUI3vEoE6Jt13X2s0bqwp9tc8=
github.com/inconshreveable/mousetrap v1.1.0/go.mod h1:vpF70FUmC8bwa3OWnCshd2FqLfsEA9PFc4w1p2J65bw=
github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 h1:BQSFePA1RWJOlocH6Fxy8MmwDt+yVQYULKfN0RoTN8A=
github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99/go.mod h1:1lJo3i6rXxKeerYnT8Nvf0QmHCRC1n8sfWVwXF2Frvo=
github.com/jung-kurt/gofpdf v1.0.0/go.mod h1:7Id9E/uU8ce6rXgefFLlgrJj/GYY22cpxn+r32jIOes=
github.com/jung-kurt/gofpdf v1.16.2 h1:jgbatWHfRlPYiK85qgevsZTHviWXKwB1TTiKdz5PtRc=
github.com/jung-kurt/gofpdf v1.16.2/go.mod h1:1hl7y57EsiPAkLbOwzpzqgx1A30nQCk/YmFV8S2vmK0=
github.com/kevinburke/ssh_config v1.2.0 h1:x584FjTGwHzMwvHx18PXxbBVzfnxogHaAReU4gf13a4=
github.com/kevinburke/ssh_config v1.2.0/go.mod h1:CT57kijsi8u/K/BOFA39wgDQJ9CxiF4nAY/ojJ6r6mM=
github.com/kr/pretty v0.1.0/go.mod h1:dAy3ld7l9f0ibDNOQOHHMYYIIbhfbHSm3C4ZsoJORNo=
github.com/kr/pretty v0.3.1 h1:flRD4NNwYAUpkphVc1HcthR4KEIFJ65n8Mw5qdRn3LE=
github.com/kr/pretty v0.3.1/go.mod h1:hoEshYVHaxMs3cyo3Yncou5ZscifuDolrwPKZanG3xk=
github.com/kr/pty v1.1.1/go.mod h1:pFQYn66WHrOpPYNljwOMqo10TkYh1fy3cYio2l3bCsQ=
github.com/kr/text v0.1.0/go.mod h1:4Jbv+DJW3UT/LiOwJeYQe1efqtUx/iVham/4vfdArNI=
github.com/kr/text v0.2.0 h1:5Nx0Ya0ZqY2ygV366QzturHI13Jq95ApcVaJBhpS+AY=
github.com/kr/text v0.2.0/go.mod h1:eLer722TekiGuMkidMxC/pM04lWEeraHUUmBw8l2grE=
github.com/lucasb-eyer/go-colorful v1.2.0 h1:1nnpGOrhyZZuNyfu1QjKiUICQ74+3FNCN69Aj6K7nkY=
github.com/lucasb-eyer/go-colorful v1.2.0/go.mod h1:R4dSotOR9KMtayYi1e77YzuveK+i7ruzyGqttikkLy0=
github.com/mattn/go-colorable v0.1.4/go.mod h1:U0ppj6V5qS13XJ6of8GYAs25YV2eR4EVcfRqFIhoBtE=
github.com/mattn/go-colorable v0.1.7/go.mod h1:u6P/XSegPjTcexA+o6vUJrdnUu04hMope9wVRipJSqc=
github.com/mattn/go-colorable v0.1.13 h1:fFA4WZxdEF4tXPZVKMLwD8oUnCTTo08duU7wxecdEvA=
github.com/mattn/go-colorable v0.1.13/go.mod h1:7S9/ev0klgBDR4GtXTXX8a3vIGJpMovkB8vQcUbaXHg=
github.com/mattn/go-isatty v0.0.8/go.mod h1:Iq45c/XA43vh69/j3iqttzPXn0bhXyGjM0Hdxcsrc5s=
github.com/mattn/go-isatty v0.0.10/go.mod h1:qgIWMr58cqv1PHHyhnkY9lrL7etaEgOFcMEpPG5Rm84=
github.com/mattn/go-isatty v0.0.12/go.mod h1:cbi8OIDigv2wuxKPP5vlRcQ1OAZbq2CE4Kysco4FUpU=
github.com/mattn/go-isatty v0.0.16/go.mod h1:kYGgaQfpe5nmfYZH+SKPsOc2e4SrIfOl2e/yFXSvRLM=
github.com/mattn/go-isatty v0.0.20 h1:xfD0iDuEKnDkl03q4limB+vH+GxLEtL/jb4xVJSWWEY=
github.com/mattn/go-isatty v0.0.20/go.mod h1:W+V8PltTTMOvKvAeJH7IuucS94S2C6jfK/D7dTCTo3Y=
github.com/mattn/go-runewidth v0.0.6/go.mod h1:H031xJmbD/WCDINGzjvQ9THkh0rPKHF+m2gUSrubnMI=
github.com/mattn/go-runewidth v0.0.9/go.mod h1:H031xJmbD/WCDINGzjvQ9THkh0rPKHF+m2gUSrubnMI=
github.com/mattn/go-runewidth v0.0.12/go.mod h1:RAqKPSqVFrSLVXbA8x7dzmKdmGzieGRCM46jaSJTDAk=
github.com/mattn/go-runewidth v0.0.16 h1:E5ScNMtiwvlvB5paMFdw9p4kSQzbXFikJ5SQO6TULQc=
github.com/mattn/go-runewidth v0.0.16/go.mod h1:Jdepj2loyihRzMpdS35Xk/zdY8IAYHsh153qUoGf23w=
github.com/mattn/go-tty v0.0.3 h1:5OfyWorkyO7xP52Mq7tB36ajHDG5OHrmBGIS/DtakQI=
github.com/mattn/go-tty v0.0.3/go.mod h1:ihxohKRERHTVzN+aSVRwACLCeqIoZAWpoICkkvrWyR0=
github.com/microcosm-cc/bluemonday v1.0.27 h1:MpEUotklkwCSLeH+Qdx1VJgNqLlpY2KXwXFM08ygZfk=
github.com/microcosm-cc/bluemonday v1.0.27/go.mod h1:jFi9vgW+H7c3V0lb6nR74Ib/DIB5OBs92Dimizgw2cA=
github.com/muesli/reflow v0.3.0 h1:IFsN6K9NfGtjeggFP+68I4chLZV2yIKsXJFNZ+eWh6s=
github.com/muesli/reflow v0.3.0/go.mod h1:pbwTDkVPibjO2kyvBQRBxTWEEGDGq0FlB1BIKtnHY/8=
github.com/muesli/termenv v0.16.0 h1:S5AlUN9dENB57rsbnkPyfdGuWIlkmzJjbFf0Tf5FWUc=
github.com/muesli/termenv v0.16.0/go.mod h1:ZRfOIKPFDYQoDFF4Olj7/QJbW60Ol/kL1pU3VfY/Cnk=
github.com/nicksnyder/go-i18n/v2 v2.5.1 h1:IxtPxYsR9Gp60cGXjfuR/llTqV8aYMsC472zD0D1vHk=
github.com/nicksnyder/go-i18n/v2 v2.5.1/go.mod h1:DrhgsSDZxoAfvVrBVLXoxZn/pN5TXqaDbq7ju94viiQ=
github.com/onsi/gomega v1.34.1 h1:EUMJIKUjM8sKjYbtxQI9A4z2o+rruxnzNvpknOXie6k=
github.com/onsi/gomega v1.34.1/go.mod h1:kU1QgUvBDLXBJq618Xvm2LUX6rSAfRaFRTcdOeDLwwY=
github.com/phpdave11/gofpdi v1.0.7/go.mod h1:vBmVV0Do6hSBHC8uKUQ71JGW+ZGQq74llk/7bXwjDoI=
github.com/pjbgf/sha1cd v0.3.2 h1:a9wb0bp1oC2TGwStyn0Umc/IGKQnEgF0vVaZ8QF8eo4=
github.com/pjbgf/sha1cd v0.3.2/go.mod h1:zQWigSxVmsHEZow5qaLtPYxpcKMMQpa09ixqBxuCS6A=
github.com/pkg/errors v0.8.1/go.mod h1:bwawxfHBFNV+L2hUp1rHADufV3IMtnDRdf1r5NINEl0=
github.com/pkg/errors v0.9.1 h1:FEBLx1zS214owpjy7qsBeixbURkuhQAwrK5UwLGTwt4=
github.com/pkg/errors v0.9.1/go.mod h1:bwawxfHBFNV+L2hUp1rHADufV3IMtnDRdf1r5NINEl0=
github.com/pkg/term v1.2.0-beta.2 h1:L3y/h2jkuBVFdWiJvNfYfKmzcCnILw7mJWm2JQuMppw=
github.com/pkg/term v1.2.0-beta.2/go.mod h1:E25nymQcrSllhX42Ok8MRm1+hyBdHY0dCeiKZ9jpNGw=
github.com/pmezard/go-difflib v1.0.0 h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=
github.com/pmezard/go-difflib v1.0.0/go.mod h1:iKH77koFhYxTK1pcRnkKkqfTogsbg7gZNVY4sRDYZ/4=
github.com/rivo/uniseg v0.1.0/go.mod h1:J6wj4VEh+S6ZtnVlnTBMWIodfgj8LQOQFoIToxlJtxc=
github.com/rivo/uniseg v0.2.0/go.mod h1:J6wj4VEh+S6ZtnVlnTBMWIodfgj8LQOQFoIToxlJtxc=
github.com/rivo/uniseg v0.4.7 h1:WUdvkW8uEhrYfLC4ZzdpI2ztxP1I582+49Oc5Mq64VQ=
github.com/rivo/uniseg v0.4.7/go.mod h1:FN3SvrM+Zdj16jyLfmOkMNblXMcoc8DfTHruCPUcx88=
github.com/rogpeppe/go-internal v1.14.1 h1:UQB4HGPB6osV0SQTLymcB4TgvyWu6ZyliaW0tI/otEQ=
github.com/rogpeppe/go-internal v1.14.1/go.mod h1:MaRKkUm5W0goXpeCfT7UZI6fk/L7L7so1lCWt35ZSgc=
github.com/russross/blackfriday/v2 v2.1.0/go.mod h1:+Rmxgy9KzJVeS9/2gXHxylqXiyQDYRxCVz55jmeOWTM=
github.com/ruudk/golang-pdf417 v0.0.0-20181029194003-1af4ab5afa58/go.mod h1:6lfFZQK844Gfx8o5WFuvpxWRwnSoipWe/p622j1v06w=
github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 h1:n661drycOFuPLCN3Uc8sB6B/s6Z4t2xvBgU1htSHuq8=
github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3/go.mod h1:A0bzQcvG0E7Rwjx0REVgAGH58e96+X0MeOfepqsbeW4=
github.com/sirupsen/logrus v1.7.0/go.mod h1:yWOB1SBYBC5VeMP7gHvWumXLIWorT60ONWic61uBYv0=
github.com/skeema/knownhosts v1.3.1 h1:X2osQ+RAjK76shCbvhHHHVl3ZlgDm8apHEHFqRjnBY8=
github.com/skeema/knownhosts v1.3.1/go.mod h1:r7KTdC8l4uxWRyK2TpQZ/1o5HaSzh06ePQNxPwTcfiY=
github.com/spf13/cobra v1.9.1 h1:CXSaggrXdbHK9CF+8ywj8Amf7PBRmPCOJugH954Nnlo=
github.com/spf13/cobra v1.9.1/go.mod h1:nDyEzZ8ogv936Cinf6g1RU9MRY64Ir93oCnqb9wxYW0=
github.com/spf13/pflag v1.0.6 h1:jFzHGLGAlb3ruxLB8MhbI6A8+AQX/2eW4qeyNZXNp2o=
github.com/spf13/pflag v1.0.6/go.mod h1:McXfInJRrz4CZXVZOBLb0bTZqETkiAhM9Iw0y3An2Bg=
github.com/stretchr/objx v0.1.0/go.mod h1:HFkY916IF+rwdDfMAkV7OtwuqBVzrE8GR6GFx+wExME=
github.com/stretchr/testify v1.2.2/go.mod h1:a8OnRcib4nhh0OaRAV+Yts87kKdq0PP7pXfy6kDkUVs=
github.com/stretchr/testify v1.4.0/go.mod h1:j7eGeouHqKxXV5pUuKE4zz7dFj8WfuZ+81PSLYec5m4=
github.com/stretchr/testify v1.10.0 h1:Xv5erBjTwe/5IxqUQTdXv5kgmIvbHo3QQyRwhJsOfJA=
github.com/stretchr/testify v1.10.0/go.mod h1:r2ic/lqez/lEtzL7wO/rwa5dbSLXVDPFyf8C91i36aY=
github.com/xanzy/ssh-agent v0.3.3 h1:+/15pJfg/RsTxqYcX6fHqOXZwwMP+2VyYWJeWM2qQFM=
github.com/xanzy/ssh-agent v0.3.3/go.mod h1:6dzNDKs0J9rVPHPhaGCukekBHKqfl+L3KghI1Bc68Uw=
github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e h1:JVG44RsyaB9T2KIHavMF/ppJZNG9ZpyihvCd0w101no=
github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e/go.mod h1:RbqR21r5mrJuqunuUZ/Dhy/avygyECGrLceyNeo4LiM=
github.com/yuin/goldmark v1.7.1/go.mod h1:uzxRWxtg69N339t3louHJ7+O03ezfj6PlliRlaOzY1E=
github.com/yuin/goldmark v1.7.8 h1:iERMLn0/QJeHFhxSt3p6PeN9mGnvIKSpG9YYorDMnic=
github.com/yuin/goldmark v1.7.8/go.mod h1:uzxRWxtg69N339t3louHJ7+O03ezfj6PlliRlaOzY1E=
github.com/yuin/goldmark-emoji v1.0.5 h1:EMVWyCGPlXJfUXBXpuMu+ii3TIaxbVBnEX9uaDC4cIk=
github.com/yuin/goldmark-emoji v1.0.5/go.mod h1:tTkZEbwu5wkPmgTcitqddVxY9osFZiavD+r4AzQrh1U=
golang.org/x/crypto v0.0.0-20220622213112-05595931fe9d/go.mod h1:IxCIyHEi3zRg3s0A5j5BB6A9Jmi73HwBIUl50j+osU4=
golang.org/x/crypto v0.35.0 h1:b15kiHdrGCHrP6LvwaQ3c03kgNhhiMgvlhxHQhmg2Xs=
golang.org/x/crypto v0.35.0/go.mod h1:dy7dXNW32cAb/6/PRuTNsix8T+vJAqvuIy5Bli/x0YQ=
golang.org/x/exp v0.0.0-20240719175910-8a7402abbf56 h1:2dVuKD2vS7b0QIHQbpyTISPd0LeHDbnYEryqj5Q1ug8=
golang.org/x/exp v0.0.0-20240719175910-8a7402abbf56/go.mod h1:M4RDyNAINzryxdtnbRXRL/OHtkFuWGRjvuhBJpk2IlY=
golang.org/x/image v0.0.0-20190910094157-69e4b8554b2a/go.mod h1:FeLwcggjj3mMvU+oOTbSwawSJRM1uh48EjtB4UJZlP0=
golang.org/x/net v0.0.0-20211112202133-69e39bad7dc2/go.mod h1:9nx3DQGgdP8bBQD5qxJ1jj9UTztislL4KSBs9R2vV5Y=
golang.org/x/net v0.35.0 h1:T5GQRQb2y08kTAByq9L4/bz8cipCdA8FbRTXewonqY8=
golang.org/x/net v0.35.0/go.mod h1:EglIi67kWsHKlRzzVMUD93VMSWGFOMSZgxFjparz1Qk=
golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e/go.mod h1:RxMgew5VJxzue5/jJTE5uejpjVlOe/izrB70Jof72aM=
golang.org/x/sys v0.0.0-20190222072716-a9d3bda3a223/go.mod h1:STP8DvDyc/dI5b8T5hshtkjS+E42TnysNCUPdjciGhY=
golang.org/x/sys v0.0.0-20191008105621-543471e840be/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
golang.org/x/sys v0.0.0-20191026070338-33540a1f6037/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
golang.org/x/sys v0.0.0-20191120155948-bd437916bb0e/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
golang.org/x/sys v0.0.0-20200116001909-b77594299b42/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
golang.org/x/sys v0.0.0-20200223170610-d5e6a3e2c0ae/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
golang.org/x/sys v0.0.0-20200909081042-eff7692f9009/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
golang.org/x/sys v0.0.0-20200918174421-af09f7315aff/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
golang.org/x/sys v0.0.0-20201119102817-f84b799fce68/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
golang.org/x/sys v0.0.0-20210124154548-22da62e12c0c/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
golang.org/x/sys v0.0.0-20210423082822-04245dca01da/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
golang.org/x/sys v0.0.0-20220811171246-fbc7d0a398ab/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
golang.org/x/sys v0.6.0/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
golang.org/x/sys v0.31.0 h1:ioabZlmFYtWhL+TRYpcnNlLwhyxaM9kWTDEmfnprqik=
golang.org/x/sys v0.31.0/go.mod h1:BJP2sWEmIv4KK5OTEluFJCKSidICx8ciO85XgH3Ak8k=
golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1/go.mod h1:bj7SfCRtBDWHUb9snDiAeCFNEtKQo2Wmx5Cou7ajbmo=
golang.org/x/term v0.30.0 h1:PQ39fJZ+mfadBm0y5WlL4vlM7Sx1Hgf13sMIY2+QS9Y=
golang.org/x/term v0.30.0/go.mod h1:NYYFdzHoI5wRh/h5tDMdMqCqPJZEuNqVR5xJLd/n67g=
golang.org/x/text v0.3.0/go.mod h1:NqM8EUOU14njkJ3fqMW+pc6Ldnwhi/IjpwHt7yyuwOQ=
golang.org/x/text v0.3.6/go.mod h1:5Zoc/QRtKVWzQhOtBMvqHzDpF6irO9z98xDceosuGiQ=
golang.org/x/text v0.23.0 h1:D71I7dUrlY+VX0gQShAThNGHFxZ13dGLBHQLVl1mJlY=
golang.org/x/text v0.23.0/go.mod h1:/BLNzu4aZCJ1+kcD0DNRotWKage4q2rGVAg4o22unh4=
golang.org/x/tools v0.0.0-20180917221912-90fa682c2a6e/go.mod h1:n7NCudcB/nEzxVGmLbDWY5pfWTLqBcC2KZ6jyYvM4mQ=
gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c h1:Hei/4ADfdWqJk1ZMxUNpqntNwaWcugrBjAiHlqqRiVk=
gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c/go.mod h1:JHkPIbrfpd72SG/EVd6muEfDQjcINNoR0C8j2r3qZ4Q=
gopkg.in/warnings.v0 v0.1.2 h1:wFXVbFY8DY5/xOe1ECiWdKCzZlxgshcYVNkBHstARME=
gopkg.in/warnings.v0 v0.1.2/go.mod h1:jksf8JmL6Qr/oQM2OXTHunEvvTAsrWBLb6OOjuVWRNI=
gopkg.in/yaml.v2 v2.2.2/go.mod h1:hI93XBmqTisBFMUTm0b8Fm+jr3Dg1NNxqwp+5A1VGuI=
gopkg.in/yaml.v2 v2.4.0/go.mod h1:RDklbk79AGWmwhnvt/jBztapEOGDOx6ZbXqjP6csGnQ=
gopkg.in/yaml.v3 v3.0.1 h1:fxVm/GzAzEWqLHuvctI91KS9hhNmmWOoWu0XTYJS7CA=
gopkg.in/yaml.v3 v3.0.1/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=

- [helper](#helper)

## helper

  - [cmd.go](#cmd.go)

### cmd.go

package helper

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/c-bata/go-prompt"
	"github.com/sjzsdu/tong/lang"
)

func ShowLoadingAnimation(done chan bool) {
	spinChars := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	i := 0
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			fmt.Print("\n") // 清除当前行
			done <- false   // 发送 false 表示动画已清理完成
			return
		case <-ticker.C:
			fmt.Printf("\r%s %s... ", spinChars[i], lang.T("Thinking"))
			i = (i + 1) % len(spinChars)
		}
	}
}

func ReadFromTerminal(promptText string) (string, error) {
	var result string
	done := make(chan struct{})
	once := &sync.Once{}

	p := prompt.New(
		func(in string) {
			result = in
			once.Do(func() { close(done) })
		},
		func(d prompt.Document) []prompt.Suggest {
			return nil
		},
		prompt.OptionPrefix(""), // 移除默认提示符
		prompt.OptionTitle("tong"),
		prompt.OptionPrefixTextColor(prompt.Blue),
		prompt.OptionInputTextColor(prompt.DefaultColor),
		prompt.OptionAddKeyBind(
			prompt.KeyBind{
				Key: prompt.ControlV,
				Fn: func(b *prompt.Buffer) {
					result = "vim"
					once.Do(func() { close(done) })
				},
			},
			prompt.KeyBind{
				Key: prompt.ControlC,
				Fn: func(b *prompt.Buffer) {
					result = "quit"
					once.Do(func() { close(done) })
				},
			},
		),
		prompt.OptionSetExitCheckerOnInput(func(in string, breakline bool) bool {
			return breakline
		}),
	)

	// 手动输出提示符
	fmt.Print(promptText)

	go p.Run()
	<-done

	return result, nil
}

func ReadPipeContent() (string, error) {
	content, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	return StripAnsiCodes(string(content)), nil
}

func ReadFromVim() (string, error) {
	// 创建一个临时文件来存储输入
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, "vim_input_"+randomString(8)+".txt")

	// 确保在函数结束时删除临时文件
	defer os.Remove(tempFile)

	// 使用 Vim 编辑临时文件，+startinsert 参数让 vim 启动后直接进入插入模式
	cmd := exec.Command("vim", "+startinsert", tempFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error running Vim: %w", err)
	}

	// 读取临时文件的内容
	content, err := os.ReadFile(tempFile)
	if err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	// 处理用户输入的内容
	userInput := strings.TrimSpace(string(content))

	return userInput, nil
}

func InputString(promptText string) (string, error) {
	input, err := ReadFromTerminal(promptText)
	if err != nil {
		return "", fmt.Errorf("error reading input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("empty input")
	}

	if input == "vim" {
		input, err = ReadFromVim()
		if err != nil {
			return "", fmt.Errorf(lang.T("Error reading vim")+": %v\n", err)
		}
		fmt.Printf(">%s\n", input)
	}

	return input, nil
}

  - [cmd_test.go](#cmd_test.go)

### cmd_test.go

package helper

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShowLoadingAnimation(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
	}{
		{
			name:     "快速取消动画",
			duration: 100 * time.Millisecond,
		},
		{
			name:     "延迟取消动画",
			duration: 500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done := make(chan bool)
			go ShowLoadingAnimation(done)

			time.Sleep(tt.duration)
			done <- true
		})
	}
}

func TestReadFromVim(t *testing.T) {
	// 跳过实际执行 vim 的测试
	if os.Getenv("TEST_WITH_VIM") != "1" {
		t.Skip("跳过需要 vim 的测试。设置 TEST_WITH_VIM=1 来运行此测试")
	}

	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "基本测试",
			content: "test content",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建一个临时文件并写入测试内容
			tempDir := os.TempDir()
			tempFile := filepath.Join(tempDir, "vim_test_"+randomString(8)+".txt")
			err := os.WriteFile(tempFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("无法创建测试文件: %v", err)
			}
			defer os.Remove(tempFile)

			// 模拟 vim 的行为
			os.Setenv("EDITOR", "echo") // 使用 echo 替代 vim 用于测试

			got, err := ReadFromVim()
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadFromVim() 错误 = %v, 期望错误 %v", err, tt.wantErr)
				return
			}

			// 由于我们使用了 echo 替代 vim，这里主要测试函数是否正常运行
			// 实际内容验证可能需要手动测试或更复杂的模拟
			if err == nil && got == "" {
				t.Error("ReadFromVim() 返回空字符串")
			}
		})
	}
}

  - [code.go](#code.go)

### code.go

package helper

// 常见的程序文件扩展名
var ProgramFileExtensions = map[string]bool{
	"go":    true,
	"py":    true,
	"js":    true,
	"ts":    true,
	"jsx":   true,
	"tsx":   true,
	"java":  true,
	"cpp":   true,
	"c":     true,
	"h":     true,
	"hpp":   true,
	"rs":    true,
	"rb":    true,
	"php":   true,
	"swift": true,
	"kt":    true,
	"scala": true,
	"cs":    true,
	"vue":   true,
	"sh":    true,
	"pl":    true,
	"r":     true,
	"m":     true,
	"mm":    true,
	"lua":   true,
}

// IsProgramFile 判断是否是程序文件
func IsProgramFile(file string) bool {
	ext := GetFileExt(file)
	return ProgramFileExtensions[ext]
}

  - [console.go](#console.go)

### console.go

package helper

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// PrintWithLabel 带标签的打印，方便调试时识别输出内容
func PrintWithLabel(label string, v ...interface{}) {
    fmt.Printf("[%s]: ", label)
    if len(v) == 0 {
        fmt.Println("nil")
        return
    }
    
    if len(v) == 1 {
        Print(v[0])
        return
    }
    
    // 处理多个参数
    fmt.Print("[ ")
    for i, item := range v {
        if i > 0 {
            fmt.Print(", ")
        }
        Print(item)
    }
    fmt.Println(" ]")
}

func Print(v interface{}) {
	switch reflect.TypeOf(v).Kind() {
	case reflect.Map, reflect.Slice, reflect.Struct, reflect.Ptr:
		formatted, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			fmt.Printf("格式化输出失败: %v\n", err)
			return
		}
		fmt.Print(string(formatted))
		fmt.Println()
	default:
		fmt.Println(v)
	}
}

// Printf 支持格式化字符串的打印
func Printf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

// Println 换行打印
func Println(v ...interface{}) {
	fmt.Println(v...)
}

  - [console_test.go](#console_test.go)

### console_test.go

package helper

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestPrintWithLabel(t *testing.T) {
	tests := []struct {
		name     string
		label    string
		input    interface{}
		expected string
	}{
		{
			name:     "打印字符串",
			label:    "测试",
			input:    "Hello World",
			expected: "[测试]: Hello World\n",
		},
		{
			name:     "打印整数",
			label:    "数字",
			input:    42,
			expected: "[数字]: 42\n",
		},
		{
			name:  "打印结构体",
			label: "用户信息",
			input: struct {
				Name string
				Age  int
			}{Name: "张三", Age: 25},
			expected: `[用户信息]: {
  "Name": "张三",
  "Age": 25
}
`,
		},
		{
			name:  "打印map",
			label: "配置",
			input: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
			},
			expected: `[配置]: {
  "key1": "value1",
  "key2": 123
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 使用bytes.Buffer捕获输出
			var buf bytes.Buffer
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			PrintWithLabel(tt.label, tt.input)

			w.Close()
			os.Stdout = old
			io.Copy(&buf, r)

			got := buf.String()
			if got != tt.expected {
				t.Errorf("PrintWithLabel() = %q, want %q", got, tt.expected)
			}
		})
	}
}

  - [content_updater.go](#content_updater.go)

### content_updater.go

package helper

import "strings"

// UpdateOperation 定义更新操作的结构体
type UpdateOperation struct {
	Operation string // 操作类型：insert, delete, replace, replaceAll
	Target    string // 源文档中的内容块
	Content   string // 新增或更新的内容
}

// ApplyChanges 利用更新数组完成对原来文档的更新
func ApplyChanges(blogContent string, changes []UpdateOperation) string {
	for _, change := range changes {
		// 统一处理空 Target 的情况
		if change.Target == "" {
			if blogContent != "" && !strings.HasSuffix(blogContent, "\n") {
				blogContent += "\n"
			}
			blogContent += change.Content
			continue
		}

		switch change.Operation {
		case "insert":
			blogContent = strings.Replace(blogContent, change.Target, change.Target+"\n"+change.Content, 1)
		case "delete":
			blogContent = strings.Replace(blogContent, change.Target, "", 1)
		case "replace":
			blogContent = strings.Replace(blogContent, change.Target, change.Content, 1)
		case "replaceAll":
			blogContent = strings.ReplaceAll(blogContent, change.Target, change.Content)
		}
	}
	return blogContent
}

  - [content_updater_test.go](#content_updater_test.go)

### content_updater_test.go

package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyChanges(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		changes  []UpdateOperation
		expected string
	}{
		{
			name:    "插入操作",
			content: "Hello World",
			changes: []UpdateOperation{
				{
					Operation: "insert",
					Target:    "Hello",
					Content:   "Beautiful",
				},
			},
			expected: "Hello\nBeautiful World",
		},
		{
			name:    "删除操作",
			content: "Hello World",
			changes: []UpdateOperation{
				{
					Operation: "delete",
					Target:    "World",
				},
			},
			expected: "Hello ",
		},
		{
			name:    "替换操作",
			content: "Hello World",
			changes: []UpdateOperation{
				{
					Operation: "replace",
					Target:    "World",
					Content:   "Golang",
				},
			},
			expected: "Hello Golang",
		},
		{
			name:    "全局替换操作",
			content: "Hello World, World!",
			changes: []UpdateOperation{
				{
					Operation: "replaceAll",
					Target:    "World",
					Content:   "Golang",
				},
			},
			expected: "Hello Golang, Golang!",
		},
		{
			name:    "多个操作组合",
			content: "Hello World",
			changes: []UpdateOperation{
				{
					Operation: "insert",
					Target:    "Hello",
					Content:   "Beautiful",
				},
				{
					Operation: "replace",
					Target:    "World",
					Content:   "Golang",
				},
			},
			expected: "Hello\nBeautiful Golang",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyChanges(tt.content, tt.changes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

  - [file.go](#file.go)

### file.go

package helper

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sjzsdu/tong/share"
)

// FileInfo 包含文件的基本信息
type FileInfo struct {
	Path string
	Info os.FileInfo
}

// WalkFunc 是一个回调函数类型，用于处理每个文件
type WalkFunc func(fileInfo FileInfo) error

// FilterFunc 是一个筛选函数类型，用于决定是否处理某个文件
type FilterFunc func(fileInfo FileInfo) bool

// WalkDirOptions 包含 WalkDir 的选项
type WalkDirOptions struct {
	DisableGitIgnore bool
	Extensions       []string
	Excludes         []string
}

func WalkDir(root string, callback WalkFunc, filter FilterFunc, options WalkDirOptions) error {
	gitignoreRules := make(map[string][]string)
	root = filepath.Clean(root)

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		fileInfo := FileInfo{
			Path: path,
			Info: info,
		}

		// 处理 .gitignore 规则
		if !options.DisableGitIgnore {
			if info.IsDir() {
				rules, err := ReadGitignore(path)
				if err == nil && rules != nil {
					gitignoreRules[path] = rules
				}
			}

			excluded, excludeErr := IsPathExcludedByGitignore(path, root, gitignoreRules)
			if excludeErr != nil {
				return excludeErr
			}
			if excluded {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// 检查文件扩展名
		if len(options.Extensions) > 0 {
			ext := strings.ToLower(filepath.Ext(fileInfo.Path))
			if len(ext) > 0 {
				ext = ext[1:] // 移除开头的点
			}
			if !StringSliceContains(options.Extensions, ext) && !StringSliceContains(options.Extensions, "*") {
				return nil
			}
		}

		// 应用自定义筛选函数
		if filter != nil && !filter(fileInfo) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		return callback(fileInfo)
	})
}

// FilterReadableFiles 使用 WalkDir 来过滤可读的文本文件
func FilterReadableFiles(root string, options WalkDirOptions) ([]string, error) {
	var files []string
	var count int

	startTime := time.Now()
	filter := func(fileInfo FileInfo) bool {
		count++
		if count%1000 == 0 {
			log.Printf("Processed %d files", count)
		}
		// 如果是目录，允许继续遍历，但排除 .git 目录
		if fileInfo.Info.IsDir() {
			return fileInfo.Info.Name() != ".git"
		}

		// 检查文件是否为可读文本文件
		if !isReadableTextFile(fileInfo.Path) {
			return false
		}

		// 检查文件扩展名
		if len(options.Extensions) > 0 {
			ext := strings.ToLower(filepath.Ext(fileInfo.Path))
			if len(ext) > 0 {
				ext = ext[1:] // 移除开头的点
			}
			if !StringSliceContains(options.Extensions, ext) && !StringSliceContains(options.Extensions, "*") {
				return false
			}
		}

		// 检查文件是否应被排除
		return !IsPathExcluded(fileInfo.Path, options.Excludes, root)
	}

	err := WalkDir(root, func(fileInfo FileInfo) error {
		if !fileInfo.Info.IsDir() {
			files = append(files, fileInfo.Path)
		}
		return nil
	}, filter, options)
	log.Printf("FilterReadableFiles completed: processed %d files, filtered %d readable files. Elapsed time: %v", count, len(files), time.Since(startTime))
	return files, err
}

var textExtensions = map[string]bool{
	".md": true, ".txt": true, ".log": true, ".json": true, ".xml": true, ".csv": true,
	".yml": true, ".yaml": true, ".go": true, ".py": true, ".js": true, ".ts": true,
	".html": true, ".css": true, ".java": true, ".c": true, ".cpp": true, ".h": true,
	".rb": true, ".php": true, ".sh": true, ".bat": true, ".ps1": true, ".sql": true,
	".r": true, ".scala": true, ".swift": true, ".mdx": true,
	".kt": true, ".groovy": true, ".dart": true, ".lua": true, ".perl": true,
	".hs": true, ".erl": true, ".ex": true, ".rs": true, ".fs": true, ".vb": true,
	".asm": true, ".s": true, ".pl": true, ".pm": true, ".t": true, ".pod": true,
	".ini": true, ".cfg": true, ".conf": true, ".properties": true, ".toml": true,
	".lock": true, ".env": true, ".gitignore": true, ".dockerignore": true,
	".editorconfig": true, ".eslintrc": true, ".prettierrc": true, ".babelrc": true,
	".tsv": true, ".tsx": true, ".jsx": true, ".vue": true, ".svelte": true,
	".graphql": true, ".gql": true, ".proto": true, ".thrift": true, ".avdl": true,
	".avpr": true, ".avsc": true, ".idl": true, ".puml": true, ".plantuml": true,
	".dot": true, ".gv": true, ".mmd": true, ".mermaid": true, ".sv": true,
	".v": true, ".vh": true, ".svh": true, ".vhd": true, ".vhdl": true,
	".tex": true, ".sty": true, ".cls": true, ".bib": true, ".bst": true,
	".adoc": true, ".asciidoc": true, ".rst": true, ".rest": true, ".wiki": true,
	".markdown": true, ".mdown": true, ".mkdn": true, ".mkd": true, ".mdwn": true,
	".mdtxt": true, ".mdtext": true, ".text": true, ".creole": true, ".mediawiki": true,
	".twig": true, ".j2": true, ".jinja": true, ".jinja2": true, ".njk": true,
	".ejs": true, ".haml": true, ".pug": true, ".slim": true, ".styl": true,
	".less": true, ".sass": true, ".scss": true, ".stylus": true, ".postcss": true,
	".pcss": true, ".sss": true, ".coffee": true, ".litcoffee": true, ".iced": true,
	".cson": true, ".pegjs": true, ".jison": true, ".jisonlex": true, ".lex": true,
	".y": true, ".yacc": true, ".ebnf": true, ".bnf": true, ".abnf": true,
	".peg": true, ".pegcoffee": true, ".pegiced": true, ".pegjison": true,
	".peglex": true, ".pegy": true, ".pegyacc": true, ".pegebnf": true, ".pegbnf": true,
	".pegabnf": true, ".pegpeg": true, ".pegpegjs": true, ".pegpegcoffee": true,
	".pegpegiced": true, ".pegpegjison": true, ".pegpeglex": true, ".pegpegy": true,
	".pegpegyacc": true, ".pegpegebnf": true, ".pegpegbnf": true, ".pegpegabnf": true,
}

// GetMimeType 根据文件扩展名获取 MIME 类型
func GetMimeType(ext string) string {
	switch strings.ToLower(ext) {
	case ".md", ".markdown", ".mdown", ".mkdn", ".mkd", ".mdwn", ".mdtxt", ".mdtext":
		return "text/markdown"
	case ".json":
		return "application/json"
	case ".go":
		return "text/x-go"
	case ".py":
		return "text/x-python"
	case ".js":
		return "application/javascript"
	case ".ts":
		return "application/typescript"
	case ".html":
		return "text/html"
	case ".css":
		return "text/css"
	case ".java":
		return "text/x-java"
	case ".c", ".h":
		return "text/x-c"
	case ".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx":
		return "text/x-c++"
	case ".rb":
		return "text/x-ruby"
	case ".php":
		return "application/x-httpd-php"
	case ".sh":
		return "application/x-sh"
	case ".bat":
		return "application/x-msdos-program"
	case ".ps1":
		return "application/x-powershell"
	case ".sql":
		return "application/sql"
	// 添加更多类型...
	default:
		return "text/plain"
	}
}

func isReadableTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := textExtensions[ext]; ok {
		return true
	}

	// 只对未知扩展名的文件进行内容检查
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}
	buffer = buffer[:n]

	return utf8.Valid(buffer)
}

// ReadGitignore 读取.gitignore文件并返回其中的规则
func ReadGitignore(dir string) ([]string, error) {
	gitignorePath := filepath.Join(dir, ".gitignore")
	file, err := os.Open(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var rules []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			rules = append(rules, line)
		}
	}

	return rules, scanner.Err()
}

// IsPathExcluded 检查给定路径是否应被排除
func IsPathExcluded(path string, excludes []string, rootDir string) bool {
	// 检查自定义排除规则
	for _, pattern := range excludes {
		// 使用完整路径进行匹配
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}

		// 检查相对路径
		relPath, err := filepath.Rel(rootDir, path)
		if err == nil {
			matched, err = filepath.Match(pattern, relPath)
			if err == nil && matched {
				return true
			}
		}

		// 对于包含 ** 的模式，需要特殊处理
		if strings.Contains(pattern, "**") {
			pattern = strings.ReplaceAll(pattern, "**", "*")
			if strings.Contains(path, pattern) {
				return true
			}
		}
	}

	// 检查.gitignore规则
	gitignoreRules, err := ReadGitignore(rootDir)
	if err != nil {
		// 如果读取.gitignore出错，我们就忽略它，继续处理
		return false
	}

	relPath, err := filepath.Rel(rootDir, path)
	if err != nil {
		// 如果无法获取相对路径，我们就忽略它，继续处理
		return false
	}

	for _, rule := range gitignoreRules {
		if strings.HasPrefix(rule, "/") {
			// 绝对路径规则
			if matched, _ := filepath.Match(rule[1:], relPath); matched {
				return true
			}
		} else {
			// 相对路径规则
			if matched, _ := filepath.Match(rule, filepath.Base(relPath)); matched {
				return true
			}
			if strings.Contains(relPath, rule) {
				return true
			}
		}
	}

	return false
}

func IsPathExcludedByGitignore(path, rootDir string, gitignoreRules map[string][]string) (bool, error) {
	relPath, err := filepath.Rel(rootDir, path)
	if err != nil {
		return false, err
	}

	// Check rules from all parent directories
	for dir := path; dir != rootDir && dir != "."; dir = filepath.Dir(dir) {
		if rules, ok := gitignoreRules[dir]; ok {
			for _, rule := range rules {
				if matchGitignoreRule(relPath, rule) {
					return true, nil
				}
			}
		}
	}

	// Check root directory rules last
	if rules, ok := gitignoreRules[rootDir]; ok {
		for _, rule := range rules {
			if matchGitignoreRule(relPath, rule) {
				return true, nil
			}
		}
	}

	return false, nil
}

func matchGitignoreRule(path, rule string) bool {
	// Skip empty rules
	if rule == "" {
		return false
	}

	// Handle directory-specific rules
	if rule != "" && strings.HasSuffix(rule, "/") {
		rule = rule[:len(rule)-1]
	}

	if strings.HasPrefix(rule, "/") {
		// Absolute path rule
		matched, _ := filepath.Match(rule[1:], path)
		return matched
	} else {
		// Relative path rule
		base := filepath.Base(path)
		matched, _ := filepath.Match(rule, base)
		if matched {
			return true
		}

		// Check if rule matches any path component
		components := strings.Split(path, string(filepath.Separator))
		for _, comp := range components {
			if matched, _ := filepath.Match(rule, comp); matched {
				return true
			}
		}
		return false
	}
}

func GetPath(subPath string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "/"
	}

	configFile := filepath.Join(homeDir, share.PATH, subPath)

	return configFile
}

func GetAbsPath(path string) (string, error) {
	if path == "" || !filepath.IsAbs(path) {
		currentDir, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("error getting current directory: %v", err)
		}
		return filepath.Join(currentDir, path), nil
	}
	return filepath.Clean(path), nil
}

func CheckFilesExist(files string) error {
	if files == "." {
		return nil
	}

	for _, file := range strings.Split(files, ",") {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", file)
		}
	}
	return nil
}

func GetFileExt(file string) string {
	ext := filepath.Ext(file)
	if len(ext) <= 1 {
		return ""
	}
	return ext[1:]
}

// WriteFileContent 将内容写入指定文件
func WriteFileContent(file string, content string) error {
	// 获取文件的绝对路径
	absPath, err := GetAbsPath(file)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// 写入内容到文件
	err = os.WriteFile(absPath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}

func GetFileContent(file string) (string, error) {
	// Get the absolute path of the file
	absPath, err := GetAbsPath(file)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %v", err)
	}

	// 检查文件是否存在
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return "", nil
	}

	// Read the content from the file
	fileContent, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	// Return the file content as a string
	return string(fileContent), nil
}

  - [file_test.go](#file_test.go)

### file_test.go

package helper

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupTestDir(t *testing.T) string {
	log.Println("Setting up test directory")
	tempDir := t.TempDir()
	os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("test file 1"), 0644)
	os.WriteFile(filepath.Join(tempDir, "file2.log"), []byte("test file 2"), 0644)
	os.WriteFile(filepath.Join(tempDir, "subdir", "file3.txt"), []byte("test file 3"), 0644)
	os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte("*.log\n"), 0644)
	log.Println("Test directory setup complete")

	// 创建 .gitignore 文件
	gitignoreContent := []byte("*.log\n")
	err := os.WriteFile(filepath.Join(tempDir, ".gitignore"), gitignoreContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create .gitignore file: %v", err)
	}
	return tempDir
}

func TestWalkDir(t *testing.T) {
	tempDir := setupTestDir(t)

	tests := []struct {
		name     string
		options  WalkDirOptions
		expected []string
	}{
		{
			name: "正常遍历所有文件",
			options: WalkDirOptions{
				DisableGitIgnore: true,
			},
			expected: []string{
				filepath.Join(tempDir, "file1.txt"),
				filepath.Join(tempDir, "file2.log"),
				filepath.Join(tempDir, "subdir", "file3.txt"),
				filepath.Join(tempDir, ".gitignore"),
			},
		},
		{
			name: "启用.gitignore规则",
			options: WalkDirOptions{
				DisableGitIgnore: false,
			},
			expected: []string{
				filepath.Join(tempDir, "file1.txt"),
				filepath.Join(tempDir, "subdir", "file3.txt"),
				filepath.Join(tempDir, ".gitignore"),
			},
		},
		{
			name: "仅筛选特定扩展名文件",
			options: WalkDirOptions{
				DisableGitIgnore: true,
				Extensions:       []string{"txt"},
			},
			expected: []string{
				filepath.Join(tempDir, "file1.txt"),
				filepath.Join(tempDir, "subdir", "file3.txt"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result []string
			done := make(chan bool)
			go func() {
				log.Printf("Starting test: %s", tt.name)
				err := WalkDir(tempDir, func(fileInfo FileInfo) error {
					if !fileInfo.Info.IsDir() {
						result = append(result, fileInfo.Path)
					}
					return nil
				}, nil, tt.options)
				assert.NoError(t, err)
				log.Printf("Finished test: %s", tt.name)
				done <- true
			}()

			<-done
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestFilterReadableFiles(t *testing.T) {
	tempDir := setupTestDir(t)

	tests := []struct {
		name     string
		options  WalkDirOptions
		expected []string
	}{
		{
			name: "筛选可读文本文件",
			options: WalkDirOptions{
				DisableGitIgnore: true,
			},
			expected: []string{
				filepath.Join(tempDir, "file1.txt"),
				filepath.Join(tempDir, "subdir", "file3.txt"),
				filepath.Join(tempDir, ".gitignore"),
			},
		},
		{
			name: "启用.gitignore规则",
			options: WalkDirOptions{
				DisableGitIgnore: false,
			},
			expected: []string{
				filepath.Join(tempDir, "file1.txt"),
				filepath.Join(tempDir, "subdir", "file3.txt"),
				filepath.Join(tempDir, ".gitignore"),
			},
		},
		{
			name: "筛选特定扩展名文件",
			options: WalkDirOptions{
				DisableGitIgnore: true,
				Extensions:       []string{"txt"},
			},
			expected: []string{
				filepath.Join(tempDir, "file1.txt"),
				filepath.Join(tempDir, "subdir", "file3.txt"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log.Printf("Starting test: %s", tt.name)
			done := make(chan bool)
			var result []string
			var err error

			go func() {
				result, err = FilterReadableFiles(tempDir, tt.options)
				done <- true
			}()

			<-done
			assert.NoError(t, err)
			assert.ElementsMatch(t, tt.expected, result)
			log.Printf("Finished test: %s", tt.name)
		})
	}
}

  - [font.go](#font.go)

### font.go

package helper

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

func FindFont() (string, error) {
	fontPaths := []string{
		"/System/Library/Fonts/Songti.ttc", // 常用中文字体
		"/System/Library/Fonts/SimSun.ttf", // 常用中文字体
		"/System/Library/Fonts/SimHei.ttf", // 常用中文字体
		"/System/Library/Fonts/PingFang.ttc",
		"/Library/Fonts/Arial Unicode.ttf",
		"/System/Library/Fonts/STHeiti Light.ttc",
		"/System/Library/Fonts/STHeiti Medium.ttc",
	}

	for _, path := range fontPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no suitable font found")
}

//go:embed fonts/*.ttf
var embeddedFonts embed.FS

const FONT_NAME = "FangZhengFangSongJianTi-1"

func UseEmbeddedFont(fontName string) (string, error) {
	if fontName == "" {
		fontName = FONT_NAME
	}
	// 修正字体路径，需要包含 fonts 目录
	fontPath := fmt.Sprintf("fonts/%s.ttf", fontName)

	// 添加调试信息
	entries, err := embeddedFonts.ReadDir("fonts")
	if err != nil {
		return "", fmt.Errorf("无法读取字体目录: %v", err)
	}

	// 检查目录中的文件
	var foundFont bool
	for _, entry := range entries {
		fmt.Printf("Found font file: %s\n", entry.Name()) // 添加调试输出
		if entry.Name() == fmt.Sprintf("%s.ttf", fontName) {
			foundFont = true
			break
		}
	}

	if !foundFont {
		return "", fmt.Errorf("字体文件 %s.ttf 不存在于嵌入的目录中", fontName)
	}

	data, err := embeddedFonts.ReadFile(fontPath)
	if err != nil {
		return "", fmt.Errorf("读取嵌入字体失败: %v", err)
	}

	if len(data) == 0 {
		return "", fmt.Errorf("字体文件为空")
	}

	localFontPath := filepath.Join(os.TempDir(), fontName+".ttf")
	err = os.WriteFile(localFontPath, data, 0644)
	if err != nil {
		return "", fmt.Errorf("写入字体到临时目录失败: %v", err)
	}

	return localFontPath, nil
}

  - [font_test.go](#font_test.go)

### font_test.go

package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindFont(t *testing.T) {
	fontPath, err := FindFont()
	
	// 在 MacOS 系统上应该能找到至少一个字体
	if assert.NoError(t, err) {
		assert.NotEmpty(t, fontPath)
		assert.Contains(t, []string{
			"/System/Library/Fonts/PingFang.ttc",
			"/Library/Fonts/Arial Unicode.ttf",
			"/System/Library/Fonts/STHeiti Light.ttc",
			"/System/Library/Fonts/STHeiti Medium.ttc",
		}, fontPath)
	}
}
  - [git.go](#git.go)

### git.go

package helper

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
)

// CloneProject 克隆指定的Git仓库到临时目录并返回克隆的路径
func CloneProject(gitURL string) (string, error) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "git-clone-")
	if err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}

	// 克隆仓库
	_, err = git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:      gitURL,
		Progress: os.Stdout, // 显示克隆进度
	})
	if err != nil {
		os.RemoveAll(tempDir) // 清理临时目录
		return "", fmt.Errorf("克隆仓库失败: %w", err)
	}

	fmt.Printf("仓库已成功克隆到临时目录: %s\n", tempDir)

	// 返回克隆的路径
	return tempDir, nil
}

  - [net.go](#net.go)

### net.go

package helper

import "strconv"

func IsValidPort(port string) bool {
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return false
	}
	return portNum > 0 && portNum <= 65535
}

  - [progress.go](#progress.go)

### progress.go

package helper

import (
	"fmt"
	"strings"
	"sync/atomic"
)

type Progress struct {
	total   int
	current int64
	width   int
	title   string
}

func NewProgress(title string, total int) *Progress {
	return &Progress{
		total: total,
		width: 50, // 进度条宽度
		title: title,
	}
}

func (p *Progress) Increment() {
	atomic.AddInt64(&p.current, 1)
	p.render()
}

// Show 显示当前进度，不增加计数
func (p *Progress) Show() {
	p.render()
}

func (p *Progress) render() {
	percent := float64(p.current) / float64(p.total) * 100
	filled := int(percent / 2) // Each "=" represents 2%

	// Ensure filled is never negative
	if filled < 0 {
		filled = 0
	}

	// Calculate remaining, ensuring it's never negative
	remaining := 50 - filled // Total width is 50
	if remaining < 0 {
		remaining = 0
	}

	bar := strings.Repeat("=", filled) + strings.Repeat(" ", remaining)
	fmt.Printf("\r%s [%s] %.1f%% (%d/%d)",
		p.title, bar, percent, p.current, p.total)
}

  - [renderer.go](#renderer.go)

### renderer.go

package helper

import (
	"fmt"
	"os"

	"github.com/sjzsdu/tong/helper/renders"
	"github.com/sjzsdu/tong/share"
)

func GetDefaultRenderer() renders.Renderer {
	render := os.Getenv("WN_RENDER")
	if render == "" {
		render = share.DEFAULT_RENDERER
	}
	return NewRenderer(render)
}

func NewRenderer(renderer string) renders.Renderer {
	switch renderer {
	case "text":
		return renders.NewTextRenderer()
	case "markdown":
		render, err := renders.NewMarkdownRenderer()
		if err != nil {
			fmt.Printf("初始化 Markdown 渲染器失败: %v，将使用文本渲染器\n", err)
			return renders.NewTextRenderer()
		}
		return render
	default:
		return renders.NewTextRenderer()
	}
}

  - [renders](#renders)

### renders

    - [markdown.go](#markdown.go)

#### markdown.go

package renders

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/sjzsdu/tong/lang"
)

// MarkdownRenderer 实现 Renderer 接口，提供 Markdown 渲染功能
type MarkdownRenderer struct {
	renderer    *glamour.TermRenderer
	buffer      strings.Builder
	mu          sync.Mutex
	isOutputing bool
}

// NewMarkdownRenderer 创建一个新的 Markdown 渲染器
func NewMarkdownRenderer() (*MarkdownRenderer, error) {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(120),
	)
	if err != nil {
		return nil, fmt.Errorf("初始化 Markdown 渲染器失败: %v", err)
	}

	return &MarkdownRenderer{
		renderer:    renderer,
		buffer:      strings.Builder{},
		isOutputing: false,
	}, nil
}

// WriteStream 实现 Renderer 接口，将内容写入缓冲区
// 如果是第一次写入，会显示 "output...." 提示
func (m *MarkdownRenderer) WriteStream(content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 第一次写入时显示提示
	if !m.isOutputing {
		fmt.Print(lang.T("Preparing..."))
		m.isOutputing = true
	}

	// 将内容添加到缓冲区
	m.buffer.WriteString(content)
	return nil
}

// Done 实现 Renderer 接口，完成输出并渲染 Markdown
func (m *MarkdownRenderer) Done() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果没有开始输出，直接返回
	if !m.isOutputing {
		return
	}

	// 清除 "output...." 提示
	fmt.Print("\r                \r")

	// 获取缓冲区内容
	content := m.buffer.String()
	if content == "" {
		m.reset()
		return
	}

	// 渲染 Markdown 内容
	rendered, err := m.renderer.Render(content)
	if err != nil {
		// 渲染失败时，输出原始内容
		fmt.Print(content)
	} else {
		// 处理渲染结果，移除多余空行
		rendered = strings.TrimSpace(rendered)
		// 将连续的多个空行替换为单个空行
		for strings.Contains(rendered, "\n\n\n") {
			rendered = strings.ReplaceAll(rendered, "\n\n\n", "\n\n")
		}
		fmt.Print(rendered)
	}

	// 重置状态
	m.reset()
}

// reset 重置渲染器状态
func (m *MarkdownRenderer) reset() {
	m.buffer.Reset()
	m.isOutputing = false
}

    - [text.go](#text.go)

#### text.go

package renders

import (
	"fmt"
)

// TextRenderer 实现 Renderer 接口，提供纯文本渲染功能
type TextRenderer struct {
}

// NewTextRenderer 创建一个新的文本渲染器
func NewTextRenderer() *TextRenderer {
	return &TextRenderer{}
}

// WriteStream 实现 Renderer 接口，将内容写入缓冲区
// 如果是第一次写入，会显示 "output...." 提示
func (t *TextRenderer) WriteStream(content string) error {
	fmt.Print(content)
	return nil
}

// Done 实现 Renderer 接口，完成输出并显示文本
func (t *TextRenderer) Done() {
	fmt.Println()
}

    - [type.go](#type.go)

#### type.go

package renders

// Renderer 定义了通用的渲染器接口
type Renderer interface {
	// 输出文字
	WriteStream(content string) error
	// 完成输出
	Done()
}

  - [root.go](#root.go)

### root.go

package helper

import (
	"fmt"
)

func GetTargetPath(cmdPath string, gitURL string) (string, error) {
	var targetPath string

	if gitURL != "" {
		// 创建临时目录
		tempDir, err := CloneProject(gitURL)
		if err != nil {
			return "", fmt.Errorf("error cloning repository: %v", err)
		}
		targetPath = tempDir
	} else {
		absPath, err := GetAbsPath(cmdPath)
		if err != nil {
			return "", err
		}
		targetPath = absPath
	}

	return targetPath, nil
}

  - [server.go](#server.go)

### server.go

package helper

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"os/exec"
	"sync"
)

var (
	content   string
	contentMu sync.RWMutex
	server    *http.Server
	etag      string // Add etag variable
)

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Blog Preview</title>
    <link rel="stylesheet" href="https://cdn.bootcdn.net/ajax/libs/github-markdown-css/5.2.0/github-markdown.min.css">
<script src="https://cdn.bootcdn.net/ajax/libs/markdown-it/12.0.4/markdown-it.min.js"></script>
<script src="https://cdn.bootcdn.net/ajax/libs/markdown-it-emoji/2.0.0/markdown-it-emoji.min.js"></script>
<script src="https://cdn.bootcdn.net/ajax/libs/markdown-it-footnote/3.0.3/markdown-it-footnote.min.js"></script>
<script src="https://cdn.bootcdn.net/ajax/libs/markdown-it-task-lists/2.1.1/markdown-it-task-lists.min.js"></script>
<script src="https://cdn.bootcdn.net/ajax/libs/markdown-it-anchor/8.4.1/markdown-it-anchor.min.js"></script>
<script src="https://cdn.bootcdn.net/ajax/libs/markdown-it-toc-done-right/4.1.0/markdown-it-toc-done-right.min.js"></script>
<script src="https://cdn.bootcdn.net/ajax/libs/mermaid/10.2.0/mermaid.min.js"></script>
<script src="https://cdn.bootcdn.net/ajax/libs/KaTeX/0.16.4/katex.min.js"></script>
<script src="https://cdn.bootcdn.net/ajax/libs/markdown-it-texmath/0.9.0/texmath.min.js"></script>
<link rel="stylesheet" href="https://cdn.bootcdn.net/ajax/libs/KaTeX/0.16.4/katex.min.css">
    <style>
        body {
            box-sizing: border-box;
            min-width: 200px;
            max-width: 980px;
            margin: 0 auto;
            padding: 45px;
            background-color: #f6f8fa;
        }
        .markdown-body {
            background-color: white;
            border-radius: 6px;
            padding: 20px;
            margin: 20px 0;
            box-shadow: 0 1px 3px rgba(0,0,0,0.12);
        }
        pre {
            background-color: #f6f8fa;
            border-radius: 6px;
            padding: 16px;
        }
    </style>
</head>
<body>
    <div class="markdown-body" id="content"></div>
    <script>
        // 初始化 markdown-it
        const md = window.markdownit({
            html: true,
            linkify: true,
            typographer: true,
            breaks: true,
            highlight: function (str, lang) {
                if (lang && lang === 'mermaid') {
                    return '<div class="mermaid">' + str + '</div>';
                }
                return '';
            }
        });

        // 确保插件已经加载后再使用
        if (window.markdownitEmoji) {
            md.use(window.markdownitEmoji);
        }
        if (window.markdownitFootnote) {
            md.use(window.markdownitFootnote);
        }
        if (window.markdownitTaskLists) {
            md.use(window.markdownitTaskLists);
        }
        if (window.markdownitAnchor) {
            md.use(window.markdownitAnchor);
        }
        if (window.markdownitTocDoneRight) {
            md.use(window.markdownitTocDoneRight);
        }
        if (window.texmath && window.katex) {
            md.use(window.texmath, { engine: window.katex });
        }

        // 初始化 Mermaid
        mermaid.initialize({
            startOnLoad: true,
            theme: 'default',
            securityLevel: 'loose',
            flowchart: {
                useMaxWidth: true,
                htmlLabels: true,
                curve: 'basis'
            }
        });

        // 渲染内容
        function renderContent(markdown) {
            const contentDiv = document.getElementById('content');
            
            // 渲染 Markdown
            contentDiv.innerHTML = md.render(markdown);
            
            // 重新初始化 Mermaid
            mermaid.init(undefined, document.querySelectorAll('.mermaid'));
        }

        let lastEtag = '';

        // 自动刷新内容
        function refreshContent() {
            fetch('/content', {
                headers: {
                    'If-None-Match': lastEtag
                }
            })
            .then(response => {
                if (response.status === 304) {
                    return null;
                }
                lastEtag = response.headers.get('ETag') || '';
                return response.text();
            })
            .then(newContent => {
                if (newContent) {
                    renderContent(newContent);
                }
            })
            .catch(error => {
                console.error('Failed to fetch content:', error);
            });
        }

        // 初始化内容
        refreshContent();

        // 每秒刷新一次
        setInterval(refreshContent, 1000);
    </script>
</body>
</html>
`

func StartPreviewServer(port int) string {
	tmpl := template.Must(template.New("preview").Parse(htmlTemplate))

	// 处理基础 HTML 页面请求
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, nil)
	})

	// 处理内容请求
	http.HandleFunc("/content", func(w http.ResponseWriter, r *http.Request) {
		contentMu.RLock()
		currentContent := content
		currentEtag := etag
		contentMu.RUnlock()

		// 检查内容是否有变化
		if r.Header.Get("If-None-Match") == currentEtag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("ETag", currentEtag)
		w.Write([]byte(currentContent))
	})

	server = &http.Server{
		Addr: fmt.Sprintf(":%d", port),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	url := fmt.Sprintf("http://localhost:%d", port)
	// 在默认浏览器中打开 URL
	go func() {
		cmd := exec.Command("open", url)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Failed to open browser: %v\n", err)
		}
	}()
	// OSC 8 格式：\033]8;;URL\007TEXT\033]8;;\007
	return fmt.Sprintf("\033]8;;%s\007%s\033]8;;\007", url, url)
}

func UpdatePreviewContent(newContent string) {
	contentMu.Lock()
	content = newContent
	// Generate MD5 hash as ETag
	hash := md5.Sum([]byte(newContent))
	etag = hex.EncodeToString(hash[:])
	contentMu.Unlock()
}

// 添加关闭服务器的函数
func StopPreviewServer() error {
	if server != nil {
		return server.Close()
	}
	return nil
}

  - [str.go](#str.go)

### str.go

package helper

import (
	"encoding/json"
	"math/rand"
	"regexp"
	"strings"
)

// StringSliceContains 检查切片中是否包含指定的字符串
func StringSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// 添加新的辅助函数来清理 ANSI 转义序列
// 修改 stripAnsiCodes 函数，确保正确处理 git diff 输出
func StripAnsiCodes(s string) string {
	// 处理 git diff 常见的颜色代码和格式控制符
	ansi := regexp.MustCompile(`\x1b\[[0-9;]*[mGKHF]`)
	return strings.TrimSpace(ansi.ReplaceAllString(s, ""))
}

// randomString 生成指定长度的随机字符串
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func StripHTMLTags(text string) string {
	var result strings.Builder
	var inTag bool

	for _, r := range text {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			result.WriteRune(r)
		}
	}

	return strings.TrimSpace(result.String())
}

func SubString(str string, count int) string {
	runes := []rune(str)
	if len(runes) > count {
		return string(runes[:count]) + "..."
	}
	return str
}

// StringToMap 将字符串转换为 map[string]interface{}
func StringToMap(s string) map[string]interface{} {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return make(map[string]interface{})
	}
	return result
}

// ToJSONString 将任意的 map、slice 或 struct 转换为字符串
func ToJSONString(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}

  - [str_test.go](#str_test.go)

### str_test.go

package helper

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringSliceContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "空切片",
			slice:    []string{},
			item:     "test",
			expected: false,
		},
		{
			name:     "包含目标字符串",
			slice:    []string{"test1", "test2", "test3"},
			item:     "test2",
			expected: true,
		},
		{
			name:     "不包含目标字符串",
			slice:    []string{"test1", "test2", "test3"},
			item:     "test4",
			expected: false,
		},
		{
			name:     "空字符串测试",
			slice:    []string{"test1", "", "test3"},
			item:     "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringSliceContains(tt.slice, tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"空字符串", 0},
		{"8位字符串", 8},
		{"16位字符串", 16},
		{"32位字符串", 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := randomString(tt.length)

			// 检查长度是否符合预期
			if len(got) != tt.length {
				t.Errorf("randomString() 长度 = %v, 期望长度 %v", len(got), tt.length)
			}

			// 检查字符是否都在允许的范围内
			pattern := "^[a-zA-Z0-9]*$"
			matched, _ := regexp.MatchString(pattern, got)
			if !matched {
				t.Errorf("randomString() = %v, 包含非法字符", got)
			}

			// 生成多个字符串检查是否有重复（当长度大于0时）
			if tt.length > 0 {
				results := make(map[string]bool)
				for i := 0; i < 100; i++ {
					str := randomString(tt.length)
					results[str] = true
				}
				// 检查是否生成了不同的字符串（允许少量重复）
				if len(results) < 90 {
					t.Errorf("randomString() 生成的随机字符串重复率过高")
				}
			}
		})
	}
}

  - [struct.go](#struct.go)

### struct.go

package helper

import (
	"encoding/json"
	"fmt"
	"reflect"
)

func MapToStruct[T any](data map[string]interface{}) (*T, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal map to json: %w", err)
	}

	var result T
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, fmt.Errorf("unmarshal json to struct: %w", err)
	}

	return &result, nil
}

func MergeStruct[T any](base T, override T) T {
	baseValue := reflect.ValueOf(&base).Elem()
	overrideValue := reflect.ValueOf(override)

	for i := 0; i < baseValue.NumField(); i++ {
		field := baseValue.Field(i)
		overrideField := overrideValue.Field(i)

		if !overrideField.IsZero() {
			if overrideField.Kind() == reflect.Struct {
				// Get the concrete type of the nested struct
				baseField := field.Interface()

				// Create a new merged value using reflection
				mergedValue := reflect.ValueOf(baseField)
				merged := reflect.New(mergedValue.Type()).Elem()
				merged.Set(mergedValue)

				// Iterate through the nested struct fields
				for j := 0; j < overrideField.NumField(); j++ {
					if !overrideField.Field(j).IsZero() {
						merged.Field(j).Set(overrideField.Field(j))
					}
				}

				field.Set(merged)
			} else {
				field.Set(overrideField)
			}
		}
	}

	return base
}

  - [struct_test.go](#struct_test.go)

### struct_test.go

package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// 测试用的结构体
type TestStruct struct {
	Name    string   `json:"name"`
	Age     int      `json:"age"`
	Hobbies []string `json:"hobbies"`
	Info    struct {
		City    string `json:"city"`
		Country string `json:"country"`
	} `json:"info"`
}

func TestMapToStruct(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]interface{}
		want    *TestStruct
		wantErr bool
	}{
		{
			name: "正常转换",
			input: map[string]interface{}{
				"name": "张三",
				"age":  25,
				"hobbies": []string{
					"读书",
					"游泳",
				},
				"info": map[string]interface{}{
					"city":    "北京",
					"country": "中国",
				},
			},
			want: &TestStruct{
				Name: "张三",
				Age:  25,
				Hobbies: []string{
					"读书",
					"游泳",
				},
				Info: struct {
					City    string `json:"city"`
					Country string `json:"country"`
				}{
					City:    "北京",
					Country: "中国",
				},
			},
			wantErr: false,
		},
		{
			name: "类型不匹配",
			input: map[string]interface{}{
				"name": "张三",
				"age":  "25", // 错误的类型：字符串而不是整数
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "空输入",
			input:   map[string]interface{}{},
			want:    &TestStruct{},
			wantErr: false,
		},
		{
			name: "嵌套结构体字段缺失",
			input: map[string]interface{}{
				"name": "李四",
				"age":  30,
				"hobbies": []string{
					"跑步",
					"游泳",
				},
				"info": map[string]interface{}{
					"city": "上海",
				},
			},
			want: &TestStruct{
				Name: "李四",
				Age:  30,
				Hobbies: []string{
					"跑步",
					"游泳",
				},
				Info: struct {
					City    string `json:"city"`
					Country string `json:"country"`
				}{
					City: "上海",
				},
			},
			wantErr: false,
		},
		{
			name: "数组字段为空",
			input: map[string]interface{}{
				"name": "王五",
				"age":  35,
				"hobbies": []string{},
				"info": map[string]interface{}{
					"city":    "广州",
					"country": "中国",
				},
			},
			want: &TestStruct{
				Name:    "王五",
				Age:     35,
				Hobbies: []string{},
				Info: struct {
					City    string `json:"city"`
					Country string `json:"country"`
				}{
					City:    "广州",
					Country: "中国",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapToStruct[TestStruct](tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
- [lang](#lang)

## lang

  - [i18n.go](#i18n.go)

### i18n.go

package lang

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var (
	bundle *i18n.Bundle
	loc    *i18n.Localizer
	// LocalePath 用于配置语言文件的路径
	LocalePath = "lang/locales"
)

// Init initializes the i18n system
func init() {
	SetupI18n("")
}

// SetupI18n 设置国际化配置并初始化语言
func SetupI18n(localePath string) {
	if localePath != "" {
		LocalePath = localePath
	}

	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	// 获取环境变量设置的语言
	lang := os.Getenv("TONG_LANG")
	if lang == "" {
		return
	}

	// 标准化语言代码
	switch lang {
	case "zh":
		lang = "zh-CN"
	case "cn":
		lang = "zh-CN"
	case "tw":
		lang = "zh-TW"
	}

	// 检查对应的语言文件是否存在
	langFile := filepath.Join(LocalePath, lang+".json")
	if _, err := os.Stat(langFile); err == nil {
		bundle.MustLoadMessageFile(langFile)
		loc = i18n.NewLocalizer(bundle, lang)
	}
}

// T translates a message, optionally with template data
func T(msgID string, data ...map[string]interface{}) string {
	// 如果未初始化 localizer，直接返回原始键
	if loc == nil {
		return msgID
	}

	config := &i18n.LocalizeConfig{
		MessageID: msgID,
	}

	// 如果提供了模板数据，则添加到配置中
	if len(data) > 0 {
		config.TemplateData = data[0]
	}

	msg, err := loc.Localize(config)
	if err != nil {
		// 如果翻译出错（比如键不存在），返回原始键
		return msgID
	}
	return msg
}

  - [i18n_test.go](#i18n_test.go)

### i18n_test.go

package lang

import (
	"os"
	"path/filepath"
	"testing"
)

func resetEnv() {
	os.Unsetenv("TONG_LANG")
	loc = nil
}

func TestI18n(t *testing.T) {
	// 设置测试用的语言文件路径
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	testLocalePath := filepath.Join(pwd, "..", "lang", "locales")

	// 每个测试前重置环境并设置路径
	resetEnv()
	SetupI18n(testLocalePath)

	// 测试默认语言（英文）
	if msg := T("test message"); msg != "test message" {
		t.Errorf("Expected original message, got %s", msg)
	}

	// 测试简体中文
	resetEnv()
	os.Setenv("TONG_LANG", "zh-CN")
	SetupI18n(testLocalePath)
	if msg := T("Pack files"); msg != "打包文件" {
		t.Errorf("Expected '打包文件', got %s", msg)
	}

	// 测试繁体中文
	resetEnv()
	os.Setenv("TONG_LANG", "zh-TW")
	SetupI18n(testLocalePath)
	if msg := T("Pack files"); msg != "打包文件" {
		t.Errorf("Expected '打包文件', got %s", msg)
	}

	// 测试不存在的语言
	resetEnv()
	os.Setenv("TONG_LANG", "fr")
	SetupI18n(testLocalePath)
	if msg := T("Pack files"); msg != "Pack files" {
		t.Errorf("Expected original message, got %s", msg)
	}

	// 测试不存在的翻译键
	resetEnv()
	os.Setenv("TONG_LANG", "zh-CN")
	SetupI18n(testLocalePath)
	if msg := T("non-existent key"); msg != "non-existent key" {
		t.Errorf("Expected original message, got %s", msg)
	}

	// 测试语言代码别名
	tests := []struct {
		lang     string
		message  string
		expected string
	}{
		{"zh", "Pack files", "打包文件"},
		{"cn", "Pack files", "打包文件"},
		{"tw", "Pack files", "打包文件"},
	}

	for _, test := range tests {
		resetEnv()
		os.Setenv("TONG_LANG", test.lang)
		SetupI18n(testLocalePath)
		if msg := T(test.message); msg != test.expected {
			t.Errorf("Lang %s: Expected %s, got %s", test.lang, test.expected, msg)
		}
	}
}

  - [locales](#locales)

### locales

    - [zh-CN.json](#zh-cn.json)

#### zh-CN.json

{
    "One or more arguments are not correct": "参数错误",
    "work directory": "工作目录",
    "Pack files": "打包文件",
    "Pack files with specified extensions into a single output file": "将指定扩展名的文件打包成单个输出文件",
    "File extensions to include": "要包含的文件扩展名",
    "Output file name": "输出文件名",
    "Glob patterns to exclude": "要排除的文件模式",
    "Git repository URL to clone and pack": "要克隆和打包的Git仓库URL",
    "Disable .gitignore rules": "禁用.gitignore规则",
    "Print version information": "打印版本信息",
    "Print detailed version information of tong": "打印 tong 的详细版本信息",
    "Set config": "设置配置",
    "Set global configuration": "设置全局配置",
    "Set language": "设置语言",
    "Set llm response render type": "设置 AI 回复的渲染方式",
    "Set default LLM provider": "设置默认大模型提供商",
    "Set default agent": "设置默认 agent",
    "Set DeepSeek API Key": "设置 DeepSeek API 密钥",
    "Set DeepSeek default model": "设置 DeepSeek 默认模型",
    "Set Openai API Key": "设置 OpenAI API 密钥",
    "Set Openai default model": "设置 OpenAI 默认模型",
    "Set Claude API Key": "设置 Claude API 密钥",
    "Set Claude default model": "设置 Claude 默认模型",
    "List all configurations": "列出所有配置",
    "Current configurations": "当前配置",
    "mcp commands": "MCP 命令",
    "mcp server and client commands for this project": "项目的 MCP 服务端和客户端命令",
    "start mcp server": "启动 MCP 服务器",
    "start mcp server with specified configuration": "使用指定配置启动 MCP 服务器",
    "start mcp client": "启动 MCP 客户端",
    "start mcp client with specified configuration": "使用指定配置启动 MCP 客户端",
    "MCP transfer layer": "MCP 传输层",
    "MCP sse port": "MCP SSE 端口",
    "MCP server command": "MCP 服务器命令",
    "MCP server environtment": "MCP 服务器环境变量",
    "MCP server command arguments": "MCP 服务器命令参数",
    "Preparing...": "准备中...",
    "Chat with AI": "AI 对话",
    "Start an interactive chat session with AI using configured LLM provider": "启动一个基于配置的 AI 大模型互动式对话会话",
    "LLM model Provider": "大模型提供商",
    "LLM model to use": "使用的大模型",
    "Maximum tokens for response": "回应的最大 token 数",
    "List available LLM providers": "列出可用的大模型提供商",
    "List available models for current provider": "列出当前提供商支持的模型",
    "AI use agent name": "AI 使用的 agent 名称"
}

    - [zh-TW.json](#zh-tw.json)

#### zh-TW.json

{
    "One or more arguments are not correct": "參數錯誤",
    "work directory": "工作目錄",
    "Pack files": "打包文件",
    "Pack files with specified extensions into a single output file": "將指定擴展名的文件打包成單個輸出文件",
    "File extensions to include": "要包含的文件擴展名",
    "Output file name": "輸出文件名",
    "Glob patterns to exclude": "要排除的文件模式",
    "Git repository URL to clone and pack": "要克隆和打包的Git倉庫URL",
    "Disable .gitignore rules": "禁用.gitignore規則",
    "Print version information": "打印版本信息",
    "Print detailed version information of tong": "打印 tong 的詳細版本信息",
    "Set config": "設置配置",
    "Set global configuration": "設置全局配置",
    "Set language": "設置語言",
    "Set llm response render type": "設置 AI 回復的渲染方式",
    "Set default LLM provider": "設置默認大模型提供商",
    "Set default agent": "設置默認 agent",
    "Set DeepSeek API Key": "設置 DeepSeek API 密鑰",
    "Set DeepSeek default model": "設置 DeepSeek 默認模型",
    "Set Openai API Key": "設置 OpenAI API 密鑰",
    "Set Openai default model": "設置 OpenAI 默認模型",
    "Set Claude API Key": "設置 Claude API 密鑰",
    "Set Claude default model": "設置 Claude 默認模型",
    "List all configurations": "列出所有配置",
    "Current configurations": "當前配置",
    "mcp commands": "MCP 命令",
    "mcp server and client commands for this project": "項目的 MCP 服務端和客戶端命令",
    "start mcp server": "啟動 MCP 服務器",
    "start mcp server with specified configuration": "使用指定配置啟動 MCP 服務器",
    "start mcp client": "啟動 MCP 客戶端",
    "start mcp client with specified configuration": "使用指定配置啟動 MCP 客戶端",
    "MCP transfer layer": "MCP 傳輸層",
    "MCP sse port": "MCP SSE 端口",
    "MCP server command": "MCP 服務器命令",
    "MCP server environtment": "MCP 服務器環境變量",
    "MCP server command arguments": "MCP 服務器命令參數",
    "Preparing...": "準備中...",
    "Chat with AI": "AI 對話",
    "Start an interactive chat session with AI using configured LLM provider": "啟動一個基於配置的 AI 大模型互動式對話會話",
    "LLM model Provider": "大模型提供商",
    "LLM model to use": "使用的大模型",
    "Maximum tokens for response": "回應的最大 token 數",
    "List available LLM providers": "列出可用的大模型提供商",
    "List available models for current provider": "列出當前提供商支持的模型",
    "AI use agent name": "AI 使用的 agent 名稱"
}
- [main.go](#main.go)

## main.go

package main

import (
	"github.com/sjzsdu/tong/cmd"
)

func main() {
	cmd.Execute()
}

- [project](#project)

## project

  - [analyzer](#analyzer)

### analyzer

    - [code_analyzer.go](#code_analyzer.go)

#### code_analyzer.go

package analyzer

import (
	"bufio"
	"bytes"
	"path/filepath"
	"strings"

	"github.com/sjzsdu/tong/project"
)

// CodeStats 代码统计信息
type CodeStats struct {
	TotalFiles      int            // 文件总数
	TotalDirs       int            // 目录总数
	TotalLines      int            // 代码总行数
	TotalSize       int64          // 总大小（字节）
	LanguageStats   map[string]int // 各语言代码行数统计
	FileTypeStats   map[string]int // 各文件类型统计
	ComplexityStats map[string]int // 复杂度统计（可选）
}

// CodeAnalyzer 代码分析器接口
type CodeAnalyzer interface {
	// 分析代码并返回统计信息
	Analyze(project *project.Project) (*CodeStats, error)
}

// DefaultCodeAnalyzer 默认代码分析器实现
type DefaultCodeAnalyzer struct {
	// 语言扩展名映射
	languageMap map[string]string
}

// NewDefaultCodeAnalyzer 创建一个新的默认代码分析器
func NewDefaultCodeAnalyzer() *DefaultCodeAnalyzer {
	return &DefaultCodeAnalyzer{
		languageMap: map[string]string{
			"go":   "Go",
			"py":   "Python",
			"js":   "JavaScript",
			"ts":   "TypeScript",
			"java": "Java",
			"c":    "C",
			"cpp":  "C++",
			"h":    "C/C++ Header",
			"hpp":  "C++ Header",
			"cs":   "C#",
			"php":  "PHP",
			"rb":   "Ruby",
			"swift":"Swift",
			"kt":   "Kotlin",
			"rs":   "Rust",
			"html": "HTML",
			"css":  "CSS",
			"scss": "SCSS",
			"sass": "Sass",
			"less": "Less",
			"xml":  "XML",
			"json": "JSON",
			"yaml": "YAML",
			"yml":  "YAML",
			"md":   "Markdown",
			"txt":  "Text",
			"sh":   "Shell",
			"bat":  "Batch",
			"ps1":  "PowerShell",
		},
	}
}

// Analyze 实现 CodeAnalyzer 接口
func (d *DefaultCodeAnalyzer) Analyze(p *project.Project) (*CodeStats, error) {
	stats := &CodeStats{
		LanguageStats:   make(map[string]int),
		FileTypeStats:   make(map[string]int),
		ComplexityStats: make(map[string]int),
	}

	// 创建访问者函数
	visitor := project.VisitorFunc(func(path string, node *project.Node, depth int) error {
		if node.IsDir {
			stats.TotalDirs++
			return nil
		}

		// 统计文件
		stats.TotalFiles++
		stats.TotalSize += int64(len(node.Content))

		// 获取文件扩展名
		ext := strings.TrimPrefix(filepath.Ext(node.Name), ".")
		stats.FileTypeStats[ext]++

		// 获取语言类型
		if lang, ok := d.languageMap[ext]; ok {
			// 统计代码行数
			lines := countCodeLines(node.Content, ext)
			stats.TotalLines += lines
			stats.LanguageStats[lang] += lines
		}

		return nil
	})

	// 遍历项目树
	traverser := project.NewTreeTraverser(p)
	err := traverser.TraverseTree(visitor)
	return stats, err
}

// countCodeLines 计算代码行数（排除空行和注释）
func countCodeLines(content []byte, ext string) int {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineCount := 0
	inComment := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行
		if line == "" {
			continue
		}

		// 根据不同语言处理注释
		switch ext {
		case "go", "c", "cpp", "java", "cs", "js", "ts":
			// 处理多行注释
			if inComment {
				if strings.Contains(line, "*/") {
					inComment = false
					line = strings.TrimSpace(strings.Split(line, "*/")[1])
					if line == "" {
						continue
					}
				} else {
					continue
				}
			}

			// 检查是否开始多行注释
			if strings.Contains(line, "/*") {
				parts := strings.Split(line, "/*")
				if !strings.Contains(parts[1], "*/") {
					inComment = true
					line = strings.TrimSpace(parts[0])
					if line == "" {
						continue
					}
				}
			}

			// 处理单行注释
			if strings.HasPrefix(line, "//") {
				continue
			}

		case "py", "rb":
			// 处理 Python/Ruby 注释
			if strings.HasPrefix(line, "#") {
				continue
			}

		case "html", "xml":
			// 处理 HTML/XML 注释
			if strings.HasPrefix(line, "<!--") && !strings.Contains(line, "-->") {
				inComment = true
				continue
			}
			if inComment {
				if strings.Contains(line, "-->") {
					inComment = false
				}
				continue
			}
		}

		lineCount++
	}

	return lineCount
}
    - [dependency_analyzer.go](#dependency_analyzer.go)

#### dependency_analyzer.go

package analyzer

import (
	"bufio"
	"bytes"
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sjzsdu/tong/project"
)

// DependencyNode 依赖节点
type DependencyNode struct {
	Name    string // 依赖名称
	Version string // 依赖版本
	Type    string // 依赖类型（直接依赖/间接依赖）
}

// DependencyGraph 依赖关系图
type DependencyGraph struct {
	Nodes map[string]*DependencyNode // 依赖节点
	Edges map[string][]string        // 依赖关系
}

// DependencyAnalyzer 依赖分析器接口
type DependencyAnalyzer interface {
	// 分析项目依赖关系
	AnalyzeDependencies(project *project.Project) (*DependencyGraph, error)
}

// LanguageDependencyAnalyzer 特定语言的依赖分析器
type LanguageDependencyAnalyzer interface {
	// 分析特定语言的依赖
	Analyze(content []byte, filePath string) ([]*DependencyNode, []string, error)
}

// DefaultDependencyAnalyzer 默认依赖分析器实现
type DefaultDependencyAnalyzer struct {
	languageAnalyzers map[string]LanguageDependencyAnalyzer
}

// NewDefaultDependencyAnalyzer 创建一个新的默认依赖分析器
func NewDefaultDependencyAnalyzer() *DefaultDependencyAnalyzer {
	return &DefaultDependencyAnalyzer{
		languageAnalyzers: map[string]LanguageDependencyAnalyzer{
			".go":     &GoDependencyAnalyzer{},
			".js":     &JSDependencyAnalyzer{},
			".json":   &JSONDependencyAnalyzer{},
			".py":     &PythonDependencyAnalyzer{},
			".java":   &JavaDependencyAnalyzer{},
			".gradle": &GradleDependencyAnalyzer{},
		},
	}
}

// AnalyzeDependencies 实现 DependencyAnalyzer 接口
func (d *DefaultDependencyAnalyzer) AnalyzeDependencies(p *project.Project) (*DependencyGraph, error) {
	graph := &DependencyGraph{
		Nodes: make(map[string]*DependencyNode),
		Edges: make(map[string][]string),
	}

	// 创建访问者函数
	visitor := project.VisitorFunc(func(path string, node *project.Node, depth int) error {
		if node.IsDir {
			return nil
		}

		// 获取文件扩展名
		ext := filepath.Ext(node.Name)
		analyzer, ok := d.languageAnalyzers[ext]
		if !ok {
			return nil
		}

		// 分析依赖
		nodes, edges, err := analyzer.Analyze(node.Content, path)
		if err != nil {
			return err
		}

		// 添加到图中
		for _, node := range nodes {
			graph.Nodes[node.Name] = node
		}

		// 添加边
		for i := 0; i < len(edges)-1; i += 2 {
			src := edges[i]
			dst := edges[i+1]
			graph.Edges[src] = append(graph.Edges[src], dst)
		}

		return nil
	})

	// 遍历项目树
	traverser := project.NewTreeTraverser(p)
	err := traverser.TraverseTree(visitor)
	return graph, err
}

// GoDependencyAnalyzer Go语言依赖分析器
type GoDependencyAnalyzer struct{}

// Analyze 实现 LanguageDependencyAnalyzer 接口
func (g *GoDependencyAnalyzer) Analyze(content []byte, filePath string) ([]*DependencyNode, []string, error) {
	var nodes []*DependencyNode
	var edges []string

	// 检查是否是 go.mod 文件
	if strings.HasSuffix(filePath, "go.mod") {
		return g.analyzeGoMod(content)
	}

	// 分析 Go 源文件中的导入
	scanner := bufio.NewScanner(bytes.NewReader(content))
	importRegex := regexp.MustCompile(`import\s+\(([^)]+)\)|import\s+([^\s]+)`)
	packageRegex := regexp.MustCompile(`package\s+(\w+)`)

	var packageName string
	for scanner.Scan() {
		line := scanner.Text()

		// 获取包名
		if packageName == "" {
			matches := packageRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				packageName = matches[1]
			}
		}

		// 查找导入
		matches := importRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			// 多行导入
			if matches[1] != "" {
				imports := strings.Split(matches[1], "\n")
				for _, imp := range imports {
					imp = strings.TrimSpace(imp)
					if imp == "" || strings.HasPrefix(imp, "//") {
						continue
					}
					// 清理引号
					imp = strings.Trim(imp, `"`)
					if imp != "" {
						nodes = append(nodes, &DependencyNode{
							Name: imp,
							Type: "import",
						})
						if packageName != "" {
							edges = append(edges, packageName, imp)
						}
					}
				}
			} else if matches[2] != "" {
				// 单行导入
				imp := strings.Trim(matches[2], `"`)
				if imp != "" {
					nodes = append(nodes, &DependencyNode{
						Name: imp,
						Type: "import",
					})
					if packageName != "" {
						edges = append(edges, packageName, imp)
					}
				}
			}
		}
	}

	return nodes, edges, nil
}

// analyzeGoMod 分析 go.mod 文件
func (g *GoDependencyAnalyzer) analyzeGoMod(content []byte) ([]*DependencyNode, []string, error) {
	var nodes []*DependencyNode
	var edges []string

	scanner := bufio.NewScanner(bytes.NewReader(content))
	moduleRegex := regexp.MustCompile(`module\s+(.+)`)
	requireRegex := regexp.MustCompile(`require\s+(.+)\s+(.+)`)
	requireBlockRegex := regexp.MustCompile(`require\s+\(`)

	var moduleName string
	inRequireBlock := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// 获取模块名
		if moduleName == "" {
			matches := moduleRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				moduleName = matches[1]
			}
		}

		// 检查是否进入 require 块
		if requireBlockRegex.MatchString(line) {
			inRequireBlock = true
			continue
		}

		// 检查是否退出 require 块
		if inRequireBlock && line == ")" {
			inRequireBlock = false
			continue
		}

		// 处理 require 块内的依赖
		if inRequireBlock {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				depName := parts[0]
				depVersion := parts[1]
				nodes = append(nodes, &DependencyNode{
					Name:    depName,
					Version: depVersion,
					Type:    "direct",
				})
				if moduleName != "" {
					edges = append(edges, moduleName, depName)
				}
			}
			continue
		}

		// 处理单行 require
		matches := requireRegex.FindStringSubmatch(line)
		if len(matches) > 2 {
			depName := matches[1]
			depVersion := matches[2]
			nodes = append(nodes, &DependencyNode{
				Name:    depName,
				Version: depVersion,
				Type:    "direct",
			})
			if moduleName != "" {
				edges = append(edges, moduleName, depName)
			}
		}
	}

	return nodes, edges, nil
}

// JSDependencyAnalyzer JavaScript依赖分析器
type JSDependencyAnalyzer struct{}

// Analyze 实现 LanguageDependencyAnalyzer 接口
func (j *JSDependencyAnalyzer) Analyze(content []byte, filePath string) ([]*DependencyNode, []string, error) {
	var nodes []*DependencyNode
	var edges []string

	// 分析 JS 源文件中的导入
	scanner := bufio.NewScanner(bytes.NewReader(content))
	importRegex := regexp.MustCompile(`import\s+.*?from\s+['"]([^'"]+)['"]|require\(['"]([^'"]+)['"]\)`)

	for scanner.Scan() {
		line := scanner.Text()

		// 查找导入
		matches := importRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			var importPath string
			if match[1] != "" {
				importPath = match[1]
			} else if match[2] != "" {
				importPath = match[2]
			}

			if importPath != "" {
				nodes = append(nodes, &DependencyNode{
					Name: importPath,
					Type: "import",
				})
				// 使用文件路径作为源节点
				edges = append(edges, filePath, importPath)
			}
		}
	}

	return nodes, edges, nil
}

// JSONDependencyAnalyzer JSON依赖分析器（主要用于package.json）
type JSONDependencyAnalyzer struct{}

// Analyze 实现 LanguageDependencyAnalyzer 接口
func (j *JSONDependencyAnalyzer) Analyze(content []byte, filePath string) ([]*DependencyNode, []string, error) {
	var nodes []*DependencyNode
	var edges []string

	// 只处理 package.json 文件
	if !strings.HasSuffix(filePath, "package.json") {
		return nodes, edges, nil
	}

	// 解析 JSON
	var packageJSON struct {
		Name            string            `json:"name"`
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	if err := json.Unmarshal(content, &packageJSON); err != nil {
		return nil, nil, err
	}

	// 处理依赖
	for name, version := range packageJSON.Dependencies {
		nodes = append(nodes, &DependencyNode{
			Name:    name,
			Version: version,
			Type:    "direct",
		})
		if packageJSON.Name != "" {
			edges = append(edges, packageJSON.Name, name)
		}
	}

	// 处理开发依赖
	for name, version := range packageJSON.DevDependencies {
		nodes = append(nodes, &DependencyNode{
			Name:    name,
			Version: version,
			Type:    "dev",
		})
		if packageJSON.Name != "" {
			edges = append(edges, packageJSON.Name, name)
		}
	}

	return nodes, edges, nil
}

// PythonDependencyAnalyzer Python依赖分析器
type PythonDependencyAnalyzer struct{}

// Analyze 实现 LanguageDependencyAnalyzer 接口
func (p *PythonDependencyAnalyzer) Analyze(content []byte, filePath string) ([]*DependencyNode, []string, error) {
	var nodes []*DependencyNode
	var edges []string

	// 检查是否是 requirements.txt 文件
	if strings.HasSuffix(filePath, "requirements.txt") {
		return p.analyzeRequirementsTxt(content, filePath)
	}

	// 分析 Python 源文件中的导入
	scanner := bufio.NewScanner(bytes.NewReader(content))
	importRegex := regexp.MustCompile(`^\s*import\s+([\w\.]+)|^\s*from\s+([\w\.]+)\s+import`)

	for scanner.Scan() {
		line := scanner.Text()

		// 跳过注释
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		// 查找导入
		matches := importRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			var importPath string
			if matches[1] != "" {
				importPath = matches[1]
			} else if matches[2] != "" {
				importPath = matches[2]
			}

			if importPath != "" {
				// 获取顶级包名
				topLevelPackage := strings.Split(importPath, ".")[0]
				nodes = append(nodes, &DependencyNode{
					Name: topLevelPackage,
					Type: "import",
				})
				// 使用文件路径作为源节点
				edges = append(edges, filePath, topLevelPackage)
			}
		}
	}

	return nodes, edges, nil
}

// analyzeRequirementsTxt 分析 requirements.txt 文件
func (p *PythonDependencyAnalyzer) analyzeRequirementsTxt(content []byte, filePath string) ([]*DependencyNode, []string, error) {
	var nodes []*DependencyNode
	var edges []string

	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 解析依赖
		parts := strings.Split(line, "==")
		if len(parts) >= 2 {
			name := strings.TrimSpace(parts[0])
			version := strings.TrimSpace(parts[1])
			nodes = append(nodes, &DependencyNode{
				Name:    name,
				Version: version,
				Type:    "direct",
			})
			// 使用文件路径作为源节点
			edges = append(edges, filePath, name)
		} else {
			// 处理没有版本的依赖
			name := strings.TrimSpace(line)
			nodes = append(nodes, &DependencyNode{
				Name: name,
				Type: "direct",
			})
			// 使用文件路径作为源节点
			edges = append(edges, filePath, name)
		}
	}

	return nodes, edges, nil
}

// JavaDependencyAnalyzer Java依赖分析器
type JavaDependencyAnalyzer struct{}

// Analyze 实现 LanguageDependencyAnalyzer 接口
func (j *JavaDependencyAnalyzer) Analyze(content []byte, filePath string) ([]*DependencyNode, []string, error) {
	var nodes []*DependencyNode
	var edges []string

	// 分析 Java 源文件中的导入
	scanner := bufio.NewScanner(bytes.NewReader(content))
	importRegex := regexp.MustCompile(`^\s*import\s+([\w\.\*]+);`)
	packageRegex := regexp.MustCompile(`^\s*package\s+([\w\.]+);`)

	var packageName string
	for scanner.Scan() {
		line := scanner.Text()

		// 获取包名
		if packageName == "" {
			matches := packageRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				packageName = matches[1]
			}
		}

		// 查找导入
		matches := importRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			importPath := matches[1]
			// 获取顶级包名
			topLevelPackage := strings.Split(importPath, ".")[0]
			nodes = append(nodes, &DependencyNode{
				Name: topLevelPackage,
				Type: "import",
			})
			if packageName != "" {
				edges = append(edges, packageName, topLevelPackage)
			}
		}
	}

	return nodes, edges, nil
}

// GradleDependencyAnalyzer Gradle依赖分析器
type GradleDependencyAnalyzer struct{}

// Analyze 实现 LanguageDependencyAnalyzer 接口
func (g *GradleDependencyAnalyzer) Analyze(content []byte, filePath string) ([]*DependencyNode, []string, error) {
	var nodes []*DependencyNode
	var edges []string

	// 只处理 build.gradle 文件
	if !strings.HasSuffix(filePath, "build.gradle") {
		return nodes, edges, nil
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	dependencyRegex := regexp.MustCompile(`\s*([\w]+)\s*['"]([^'"]+):([^'"]+):([^'"]+)['"]`)
	inDependenciesBlock := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 检查是否进入 dependencies 块
		if strings.HasPrefix(line, "dependencies {") {
			inDependenciesBlock = true
			continue
		}

		// 检查是否退出 dependencies 块
		if inDependenciesBlock && line == "}" {
			inDependenciesBlock = false
			continue
		}

		// 处理 dependencies 块内的依赖
		if inDependenciesBlock {
			matches := dependencyRegex.FindStringSubmatch(line)
			if len(matches) > 4 {
				depType := matches[1]  // implementation, api, etc.
				group := matches[2]    // group ID
				artifact := matches[3] // artifact ID
				version := matches[4]  // version

				depName := group + ":" + artifact
				nodes = append(nodes, &DependencyNode{
					Name:    depName,
					Version: version,
					Type:    depType,
				})
				// 使用文件路径作为源节点
				edges = append(edges, filePath, depName)
			}
		}
	}

	return nodes, edges, nil
}

  - [builder.go](#builder.go)

### builder.go

package project

import (
	"os"
	"path/filepath"

	"github.com/sjzsdu/tong/helper"
)

// 需要排除的系统和开发工具目录
var excludedDirs = map[string]bool{
	".git":         true,
	".vscode":      true,
	".idea":        true,
	"node_modules": true,
	".svn":         true,
	".hg":          true,
	".DS_Store":    true,
	"__pycache__":  true,
	"bin":          true,
	"obj":          true,
	"dist":         true,
	"build":        true,
	"target":       true,
	"fonts":        true,
}

// BuildProjectTree 构建项目树
func BuildProjectTree(targetPath string, options helper.WalkDirOptions) (*Project, error) {
	doc := NewProject(targetPath)
	gitignoreRules := make(map[string][]string)
	targetPath = filepath.Clean(targetPath)

	err := filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 检查是否是需要排除的目录
		if info.IsDir() {
			name := info.Name()
			// 排除 . 和 .. 目录
			if name == "." || name == ".." {
				return nil
			}

			// 对于非根目录的情况才检查排除规则
			if path != targetPath && excludedDirs[name] {
				return filepath.SkipDir
			}

			if excludedDirs[name] {
				return filepath.SkipDir
			}

			// 处理 .gitignore 规则
			if !options.DisableGitIgnore {
				rules, err := helper.ReadGitignore(path)
				if err == nil && rules != nil {
					gitignoreRules[path] = rules
				}
			}
		}

		// 处理 .gitignore 规则
		if !options.DisableGitIgnore {
			excluded, excludeErr := helper.IsPathExcludedByGitignore(path, targetPath, gitignoreRules)
			if excludeErr != nil {
				return excludeErr
			}
			if excluded {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// 获取相对路径
		relPath, err := filepath.Rel(targetPath, path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			if info.Name() == "." {
				return nil
			}
			// 创建目录节点
			return doc.CreateDir(relPath, info)
		}

		// 检查文件扩展名
		if len(options.Extensions) > 0 {
			ext := filepath.Ext(path)
			if len(ext) > 0 {
				ext = ext[1:] // 移除开头的点
			}
			if !helper.StringSliceContains(options.Extensions, ext) && !helper.StringSliceContains(options.Extensions, "*") {
				return nil
			}
		}

		// 检查排除规则
		if helper.IsPathExcluded(path, options.Excludes, targetPath) {
			return nil
		}

		// 读取文件内容
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // 跳过无法读取的文件
		}

		// 创建文件节点
		return doc.CreateFile(relPath, content, info)
	})

	if err != nil {
		return nil, err
	}

	return doc, nil
}

  - [exporter.go](#exporter.go)

### exporter.go

package project

// ContentCollector 定义内容收集的接口
type ContentCollector interface {
	// AddTitle 添加标题
	AddTitle(title string, level int) error
	// AddContent 添加内容
	AddContent(content string) error
	// AddTOCItem 添加目录项
	AddTOCItem(title string, level int) error
	// Render 渲染最终结果
	Render(outputPath string) error
}

// Exporter 定义了项目导出器的接口
type Exporter interface {
	NodeVisitor
	Export(outputPath string) error
}

// BaseExporter 提供了基本的导出功能
type BaseExporter struct {
	project   *Project
	collector ContentCollector
}

// NewBaseExporter 创建一个基本导出器
func NewBaseExporter(p *Project, collector ContentCollector) *BaseExporter {
	return &BaseExporter{
		project:   p,
		collector: collector,
	}
}

// Export 实现通用的导出逻辑
func (b *BaseExporter) Export(outputPath string) error {
	if b.project.root == nil || len(b.project.root.Children) == 0 {
		return b.collector.AddTitle("空项目", 1)
	}

	traverser := NewTreeTraverser(b.project)
	traverser.SetTraverseOrder(PreOrder).TraverseTree(b)
	return b.collector.Render(outputPath)
}

// VisitDirectory 实现通用的目录访问逻辑
func (b *BaseExporter) VisitDirectory(node *Node, path string, level int) error {
	if path == "/" {
		return nil
	}

	// 尝试添加目录项（如果收集器支持的话）
	if tocCollector, ok := b.collector.(interface{ AddTOCItem(string, int) error }); ok {
		if err := tocCollector.AddTOCItem(node.Name, level); err != nil {
			return err
		}
	}

	return b.collector.AddTitle(node.Name, level)
}

// VisitFile 实现通用的文件访问逻辑
func (b *BaseExporter) VisitFile(node *Node, path string, level int) error {
	// 尝试添加目录项（如果收集器支持的话）
	if tocCollector, ok := b.collector.(interface{ AddTOCItem(string, int) error }); ok {
		if err := tocCollector.AddTOCItem(node.Name, level); err != nil {
			return err
		}
	}

	if err := b.collector.AddTitle(node.Name, level); err != nil {
		return err
	}
	return b.collector.AddContent(string(node.Content))
}

  - [health](#health)

### health

    - [code_quality.go](#code_quality.go)

#### code_quality.go

package health

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/sjzsdu/tong/project"
)

// CodeQualityMetric 代码质量指标
type CodeQualityMetric string

const (
	CyclomaticComplexity CodeQualityMetric = "cyclomatic_complexity" // 圈复杂度
	MaintainabilityIndex CodeQualityMetric = "maintainability_index" // 可维护性指数
	CommentRatio         CodeQualityMetric = "comment_ratio"         // 注释比例
	DuplicationRatio     CodeQualityMetric = "duplication_ratio"     // 重复代码比例
	TestCoverage         CodeQualityMetric = "test_coverage"         // 测试覆盖率
	CodeSmells           CodeQualityMetric = "code_smells"           // 代码异味
)

// MetricSeverity 指标严重程度
type MetricSeverity string

const (
	Info    MetricSeverity = "info"    // 信息
	Warning MetricSeverity = "warning" // 警告
	Error   MetricSeverity = "error"   // 错误
)

// MetricResult 指标结果
type MetricResult struct {
	Metric      CodeQualityMetric // 指标
	Value       float64           // 值
	Threshold   float64           // 阈值
	Severity    MetricSeverity    // 严重程度
	Description string            // 描述
}

// FileQualityResult 文件质量结果
type FileQualityResult struct {
	FilePath string                             // 文件路径
	Metrics  map[CodeQualityMetric]MetricResult // 指标结果
	Issues   []CodeIssue                        // 问题列表
}

// CodeIssue 代码问题
type CodeIssue struct {
	FilePath    string         // 文件路径
	Line        int            // 行号
	Column      int            // 列号
	Message     string         // 消息
	Severity    MetricSeverity // 严重程度
	Rule        string         // 规则
	Description string         // 描述
}

// ProjectQualityResult 项目质量结果
type ProjectQualityResult struct {
	Files       map[string]FileQualityResult       // 文件质量结果
	TotalIssues int                                // 总问题数
	Metrics     map[CodeQualityMetric]MetricResult // 项目级指标
	Score       float64                            // 总分
}

// CodeQualityAnalyzer 代码质量分析器接口
type CodeQualityAnalyzer interface {
	// 分析项目代码质量
	Analyze() (ProjectQualityResult, error)
	// 分析文件代码质量
	AnalyzeFile(filePath string) (FileQualityResult, error)
	// 获取支持的指标
	GetSupportedMetrics() []CodeQualityMetric
	// 设置指标阈值
	SetThreshold(metric CodeQualityMetric, threshold float64)
	// 获取指标阈值
	GetThreshold(metric CodeQualityMetric) float64
}

// DefaultCodeQualityAnalyzer 默认代码质量分析器
type DefaultCodeQualityAnalyzer struct {
	project    *project.Project                       // 项目
	thresholds map[CodeQualityMetric]float64          // 阈值
	metrics    map[CodeQualityMetric]MetricCalculator // 指标计算器
	mu         sync.RWMutex                           // 读写锁
}

// MetricCalculator 指标计算器接口
type MetricCalculator interface {
	// 计算文件指标
	CalculateFileMetric(filePath string, content []byte) (float64, []CodeIssue, error)
	// 计算项目指标
	CalculateProjectMetric(fileResults map[string]FileQualityResult) (float64, error)
	// 获取指标描述
	GetDescription() string
	// 获取默认阈值
	GetDefaultThreshold() float64
	// 评估指标值的严重程度
	EvaluateSeverity(value float64, threshold float64) MetricSeverity
}

// NewCodeQualityAnalyzer 创建一个新的代码质量分析器
func NewCodeQualityAnalyzer(p *project.Project) *DefaultCodeQualityAnalyzer {
	analyzer := &DefaultCodeQualityAnalyzer{
		project:    p,
		thresholds: make(map[CodeQualityMetric]float64),
		metrics:    make(map[CodeQualityMetric]MetricCalculator),
	}

	// 注册指标计算器
	analyzer.RegisterMetric(CyclomaticComplexity, NewCyclomaticComplexityCalculator())
	analyzer.RegisterMetric(MaintainabilityIndex, NewMaintainabilityIndexCalculator())
	analyzer.RegisterMetric(CommentRatio, NewCommentRatioCalculator())
	analyzer.RegisterMetric(DuplicationRatio, NewDuplicationRatioCalculator())
	analyzer.RegisterMetric(TestCoverage, NewTestCoverageCalculator())
	analyzer.RegisterMetric(CodeSmells, NewCodeSmellsCalculator())

	return analyzer
}

// RegisterMetric 注册指标计算器
func (a *DefaultCodeQualityAnalyzer) RegisterMetric(metric CodeQualityMetric, calculator MetricCalculator) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.metrics[metric] = calculator
	a.thresholds[metric] = calculator.GetDefaultThreshold()
}

// Analyze 实现 CodeQualityAnalyzer 接口
func (a *DefaultCodeQualityAnalyzer) Analyze() (ProjectQualityResult, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// 初始化结果
	result := ProjectQualityResult{
		Files:   make(map[string]FileQualityResult),
		Metrics: make(map[CodeQualityMetric]MetricResult),
	}

	// 创建访问者函数
	visitor := project.VisitorFunc(func(path string, node *project.Node, depth int) error {
		if node.IsDir {
			return nil
		}

		// 分析文件
		fileResult, err := a.AnalyzeFile(path)
		if err != nil {
			return err
		}

		// 添加到结果
		result.Files[path] = fileResult
		result.TotalIssues += len(fileResult.Issues)

		return nil
	})

	// 遍历项目树
	traverser := project.NewTreeTraverser(a.project)
	err := traverser.TraverseTree(visitor)
	if err != nil {
		return result, err
	}

	// 计算项目级指标
	for metric, calculator := range a.metrics {
		value, err := calculator.CalculateProjectMetric(result.Files)
		if err != nil {
			continue
		}

		threshold := a.thresholds[metric]
		severity := calculator.EvaluateSeverity(value, threshold)

		result.Metrics[metric] = MetricResult{
			Metric:      metric,
			Value:       value,
			Threshold:   threshold,
			Severity:    severity,
			Description: calculator.GetDescription(),
		}
	}

	// 计算总分
	result.Score = a.calculateOverallScore(result)

	return result, nil
}

// AnalyzeFile 实现 CodeQualityAnalyzer 接口
func (a *DefaultCodeQualityAnalyzer) AnalyzeFile(filePath string) (FileQualityResult, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// 初始化结果
	result := FileQualityResult{
		FilePath: filePath,
		Metrics:  make(map[CodeQualityMetric]MetricResult),
		Issues:   make([]CodeIssue, 0),
	}

	// 检查文件是否存在
	node, err := a.project.FindNode(filePath)
	if err != nil || node == nil || node.IsDir {
		return result, fmt.Errorf("文件不存在: %s", filePath)
	}

	// 获取文件内容
	content := node.Content

	// 计算每个指标
	for metric, calculator := range a.metrics {
		value, issues, err := calculator.CalculateFileMetric(filePath, content)
		if err != nil {
			continue
		}

		threshold := a.thresholds[metric]
		severity := calculator.EvaluateSeverity(value, threshold)

		result.Metrics[metric] = MetricResult{
			Metric:      metric,
			Value:       value,
			Threshold:   threshold,
			Severity:    severity,
			Description: calculator.GetDescription(),
		}

		// 添加问题
		result.Issues = append(result.Issues, issues...)
	}

	return result, nil
}

// GetSupportedMetrics 实现 CodeQualityAnalyzer 接口
func (a *DefaultCodeQualityAnalyzer) GetSupportedMetrics() []CodeQualityMetric {
	a.mu.RLock()
	defer a.mu.RUnlock()

	metrics := make([]CodeQualityMetric, 0, len(a.metrics))
	for metric := range a.metrics {
		metrics = append(metrics, metric)
	}

	return metrics
}

// SetThreshold 实现 CodeQualityAnalyzer 接口
func (a *DefaultCodeQualityAnalyzer) SetThreshold(metric CodeQualityMetric, threshold float64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.thresholds[metric] = threshold
}

// GetThreshold 实现 CodeQualityAnalyzer 接口
func (a *DefaultCodeQualityAnalyzer) GetThreshold(metric CodeQualityMetric) float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.thresholds[metric]
}

// calculateOverallScore 计算总分
func (a *DefaultCodeQualityAnalyzer) calculateOverallScore(result ProjectQualityResult) float64 {
	// 权重
	weights := map[CodeQualityMetric]float64{
		CyclomaticComplexity: 0.2,
		MaintainabilityIndex: 0.3,
		CommentRatio:         0.1,
		DuplicationRatio:     0.2,
		TestCoverage:         0.1,
		CodeSmells:           0.1,
	}

	// 计算加权分数
	totalWeight := 0.0
	totalScore := 0.0

	for metric, metricResult := range result.Metrics {
		weight, ok := weights[metric]
		if !ok {
			continue
		}

		// 计算指标得分（0-100）
		var score float64
		switch metric {
		case CyclomaticComplexity:
			// 圈复杂度越低越好
			score = 100 * math.Max(0, 1-metricResult.Value/metricResult.Threshold)
		case MaintainabilityIndex:
			// 可维护性指数越高越好
			score = metricResult.Value
		case CommentRatio:
			// 注释比例越高越好，但不超过50%
			score = 100 * math.Min(metricResult.Value/0.3, 1)
		case DuplicationRatio:
			// 重复代码比例越低越好
			score = 100 * math.Max(0, 1-metricResult.Value/metricResult.Threshold)
		case TestCoverage:
			// 测试覆盖率越高越好
			score = metricResult.Value * 100
		case CodeSmells:
			// 代码异味越少越好
			score = 100 * math.Max(0, 1-metricResult.Value/metricResult.Threshold)
		}

		totalScore += score * weight
		totalWeight += weight
	}

	// 计算最终分数
	if totalWeight > 0 {
		return totalScore / totalWeight
	}

	return 0
}

// CyclomaticComplexityCalculator 圈复杂度计算器
type CyclomaticComplexityCalculator struct{}

// NewCyclomaticComplexityCalculator 创建一个新的圈复杂度计算器
func NewCyclomaticComplexityCalculator() *CyclomaticComplexityCalculator {
	return &CyclomaticComplexityCalculator{}
}

// CalculateFileMetric 实现 MetricCalculator 接口
func (c *CyclomaticComplexityCalculator) CalculateFileMetric(filePath string, content []byte) (float64, []CodeIssue, error) {
	// 只处理Go文件
	if !strings.HasSuffix(filePath, ".go") {
		return 0, nil, nil
	}

	// 解析文件
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return 0, nil, err
	}

	// 计算每个函数的圈复杂度
	funcComplexities := make(map[string]int)
	totalComplexity := 0
	funcCount := 0

	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			// 计算函数的圈复杂度
			complexity := 1 // 基础复杂度

			// 遍历函数体，计算分支数
			ast.Inspect(node.Body, func(n ast.Node) bool {
				switch n.(type) {
				case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.CaseClause, *ast.CommClause, *ast.BinaryExpr:
					// 二元表达式中的 &&, || 也增加复杂度
					if expr, ok := n.(*ast.BinaryExpr); ok {
						if expr.Op == token.LAND || expr.Op == token.LOR {
							complexity++
						}
					} else {
						complexity++
					}
				}
				return true
			})

			// 记录函数复杂度
			funcName := node.Name.Name
			if node.Recv != nil {
				// 方法
				recvType := ""
				if len(node.Recv.List) > 0 {
					recvType = fmt.Sprintf("%s", node.Recv.List[0].Type)
				}
				funcName = fmt.Sprintf("%s.%s", recvType, funcName)
			}

			funcComplexities[funcName] = complexity
			totalComplexity += complexity
			funcCount++
		}
		return true
	})

	// 计算平均复杂度
	averageComplexity := 0.0
	if funcCount > 0 {
		averageComplexity = float64(totalComplexity) / float64(funcCount)
	}

	// 创建问题列表
	issues := make([]CodeIssue, 0)
	threshold := c.GetDefaultThreshold()

	for funcName, complexity := range funcComplexities {
		if float64(complexity) > threshold {
			// 查找函数位置
			var line, column int
			ast.Inspect(f, func(n ast.Node) bool {
				if funcDecl, ok := n.(*ast.FuncDecl); ok {
					currentName := funcDecl.Name.Name
					if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
						recvType := fmt.Sprintf("%s", funcDecl.Recv.List[0].Type)
						currentName = fmt.Sprintf("%s.%s", recvType, currentName)
					}

					if currentName == funcName {
						pos := fset.Position(funcDecl.Pos())
						line = pos.Line
						column = pos.Column
						return false
					}
				}
				return true
			})

			// 添加问题
			issues = append(issues, CodeIssue{
				FilePath:    filePath,
				Line:        line,
				Column:      column,
				Message:     fmt.Sprintf("函数 %s 的圈复杂度为 %d，超过阈值 %.1f", funcName, complexity, threshold),
				Severity:    c.EvaluateSeverity(float64(complexity), threshold),
				Rule:        "high-complexity",
				Description: fmt.Sprintf("高圈复杂度函数难以理解和维护，建议重构拆分为多个小函数"),
			})
		}
	}

	return averageComplexity, issues, nil
}

// CalculateProjectMetric 实现 MetricCalculator 接口
func (c *CyclomaticComplexityCalculator) CalculateProjectMetric(fileResults map[string]FileQualityResult) (float64, error) {
	totalComplexity := 0.0
	fileCount := 0

	for _, result := range fileResults {
		if metric, ok := result.Metrics[CyclomaticComplexity]; ok {
			totalComplexity += metric.Value
			fileCount++
		}
	}

	if fileCount > 0 {
		return totalComplexity / float64(fileCount), nil
	}

	return 0, nil
}

// GetDescription 实现 MetricCalculator 接口
func (c *CyclomaticComplexityCalculator) GetDescription() string {
	return "圈复杂度是衡量代码复杂性的指标，表示代码中的线性独立路径数量。较高的圈复杂度表示代码更难理解和测试。"
}

// GetDefaultThreshold 实现 MetricCalculator 接口
func (c *CyclomaticComplexityCalculator) GetDefaultThreshold() float64 {
	return 10.0 // 一般认为10以下是合理的
}

// EvaluateSeverity 实现 MetricCalculator 接口
func (c *CyclomaticComplexityCalculator) EvaluateSeverity(value float64, threshold float64) MetricSeverity {
	if value <= threshold {
		return Info
	} else if value <= threshold*1.5 {
		return Warning
	} else {
		return Error
	}
}

// MaintainabilityIndexCalculator 可维护性指数计算器
type MaintainabilityIndexCalculator struct{}

// NewMaintainabilityIndexCalculator 创建一个新的可维护性指数计算器
func NewMaintainabilityIndexCalculator() *MaintainabilityIndexCalculator {
	return &MaintainabilityIndexCalculator{}
}

// CalculateFileMetric 实现 MetricCalculator 接口
func (m *MaintainabilityIndexCalculator) CalculateFileMetric(filePath string, content []byte) (float64, []CodeIssue, error) {
	// 只处理特定类型的文件
	ext := filepath.Ext(filePath)
	if ext != ".go" && ext != ".js" && ext != ".py" && ext != ".java" && ext != ".c" && ext != ".cpp" {
		return 0, nil, nil
	}

	// 计算代码行数
	lines := bytes.Count(content, []byte{'\n'}) + 1

	// 计算代码体积（字节数）
	volume := len(content)

	// 计算注释行数
	commentLines := countCommentLines(filePath, content)

	// 计算圈复杂度（简化版）
	complexity := 1.0
	if ext == ".go" {
		// 使用之前的圈复杂度计算器
		complexityCalculator := NewCyclomaticComplexityCalculator()
		complexity, _, _ = complexityCalculator.CalculateFileMetric(filePath, content)
	} else {
		// 简单估计：根据条件语句数量
		conditionPatterns := []string{
			"if\\s*\\(", "for\\s*\\(", "while\\s*\\(", "switch\\s*\\(", "case\\s+", "\\?\\s*:",
		}

		for _, pattern := range conditionPatterns {
			re := regexp.MustCompile(pattern)
			matches := re.FindAllIndex(content, -1)
			complexity += float64(len(matches))
		}
	}

	// 计算可维护性指数
	// MI = 171 - 5.2 * ln(volume) - 0.23 * complexity - 16.2 * ln(lines) + 50 * sin(sqrt(2.4 * commentRatio))
	commentRatio := 0.0
	if lines > 0 {
		commentRatio = float64(commentLines) / float64(lines)
	}

	mi := 171 - 5.2*math.Log(float64(volume)) - 0.23*complexity - 16.2*math.Log(float64(lines))
	mi += 50 * math.Sin(math.Sqrt(2.4*commentRatio))

	// 归一化到0-100
	mi = math.Max(0, math.Min(100, mi))

	// 创建问题列表
	issues := make([]CodeIssue, 0)
	threshold := m.GetDefaultThreshold()

	if mi < threshold {
		issues = append(issues, CodeIssue{
			FilePath:    filePath,
			Line:        1,
			Column:      1,
			Message:     fmt.Sprintf("文件的可维护性指数为 %.1f，低于阈值 %.1f", mi, threshold),
			Severity:    m.EvaluateSeverity(mi, threshold),
			Rule:        "low-maintainability",
			Description: "低可维护性指数表示代码难以维护，建议重构以提高可读性和可维护性",
		})
	}

	return mi, issues, nil
}

// CalculateProjectMetric 实现 MetricCalculator 接口
func (m *MaintainabilityIndexCalculator) CalculateProjectMetric(fileResults map[string]FileQualityResult) (float64, error) {
	totalMI := 0.0
	fileCount := 0

	for _, result := range fileResults {
		if metric, ok := result.Metrics[MaintainabilityIndex]; ok {
			totalMI += metric.Value
			fileCount++
		}
	}

	if fileCount > 0 {
		return totalMI / float64(fileCount), nil
	}

	return 0, nil
}

// GetDescription 实现 MetricCalculator 接口
func (m *MaintainabilityIndexCalculator) GetDescription() string {
	return "可维护性指数是衡量代码可维护性的综合指标，考虑了代码量、复杂度、注释等因素。较高的值表示代码更易于维护。"
}

// GetDefaultThreshold 实现 MetricCalculator 接口
func (m *MaintainabilityIndexCalculator) GetDefaultThreshold() float64 {
	return 65.0 // 一般认为65以上是可维护的
}

// EvaluateSeverity 实现 MetricCalculator 接口
func (m *MaintainabilityIndexCalculator) EvaluateSeverity(value float64, threshold float64) MetricSeverity {
	if value >= threshold {
		return Info
	} else if value >= threshold*0.8 {
		return Warning
	} else {
		return Error
	}
}

// CommentRatioCalculator 注释比例计算器
type CommentRatioCalculator struct{}

// NewCommentRatioCalculator 创建一个新的注释比例计算器
func NewCommentRatioCalculator() *CommentRatioCalculator {
	return &CommentRatioCalculator{}
}

// CalculateFileMetric 实现 MetricCalculator 接口
func (c *CommentRatioCalculator) CalculateFileMetric(filePath string, content []byte) (float64, []CodeIssue, error) {
	// 只处理特定类型的文件
	ext := filepath.Ext(filePath)
	if ext != ".go" && ext != ".js" && ext != ".py" && ext != ".java" && ext != ".c" && ext != ".cpp" {
		return 0, nil, nil
	}

	// 计算总行数
	lines := bytes.Count(content, []byte{'\n'}) + 1

	// 计算注释行数
	commentLines := countCommentLines(filePath, content)

	// 计算注释比例
	ratio := 0.0
	if lines > 0 {
		ratio = float64(commentLines) / float64(lines)
	}

	// 创建问题列表
	issues := make([]CodeIssue, 0)
	threshold := c.GetDefaultThreshold()

	if ratio < threshold {
		issues = append(issues, CodeIssue{
			FilePath:    filePath,
			Line:        1,
			Column:      1,
			Message:     fmt.Sprintf("文件的注释比例为 %.1f%%，低于阈值 %.1f%%", ratio*100, threshold*100),
			Severity:    c.EvaluateSeverity(ratio, threshold),
			Rule:        "low-comment-ratio",
			Description: "注释不足会降低代码可读性，建议添加适当的注释解释代码逻辑和意图",
		})
	}

	return ratio, issues, nil
}

// CalculateProjectMetric 实现 MetricCalculator 接口
func (c *CommentRatioCalculator) CalculateProjectMetric(fileResults map[string]FileQualityResult) (float64, error) {
	totalRatio := 0.0
	fileCount := 0

	for _, result := range fileResults {
		if metric, ok := result.Metrics[CommentRatio]; ok {
			totalRatio += metric.Value
			fileCount++
		}
	}

	if fileCount > 0 {
		return totalRatio / float64(fileCount), nil
	}

	return 0, nil
}

// GetDescription 实现 MetricCalculator 接口
func (c *CommentRatioCalculator) GetDescription() string {
	return "注释比例是代码中注释行数与总行数的比值。适当的注释有助于理解代码，但过多的注释可能表明代码本身不够清晰。"
}

// GetDefaultThreshold 实现 MetricCalculator 接口
func (c *CommentRatioCalculator) GetDefaultThreshold() float64 {
	return 0.15 // 建议至少15%的注释率
}

// EvaluateSeverity 实现 MetricCalculator 接口
func (c *CommentRatioCalculator) EvaluateSeverity(value float64, threshold float64) MetricSeverity {
	if value >= threshold {
		return Info
	} else if value >= threshold*0.7 {
		return Warning
	} else {
		return Error
	}
}

// DuplicationRatioCalculator 重复代码比例计算器
type DuplicationRatioCalculator struct{}

// NewDuplicationRatioCalculator 创建一个新的重复代码比例计算器
func NewDuplicationRatioCalculator() *DuplicationRatioCalculator {
	return &DuplicationRatioCalculator{}
}

// CalculateFileMetric 实现 MetricCalculator 接口
func (d *DuplicationRatioCalculator) CalculateFileMetric(filePath string, content []byte) (float64, []CodeIssue, error) {
	// 只处理特定类型的文件
	ext := filepath.Ext(filePath)
	if ext != ".go" && ext != ".js" && ext != ".py" && ext != ".java" && ext != ".c" && ext != ".cpp" {
		return 0, nil, nil
	}

	// 计算总行数
	lines := bytes.Split(content, []byte{'\n'})
	totalLines := len(lines)

	// 简化的重复代码检测：检测连续的N行是否在其他地方重复出现
	duplicateLines := 0
	minDuplicateLength := 6 // 最小重复行数

	// 创建行内容到行号的映射
	lineMap := make(map[string][]int)
	for i, line := range lines {
		// 忽略空行和注释行
		trimmedLine := bytes.TrimSpace(line)
		if len(trimmedLine) == 0 || isCommentLine(string(trimmedLine)) {
			continue
		}

		lineMap[string(trimmedLine)] = append(lineMap[string(trimmedLine)], i)
	}

	// 检测重复块
	duplicateBlocks := make(map[int]bool) // 记录已经标记为重复的行

	for i := 0; i < totalLines-minDuplicateLength+1; i++ {
		// 如果当前行已经被标记为重复，跳过
		if duplicateBlocks[i] {
			continue
		}

		// 尝试找到重复块
		for j := i + 1; j < totalLines-minDuplicateLength+1; j++ {
			// 检查从i和j开始的minDuplicateLength行是否相同
			match := true
			for k := 0; k < minDuplicateLength; k++ {
				if i+k >= totalLines || j+k >= totalLines {
					match = false
					break
				}

				// 忽略空行和注释行
				lineI := bytes.TrimSpace(lines[i+k])
				lineJ := bytes.TrimSpace(lines[j+k])

				if len(lineI) == 0 || isCommentLine(string(lineI)) {
					// 延长匹配
					if i+k+1 < totalLines {
						k--
						i++
					}
					continue
				}

				if len(lineJ) == 0 || isCommentLine(string(lineJ)) {
					// 延长匹配
					if j+k+1 < totalLines {
						k--
						j++
					}
					continue
				}

				if !bytes.Equal(lineI, lineJ) {
					match = false
					break
				}
			}

			if match {
				// 标记重复块
				for k := 0; k < minDuplicateLength; k++ {
					if !duplicateBlocks[i+k] {
						duplicateBlocks[i+k] = true
						duplicateLines++
					}
					if !duplicateBlocks[j+k] {
						duplicateBlocks[j+k] = true
						duplicateLines++
					}
				}
			}
		}
	}

	// 计算重复比例
	ratio := 0.0
	if totalLines > 0 {
		ratio = float64(duplicateLines) / float64(totalLines)
	}

	// 创建问题列表
	issues := make([]CodeIssue, 0)
	threshold := d.GetDefaultThreshold()

	if ratio > threshold {
		issues = append(issues, CodeIssue{
			FilePath:    filePath,
			Line:        1,
			Column:      1,
			Message:     fmt.Sprintf("文件的重复代码比例为 %.1f%%，超过阈值 %.1f%%", ratio*100, threshold*100),
			Severity:    d.EvaluateSeverity(ratio, threshold),
			Rule:        "high-duplication",
			Description: "高重复代码比例表明代码存在冗余，建议提取公共方法或使用设计模式减少重复",
		})
	}

	return ratio, issues, nil
}

// CalculateProjectMetric 实现 MetricCalculator 接口
func (d *DuplicationRatioCalculator) CalculateProjectMetric(fileResults map[string]FileQualityResult) (float64, error) {
	totalRatio := 0.0
	fileCount := 0

	for _, result := range fileResults {
		if metric, ok := result.Metrics[DuplicationRatio]; ok {
			totalRatio += metric.Value
			fileCount++
		}
	}

	if fileCount > 0 {
		return totalRatio / float64(fileCount), nil
	}

	return 0, nil
}

// GetDescription 实现 MetricCalculator 接口
func (d *DuplicationRatioCalculator) GetDescription() string {
	return "重复代码比例是代码中重复行数与总行数的比值。高重复度表明代码存在冗余，可能需要重构以提高可维护性。"
}

// GetDefaultThreshold 实现 MetricCalculator 接口
func (d *DuplicationRatioCalculator) GetDefaultThreshold() float64 {
	return 0.1 // 建议重复代码不超过10%
}

// EvaluateSeverity 实现 MetricCalculator 接口
func (d *DuplicationRatioCalculator) EvaluateSeverity(value float64, threshold float64) MetricSeverity {
	if value <= threshold {
		return Info
	} else if value <= threshold*1.5 {
		return Warning
	} else {
		return Error
	}
}

// TestCoverageCalculator 测试覆盖率计算器
type TestCoverageCalculator struct{}

// NewTestCoverageCalculator 创建一个新的测试覆盖率计算器
func NewTestCoverageCalculator() *TestCoverageCalculator {
	return &TestCoverageCalculator{}
}

// CalculateFileMetric 实现 MetricCalculator 接口
func (t *TestCoverageCalculator) CalculateFileMetric(filePath string, content []byte) (float64, []CodeIssue, error) {
	// 测试覆盖率通常是项目级指标，而不是文件级指标
	// 这里简单地检查是否有对应的测试文件

	// 只处理Go文件，且不是测试文件
	if !strings.HasSuffix(filePath, ".go") || strings.HasSuffix(filePath, "_test.go") {
		return 0, nil, nil
	}

	// 检查是否有对应的测试文件
	testFilePath := strings.TrimSuffix(filePath, ".go") + "_test.go"
	hasTestFile := false

	// 尝试在项目中查找测试文件
	_, err := os.Stat(testFilePath)
	hasTestFile = err == nil

	// 如果找不到测试文件，覆盖率为0
	coverage := 0.0
	if hasTestFile {
		// 简单估计：有测试文件则假设覆盖率为50%
		// 实际情况下应该使用测试覆盖率工具的结果
		coverage = 0.5
	}

	// 创建问题列表
	issues := make([]CodeIssue, 0)
	threshold := t.GetDefaultThreshold()

	if coverage < threshold {
		issues = append(issues, CodeIssue{
			FilePath:    filePath,
			Line:        1,
			Column:      1,
			Message:     fmt.Sprintf("文件的测试覆盖率估计为 %.1f%%，低于阈值 %.1f%%", coverage*100, threshold*100),
			Severity:    t.EvaluateSeverity(coverage, threshold),
			Rule:        "low-test-coverage",
			Description: "低测试覆盖率可能导致代码质量问题无法及时发现，建议增加测试用例",
		})
	}

	return coverage, issues, nil
}

// CalculateProjectMetric 实现 MetricCalculator 接口
func (t *TestCoverageCalculator) CalculateProjectMetric(fileResults map[string]FileQualityResult) (float64, error) {
	totalCoverage := 0.0
	fileCount := 0

	for _, result := range fileResults {
		if metric, ok := result.Metrics[TestCoverage]; ok {
			totalCoverage += metric.Value
			fileCount++
		}
	}

	if fileCount > 0 {
		return totalCoverage / float64(fileCount), nil
	}

	return 0, nil
}

// GetDescription 实现 MetricCalculator 接口
func (t *TestCoverageCalculator) GetDescription() string {
	return "测试覆盖率是代码被测试用例覆盖的比例。高测试覆盖率有助于及早发现问题并确保代码质量。"
}

// GetDefaultThreshold 实现 MetricCalculator 接口
func (t *TestCoverageCalculator) GetDefaultThreshold() float64 {
	return 0.7 // 建议至少70%的测试覆盖率
}

// EvaluateSeverity 实现 MetricCalculator 接口
func (t *TestCoverageCalculator) EvaluateSeverity(value float64, threshold float64) MetricSeverity {
	if value >= threshold {
		return Info
	} else if value >= threshold*0.7 {
		return Warning
	} else {
		return Error
	}
}

// CodeSmellsCalculator 代码异味计算器
type CodeSmellsCalculator struct{}

// NewCodeSmellsCalculator 创建一个新的代码异味计算器
func NewCodeSmellsCalculator() *CodeSmellsCalculator {
	return &CodeSmellsCalculator{}
}

// CalculateFileMetric 实现 MetricCalculator 接口
func (c *CodeSmellsCalculator) CalculateFileMetric(filePath string, content []byte) (float64, []CodeIssue, error) {
	// 只处理特定类型的文件
	ext := filepath.Ext(filePath)
	if ext != ".go" && ext != ".js" && ext != ".py" && ext != ".java" && ext != ".c" && ext != ".cpp" {
		return 0, nil, nil
	}

	// 代码异味规则
	type SmellRule struct {
		Name        string
		Description string
		Pattern     *regexp.Regexp
		Severity    MetricSeverity
	}

	// 定义代码异味规则
	rules := []SmellRule{
		{
			Name:        "magic-number",
			Description: "魔法数字使代码难以理解和维护，应该使用命名常量",
			Pattern:     regexp.MustCompile(`[^\w"]\d{2,}[^\w"]`),
			Severity:    Warning,
		},
		{
			Name:        "long-line",
			Description: "过长的行降低了代码可读性，应该拆分为多行",
			Pattern:     regexp.MustCompile(`.{100,}`),
			Severity:    Warning,
		},
		{
			Name:        "todo-comment",
			Description: "TODO注释表示代码不完整，应该及时处理",
			Pattern:     regexp.MustCompile(`(?i)\bTODO\b`),
			Severity:    Info,
		},
		{
			Name:        "fixme-comment",
			Description: "FIXME注释表示代码存在问题，应该优先修复",
			Pattern:     regexp.MustCompile(`(?i)\bFIXME\b`),
			Severity:    Warning,
		},
	}

	// 添加语言特定的规则
	switch ext {
	case ".go":
		rules = append(rules, SmellRule{
			Name:        "naked-return",
			Description: "裸返回使代码难以理解，应该显式指定返回值",
			Pattern:     regexp.MustCompile(`return\s*$`),
			Severity:    Warning,
		})
	case ".js":
		rules = append(rules, SmellRule{
			Name:        "eval-usage",
			Description: "eval函数存在安全风险，应该避免使用",
			Pattern:     regexp.MustCompile(`\beval\s*\(`),
			Severity:    Error,
		})
	case ".py":
		rules = append(rules, SmellRule{
			Name:        "global-variable",
			Description: "全局变量使代码难以测试和维护，应该避免使用",
			Pattern:     regexp.MustCompile(`\bglobal\b`),
			Severity:    Warning,
		})
	}

	// 检测代码异味
	issues := make([]CodeIssue, 0)
	lines := bytes.Split(content, []byte{'\n'})

	for i, line := range lines {
		lineStr := string(line)

		for _, rule := range rules {
			matches := rule.Pattern.FindAllStringIndex(lineStr, -1)
			for _, match := range matches {
				issues = append(issues, CodeIssue{
					FilePath:    filePath,
					Line:        i + 1,
					Column:      match[0] + 1,
					Message:     fmt.Sprintf("发现代码异味：%s", rule.Name),
					Severity:    rule.Severity,
					Rule:        rule.Name,
					Description: rule.Description,
				})
			}
		}
	}

	// 计算代码异味密度（每千行代码的异味数）
	density := 0.0
	if len(lines) > 0 {
		density = float64(len(issues)) * 1000 / float64(len(lines))
	}

	return density, issues, nil
}

// CalculateProjectMetric 实现 MetricCalculator 接口
func (c *CodeSmellsCalculator) CalculateProjectMetric(fileResults map[string]FileQualityResult) (float64, error) {
	totalDensity := 0.0
	fileCount := 0

	for _, result := range fileResults {
		if metric, ok := result.Metrics[CodeSmells]; ok {
			totalDensity += metric.Value
			fileCount++
		}
	}

	if fileCount > 0 {
		return totalDensity / float64(fileCount), nil
	}

	return 0, nil
}

// GetDescription 实现 MetricCalculator 接口
func (c *CodeSmellsCalculator) GetDescription() string {
	return "代码异味是代码中可能导致问题的模式。代码异味密度表示每千行代码中的异味数量，较低的值表示代码质量更好。"
}

// GetDefaultThreshold 实现 MetricCalculator 接口
func (c *CodeSmellsCalculator) GetDefaultThreshold() float64 {
	return 5.0 // 每千行代码不超过5个异味
}

// EvaluateSeverity 实现 MetricCalculator 接口
func (c *CodeSmellsCalculator) EvaluateSeverity(value float64, threshold float64) MetricSeverity {
	if value <= threshold {
		return Info
	} else if value <= threshold*2 {
		return Warning
	} else {
		return Error
	}
}

// 辅助函数

// countCommentLines 计算注释行数
func countCommentLines(filePath string, content []byte) int {
	ext := filepath.Ext(filePath)
	commentLines := 0

	scanner := bufio.NewScanner(bytes.NewReader(content))
	inMultilineComment := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		if len(trimmedLine) == 0 {
			continue
		}

		switch ext {
		case ".go":
			// 处理Go注释
			if inMultilineComment {
				commentLines++
				if strings.Contains(trimmedLine, "*/") {
					inMultilineComment = false
				}
			} else if strings.HasPrefix(trimmedLine, "//") {
				commentLines++
			} else if strings.HasPrefix(trimmedLine, "/*") {
				commentLines++
				if !strings.Contains(trimmedLine, "*/") {
					inMultilineComment = true
				}
			}
		case ".js", ".java", ".c", ".cpp":
			// 处理C风格注释
			if inMultilineComment {
				commentLines++
				if strings.Contains(trimmedLine, "*/") {
					inMultilineComment = false
				}
			} else if strings.HasPrefix(trimmedLine, "//") {
				commentLines++
			} else if strings.HasPrefix(trimmedLine, "/*") {
				commentLines++
				if !strings.Contains(trimmedLine, "*/") {
					inMultilineComment = true
				}
			}
		case ".py":
			// 处理Python注释
			if inMultilineComment {
				commentLines++
				if strings.Contains(trimmedLine, "\"\"\"") || strings.Contains(trimmedLine, "'''") {
					inMultilineComment = false
				}
			} else if strings.HasPrefix(trimmedLine, "#") {
				commentLines++
			} else if strings.HasPrefix(trimmedLine, "\"\"\"") || strings.HasPrefix(trimmedLine, "'''") {
				commentLines++
				if !strings.Contains(trimmedLine[3:], "\"\"\"") && !strings.Contains(trimmedLine[3:], "'''") {
					inMultilineComment = true
				}
			}
		}
	}

	return commentLines
}

// isCommentLine 判断一行是否为注释行
func isCommentLine(line string) bool {
	// 检查是否为空行
	if len(strings.TrimSpace(line)) == 0 {
		return false
	}

	// 检查是否为注释行
	if strings.HasPrefix(strings.TrimSpace(line), "//") ||
		strings.HasPrefix(strings.TrimSpace(line), "/*") ||
		strings.HasPrefix(strings.TrimSpace(line), "*") ||
		strings.HasPrefix(strings.TrimSpace(line), "#") ||
		strings.HasPrefix(strings.TrimSpace(line), "\"\"\"") ||
		strings.HasPrefix(strings.TrimSpace(line), "'''") {
		return true
	}

	return false
}

  - [node.go](#node.go)

### node.go

package project

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// CalculateHash 计算节点的哈希值
func (node *Node) CalculateHash() (string, error) {
	if node.IsDir {
		return node.calculateDirHash()
	}
	return node.calculateFileHash()
}

// calculateFileHash 计算文件内容的哈希值
func (node *Node) calculateFileHash() (string, error) {
	if node.Content == nil {
		return "", nil
	}
	hash := sha256.Sum256(node.Content)
	return hex.EncodeToString(hash[:]), nil
}

// calculateDirHash 计算目录的哈希值
func (node *Node) calculateDirHash() (string, error) {
	var hashes []string
	// 先对 Children 按名称排序
	sortedChildren := make([]*Node, len(node.Children))
	i := 0
	for _, child := range node.Children {
		sortedChildren[i] = child
		i++
	}
	sort.Slice(sortedChildren, func(i, j int) bool {
		return sortedChildren[i].Name < sortedChildren[j].Name
	})

	// 使用排序后的切片计算哈希
	for _, child := range sortedChildren {
		hash, err := child.CalculateHash()
		if err != nil {
			return "", err
		}
		hashes = append(hashes, hash)
	}

	combined := []byte(strings.Join(hashes, ""))
	hash := sha256.Sum256(combined)
	return hex.EncodeToString(hash[:]), nil
}

func countNodes(node *Node) int {
	if node == nil || node.Name == "." {
		return 0
	}

	// 检查是否是特殊目录
	if node.Info != nil && node.Info.IsDir() && node.Info.Name() == "." {
		return 0
	}

	count := 1 // 当前节点
	for _, child := range node.Children {
		count += countNodes(child)
	}
	return count
}

  - [options.go](#options.go)

### options.go

package project

import (
	"github.com/sjzsdu/tong/helper"
)

// DefaultWalkDirOptions 返回默认的文件遍历选项
func DefaultWalkDirOptions() helper.WalkDirOptions {
	return helper.WalkDirOptions{
		DisableGitIgnore: false,
		Extensions:       []string{"*"}, // 所有文件类型
		Excludes:         []string{},    // 不排除任何文件
	}
}
  - [output](#output)

### output

    - [exporter.go](#exporter.go)

#### exporter.go

package output

import (
	"github.com/sjzsdu/tong/project"
)

// Exporter 定义了导出器接口，与project.Exporter保持一致
type Exporter interface {
	// Export 将项目导出到指定路径
	Export(outputPath string) error
}

type BaseExporter struct {
	*project.BaseExporter
}

// NewBaseExporter 创建一个基本导出器
func NewBaseExporter(p *project.Project, collector project.ContentCollector) *BaseExporter {
	return &BaseExporter{
		BaseExporter: project.NewBaseExporter(p, collector),
	}
}

// Export 实现Exporter接口
func (b *BaseExporter) Export(outputPath string) error {
	return b.BaseExporter.Export(outputPath)
}

    - [factory.go](#factory.go)

#### factory.go

package output

import (
	"fmt"
	"path/filepath"

	"github.com/sjzsdu/tong/project"
)

// Output 将项目导出为指定格式的文件
func Output(doc *project.Project, outputFile string) error {
	// 获取导出器
	exporter, err := GetExporter(doc, outputFile)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return err
	}
	
	// 执行导出
	if err := exporter.Export(outputFile); err != nil {
		fmt.Printf("Error exporting to %s: %v\n", outputFile, err)
		return err
	}

	fmt.Printf("Successfully exported project to %s\n", outputFile)
	return nil
}

// GetExporter 根据输出文件类型返回对应的导出器
func GetExporter(doc *project.Project, outputFile string) (Exporter, error) {
	switch filepath.Ext(outputFile) {
	case ".md":
		return NewMarkdownExporter(doc), nil
	case ".pdf":
		return NewPDFExporter(doc)
	case ".xml":
		return NewXMLExporter(doc), nil
	default:
		return nil, fmt.Errorf("unsupported output format: %s", filepath.Ext(outputFile))
	}
}

    - [markdown.go](#markdown.go)

#### markdown.go

package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/sjzsdu/tong/project"
)

type MarkdownCollector struct {
	content strings.Builder
}

func NewMarkdownCollector() *MarkdownCollector {
	return &MarkdownCollector{
		content: strings.Builder{},
	}
}

func (m *MarkdownCollector) AddTOCItem(title string, level int) error {
	if level > 0 {
		indent := strings.Repeat("  ", level-1)
		anchor := strings.ToLower(strings.ReplaceAll(title, " ", "-"))
		m.content.WriteString(fmt.Sprintf("%s- [%s](#%s)\n",
			indent, title, anchor))
	}
	return nil
}

func (m *MarkdownCollector) AddTitle(title string, level int) error {
	m.content.WriteString(fmt.Sprintf("\n%s %s\n\n", strings.Repeat("#", level+1), title))
	return nil
}

func (m *MarkdownCollector) AddContent(content string) error {
	m.content.WriteString(content)
	m.content.WriteString("\n")
	return nil
}

func (m *MarkdownCollector) Render(outputPath string) error {
	return os.WriteFile(outputPath, []byte(m.content.String()), 0644)
}

type MarkdownExporter struct {
	*BaseExporter
}

func NewMarkdownExporter(p *project.Project) *MarkdownExporter {
	return &MarkdownExporter{
		BaseExporter: NewBaseExporter(p, NewMarkdownCollector()),
	}
}

// Export 实现Exporter接口
func (m *MarkdownExporter) Export(outputPath string) error {
	return m.BaseExporter.Export(outputPath)
}

    - [pdf.go](#pdf.go)

#### pdf.go

package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/project"
)

// PDFCollector 实现 ContentCollector 接口
type PDFCollector struct {
	pdf      *gofpdf.Fpdf
	tocItems []struct {
		title string
		page  int
		level int
	}
	fontName string
	tocPage  int
}

// 添加辅助函数用于清理文本
func cleanText(text string) string {
	runes := []rune(text)
	result := make([]rune, 0, len(runes))
	for _, r := range runes {
		if r <= 0xFFFF {
			result = append(result, r)
		}
	}
	return string(result)
}

func NewPDFCollector() (*PDFCollector, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 10)

	// 使用嵌入式字体
	fontPath, err := helper.UseEmbeddedFont("")
	if err != nil {
		return nil, fmt.Errorf("error loading embedded font: %v", err)
	}

	fontName := helper.FONT_NAME

	// 读取字体文件
	fontData, err := os.ReadFile(fontPath)
	if err != nil {
		return nil, fmt.Errorf("error reading font file: %v", err)
	}

	// 添加字体
	pdf.AddUTF8FontFromBytes(fontName, "", fontData)
	pdf.SetFont(fontName, "", 12)

	// 添加目录页
	pdf.AddPage()

	return &PDFCollector{
		pdf: pdf,
		tocItems: make([]struct {
			title string
			page  int
			level int
		}, 0),
		fontName: fontName,
		tocPage:  1,
	}, nil
}

// AddTitle 实现 ContentCollector 接口
func (p *PDFCollector) AddTitle(title string, level int) error {
	// 只有在不是第一页时才添加新页面
	if p.pdf.PageNo() > 0 {
		p.pdf.AddPage()
	}
	cleanTitle := cleanText(title)
	p.pdf.SetFont(p.fontName, "", 14+float64(4-level))
	p.pdf.CellFormat(190, 10, cleanTitle, "", 1, "L", false, 0, "")
	p.pdf.SetFont(p.fontName, "", 12)
	return nil
}

// AddContent 实现 ContentCollector 接口
func (p *PDFCollector) AddContent(content string) error {
	// 确保当前页面存在
	if p.pdf.PageNo() == 0 {
		p.pdf.AddPage()
	}
	cleanContent := cleanText(content)
	p.pdf.MultiCell(190, 5, cleanContent, "", "L", false)
	return nil
}

// AddTOCItem 实现 ContentCollector 接口
func (p *PDFCollector) AddTOCItem(title string, level int) error {
	cleanTitle := cleanText(title)
	p.tocItems = append(p.tocItems, struct {
		title string
		page  int
		level int
	}{
		title: cleanTitle,
		page:  p.pdf.PageNo() + 1, // 修正页码计算
		level: level,
	})
	return nil
}

// writeTOC 生成目录
func (p *PDFCollector) writeTOC() error {
	if len(p.tocItems) == 0 {
		return nil
	}

	p.pdf.SetPage(p.tocPage)
	p.pdf.SetY(30)
	p.pdf.SetFont(p.fontName, "", 16)
	p.pdf.CellFormat(190, 10, "目录", "", 1, "C", false, 0, "")
	p.pdf.SetFont(p.fontName, "", 12)

	for _, item := range p.tocItems {
		indent := strings.Repeat(" ", item.level)
		titleWidth := 150 - float64(item.level*5)

		x, y := p.pdf.GetXY()

		p.pdf.CellFormat(float64(item.level*5), 5, "", "", 0, "L", false, 0, "")
		p.pdf.CellFormat(titleWidth, 5, indent+item.title, "", 0, "L", false, 0, "")
		p.pdf.CellFormat(40, 5, fmt.Sprintf("%d", item.page), "", 1, "R", false, 0, "")

		link := p.pdf.AddLink()
		p.pdf.SetLink(link, 0, item.page) // 不需要再加1
		p.pdf.Link(x, y, 190, 5, link)
	}

	return nil
}

// Render 实现 ContentCollector 接口
func (p *PDFCollector) Render(outputPath string) error {
	if err := p.writeTOC(); err != nil {
		return err
	}

	// 确保输出目录存在
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	return p.pdf.OutputFileAndClose(outputPath)
}

// PDFExporter 使用 BaseExporter 和 PDFCollector
type PDFExporter struct {
	*project.BaseExporter
}

// NewPDFExporter 创建一个新的 PDF 导出器
func NewPDFExporter(p *project.Project) (*PDFExporter, error) {
	collector, err := NewPDFCollector()
	if err != nil {
		return nil, err
	}

	return &PDFExporter{
		BaseExporter: project.NewBaseExporter(p, collector),
	}, nil
}

// Export 实现Exporter接口
func (p *PDFExporter) Export(outputPath string) error {
	return p.BaseExporter.Export(outputPath)
}

    - [xml.go](#xml.go)

#### xml.go

package output

import (
	"encoding/xml"
	"os"

	"github.com/sjzsdu/tong/project"
)

type XMLNode struct {
	XMLName  xml.Name  `xml:"node"`
	Name     string    `xml:"name,attr"`
	Type     string    `xml:"type,attr"`
	Content  *string   `xml:"content,omitempty"`
	Children []XMLNode `xml:"nodes>node,omitempty"`
}

type XMLDocument struct {
	XMLName xml.Name  `xml:"document"`
	TOC     []XMLNode `xml:"toc>item,omitempty"`
	Nodes   []XMLNode `xml:"nodes>node"`
}

type XMLCollector struct {
	doc XMLDocument
}

func NewXMLCollector() *XMLCollector {
	return &XMLCollector{
		doc: XMLDocument{},
	}
}

func (x *XMLCollector) AddTitle(title string, level int) error {
	node := XMLNode{
		Name: title,
		Type: "directory",
	}
	x.doc.Nodes = append(x.doc.Nodes, node)
	return nil
}

func (x *XMLCollector) AddContent(content string) error {
	node := XMLNode{
		Type:    "file",
		Content: &content,
	}
	x.doc.Nodes = append(x.doc.Nodes, node)
	return nil
}

// AddTOCItem 实现可选的目录项添加
func (x *XMLCollector) AddTOCItem(title string, level int) error {
	node := XMLNode{
		Name: title,
		Type: "toc",
		// 由于 XMLNode 结构体中没有 Level 字段，需要将 level 信息存储在其他现有字段中
		// 这里我们可以考虑将 level 信息编码到 Name 或 Content 中，或者添加自定义属性
	}
	x.doc.TOC = append(x.doc.TOC, node)
	return nil
}

func (x *XMLCollector) Render(outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	file.WriteString(xml.Header)
	encoder := xml.NewEncoder(file)
	encoder.Indent("", "  ")
	return encoder.Encode(x.doc)
}

type XMLExporter struct {
	*BaseExporter
}

func NewXMLExporter(p *project.Project) *XMLExporter {
	return &XMLExporter{
		BaseExporter: NewBaseExporter(p, NewXMLCollector()),
	}
}

// Export 实现Exporter接口
func (x *XMLExporter) Export(outputPath string) error {
	return x.BaseExporter.Export(outputPath)
}

  - [project.go](#project.go)

### project.go

package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// NewProject 创建一个新的文档树
func NewProject(rootPath string) *Project {
	return &Project{
		root: &Node{
			Name:     "/",
			IsDir:    true,
			Children: make(map[string]*Node),
		},
		rootPath: rootPath,
	}
}

func (d *Project) GetRootPath() string {
	return d.rootPath
}

// CreateDir 创建一个新目录
func (d *Project) CreateDir(path string, info os.FileInfo) error {
	if path == "." {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	parent, name, err := d.resolvePath(path)
	if err != nil {
		return err
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	if _, exists := parent.Children[name]; exists {
		return errors.New("directory already exists")
	}

	parent.Children[name] = &Node{
		Name:     name,
		IsDir:    true,
		Info:     info,
		Children: make(map[string]*Node),
		Parent:   parent,
	}

	return nil
}

// CreateFile 创建一个新文件
func (d *Project) CreateFile(path string, content []byte, info os.FileInfo) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	parent, name, err := d.resolvePath(path)
	if err != nil {
		return err
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	if _, exists := parent.Children[name]; exists {
		return errors.New("file already exists")
	}

	parent.Children[name] = &Node{
		Name:     name,
		IsDir:    false,
		Info:     info,
		Content:  content,
		Parent:   parent,
		Children: make(map[string]*Node),
	}

	return nil
}

// ReadFile 读取文件内容
func (d *Project) ReadFile(path string) ([]byte, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	node, err := d.findNode(path)
	if err != nil {
		return nil, err
	}

	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.IsDir {
		return nil, errors.New("cannot read directory")
	}

	return node.Content, nil
}

// WriteFile 写入文件内容
func (d *Project) WriteFile(path string, content []byte) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	node, err := d.findNode(path)
	if err != nil {
		return err
	}

	node.mu.Lock()
	defer node.mu.Unlock()

	if node.IsDir {
		return errors.New("cannot write to directory")
	}

	node.Content = content
	return nil
}

// 辅助函数，用于解析路径
func (d *Project) resolvePath(path string) (*Node, string, error) {
	// 处理根路径
	if path == "/" || path == "" {
		return d.root, "", nil
	}

	// 清理路径
	path = filepath.Clean(path)
	// 移除开头的 /
	if path[0] == '/' {
		path = path[1:]
	}

	// 分割路径组件
	components := strings.Split(path, string(filepath.Separator))
	parent := d.root

	// 遍历到倒数第二个组件
	for i := 0; i < len(components)-1; i++ {
		comp := components[i]
		if comp == "" {
			continue
		}

		parent.mu.RLock()
		child, ok := parent.Children[comp]
		parent.mu.RUnlock()

		if !ok {
			return parent, components[len(components)-1], nil
		}
		if !child.IsDir {
			return nil, "", errors.New("path component is not a directory")
		}
		parent = child
	}

	return parent, components[len(components)-1], nil
}

// 辅助函数，用于查找节点
func (d *Project) findNode(path string) (*Node, error) {
	// 处理根路径
	if path == "/" || path == "" {
		return d.root, nil
	}

	// 清理路径
	path = filepath.Clean(path)
	// 移除开头的 /
	if path[0] == '/' {
		path = path[1:]
	}

	// 分割路径组件
	components := strings.Split(path, string(filepath.Separator))
	current := d.root

	// 遍历所有组件
	for _, comp := range components {
		if comp == "" {
			continue
		}

		current.mu.RLock()
		child, ok := current.Children[comp]
		current.mu.RUnlock()

		if !ok {
			return nil, errors.New("path not found")
		}
		current = child
	}

	return current, nil
}

// IsEmpty 检查项目是否为空
func (d *Project) IsEmpty() bool {
	if d == nil || d.root == nil {
		return true
	}

	d.root.mu.RLock()
	defer d.root.mu.RUnlock()

	return len(d.root.Children) == 0
}

func (p *Project) GetAbsolutePath(path string) string {
	return filepath.Join(p.rootPath, path)
}

// GetTotalNodes 计算项目中的总节点数（文件+目录）
func (p *Project) GetTotalNodes() int {
	if p.root == nil {
		return 0
	}
	return countNodes(p.root)
}

// GetAllFiles 返回项目中所有文件的相对路径
func (p *Project) GetAllFiles() ([]string, error) {
	if p.root == nil {
		return nil, fmt.Errorf("project root is nil")
	}

	var files []string
	traverser := NewTreeTraverser(p)
	visitor := VisitorFunc(func(path string, node *Node, depth int) error {
		if node.IsDir {
			return nil
		}
		files = append(files, path)
		return nil
	})
	err := traverser.TraverseTree(visitor)

	if err != nil {
		return nil, err
	}
	return files, nil
}

func (p *Project) GetName() string {
	if p.rootPath == "" {
		return "root"
	}
	return filepath.Base(p.rootPath)
}

// FindNode 查找指定路径的节点（公开方法）
func (p *Project) FindNode(path string) (*Node, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return p.findNode(path)
}

  - [search](#search)

### search

    - [search_engine.go](#search_engine.go)

#### search_engine.go

package search

import (
	"bufio"
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/sjzsdu/tong/project"
)

// SearchOptions 搜索选项
type SearchOptions struct {
	CaseSensitive bool     // 区分大小写
	WholeWord     bool     // 全词匹配
	RegexMode     bool     // 正则表达式模式
	FileTypes     []string // 限定文件类型
	MaxResults    int      // 最大结果数
}

// SearchResult 搜索结果
type SearchResult struct {
	FilePath    string // 文件路径
	LineNumber  int    // 行号
	ColumnStart int    // 列开始位置
	ColumnEnd   int    // 列结束位置
	LineContent string // 行内容
	Context     string // 上下文内容
}

// SearchEngine 搜索引擎接口
type SearchEngine interface {
	// 构建搜索索引
	BuildIndex(project *project.Project) error
	// 搜索关键词
	Search(query string, options SearchOptions) ([]SearchResult, error)
}

// DefaultSearchEngine 默认搜索引擎实现
type DefaultSearchEngine struct {
	project     *project.Project
	indexed     bool
	fileContent map[string][]byte
	mu          sync.RWMutex
}

// NewDefaultSearchEngine 创建一个新的默认搜索引擎
func NewDefaultSearchEngine() *DefaultSearchEngine {
	return &DefaultSearchEngine{
		fileContent: make(map[string][]byte),
		indexed:     false,
	}
}

// BuildIndex 实现 SearchEngine 接口
func (s *DefaultSearchEngine) BuildIndex(p *project.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.project = p
	s.fileContent = make(map[string][]byte)

	// 创建访问者函数
	visitor := project.VisitorFunc(func(path string, node *project.Node, depth int) error {
		if !node.IsDir {
			s.fileContent[path] = node.Content
		}
		return nil
	})

	// 遍历项目树
	traverser := project.NewTreeTraverser(p)
	err := traverser.TraverseTree(visitor)
	if err != nil {
		return err
	}

	s.indexed = true
	return nil
}

// Search 实现 SearchEngine 接口
func (s *DefaultSearchEngine) Search(query string, options SearchOptions) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.indexed {
		return nil, fmt.Errorf("搜索引擎尚未建立索引")
	}

	var results []SearchResult
	var wg sync.WaitGroup
	var resultsMu sync.Mutex

	// 准备正则表达式
	var re *regexp.Regexp
	var err error
	if options.RegexMode {
		// 使用用户提供的正则表达式
		re, err = regexp.Compile(query)
		if err != nil {
			return nil, fmt.Errorf("无效的正则表达式: %v", err)
		}
	} else {
		// 构建搜索模式
		pattern := regexp.QuoteMeta(query)
		if options.WholeWord {
			pattern = fmt.Sprintf("\\b%s\\b", pattern)
		}
		if options.CaseSensitive {
			re, err = regexp.Compile(pattern)
		} else {
			re, err = regexp.Compile("(?i)" + pattern)
		}
		if err != nil {
			return nil, fmt.Errorf("无法编译搜索模式: %v", err)
		}
	}

	// 创建通道用于限制并发数
	semaphore := make(chan struct{}, 10) // 最多10个并发搜索

	// 遍历所有文件
	for path, content := range s.fileContent {
		// 检查文件类型
		if len(options.FileTypes) > 0 {
			ext := strings.TrimPrefix(filepath.Ext(path), ".")
			if !contains(options.FileTypes, ext) && !contains(options.FileTypes, "*") {
				continue
			}
		}

		wg.Add(1)
		go func(filePath string, fileContent []byte) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 搜索文件内容
			fileResults := s.searchInFile(filePath, fileContent, re, options)

			// 添加到结果集
			if len(fileResults) > 0 {
				resultsMu.Lock()
				results = append(results, fileResults...)
				resultsMu.Unlock()
			}
		}(path, content)
	}

	wg.Wait()

	// 限制结果数量
	if options.MaxResults > 0 && len(results) > options.MaxResults {
		results = results[:options.MaxResults]
	}

	return results, nil
}

// searchInFile 在单个文件中搜索
func (s *DefaultSearchEngine) searchInFile(filePath string, content []byte, re *regexp.Regexp, options SearchOptions) []SearchResult {
	var results []SearchResult

	// 按行读取文件内容
	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNumber := 0
	contextLines := make([]string, 0, 5) // 保存上下文行

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// 更新上下文行
		if len(contextLines) >= 4 {
			contextLines = append(contextLines[1:], line)
		} else {
			contextLines = append(contextLines, line)
		}

		// 查找匹配
		matches := re.FindAllStringIndex(line, -1)
		if len(matches) > 0 {
			for _, match := range matches {
				// 构建上下文
				context := strings.Join(contextLines[:len(contextLines)-1], "\n")

				// 创建搜索结果
				result := SearchResult{
					FilePath:    filePath,
					LineNumber:  lineNumber,
					ColumnStart: match[0] + 1, // 1-indexed
					ColumnEnd:   match[1] + 1, // 1-indexed
					LineContent: line,
					Context:     context,
				}
				results = append(results, result)
			}
		}
	}

	return results
}

// contains 检查切片是否包含指定元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// FormatSearchResults 格式化搜索结果
func FormatSearchResults(results []SearchResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("找到 %d 个结果:\n\n", len(results)))

	// 按文件分组
	fileGroups := make(map[string][]SearchResult)
	for _, result := range results {
		fileGroups[result.FilePath] = append(fileGroups[result.FilePath], result)
	}

	// 输出每个文件的结果
	for filePath, fileResults := range fileGroups {
		sb.WriteString(fmt.Sprintf("文件: %s (%d 个匹配)\n", filePath, len(fileResults)))
		for _, result := range fileResults {
			sb.WriteString(fmt.Sprintf("  行 %d, 列 %d-%d: %s\n", 
				result.LineNumber, 
				result.ColumnStart, 
				result.ColumnEnd,
				result.LineContent))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// MarkdownSearchFormatter Markdown格式的搜索结果格式化器
type MarkdownSearchFormatter struct{}

// Format 格式化搜索结果为Markdown
func (m *MarkdownSearchFormatter) Format(results []SearchResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# 搜索结果\n\n"))
	sb.WriteString(fmt.Sprintf("找到 **%d** 个结果\n\n", len(results)))

	// 按文件分组
	fileGroups := make(map[string][]SearchResult)
	for _, result := range results {
		fileGroups[result.FilePath] = append(fileGroups[result.FilePath], result)
	}

	// 输出每个文件的结果
	for filePath, fileResults := range fileGroups {
		sb.WriteString(fmt.Sprintf("## %s\n\n", filePath))
		sb.WriteString(fmt.Sprintf("*%d 个匹配*\n\n", len(fileResults)))

		for _, result := range fileResults {
			// 高亮显示匹配部分
			highlightedLine := result.LineContent
			prefix := highlightedLine[:result.ColumnStart-1]
			match := highlightedLine[result.ColumnStart-1:result.ColumnEnd-1]
			suffix := highlightedLine[result.ColumnEnd-1:]
			highlightedLine = fmt.Sprintf("%s**%s**%s", prefix, match, suffix)

			sb.WriteString(fmt.Sprintf("- 行 **%d**, 列 %d-%d:\n  ```\n  %s\n  ```\n\n", 
				result.LineNumber, 
				result.ColumnStart, 
				result.ColumnEnd,
				highlightedLine))
		}
	}

	return sb.String()
}

// HTMLSearchFormatter HTML格式的搜索结果格式化器
type HTMLSearchFormatter struct{}

// Format 格式化搜索结果为HTML
func (h *HTMLSearchFormatter) Format(results []SearchResult) string {
	var sb strings.Builder

	// 添加HTML头部
	sb.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	sb.WriteString("<title>搜索结果</title>\n")
	sb.WriteString("<style>\n")
	sb.WriteString("body { font-family: Arial, sans-serif; margin: 20px; }\n")
	sb.WriteString("h1, h2 { color: #333; }\n")
	sb.WriteString(".summary { margin-bottom: 20px; }\n")
	sb.WriteString(".file { margin-bottom: 30px; }\n")
	sb.WriteString(".file-path { font-weight: bold; color: #0066cc; }\n")
	sb.WriteString(".match-count { color: #666; font-style: italic; }\n")
	sb.WriteString(".result { margin: 10px 0; padding: 5px; border-left: 3px solid #ccc; }\n")
	sb.WriteString(".line-number { color: #999; margin-right: 10px; }\n")
	sb.WriteString(".line-content { font-family: monospace; white-space: pre; }\n")
	sb.WriteString(".highlight { background-color: #ffff00; font-weight: bold; }\n")
	sb.WriteString("</style>\n")
	sb.WriteString("</head>\n<body>\n")

	// 添加标题和摘要
	sb.WriteString("<h1>搜索结果</h1>\n")
	sb.WriteString(fmt.Sprintf("<div class=\"summary\">找到 <strong>%d</strong> 个结果</div>\n", len(results)))

	// 按文件分组
	fileGroups := make(map[string][]SearchResult)
	for _, result := range results {
		fileGroups[result.FilePath] = append(fileGroups[result.FilePath], result)
	}

	// 输出每个文件的结果
	for filePath, fileResults := range fileGroups {
		sb.WriteString(fmt.Sprintf("<div class=\"file\">\n"))
		sb.WriteString(fmt.Sprintf("<h2><span class=\"file-path\">%s</span> <span class=\"match-count\">(%d 个匹配)</span></h2>\n", filePath, len(fileResults)))

		for _, result := range fileResults {
			// 高亮显示匹配部分
			highlightedLine := result.LineContent
			prefix := highlightedLine[:result.ColumnStart-1]
			match := highlightedLine[result.ColumnStart-1:result.ColumnEnd-1]
			suffix := highlightedLine[result.ColumnEnd-1:]
			highlightedLine = fmt.Sprintf("%s<span class=\"highlight\">%s</span>%s", prefix, match, suffix)

			sb.WriteString(fmt.Sprintf("<div class=\"result\">\n"))
			sb.WriteString(fmt.Sprintf("<div><span class=\"line-number\">行 %d, 列 %d-%d:</span></div>\n", 
				result.LineNumber, 
				result.ColumnStart, 
				result.ColumnEnd))
			sb.WriteString(fmt.Sprintf("<div class=\"line-content\">%s</div>\n", highlightedLine))
			sb.WriteString("</div>\n")
		}

		sb.WriteString("</div>\n")
	}

	// 添加HTML尾部
	sb.WriteString("</body>\n</html>")

	return sb.String()
}
  - [traverser.go](#traverser.go)

### traverser.go

package project

import (
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// NodeVisitor 定义了节点访问器的接口
type NodeVisitor interface {
	// VisitDirectory 访问目录节点
	VisitDirectory(node *Node, path string, level int) error
	// VisitFile 访问文件节点
	VisitFile(node *Node, path string, level int) error
}

// TraverseOrder 定义遍历顺序
type TraverseOrder int

const (
	PreOrder  TraverseOrder = iota // 前序遍历
	PostOrder                      // 后序遍历
	InOrder                        // 中序遍历
)

// TraverseOption 定义遍历选项
type TraverseOption struct {
	ContinueOnError bool    // 遇到错误时是否继续
	Errors          []error // 记录所有错误
}

// TreeTraverser 提供了树遍历的基本功能
type TreeTraverser struct {
	project *Project
	order   TraverseOrder
	option  *TraverseOption
	wg      sync.WaitGroup // 添加等待组
}

// SetOption 设置遍历选项
func (t *TreeTraverser) SetOption(option *TraverseOption) {
	t.option = option
}

// NewTreeTraverser 创建一个树遍历器，默认使用前序遍历
func NewTreeTraverser(p *Project) *TreeTraverser {
	return &TreeTraverser{
		project: p,
		order:   PreOrder,
		option:  nil,
	}
}

// SetTraverseOrder 设置遍历顺序
func (t *TreeTraverser) SetTraverseOrder(order TraverseOrder) *TreeTraverser {
	t.order = order
	return t
}

// TraverseTree 遍历整个项目树
func (t *TreeTraverser) TraverseTree(visitor NodeVisitor) error {
	if t.project.root == nil {
		return nil
	}
	return t.Traverse(t.project.root, "/", 0, visitor)
}

// traversePreOrder 处理前序遍历
func (t *TreeTraverser) traversePreOrder(node *Node, children []*Node, path string, level int, visitor NodeVisitor) error {
	if err := visitor.VisitDirectory(node, path, level); err != nil {
		return err
	}
	for _, child := range children {
		childPath := filepath.Join(path, child.Name)
		if err := t.Traverse(child, childPath, level+1, visitor); err != nil {
			return err
		}
	}
	return nil
}

// traverseError 封装遍历过程中的错误信息
type traverseError struct {
	Path     string
	NodeName string
	Err      error
}

func (e *traverseError) Error() string {
	return fmt.Sprintf("遍历错误 [%s] 在节点 '%s': %v", e.Path, e.NodeName, e.Err)
}

// 添加一个用于限制并发的常量
const maxConcurrentTraversals = 10

// traversePostOrder 处理后序遍历
func (t *TreeTraverser) traversePostOrder(node *Node, children []*Node, path string, level int, visitor NodeVisitor) error {
	// 初始化选项
	if t.option == nil {
		t.option = &TraverseOption{
			ContinueOnError: false,
			Errors:          make([]error, 0),
		}
	}

	var wg sync.WaitGroup
	errChan := make(chan *traverseError, len(children))

	// 使用信号量限制并发
	sem := make(chan struct{}, maxConcurrentTraversals)

	// 处理子节点
	for _, child := range children {
		childPath := filepath.Join(path, child.Name)
		wg.Add(1)
		go func(c *Node, p string) {
			// 获取信号量
			sem <- struct{}{}
			defer func() {
				<-sem // 释放信号量
				if r := recover(); r != nil {
					errChan <- &traverseError{
						Path:     p,
						NodeName: c.Name,
						Err:      fmt.Errorf("panic in traversal: %v", r),
					}
				}
				wg.Done()
			}()

			if err := t.Traverse(c, p, level+1, visitor); err != nil {
				errChan <- &traverseError{
					Path:     p,
					NodeName: c.Name,
					Err:      err,
				}
			}
		}(child, childPath)
	}

	// 等待所有子节点完成并收集错误
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
		close(errChan)
	}()

	// 收集所有错误，设置超时
	var errs []error
	timeout := time.After(5 * time.Minute) // 设置合理的超时时间

	for {
		select {
		case err, ok := <-errChan:
			if !ok {
				goto PROCESS_DIRECTORY
			}
			if err != nil {
				if t.option.ContinueOnError {
					errs = append(errs, err)
				} else {
					return err
				}
			}
		case <-timeout:
			return fmt.Errorf("遍历超时: 路径 '%s'", path)
		case <-done:
			goto PROCESS_DIRECTORY
		}
	}

PROCESS_DIRECTORY:
	// 如果有错误且设置了继续执行
	if len(errs) > 0 {
		t.option.Errors = append(t.option.Errors, errs...)
		if !t.option.ContinueOnError {
			return fmt.Errorf("遍历过程中发生 %d 个错误", len(errs))
		}
	}

	// 所有子节点处理完成后，处理当前目录
	if err := visitor.VisitDirectory(node, path, level); err != nil {
		return &traverseError{
			Path:     path,
			NodeName: node.Name,
			Err:      err,
		}
	}

	return nil
}

// traverseInOrder 处理中序遍历
func (t *TreeTraverser) traverseInOrder(node *Node, children []*Node, path string, level int, visitor NodeVisitor) error {
	mid := len(children) / 2

	// 前半部分
	for i := 0; i < mid; i++ {
		childPath := filepath.Join(path, children[i].Name)
		if err := t.Traverse(children[i], childPath, level+1, visitor); err != nil {
			return err
		}
	}

	// 当前节点
	if err := visitor.VisitDirectory(node, path, level); err != nil {
		return err
	}

	// 后半部分
	for i := mid; i < len(children); i++ {
		childPath := filepath.Join(path, children[i].Name)
		if err := t.Traverse(children[i], childPath, level+1, visitor); err != nil {
			return err
		}
	}
	return nil
}

// Traverse 遍历节点的通用方法
func (t *TreeTraverser) Traverse(node *Node, path string, level int, visitor NodeVisitor) error {
	if node == nil {
		return nil
	}

	if !node.IsDir {
		if err := visitor.VisitFile(node, path, level); err != nil {
			fmt.Println("visit file error:", err)
			return err
		}
		return nil
	}

	if node.Name == "." {
		return nil
	}

	// 对子节点进行排序
	children := make([]*Node, 0, len(node.Children))
	for _, child := range node.Children {
		children = append(children, child)
	}
	sort.Slice(children, func(i, j int) bool {
		return children[i].Name < children[j].Name
	})

	// 根据遍历顺序选择相应的处理方法
	switch t.order {
	case PreOrder:
		return t.traversePreOrder(node, children, path, level, visitor)
	case PostOrder:
		return t.traversePostOrder(node, children, path, level, visitor)
	case InOrder:
		return t.traverseInOrder(node, children, path, level, visitor)
	}

	return nil
}

  - [type.go](#type.go)

### type.go

package project

import (
	"os"
	"sync"
)

type Node struct {
	Name     string
	IsDir    bool
	Info     os.FileInfo
	Content  []byte
	Children map[string]*Node
	Parent   *Node
	mu       sync.RWMutex
}

// Project 表示整个文档树
type Project struct {
	root     *Node
	rootPath string
	mu       sync.RWMutex
}

type Item struct {
	Name    string `json:"name"`
	Feature string `json:"feature"`
}

type Response struct {
	Functions    []Item `json:"functions"`
	Classes      []Item `json:"classes"`
	Interfaces   []Item `json:"interfaces"`
	Variables    []Item `json:"variables"`
	OtherSymbols []Item `json:"other_symbols"`
}

  - [visitor.go](#visitor.go)

### visitor.go

package project

// VisitorFunc 定义了访问节点的函数类型
type VisitorFunc func(path string, node *Node, depth int) error

// VisitFile 实现 NodeVisitor 接口
func (f VisitorFunc) VisitFile(node *Node, path string, level int) error {
	return f(path, node, level)
}

// VisitDirectory 实现 NodeVisitor 接口
func (f VisitorFunc) VisitDirectory(node *Node, path string, level int) error {
	return f(path, node, level)
}

- [setup.sh](#setup.sh)

## setup.sh

#!/bin/bash

# 编译项目
go build .

# 移动编译后的二进制文件到 bin 目录
mv tong /Users/juzhongsun/.local/bin/tong
echo "安装完成！"
- [share](#share)

## share

  - [const.go](#const.go)

### const.go

package share

import "time"

// VERSION 版本号
const VERSION = "1.0.0"

// BUILDNAME 制品名称
const BUILDNAME = "tong"

const PREFIX = "TONG_"

const PATH = ".tong"

const TIMEOUT = time.Second * 60 * 5

const MAX_TOKENS = 8192

const CACHE_TYPE = "json"

const NOT_PROGRAM_TIP = "This is not a program file."

const DEFAULT_RENDERER = "markdown"

const SERVER_PORT = 3000

  - [debug.go](#debug.go)

### debug.go

package share

var debug = false

func SetDebug(d bool) {
	debug = d
}

func GetDebug() bool {
	return debug
}

