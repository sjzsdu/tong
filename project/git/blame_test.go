package git

import (
	"os"
	"testing"
	"time"

	"github.com/sjzsdu/tong/project"
	"github.com/stretchr/testify/assert"
)

// TestBlameImplementation 测试 go-git 的 blame 实现
func TestBlameImplementation(t *testing.T) {
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

	// 测试 go-git 实现
	blamer, err := NewGitBlamer(p)
	assert.NoError(t, err, "go-git实现应该成功创建")
	blameInfo, err := blamer.Blame(filePath)
	assert.NoError(t, err, "go-git实现应该成功")
	assert.NotNil(t, blameInfo, "go-git实现应该返回blame信息")
	assert.Greater(t, blameInfo.TotalLines, 0, "go-git实现应该返回行数大于0")

	// 验证 BlameInfo 结构
	assert.NotEmpty(t, blameInfo.Authors, "应该包含作者信息")
	assert.NotEmpty(t, blameInfo.Dates, "应该包含日期信息")
	assert.Equal(t, filePath, blameInfo.FilePath, "文件路径应该匹配")
}

// TestBlameFile 测试对单个文件的 blame 分析
func TestBlameFile(t *testing.T) {
	// 跳过测试，因为需要一个真实的Git仓库环境
	t.Skip("此测试需要在真实的Git仓库环境中运行")

	// 创建项目实例 - 使用当前目录的绝对路径
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("无法获取当前工作目录: %v", err)
	}
	p := project.NewProject(cwd)

	// 创建 blamer
	blamer, err := NewGitBlamer(p)
	assert.NoError(t, err, "应该成功创建 blamer")

	// 测试对当前文件的 blame
	blameInfo, err := blamer.BlameFile(p, "blame.go")
	assert.NoError(t, err, "BlameFile 应该成功执行")
	assert.NotNil(t, blameInfo, "应该返回 blame 信息")
	assert.Greater(t, len(blameInfo.Lines), 0, "应该包含行信息")

	// 验证行信息
	for _, line := range blameInfo.Lines {
		assert.NotEmpty(t, line.Author, "每行应该有作者信息")
		assert.NotEmpty(t, line.CommitID, "每行应该有提交ID")
		assert.False(t, line.CommitTime.IsZero(), "每行应该有提交时间")
	}
}

// TestBlameDirectory 测试对目录的 blame 分析
func TestBlameDirectory(t *testing.T) {
	// 跳过测试，因为需要一个真实的Git仓库环境
	t.Skip("此测试需要在真实的Git仓库环境中运行")

	// 创建项目实例 - 使用当前目录的绝对路径
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("无法获取当前工作目录: %v", err)
	}
	p := project.NewProject(cwd)

	// 创建 blamer
	blamer, err := NewGitBlamer(p)
	assert.NoError(t, err, "应该成功创建 blamer")

	// 测试对当前目录的 blame
	results, err := blamer.BlameDirectory(p, ".")
	assert.NoError(t, err, "BlameDirectory 应该成功执行")
	assert.NotEmpty(t, results, "应该返回 blame 结果")

	// 验证结果包含当前文件
	_, hasCurrentFile := results["blame.go"]
	assert.True(t, hasCurrentFile || len(results) > 0, "结果应该包含当前文件或至少一个文件")
}

// TestHandleUncommittedFile 测试对未提交文件的处理
func TestHandleUncommittedFile(t *testing.T) {
	// 跳过测试，因为需要一个真实的Git仓库环境
	t.Skip("此测试需要在真实的Git仓库环境中运行")

	// 创建项目实例 - 使用当前目录的绝对路径
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("无法获取当前工作目录: %v", err)
	}
	p := project.NewProject(cwd)

	// 创建临时文件
	tempFile := "temp_test_file.txt"
	err = os.WriteFile(tempFile, []byte("这是一个测试文件\n用于测试未提交文件的blame处理"), 0644)
	assert.NoError(t, err, "应该成功创建临时文件")
	defer os.Remove(tempFile) // 测试结束后删除临时文件

	// 创建 blamer
	blamer, err := NewGitBlamer(p)
	assert.NoError(t, err, "应该成功创建 blamer")

	// 测试对未提交文件的 blame
	blameInfo, err := blamer.Blame(tempFile)
	assert.NoError(t, err, "对未提交文件的 blame 应该成功")
	assert.NotNil(t, blameInfo, "应该返回 blame 信息")
	assert.Equal(t, 2, blameInfo.TotalLines, "应该有2行")

	// 验证未提交文件的 blame 信息
	for _, line := range blameInfo.Lines {
		assert.Equal(t, "未提交", line.CommitID, "未提交文件的提交ID应该是'未提交'")
		assert.WithinDuration(t, time.Now(), line.CommitTime, 5*time.Second, "提交时间应该是当前时间")
	}
}