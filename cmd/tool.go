package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/project"
	"github.com/sjzsdu/tong/schema"
)

var sharedProject *project.Project
var shareConfig *schema.SchemaConfig

func GetConfig() (*schema.SchemaConfig, error) {
	if shareConfig != nil {
		return shareConfig, nil
	}
	targetPath, err := helper.GetTargetPath(workDir, repoURL)
	if err != nil {
		fmt.Printf("failed to get target path: %v\n", err)
		return nil, err
	}
	config, err := schema.LoadMCPConfig(targetPath, configFile)
	if err != nil {
		fmt.Printf("failed to create schema config: %v\n", err)
		return nil, err
	}
	shareConfig = config
	return config, err
}

func GetProject() (*project.Project, error) {
	if sharedProject != nil {
		return sharedProject, nil
	}
	targetPath, err := helper.GetTargetPath(workDir, repoURL)
	if err != nil {
		fmt.Printf("failed to get target path: %v\n", err)
		return nil, err
	}

	options := helper.WalkDirOptions{
		DisableGitIgnore: skipGitIgnore,
		Extensions:       extensions,
		Excludes:         excludePatterns,
	}

	// 构建项目树
	project, err := buildProjectTreeWithOptions(targetPath, options)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}
	sharedProject = project
	return project, nil
}

// 构建项目树并返回
func buildProjectTreeWithOptions(targetPath string, options helper.WalkDirOptions) (*project.Project, error) {
	// 构建项目树
	project, err := project.BuildProjectTree(targetPath, options)
	if err != nil {
		return nil, fmt.Errorf("failed to build project tree: %v", err)
	}
	return project, nil
}

// IsGitRoot 判断指定路径是否为 git 项目的根目录
func IsGitRoot() bool {
	targetPath, err := helper.GetTargetPath(workDir, repoURL)
	if err != nil {
		fmt.Printf("failed to get target path: %v\n", err)
		return false
	}
	return helper.IsGitRoot(targetPath)
}

func GetProjectName() string {
	targetPath, err := helper.GetTargetPath(workDir, repoURL)
	if err != nil {
		fmt.Printf("failed to get target path: %v\n", err)
		return "unknown"
	}
	return filepath.Base(targetPath)
}

func GetCwdName() string {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("failed to get current working directory: %v\n", err)
		return "unknown"
	}
	return filepath.Base(cwd)
}

// GetTargetNode 根据路径参数获取对应的项目节点
func GetTargetNode(targetPath string) (*project.Node, error) {

	proj, err := GetProject()
	// 必须使用共享的项目实例
	if proj == nil {
		fmt.Printf("错误: 未找到共享的项目实例\n")
		os.Exit(1)
	}

	// 转换为绝对路径
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return nil, fmt.Errorf("无法获取绝对路径: %v", err)
	}

	projectRoot := proj.GetRootPath()

	// 如果目标路径与项目根路径不同，需要找到对应的节点
	if absPath != projectRoot {
		// 计算相对路径
		relPath, err := filepath.Rel(projectRoot, absPath)
		if err != nil {
			return nil, fmt.Errorf("无法计算相对路径: %v", err)
		}

		// 查找目标节点
		targetNode, err := proj.FindNode("/" + relPath)
		if err != nil {
			return nil, fmt.Errorf("找不到目标路径: %v", err)
		}
		return targetNode, nil
	} else {
		// 目标路径就是项目根路径，使用根节点
		targetNode, err := proj.FindNode("/")
		if err != nil {
			return nil, fmt.Errorf("无法获取根节点: %v", err)
		}
		return targetNode, nil
	}
}
