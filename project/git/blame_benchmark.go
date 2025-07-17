package git

import (
	"fmt"
	"time"

	"github.com/sjzsdu/tong/project"
)

// BenchmarkResult 存储基准测试结果
type BenchmarkResult struct {
	Implementation string        // 实现名称
	FilePath       string        // 测试文件路径
	Duration       time.Duration // 执行时间
	LinesCount     int           // 处理的行数
	Error          error         // 执行过程中的错误
}

// BenchmarkBlame 对比两种Git blame实现的性能
func BenchmarkBlame(p *project.Project, filePath string) ([]BenchmarkResult, error) {
	results := make([]BenchmarkResult, 0, 2)

	// 测试命令行实现
	cmdLineBlamer := NewDefaultGitBlamer(p)
	cmdLineStart := time.Now()
	cmdLineInfo, cmdLineErr := cmdLineBlamer.Blame(filePath)
	cmdLineDuration := time.Since(cmdLineStart)

	cmdLineResult := BenchmarkResult{
		Implementation: "命令行 Git",
		FilePath:       filePath,
		Duration:       cmdLineDuration,
		Error:          cmdLineErr,
	}
	if cmdLineInfo != nil {
		cmdLineResult.LinesCount = cmdLineInfo.TotalLines
	}
	results = append(results, cmdLineResult)

	// 测试go-git实现
	goGitBlamer, goGitErr := NewGoGitBlamer(p)
	if goGitErr != nil {
		return results, fmt.Errorf("无法创建go-git实现: %w", goGitErr)
	}

	goGitStart := time.Now()
	goGitInfo, goGitErr := goGitBlamer.Blame(filePath)
	goGitDuration := time.Since(goGitStart)

	goGitResult := BenchmarkResult{
		Implementation: "go-git 库",
		FilePath:       filePath,
		Duration:       goGitDuration,
		Error:          goGitErr,
	}
	if goGitInfo != nil {
		goGitResult.LinesCount = goGitInfo.TotalLines
	}
	results = append(results, goGitResult)

	return results, nil
}

// FormatBenchmarkResults 格式化基准测试结果为可读字符串
func FormatBenchmarkResults(results []BenchmarkResult) string {
	if len(results) == 0 {
		return "没有基准测试结果"
	}

	output := fmt.Sprintf("文件: %s\n\n", results[0].FilePath)
	output += "实现\t耗时\t行数\t状态\n"
	output += "----\t----\t----\t----\n"

	for _, result := range results {
		status := "成功"
		if result.Error != nil {
			status = fmt.Sprintf("错误: %v", result.Error)
		}
		output += fmt.Sprintf("%s\t%v\t%d\t%s\n", 
			result.Implementation, 
			result.Duration, 
			result.LinesCount, 
			status)
	}

	// 如果两种实现都成功，计算性能提升百分比
	if len(results) >= 2 && results[0].Error == nil && results[1].Error == nil {
		cmdLineDuration := results[0].Duration
		goGitDuration := results[1].Duration

		var improvement float64
		var faster string

		if cmdLineDuration > goGitDuration {
			improvement = float64(cmdLineDuration) / float64(goGitDuration)
			faster = "go-git 比命令行快"
		} else {
			improvement = float64(goGitDuration) / float64(cmdLineDuration)
			faster = "命令行比 go-git 快"
		}

		output += fmt.Sprintf("\n性能比较: %s %.2f 倍", faster, improvement)
	}

	return output
}