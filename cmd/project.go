package cmd

import (
	"fmt"
	"strings"

	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/project"
	"github.com/sjzsdu/tong/project/analyzer"
	"github.com/sjzsdu/tong/project/git"
	"github.com/sjzsdu/tong/project/health"
	"github.com/sjzsdu/tong/project/output"
	"github.com/sjzsdu/tong/project/search"
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

// createProgressWithCallback 创建进度条和进度回调函数
func createProgressWithCallback(title string, totalFiles int) (*helper.Progress, func(int, string)) {
	progress := helper.NewProgress(title, totalFiles, helper.WithETA(), helper.WithPercent())
	progressCallback := func(current int, filePath string) {
		progress.Update(current)
	}
	return progress, progressCallback
}

// createSimpleProgress 创建简单的进度条（用于不支持进度回调的分析器）
func createSimpleProgress(title string, totalFiles int) *helper.Progress {
	return helper.NewProgress(title, totalFiles, helper.WithETA(), helper.WithPercent())
}

// createProgressForFiles 为指定数量的文件创建进度条
func createProgressForFiles(title string, fileCount int) *helper.Progress {
	return helper.NewProgress(title, fileCount, helper.WithETA(), helper.WithPercent())
}

// updateProgressAndContinueOnError 更新进度并在错误时继续处理
func updateProgressAndContinueOnError(progress *helper.Progress, index int) {
	progress.Update(index + 1)
}

// handleAnalysisError 统一处理分析错误
func handleAnalysisError(progress *helper.Progress, err error, operation string) error {
	progress.Finish()
	return fmt.Errorf("%s失败: %v", operation, err)
}

// 导出项目文件
func packProject(doc *project.Project, outputFilePath string) error {
	if outputFilePath == "" {
		return fmt.Errorf("output file path is required")
	}

	if err := output.Output(doc, outputFilePath); err != nil {
		return fmt.Errorf("导出失败: %v", err)
	}

	return nil
}

// 代码分析函数
func analyzeCode(doc *project.Project) error {
	// 创建进度条和回调函数
	progress, progressCallback := createProgressWithCallback("代码分析", doc.GetTotalFiles())

	// 执行分析（使用统一的 Analyze 方法，传入进度回调）
	analyzer := analyzer.NewDefaultCodeAnalyzer()
	stats, err := analyzer.Analyze(doc, progressCallback)
	if err != nil {
		return handleAnalysisError(progress, err, "代码分析")
	}

	progress.Finish()

	// 输出分析结果
	fmt.Printf("\n=== 代码分析结果 ===\n")
	fmt.Printf("文件总数: %d\n", stats.TotalFiles)
	fmt.Printf("目录总数: %d\n", stats.TotalDirs)
	fmt.Printf("代码总行数: %d\n", stats.TotalLines)
	fmt.Printf("总大小: %.2f KB\n", float64(stats.TotalSize)/1024)

	fmt.Printf("\n=== 语言统计 ===\n")
	for lang, lines := range stats.LanguageStats {
		fmt.Printf("%s: %d 行\n", lang, lines)
	}

	fmt.Printf("\n=== 文件类型统计 ===\n")
	for ext, count := range stats.FileTypeStats {
		if ext == "" {
			ext = "无扩展名"
		}
		fmt.Printf(".%s: %d 个文件\n", ext, count)
	}

	return nil
}

// 分析项目依赖关系
func analyzeDependencies(doc *project.Project) error {
	// 创建进度条和回调函数
	progress, progressCallback := createProgressWithCallback("依赖分析", doc.GetTotalFiles())

	// 使用依赖分析器（使用统一的 AnalyzeDependencies 方法，传入进度回调）
	depsAnalyzer := analyzer.NewDefaultDependencyAnalyzer()
	graph, err := depsAnalyzer.AnalyzeDependencies(doc, progressCallback)
	if err != nil {
		return handleAnalysisError(progress, err, "依赖分析")
	}

	// 完成进度条
	progress.Finish()

	// 创建依赖可视化器
	visualizer := analyzer.NewDependencyVisualizer(graph)

	// 检查是否需要生成DOT文件
	if outputFile != "" && strings.HasSuffix(outputFile, ".dot") {
		return visualizer.GenerateDotFile(outputFile)
	}

	// 打印依赖分析结果
	fmt.Println("\n依赖分析结果:")
	visualizer.PrintDependencies()

	return nil
}

// 分析代码质量
func analyzeCodeQuality(doc *project.Project) error {
	// 创建进度条
	progress := createSimpleProgress("质量分析", doc.GetTotalFiles())

	// 使用代码质量分析器
	qualityAnalyzer := health.NewCodeQualityAnalyzer(doc)
	result, err := qualityAnalyzer.Analyze()
	if err != nil {
		return handleAnalysisError(progress, err, "代码质量分析")
	}

	// 完成进度条
	progress.Finish()

	// 输出分析结果
	fmt.Println("\n代码质量分析结果:")
	fmt.Printf("总分: %.2f/100\n", result.Score)
	fmt.Printf("总问题数: %d\n", result.TotalIssues)

	fmt.Println("\n项目级指标:")
	for metric, metricResult := range result.Metrics {
		fmt.Printf("%s: %.2f (阈值: %.2f, 严重程度: %s)\n",
			metric, metricResult.Value, metricResult.Threshold, metricResult.Severity)
	}

	// 输出严重问题
	fmt.Println("\n严重问题:")
	printedIssues := 0
	for _, fileResult := range result.Files {
		for _, issue := range fileResult.Issues {
			if issue.Severity == health.Error {
				fmt.Printf("%s:%d:%d - %s (%s)\n",
					issue.FilePath, issue.Line, issue.Column, issue.Message, issue.Rule)
				printedIssues++
				if printedIssues >= 10 {
					fmt.Println("...更多问题省略")
					break
				}
			}
		}
		if printedIssues >= 10 {
			break
		}
	}

	return nil
}

// 搜索项目代码
func searchProject(doc *project.Project, query string) error {
	// 创建进度条
	progress := createSimpleProgress("搜索索引构建", doc.GetTotalFiles())

	// 设置搜索选项
	options := search.SearchOptions{
		CaseSensitive: false,
		WholeWord:     false,
		RegexMode:     false,
		FileTypes:     extensions,
		MaxResults:    50,
	}

	// 创建搜索引擎
	searchEngine := search.NewDefaultSearchEngine()

	// 构建索引
	err := searchEngine.BuildIndex(doc)
	if err != nil {
		return handleAnalysisError(progress, err, "构建搜索索引")
	}

	// 完成索引构建进度
	progress.Finish()

	// 执行搜索
	fmt.Printf("正在搜索: %s\n", query)
	results, err := searchEngine.Search(query, options)
	if err != nil {
		return fmt.Errorf("搜索失败: %v", err)
	}

	// 格式化并输出结果
	formatter := &search.MarkdownSearchFormatter{}
	output := formatter.Format(results)
	fmt.Println(output)

	return nil
}

// Git blame 分析
func analyzeBlame(doc *project.Project, filePath string) error {
	// 创建 Git blame 分析器
	blamer, err := git.NewGitBlamer(doc)
	if err != nil {
		return fmt.Errorf("创建 Git blame 分析器失败: %v", err)
	}

	if filePath == "" {
		filePath = "/"
	}

	// 如果是目录，获取所有文件进行批量分析
	node, err := doc.FindNode(filePath)
	if err != nil {
		return fmt.Errorf("查找文件失败: %v", err)
	}

	var filesToAnalyze []string
	if node.IsDir {
		// 收集所有文件
		err := doc.Traverse(func(n *project.Node) error {
			if !n.IsDir {
				path := doc.GetNodePath(n)
				filesToAnalyze = append(filesToAnalyze, path)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("遍历文件失败: %v", err)
		}
	} else {
		filesToAnalyze = []string{filePath}
	}

	// 创建进度条
	progress := createProgressForFiles("Blame 分析", len(filesToAnalyze))

	// 分析每个文件
	allBlameInfo := make([]*git.BlameInfo, 0, len(filesToAnalyze))
	for i, file := range filesToAnalyze {
		blameInfo, err := blamer.Blame(file)
		if err != nil {
			// 对于单个文件错误，继续处理其他文件
			continue
		}
		allBlameInfo = append(allBlameInfo, blameInfo)
		updateProgressAndContinueOnError(progress, i)
	}

	progress.Finish()

	if len(allBlameInfo) == 0 {
		return fmt.Errorf("blame 分析失败: 没有找到可分析的文件\n请确保当前目录是一个有效的 Git 仓库，且指定的文件路径存在")
	}

	// 汇总输出结果
	if len(allBlameInfo) == 1 {
		// 单个文件的详细输出
		blameInfo := allBlameInfo[0]
		fmt.Printf("\n文件: %s\n", blameInfo.FilePath)
		fmt.Printf("总行数: %d\n\n", blameInfo.TotalLines)

		// 输出作者贡献统计
		fmt.Println("作者贡献统计:")
		for author, lines := range blameInfo.Authors {
			fmt.Printf("%s: %d 行 (%.2f%%)\n", author, lines, float64(lines)/float64(blameInfo.TotalLines)*100)
		}

		// 输出日期统计
		fmt.Println("\n日期统计:")
		for date, lines := range blameInfo.Dates {
			fmt.Printf("%s: %d 行\n", date, lines)
		}
	} else {
		// 多个文件的汇总输出
		fmt.Printf("\n分析了 %d 个文件\n\n", len(allBlameInfo))

		// 汇总作者贡献
		totalAuthors := make(map[string]int)
		totalLines := 0
		for _, blameInfo := range allBlameInfo {
			totalLines += blameInfo.TotalLines
			for author, lines := range blameInfo.Authors {
				totalAuthors[author] += lines
			}
		}

		fmt.Println("总体作者贡献统计:")
		for author, lines := range totalAuthors {
			fmt.Printf("%s: %d 行 (%.2f%%)\n", author, lines, float64(lines)/float64(totalLines)*100)
		}
		fmt.Printf("\n总代码行数: %d\n", totalLines)
	}

	return nil
}

func runproject(cmd *cobra.Command, args []string) {
	doc, err := GetProject()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	// 检查参数是否存在
	if len(args) == 0 {
		fmt.Println("请指定操作类型: pack, code, deps, quality, search, blame")
		return
	}

	switch args[0] {
	case "pack":
		if outputFile == "" {
			fmt.Printf("Output is required\n")
			return
		}
		if err := packProject(doc, outputFile); err != nil {
			fmt.Printf("%v\n", err)
		} else {
			fmt.Printf("成功导出到: %s\n", outputFile)
		}

	case "code", "analyze-code": // 兼容旧命令
		if err := analyzeCode(doc); err != nil {
			fmt.Printf("%v\n", err)
		}

	case "deps", "analyze-deps": // 兼容旧命令
		if err := analyzeDependencies(doc); err != nil {
			fmt.Printf("%v\n", err)
		}

	case "quality", "analyze-quality": // 兼容旧命令
		if err := analyzeCodeQuality(doc); err != nil {
			fmt.Printf("%v\n", err)
		}

	case "search":
		if len(args) < 2 {
			fmt.Println("请提供搜索关键词")
			return
		}

		// 设置搜索选项
		if err := searchProject(doc, args[1]); err != nil {
			fmt.Printf("%v\n", err)
		}

	case "blame":
		var filePath string
		if len(args) > 1 {
			filePath = args[1]
		}

		if err := analyzeBlame(doc, filePath); err != nil {
			fmt.Printf("%v\n", err)
		}

	default:
		fmt.Printf("未知的操作类型: %s\n", args[0])
		fmt.Println("支持的操作: pack, code, deps, quality, search, blame")
	}
}
