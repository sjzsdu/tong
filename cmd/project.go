package cmd

import (
	"fmt"

	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/project"
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
	doc, err := project.BuildProjectTree(targetPath, options)
	if err != nil {
		fmt.Printf("failed to build project tree: %v\n", err)
		return
	}

	fmt.Printf("project tree: %v\n", doc)
}
