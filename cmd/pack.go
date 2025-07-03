package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/project"
	"github.com/sjzsdu/tong/project/output"
	"github.com/spf13/cobra"
)

var packCmd = &cobra.Command{
	Use:   "pack",
	Short: lang.T("Pack files"),
	Long:  lang.T("Pack files with specified extensions into a single output file"),
	Run:   runPack,
}

func init() {
	rootCmd.AddCommand(packCmd)
}

func runPack(cmd *cobra.Command, args []string) {
	if outputFile == "" {
		fmt.Printf("Output is required")
		return
	}
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
	doc, err := project.BuildProjectTree(targetPath, options)
	if err != nil {
		fmt.Printf("failed to build project tree: %v\n", err)
		return
	}

	// 检查项目树是否为空
	if doc.IsEmpty() {
		fmt.Printf("No files to pack\n")
		return
	}

	// 根据输出文件扩展名选择导出格式
	switch filepath.Ext(outputFile) {
	case ".md":
		exporter := output.NewMarkdownExporter(doc)
		err = exporter.Export(outputFile)
	case ".pdf":
		exporter, err := output.NewPDFExporter(doc)
		if err != nil {
			fmt.Printf("Error creating PDF exporter: %v\n", err)
			return
		}
		exporter.Export(outputFile)
	case ".xml":
		exporter := output.NewXMLExporter(doc)
		err = exporter.Export(outputFile)
	default:
		fmt.Printf("Unsupported output format: %s\n", filepath.Ext(outputFile))
		return
	}

	if err != nil {
		fmt.Printf("Error packing files: %v\n", err)
		return
	}

	fmt.Printf("Successfully packed files into %s\n", outputFile)
}
