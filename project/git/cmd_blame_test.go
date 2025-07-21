package git

import (
	"os"
	"testing"

	"github.com/sjzsdu/tong/project"
	"github.com/stretchr/testify/assert"
)

// TestCmdGitBlamer 测试命令行git blame实现
func TestCmdGitBlamer(t *testing.T) {
	// 跳过测试，因为需要一个真实的Git仓库环境和git命令
	t.Skip("此测试需要在真实的Git仓库环境中运行，且需要git命令")

	// 创建项目实例 - 使用当前目录的绝对路径
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("无法获取当前工作目录: %v", err)
	}
	p := project.NewProject(cwd)

	// 创建CmdGitBlamer实例
	blamer, err := NewCmdGitBlamer(p)
	if err != nil {
		t.Fatalf("创建CmdGitBlamer失败: %v", err)
	}

	// 测试分析当前文件
	blameInfo, err := blamer.BlameFile(p, "cmd_blame_test.go")
	if err != nil {
		t.Fatalf("分析文件失败: %v", err)
	}

	// 验证基本信息
	assert.NotNil(t, blameInfo)
	assert.Greater(t, blameInfo.TotalLines, 0)
	assert.Equal(t, "cmd_blame_test.go", blameInfo.FilePath)
	assert.NotEmpty(t, blameInfo.Authors)

	// 打印blame信息
	t.Logf("文件: %s, 总行数: %d", blameInfo.FilePath, blameInfo.TotalLines)
	for author, lines := range blameInfo.Authors {
		t.Logf("作者: %s, 贡献行数: %d", author, lines)
	}
}

// TestFactoryBlamer 测试工厂模式创建不同类型的Blamer
func TestFactoryBlamer(t *testing.T) {
	// 跳过测试，因为需要一个真实的Git仓库环境
	t.Skip("此测试需要在真实的Git仓库环境中运行")

	// 创建项目实例
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("无法获取当前工作目录: %v", err)
	}
	p := project.NewProject(cwd)

	// 获取可用的blame分析器类型
	blamerTypes := GetAvailableBlamerTypes()
	t.Logf("可用的blame分析器类型: %v", blamerTypes)

	// 测试每种类型的Blamer
	for _, blamerType := range blamerTypes {
		t.Run(string(blamerType), func(t *testing.T) {
			// 创建Blamer
			blamer, err := NewBlamer(p, blamerType)
			if err != nil {
				t.Fatalf("创建%s类型的Blamer失败: %v", blamerType, err)
			}

			// 测试分析当前文件
			blameInfo, err := blamer.BlameFile(p, "cmd_blame_test.go")
			if err != nil {
				t.Fatalf("分析文件失败: %v", err)
			}

			// 验证基本信息
			assert.NotNil(t, blameInfo)
			assert.Greater(t, blameInfo.TotalLines, 0)
			assert.Equal(t, "cmd_blame_test.go", blameInfo.FilePath)
			assert.NotEmpty(t, blameInfo.Authors)

			// 打印blame信息
			t.Logf("使用%s分析器 - 文件: %s, 总行数: %d", 
				blamerType, blameInfo.FilePath, blameInfo.TotalLines)
		})
	}

	// 测试性能比较（可选）
	if len(blamerTypes) > 1 {
		t.Run("性能比较", func(t *testing.T) {
			// 这里可以添加性能比较测试
			// 例如，使用testing.Benchmark或简单计时比较两种实现的性能
		})
	}
}