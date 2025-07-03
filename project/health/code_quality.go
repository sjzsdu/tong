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
