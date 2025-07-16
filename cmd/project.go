package cmd

import (
	"fmt"
	"strings"

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

// 分析代码统计信息
func analyzeCode(doc *project.Project) error {
	// 使用代码分析器
	codeAnalyzer := analyzer.NewDefaultCodeAnalyzer()
	stats, err := codeAnalyzer.Analyze(doc)
	if err != nil {
		return fmt.Errorf("代码分析失败: %v", err)
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

	return nil
}

// 分析项目依赖关系
func analyzeDependencies(doc *project.Project) error {
	// 使用依赖分析器
	depsAnalyzer := analyzer.NewDefaultDependencyAnalyzer()
	graph, err := depsAnalyzer.AnalyzeDependencies(doc)
	if err != nil {
		return fmt.Errorf("依赖分析失败: %v", err)
	}

	// 创建依赖可视化器
	visualizer := analyzer.NewDependencyVisualizer(graph)

	// 检查是否需要生成DOT文件
	if outputFile != "" && strings.HasSuffix(outputFile, ".dot") {
		return visualizer.GenerateDotFile(outputFile)
	}

	// 打印依赖分析结果
	visualizer.PrintDependencies()

	return nil
}

// 分析代码质量
func analyzeCodeQuality(doc *project.Project) error {
	// 使用代码质量分析器
	qualityAnalyzer := health.NewCodeQualityAnalyzer(doc)
	result, err := qualityAnalyzer.Analyze()
	if err != nil {
		return fmt.Errorf("代码质量分析失败: %v", err)
	}

	// 输出分析结果
	fmt.Println("代码质量分析结果:")
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
func searchProject(doc *project.Project, query string, options search.SearchOptions) error {
	// 创建搜索引擎
	searchEngine := search.NewDefaultSearchEngine()

	// 构建索引
	err := searchEngine.BuildIndex(doc)
	if err != nil {
		return fmt.Errorf("构建搜索索引失败: %v", err)
	}

	// 执行搜索
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
		options := search.SearchOptions{
			CaseSensitive: false,
			WholeWord:     false,
			RegexMode:     false,
			FileTypes:     extensions,
			MaxResults:    50,
		}

		if err := searchProject(doc, args[1], options); err != nil {
			fmt.Printf("%v\n", err)
		}

	case "blame":

		// 创建 Git blame 分析器
		blamer := git.NewDefaultGitBlamer(doc)

		var blameInfo *git.BlameInfo
		var err error

		filePath := "/"
		if len(args) > 1 && args[1] != "" {
			filePath = args[1]
		}

		blameInfo, err = blamer.Blame(filePath)
		if err != nil {
			fmt.Printf("Blame 分析失败: %v\n", err)
			fmt.Println("请确保当前目录是一个有效的 Git 仓库，且指定的文件路径存在")
			return
		}

		// 输出 blame 分析结果
		fmt.Printf("文件: %s\n", blameInfo.FilePath)
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

		// 检查是否需要显示详细行信息
		if len(args) > 2 && args[2] == "--detail" {
			fmt.Println("\n详细行信息:")
			for _, line := range blameInfo.Lines {
				fmt.Printf("行 %d: %s (%s) - %s\n",
					line.LineNum,
					line.Author,
					line.CommitTime.Format("2006-01-02"),
					line.Content)
			}
		} else {
			fmt.Println("\n提示: 使用 'project blame <文件路径> --detail' 可查看详细行信息")
		}

	default:
		fmt.Printf("未知的操作类型: %s\n", args[0])
		fmt.Println("支持的操作: pack, code, deps, quality, search, blame")
	}
}
