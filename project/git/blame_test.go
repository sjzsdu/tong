package git

import (
	"fmt"
	"os"
	"testing"

	"github.com/sjzsdu/tong/project"
	"github.com/stretchr/testify/assert"
)

// TestBlameImplementations 测试两种blame实现的功能和性能
func TestBlameImplementations(t *testing.T) {
	// 跳过测试，因为需要一个真实的Git仓库环境
	t.Skip("此测试需要在真实的Git仓库环境中运行")

	// 创建项目实例 - 使用当前目录的绝对路径
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("无法获取当前工作目录: %v", err)
	}
	p := project.NewProject(cwd)

	// 测试文件路径 - 使用当前文件作为测试
	filePath := "blame.go" // 使用相对路径，因为Git blame通常使用相对路径

	// 测试命令行实现
	cmdLineBlamer := NewDefaultGitBlamer(p)
	cmdLineInfo, cmdLineErr := cmdLineBlamer.Blame(filePath)
	assert.NoError(t, cmdLineErr, "命令行实现应该成功")
	assert.NotNil(t, cmdLineInfo, "命令行实现应该返回blame信息")
	assert.Greater(t, cmdLineInfo.TotalLines, 0, "命令行实现应该返回行数大于0")

	// 测试go-git实现
	goGitBlamer, goGitErr := NewGoGitBlamer(p)
	assert.NoError(t, goGitErr, "go-git实现应该成功创建")
	goGitInfo, goGitErr := goGitBlamer.Blame(filePath)
	assert.NoError(t, goGitErr, "go-git实现应该成功")
	assert.NotNil(t, goGitInfo, "go-git实现应该返回blame信息")
	assert.Greater(t, goGitInfo.TotalLines, 0, "go-git实现应该返回行数大于0")

	// 验证两种实现返回的行数相同
	assert.Equal(t, cmdLineInfo.TotalLines, goGitInfo.TotalLines, "两种实现应该返回相同的行数")

	// 运行性能基准测试
	results, err := BenchmarkBlame(p, filePath)
	assert.NoError(t, err, "基准测试应该成功")
	assert.Len(t, results, 2, "基准测试应该返回两个结果")

	// 输出性能比较结果
	fmt.Println(FormatBenchmarkResults(results))
}

// TestBlamerFactory 测试GitBlamer工厂函数
func TestBlamerFactory(t *testing.T) {
	// 跳过测试，因为需要一个真实的Git仓库环境
	t.Skip("此测试需要在真实的Git仓库环境中运行")

	// 创建项目实例 - 使用当前目录的绝对路径
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("无法获取当前工作目录: %v", err)
	}
	p := project.NewProject(cwd)

	// 测试命令行实现
	cmdLineBlamer, err := NewGitBlamer(p, CommandLineBlamer)
	assert.NoError(t, err, "应该成功创建命令行实现")
	assert.IsType(t, &DefaultGitBlamer{}, cmdLineBlamer, "应该返回DefaultGitBlamer类型")

	// 测试go-git实现
	goGitBlamer, err := NewGitBlamer(p, GoGitLibraryBlamer)
	assert.NoError(t, err, "应该成功创建go-git实现")
	assert.IsType(t, &GoGitBlamer{}, goGitBlamer, "应该返回GoGitBlamer类型")

	// 测试默认实现
	defaultBlamer, err := NewGitBlamer(p, "")
	assert.NoError(t, err, "应该成功创建默认实现")
	assert.IsType(t, &DefaultGitBlamer{}, defaultBlamer, "默认应该返回DefaultGitBlamer类型")
}