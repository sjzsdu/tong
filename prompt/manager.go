package prompt

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/lang"
)

const (
	AGENT_EXT = ".md"
)

//go:embed prompts/*.md
var embeddedPrompts embed.FS

var (
	systemPrompts = make(map[string]string) // 系统级别的prompts
	userPrompts   = make(map[string]string) // 用户级别的prompts
)

func init() {
	_, userDir := getPromptDirs()

	// 创建用户目录
	if err := os.MkdirAll(userDir, 0755); err != nil {
		fmt.Printf("Warning - Failed to create user dir: %v\n", err)
	}

	// 初始化系统prompts（从embed文件系统读取）
	loadSystemPrompts()
	// 初始化用户prompts（从用户目录读取）
	loadUserPrompts(userDir)
}

// 从embed文件系统加载系统prompts
func loadSystemPrompts() {
	entries, err := embeddedPrompts.ReadDir("prompts")
	if err != nil {
		fmt.Printf("Warning - Failed to read embedded prompts: %v\n", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), AGENT_EXT) {
			name := strings.TrimSuffix(entry.Name(), AGENT_EXT)
			content, err := embeddedPrompts.ReadFile(filepath.Join("prompts", entry.Name()))
			if err == nil {
				systemPrompts[name] = string(content)
			}
		}
	}
}

// 从用户目录加载用户prompts
func loadUserPrompts(dir string) {
	files, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("Warning - Failed to read dir %s: %v\n", dir, err)
		}
		return
	}

	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), AGENT_EXT) {
			name := strings.TrimSuffix(f.Name(), AGENT_EXT)
			content, err := os.ReadFile(filepath.Join(dir, f.Name()))
			if err == nil {
				userPrompts[name] = string(content)
			} else {
				fmt.Printf("Warning - Failed to read file %s: %v\n", f.Name(), err)
			}
		}
	}
}

// 可以删除原来的 loadPromptsFromDir 函数
// ListPrompts 列出所有代理
func ListPrompts() {
	var output strings.Builder
	output.WriteString(lang.T("System Prompts:"))
	if prompts := listSystemPrompts(); len(prompts) > 0 {
		output.WriteString("\n" + strings.Join(prompts, "\n"))
	}
	output.WriteString("\n" + lang.T("User Prompts:"))
	if prompts := listUserPrompts(); len(prompts) > 0 {
		output.WriteString("\n" + strings.Join(prompts, "\n"))
	}
	output.WriteString("\n")
	fmt.Print(output.String())
}

func listSystemPrompts() []string {
	var prompts []string
	entries, err := embeddedPrompts.ReadDir("prompts")
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), AGENT_EXT) {
				prompts = append(prompts, "- "+strings.TrimSuffix(entry.Name(), AGENT_EXT))
			}
		}
	}
	return prompts
}

func listUserPrompts() []string {
	var prompts []string
	_, userDir := getPromptDirs()
	files, err := os.ReadDir(userDir)
	if err != nil {
		return prompts
	}

	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), AGENT_EXT) {
			prompts = append(prompts, "- "+strings.TrimSuffix(f.Name(), AGENT_EXT))
		}
	}
	return prompts
}

// CreateNewPrompt 创建新代理
func CreateNewPrompt(name, content string) {
	if _, exists := systemPrompts[name]; exists {
		fmt.Println(lang.T("Prompt already exists in system prompts"))
		return
	}
	if _, exists := userPrompts[name]; exists {
		fmt.Println(lang.T("Prompt already exists in user prompts"))
		return
	}

	if content == "" {
		stdinContent, err := os.ReadFile(os.Stdin.Name())
		if err != nil {
			fmt.Printf(lang.T("Failed to read input: %v\n"), err)
			return
		}
		content = string(stdinContent)
	}

	// 保存到内存
	userPrompts[name] = content

	// 保存到文件
	_, userDir := getPromptDirs()
	os.MkdirAll(userDir, 0755)
	err := os.WriteFile(filepath.Join(userDir, name+AGENT_EXT), []byte(content), 0644)
	if err != nil {
		fmt.Printf(lang.T("Failed to create prompt: %v\n"), err)
		delete(userPrompts, name)
		return
	}
	fmt.Println(lang.T("Prompt created successfully"))
}

// UpdateExistingPrompt 更新现有代理
func UpdateExistingPrompt(name, content string) {
	if _, exists := systemPrompts[name]; exists {
		fmt.Println(lang.T("Cannot update system prompt"))
		return
	}
	if _, exists := userPrompts[name]; !exists {
		fmt.Println(lang.T("Prompt not found"))
		return
	}

	if content == "" {
		stdinContent, err := os.ReadFile(os.Stdin.Name())
		if err != nil {
			fmt.Printf(lang.T("Failed to read input: %v\n"), err)
			return
		}
		content = string(stdinContent)
	}

	// 更新内存
	userPrompts[name] = content

	// 更新文件
	_, userDir := getPromptDirs()
	err := os.WriteFile(filepath.Join(userDir, name+AGENT_EXT), []byte(content), 0644)
	if err != nil {
		fmt.Printf(lang.T("Failed to update prompt: %v\n"), err)
		return
	}
	fmt.Println(lang.T("Prompt updated successfully"))
}

// DeleteExistingPrompt 删除现有代理
func DeleteExistingPrompt(name string) {

	if _, exists := userPrompts[name]; !exists {
		fmt.Println(lang.T("Prompt not found"))
		return
	}

	// 从内存中删除
	delete(userPrompts, name)

	// 删除文件
	_, userDir := getPromptDirs()
	err := os.Remove(filepath.Join(userDir, name+AGENT_EXT))
	if err != nil {
		fmt.Printf(lang.T("Failed to delete prompt file: %v\n"), err)
		return
	}
	fmt.Println(lang.T("Prompt deleted successfully"))
}

// GetPromptContent 获取代理内容，优先从用户目录获取
func GetPromptContent(name string) string {
	// 优先检查用户代理
	if content, exists := userPrompts[name]; exists {
		return content
	}
	// 找不到再检查系统代理
	if content, exists := systemPrompts[name]; exists {
		return content
	}
	return ""
}

// ShowPromptContent 显示代理内容
func ShowPromptContent(name string) string {
	content := GetPromptContent(name)
	if content == "" {
		fmt.Printf(lang.T("Prompt not found: %s\n"), name)
	}
	return content
}

func getPromptDirs() (string, string) {
	userDir := helper.GetPath("prompts")
	return "prompts", userDir
}

// SavePrompt 创建或更新代理
func SavePrompt(name, content string) {
	if content == "" {
		stdinContent, err := os.ReadFile(os.Stdin.Name())
		if err != nil {
			fmt.Printf(lang.T("Failed to read input: %v\n"), err)
			return
		}
		content = string(stdinContent)
	}

	// 检查是否是系统代理
	if _, exists := systemPrompts[name]; exists {
		fmt.Println(lang.T("Cannot modify system prompt"))
		return
	}

	// 保存到内存
	userPrompts[name] = content

	// 保存到文件
	_, userDir := getPromptDirs()
	os.MkdirAll(userDir, 0755)
	err := os.WriteFile(filepath.Join(userDir, name+AGENT_EXT), []byte(content), 0644)
	if err != nil {
		fmt.Printf(lang.T("Failed to save prompt: %v\n"), err)
		delete(userPrompts, name)
		return
	}

	if _, existed := userPrompts[name]; existed {
		fmt.Println(lang.T("Prompt updated successfully"))
	} else {
		fmt.Println(lang.T("Prompt created successfully"))
	}
}

// 删除 CreateNewPrompt 和 UpdateExistingPrompt 函数
