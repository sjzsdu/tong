package project

import (
	"os"
	"path/filepath"
	"testing"
)

// GoProject 是一个Go项目的测试环境
type GoProject struct {
	// 项目根目录路径
	RootPath string
	// 项目对象
	Project *Project
}

// 全局共享的项目实例
var sharedProject *GoProject

// GetSharedProject 获取全局共享的项目实例
// 如果项目还未初始化，则使用提供的路径初始化项目
// 如果路径为空，则使用当前工作目录
func GetSharedProject(t *testing.T, projectPath string) *GoProject {
	if sharedProject != nil {
		return sharedProject
	}

	// 如果未提供路径，使用当前工作目录或示例项目目录
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			t.Fatalf("无法获取当前工作目录: %v", err)
		}
		// 如果当前目录是测试目录，则使用上一级目录
		if filepath.Base(projectPath) == "testutil" {
			projectPath = filepath.Dir(projectPath)
		}
	}

	// 初始化项目
	sharedProject = &GoProject{
		RootPath: projectPath,
	}

	// 构建项目树
	options := DefaultWalkDirOptions()
	// 添加额外的排除项
	options.Excludes = []string{".git", "node_modules", "vendor"}

	var err error
	project, err := BuildProjectTree(projectPath, options)
	if err == nil {
		sharedProject.Project = project
	}
	if err != nil {
		t.Fatalf("构建项目树失败: %v", err)
	}

	return sharedProject
}

// GetProject 获取项目对象
func (gp *GoProject) GetProject() *Project {
	return gp.Project
}

// GetAbsolutePath 获取项目中文件的绝对路径
func (gp *GoProject) GetAbsolutePath(relativePath string) string {
	// 如果路径以 / 开头，则移除开头的 /
	if len(relativePath) > 0 && relativePath[0] == '/' {
		relativePath = relativePath[1:]
	}
	return filepath.Join(gp.RootPath, relativePath)
}

// CreateExampleGoProject 创建一个示例项目
// 这个函数用于创建一个简单的Go项目，可以用于测试
// 返回项目的根目录路径
func CreateExampleGoProject(t *testing.T) string {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "example-project-*")
	if err != nil {
		t.Fatalf("无法创建临时测试目录: %v", err)
	}

	// 创建基本的Go项目结构
	createGoModFile(t, tempDir)
	createMainFile(t, tempDir)
	createPackageFiles(t, tempDir)
	createTestFiles(t, tempDir)
	createConfigFiles(t, tempDir)
	createDocFiles(t, tempDir)

	return tempDir
}

// 创建go.mod文件
func createGoModFile(t *testing.T, projectDir string) {
	content := []byte(`module example.com/myproject

go 1.16

require (
	github.com/stretchr/testify v1.7.0
)
`)
	writeFile(t, filepath.Join(projectDir, "go.mod"), content)
}

// 创建main.go文件
func createMainFile(t *testing.T, projectDir string) {
	content := []byte(`package main

import (
	"fmt"
	"example.com/myproject/pkg/utils"
)

func main() {
	fmt.Println("Hello, World!")
	fmt.Println(utils.Greeting("User"))
}
`)
	writeFile(t, filepath.Join(projectDir, "main.go"), content)
}

// 创建包文件
func createPackageFiles(t *testing.T, projectDir string) {
	// utils包
	utilsDir := filepath.Join(projectDir, "pkg", "utils")
	os.MkdirAll(utilsDir, 0755)

	// utils/greeting.go
	greetingContent := []byte(`package utils

import "fmt"

// Greeting 返回一个问候语
func Greeting(name string) string {
	return "Hello, " + name + "!"
}

// FormatMessage 格式化消息
func FormatMessage(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}
`)
	writeFile(t, filepath.Join(utilsDir, "greeting.go"), greetingContent)

	// utils/math.go
	mathContent := []byte(`package utils

// Add 返回两个数的和
func Add(a, b int) int {
	return a + b
}

// Multiply 返回两个数的乘积
func Multiply(a, b int) int {
	return a * b
}
`)
	writeFile(t, filepath.Join(utilsDir, "math.go"), mathContent)

	// config包
	configDir := filepath.Join(projectDir, "pkg", "config")
	os.MkdirAll(configDir, 0755)

	// config/config.go
	configContent := []byte(`package config

import "os"

// Config 配置结构体
type Config struct {
	AppName string
	Version string
	Debug   bool
}

// NewConfig 创建一个新的配置
func NewConfig() *Config {
	return &Config{
		AppName: "MyApp",
		Version: "1.0.0",
		Debug:   os.Getenv("DEBUG") == "true",
	}
}
`)
	writeFile(t, filepath.Join(configDir, "config.go"), configContent)
}

// 创建测试文件
func createTestFiles(t *testing.T, projectDir string) {
	// utils/greeting_test.go
	greetingTestContent := []byte(`package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGreeting(t *testing.T) {
	result := Greeting("Test")
	assert.Equal(t, "Hello, Test!", result)
}
`)
	writeFile(t, filepath.Join(projectDir, "pkg", "utils", "greeting_test.go"), greetingTestContent)

	// utils/math_test.go
	mathTestContent := []byte(`package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdd(t *testing.T) {
	result := Add(2, 3)
	assert.Equal(t, 5, result)
}

func TestMultiply(t *testing.T) {
	result := Multiply(2, 3)
	assert.Equal(t, 6, result)
}
`)
	writeFile(t, filepath.Join(projectDir, "pkg", "utils", "math_test.go"), mathTestContent)
}

// 创建配置文件
func createConfigFiles(t *testing.T, projectDir string) {
	// config.json
	configContent := []byte(`{
	"app_name": "MyApp",
	"version": "1.0.0",
	"debug": false,
	"database": {
		"host": "localhost",
		"port": 5432,
		"user": "postgres",
		"password": "password",
		"name": "myapp"
	}
}`)
	writeFile(t, filepath.Join(projectDir, "config.json"), configContent)

	// .env
	envContent := []byte(`DEBUG=false
APP_PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=myapp
`)
	writeFile(t, filepath.Join(projectDir, ".env"), envContent)
}

// 创建文档文件
func createDocFiles(t *testing.T, projectDir string) {
	// README.md
	readmeContent := []byte(`# MyProject

这是一个示例项目，用于测试。

## 功能

- 基本的问候功能
- 简单的数学运算
- 配置管理

## 使用方法

` + "`" + `go
package main

import (
	"fmt"
	"example.com/myproject/pkg/utils"
)

func main() {
	fmt.Println(utils.Greeting("User"))
}
` + "`" + `
`)
	writeFile(t, filepath.Join(projectDir, "README.md"), readmeContent)

	// docs目录
	docsDir := filepath.Join(projectDir, "docs")
	os.MkdirAll(docsDir, 0755)

	// docs/api.md
	apiContent := []byte(`# API 文档

## 问候 API

### Greeting

返回一个问候语。

**参数：**

- name: 用户名

**返回：**

- 格式化的问候语

**示例：**

` + "`" + `go
result := utils.Greeting("User")
// 返回: "Hello, User!"
` + "`" + `
`)
	writeFile(t, filepath.Join(docsDir, "api.md"), apiContent)
}

// 辅助函数：写入文件
func writeFile(t *testing.T, path string, content []byte) {
	// 确保父目录存在
	parentDir := filepath.Dir(path)
	err := os.MkdirAll(parentDir, 0755)
	if err != nil {
		t.Fatalf("无法创建目录 %s: %v", parentDir, err)
	}

	// 写入文件内容
	err = os.WriteFile(path, content, 0644)
	if err != nil {
		t.Fatalf("无法写入文件 %s: %v", path, err)
	}
}
