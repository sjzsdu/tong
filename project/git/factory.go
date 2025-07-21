package git

import (
	"fmt"

	"github.com/sjzsdu/tong/project"
)

// BlamerType 定义Git blame分析器的类型
type BlamerType string

const (
	// LibraryBlamer 使用go-git库实现的blame分析器
	LibraryBlamer BlamerType = "library"
	// CommandBlamer 使用命令行git命令实现的blame分析器
	CommandBlamer BlamerType = "command"
)

// Blamer 定义Git blame分析器的通用接口
type Blamer interface {
	// Blame 分析文件或目录的blame信息
	Blame(filePath string) (*BlameInfo, error)
	// BlameFile 分析单个文件的blame信息
	BlameFile(p *project.Project, filePath string) (*BlameInfo, error)
	// BlameDirectory 分析目录中所有文件的blame信息
	BlameDirectory(p *project.Project, dirPath string) (map[string]*BlameInfo, error)
	// BlameProject 分析整个项目的blame信息
	BlameProject(p *project.Project) (map[string]*BlameInfo, error)
}

// NewBlamer 创建一个新的Git blame分析器
// blamerType 指定使用哪种实现方式：
// - LibraryBlamer: 使用go-git库实现
// - CommandBlamer: 使用命令行git命令实现
func NewBlamer(p *project.Project, blamerType BlamerType) (Blamer, error) {
	switch blamerType {
	case LibraryBlamer:
		return NewGitBlamer(p)
	case CommandBlamer:
		return NewCmdGitBlamer(p)
	default:
		// 默认使用go-git库实现
		return NewGitBlamer(p)
	}
}

// GetAvailableBlamerTypes 获取当前环境下可用的blame分析器类型
func GetAvailableBlamerTypes() []BlamerType {
	types := []BlamerType{LibraryBlamer}
	
	// 检查命令行git是否可用
	cmdBlamer, err := NewCmdGitBlamer(nil)
	if err == nil && cmdBlamer != nil {
		types = append(types, CommandBlamer)
	}
	
	return types
}

// GetBlamerTypeDescription 获取指定blame分析器类型的描述
func GetBlamerTypeDescription(blamerType BlamerType) string {
	switch blamerType {
	case LibraryBlamer:
		return "使用go-git库实现的Git blame分析器，无需外部依赖"
	case CommandBlamer:
		return "使用命令行git命令实现的Git blame分析器，需要系统安装git"
	default:
		return fmt.Sprintf("未知的Git blame分析器类型: %s", blamerType)
	}
}