package project

import (
	"path/filepath"
	"strings"
	"sync"
)

// Option 定义项目选项的函数类型
type Option func(*Project)

// 基础信息相关方法

func (p *Project) Root() *Node {
	return p.root
}

// GetRootPath 获取项目根路径
func (p *Project) GetRootPath() string {
	return p.rootPath
}

// GetName 获取项目名称
func (p *Project) GetName() string {
	return filepath.Base(p.rootPath)
}

// IsEmpty 检查项目是否为空
func (p *Project) IsEmpty() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.root == nil {
		return true
	}

	return len(p.root.Children) == 0
}

// GetTotalNodes 获取项目中的节点总数
func (p *Project) GetTotalNodes() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.root == nil {
		return 0
	}

	return p.root.CountNodes()
}

// IsInGit 检查项目是否在 Git 仓库中
func (p *Project) IsInGit() bool {
	return p.inGit
}

// SetInGit 设置项目是否在 Git 仓库中
func (p *Project) SetInGit(inGit bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.inGit = inGit
}

// GetAbsolutePath 获取项目中节点的绝对路径
func (p *Project) GetAbsolutePath(relativePath string) string {
	// 统一为相对路径（去掉前导 /）
	clean := strings.TrimPrefix(relativePath, "/")
	return filepath.Join(p.rootPath, clean)
}

// 项目注册表，用于根据根节点查找项目实例
var (
	projectRegistry = make(map[*Node]*Project)
	registryMu      sync.RWMutex
)

// RegisterProject 将项目实例注册到全局注册表
func RegisterProject(root *Node, project *Project) {
	registryMu.Lock()
	defer registryMu.Unlock()
	projectRegistry[root] = project
}

// UnregisterProject 从全局注册表中移除项目实例
func UnregisterProject(root *Node) {
	registryMu.Lock()
	defer registryMu.Unlock()
	delete(projectRegistry, root)
}

// GetProjectByRoot 根据根节点查找项目实例
func GetProjectByRoot(root *Node) *Project {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return projectRegistry[root]
}
