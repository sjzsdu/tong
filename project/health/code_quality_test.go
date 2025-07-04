package health

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sjzsdu/tong/project"
	"github.com/stretchr/testify/assert"
)

func TestCodeQualityAnalyzer(t *testing.T) {
	// 使用共享项目
	goProject := project.GetSharedProject(t, "")
	proj := goProject.GetProject()

	// 创建代码质量分析器
	analyzer := NewCodeQualityAnalyzer(proj)
	assert.NotNil(t, analyzer)

	// 执行分析
	result, err := analyzer.Analyze()
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, len(result.Files), 1)

	// 检查项目级别指标是否已计算
	assert.NotEmpty(t, result.Metrics)
	assert.GreaterOrEqual(t, result.Score, 0.0)
}

func TestCodeQualityAnalyzerWithExampleProject(t *testing.T) {
	// 创建一个示例项目
	projectPath := project.CreateExampleGoProject(t)
	defer os.RemoveAll(projectPath) // 测试结束后清理

	goProject := project.GetSharedProject(t, projectPath)
	proj := goProject.GetProject()

	// 创建代码质量分析器
	analyzer := NewCodeQualityAnalyzer(proj)

	// 执行分析
	result, err := analyzer.Analyze()
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// 检查是否有文件被分析
	assert.NotEmpty(t, result.Files)

	// 检查主要的指标是否存在
	metricNames := []CodeQualityMetric{
		CyclomaticComplexity,
		MaintainabilityIndex,
		CommentRatio,
	}

	for _, metricName := range metricNames {
		_, exists := result.Metrics[metricName]
		assert.True(t, exists, "指标 %s 应该存在", metricName)
	}
}

func TestSingleFileAnalysis(t *testing.T) {
	// 使用共享项目
	goProject := project.GetSharedProject(t, "")
	proj := goProject.GetProject()

	analyzer := NewCodeQualityAnalyzer(proj)

	// 寻找一个Go文件进行分析
	var fileToAnalyze string
	visitor := project.VisitorFunc(func(path string, node *project.Node, depth int) error {
		if !node.IsDir && len(node.Content) > 0 && filepath.Ext(path) == ".go" {
			fileToAnalyze = path
			return fmt.Errorf("stop traversal") // 自定义错误来停止遍历
		}
		return nil
	})

	traverser := project.NewTreeTraverser(proj)
	_ = traverser.TraverseTree(visitor)

	if fileToAnalyze != "" {
		// 分析单个文件
		fileResult, err := analyzer.AnalyzeFile(fileToAnalyze)
		assert.NoError(t, err)
		assert.NotNil(t, fileResult)
		assert.Equal(t, fileToAnalyze, fileResult.FilePath)
		assert.NotEmpty(t, fileResult.Metrics)
	} else {
		t.Skip("找不到Go文件进行测试")
	}
}

func TestMetricThresholds(t *testing.T) {
	// 使用共享项目
	goProject := project.GetSharedProject(t, "")
	proj := goProject.GetProject()

	analyzer := NewCodeQualityAnalyzer(proj)

	// 测试设置和获取阈值
	testMetric := CyclomaticComplexity
	testThreshold := 15.0

	// 获取原始阈值
	originalThreshold := analyzer.GetThreshold(testMetric)

	// 设置新阈值
	analyzer.SetThreshold(testMetric, testThreshold)

	// 验证阈值已更新
	newThreshold := analyzer.GetThreshold(testMetric)
	assert.Equal(t, testThreshold, newThreshold)

	// 恢复原始阈值
	analyzer.SetThreshold(testMetric, originalThreshold)
}

func TestSupportedMetrics(t *testing.T) {
	// 使用共享项目
	goProject := project.GetSharedProject(t, "")
	proj := goProject.GetProject()

	analyzer := NewCodeQualityAnalyzer(proj)

	// 获取支持的指标
	metrics := analyzer.GetSupportedMetrics()
	assert.NotEmpty(t, metrics)

	// 验证关键指标是否存在
	expectedMetrics := []CodeQualityMetric{
		CyclomaticComplexity,
		MaintainabilityIndex,
		CommentRatio,
	}

	for _, expectedMetric := range expectedMetrics {
		found := false
		for _, metric := range metrics {
			if metric == expectedMetric {
				found = true
				break
			}
		}
		assert.True(t, found, "应支持指标 %s", expectedMetric)
	}
}
