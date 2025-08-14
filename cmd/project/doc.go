package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sjzsdu/tong/lang"
	"github.com/spf13/cobra"
)

var (
	docOutDir   string
	docSections string
)

var DocCmd = &cobra.Command{
	Use:   "doc",
	Short: lang.T("生成最小版文档"),
	Long:  lang.T("生成 ARCHITECTURE.md 与 MODULES.md 的基础版本"),
	Run:   runDoc,
}

func init() {
	DocCmd.Flags().StringVar(&docOutDir, "out", "docs", lang.T("输出目录"))
	DocCmd.Flags().StringVar(&docSections, "sections", "arch,modules", lang.T("要生成的文档段落"))
}

func runDoc(cmd *cobra.Command, args []string) {
	if sharedProject == nil {
		fmt.Println("错误: 未找到共享的项目实例")
		os.Exit(1)
	}
	root := sharedProject.GetRootPath()
	name := sharedProject.GetName()

	// 准备输出目录
	out := filepath.Join(root, docOutDir)
	_ = os.MkdirAll(out, 0o755)

	// 简化：只生成两份基础文档骨架
	arch := fmt.Sprintf("# Architecture\n\nProject: %s\n\n- Overview: auto-generated skeleton.\n- Modules: see MODULES.md.\n", name)
	var mods strings.Builder
	mods.WriteString("# Modules\n\n")
	// 基于顶层目录列模块
	if n, err := sharedProject.FindNode("/"); err == nil && n != nil {
		for _, c := range n.Children {
			if c.IsDir {
				mods.WriteString(fmt.Sprintf("- %s/\n", c.Name))
			}
		}
	}

	_ = os.WriteFile(filepath.Join(out, "ARCHITECTURE.md"), []byte(arch), 0o644)
	_ = os.WriteFile(filepath.Join(out, "MODULES.md"), []byte(mods.String()), 0o644)

	fmt.Printf(lang.T("文档已生成到: %s\n"), out)
}
