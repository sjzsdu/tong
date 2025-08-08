package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sjzsdu/tong/project"
)

// 共享的项目实例
var sharedProject *project.Project

// SetSharedProject 设置共享的项目实例
func SetSharedProject(proj *project.Project) {
	sharedProject = proj
}

// GetTargetNode 根据路径参数获取对应的项目节点
// 这是一个通用函数，可以被多个子命令使用
func GetTargetNode(targetPath string) (*project.Node, error) {
	// 必须使用共享的项目实例
	if sharedProject == nil {
		fmt.Printf("错误: 未找到共享的项目实例\n")
		os.Exit(1)
	}

	// 转换为绝对路径
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return nil, fmt.Errorf("无法获取绝对路径: %v", err)
	}

	proj := sharedProject
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
