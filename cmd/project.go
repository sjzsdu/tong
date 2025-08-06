package cmd

import (
	"fmt"

	"github.com/sjzsdu/tong/lang"
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

	projectCmd.PersistentFlags().StringVarP(&workDir, "directory", "d", ".", lang.T("Work directory path"))
	projectCmd.PersistentFlags().StringSliceVarP(&extensions, "extensions", "e", []string{"*"}, lang.T("File extensions to include"))
	projectCmd.PersistentFlags().StringVarP(&outputFile, "out", "o", "", lang.T("Output file name"))
	projectCmd.PersistentFlags().StringSliceVarP(&excludePatterns, "exclude", "x", []string{}, lang.T("Glob patterns to exclude"))
	projectCmd.PersistentFlags().StringVarP(&repoURL, "repository", "r", "", lang.T("Git repository URL to clone and pack"))
	projectCmd.PersistentFlags().BoolVarP(&skipGitIgnore, "no-gitignore", "n", false, lang.T("Disable .gitignore rules"))
}

func runproject(cmd *cobra.Command, args []string) {
	_, err := GetProject()
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
	default:
		fmt.Println("支持的操作: pack, code, deps, quality, search, blame")
	}
}
