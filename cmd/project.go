package cmd

import (
	"fmt"

	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/project"
	"github.com/sjzsdu/tong/project/analyzer"
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

// 构建项目树并返回
func buildProjectTreeWithOptions(targetPath string, options helper.WalkDirOptions) (*project.Project, error) {
	// 构建项目树
	doc, err := project.BuildProjectTree(targetPath, options)
	if err != nil {
		return nil, fmt.Errorf("failed to build project tree: %v", err)
	}
	return doc, nil
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
	doc, err := buildProjectTreeWithOptions(targetPath, options)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	// 检查参数是否存在
	if len(args) == 0 {
		fmt.Println("请指定操作类型: pack, code, deps, quality, search")
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
	
	default:
		fmt.Printf("未知的操作类型: %s\n", args[0])
		fmt.Println("支持的操作: pack, code, deps, quality, search")
	}
}
