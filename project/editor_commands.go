package project

import (
	"fmt"
	"sort"
	"strings"
)

// CommandType 表示编辑器命令类型
type CommandType string

const (
	CommandFormat     CommandType = "format"     // 格式化代码
	CommandRefactor   CommandType = "refactor"   // 重构代码
	CommandOrganize   CommandType = "organize"   // 整理导入
	CommandGenerate   CommandType = "generate"   // 生成代码
	CommandBuildIndex CommandType = "buildIndex" // 构建索引
	CommandRename     CommandType = "rename"     // 重命名符号
	CommandCustom     CommandType = "custom"     // 自定义命令
)

// EditorCommand 表示编辑器命令
type EditorCommand struct {
	Type      CommandType                                                                        // 命令类型
	Name      string                                                                             // 命令名称
	Args      map[string]interface{}                                                             // 命令参数
	ApplyFunc func(editor *EditorAPI, session *EditorSession, args map[string]interface{}) error // 命令执行函数
}

// CommandResult 表示命令执行结果
type CommandResult struct {
	Success bool                   // 是否成功
	Message string                 // 结果消息
	Changes []TextEdit             // 产生的变更
	Data    map[string]interface{} // 额外数据
}

// CommandRegistry 命令注册表
type CommandRegistry struct {
	commands map[string]EditorCommand
}

// NewCommandRegistry 创建新的命令注册表
func NewCommandRegistry() *CommandRegistry {
	registry := &CommandRegistry{
		commands: make(map[string]EditorCommand),
	}

	// 注册内置命令
	registry.registerBuiltinCommands()

	return registry
}

// RegisterCommand 注册命令
func (r *CommandRegistry) RegisterCommand(cmd EditorCommand) {
	r.commands[string(cmd.Type)+"."+cmd.Name] = cmd
}

// GetCommand 获取命令
func (r *CommandRegistry) GetCommand(cmdType CommandType, name string) (EditorCommand, bool) {
	cmd, ok := r.commands[string(cmdType)+"."+name]
	return cmd, ok
}

// ExecuteCommand 执行命令
func (r *CommandRegistry) ExecuteCommand(
	cmdType CommandType,
	name string,
	editor *EditorAPI,
	session *EditorSession,
	args map[string]interface{}) (*CommandResult, error) {

	cmd, ok := r.GetCommand(cmdType, name)
	if !ok {
		return nil, fmt.Errorf("未找到命令: %s.%s", cmdType, name)
	}

	if cmd.ApplyFunc == nil {
		return nil, fmt.Errorf("命令没有实现: %s.%s", cmdType, name)
	}

	// 执行命令
	err := cmd.ApplyFunc(editor, session, args)

	result := &CommandResult{
		Success: err == nil,
		Message: "命令执行成功",
		Changes: []TextEdit{},
		Data:    make(map[string]interface{}),
	}

	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("命令执行失败: %v", err)
	}

	return result, nil
}

// registerBuiltinCommands 注册内置命令
func (r *CommandRegistry) registerBuiltinCommands() {
	// 格式化代码命令
	r.RegisterCommand(EditorCommand{
		Type: CommandFormat,
		Name: "document",
		Args: map[string]interface{}{},
		ApplyFunc: func(editor *EditorAPI, session *EditorSession, args map[string]interface{}) error {
			if session == nil {
				return fmt.Errorf("需要有效的编辑器会话")
			}

			// 获取文件行数
			lineCount, err := editor.GetLineCount(session.filePath)
			if err != nil {
				return err
			}

			// 格式化整个文档
			return session.FormatCode(0, lineCount-1)
		},
	})

	// 整理导入命令
	r.RegisterCommand(EditorCommand{
		Type: CommandOrganize,
		Name: "imports",
		Args: map[string]interface{}{},
		ApplyFunc: func(editor *EditorAPI, session *EditorSession, args map[string]interface{}) error {
			if session == nil {
				return fmt.Errorf("需要有效的编辑器会话")
			}

			// 读取文件内容
			content, err := editor.project.ReadFile(session.filePath)
			if err != nil {
				return err
			}

			lines := strings.Split(string(content), "\n")

			// 查找导入区块
			importStart := -1
			importEnd := -1
			inImportBlock := false

			for i, line := range lines {
				trimmed := strings.TrimSpace(line)

				if strings.HasPrefix(trimmed, "import (") {
					importStart = i
					inImportBlock = true
				} else if inImportBlock && trimmed == ")" {
					importEnd = i
					break
				} else if strings.HasPrefix(trimmed, "import ") && !inImportBlock {
					importStart = i
					importEnd = i
					break
				}
			}

			if importStart == -1 {
				return fmt.Errorf("未找到导入区块")
			}

			// 提取导入语句
			var imports []string
			if importStart == importEnd {
				// 单行导入
				line := lines[importStart]
				importStr := strings.TrimSpace(line[6:]) // 移除 "import "
				imports = append(imports, importStr)
			} else {
				// 多行导入
				for i := importStart + 1; i < importEnd; i++ {
					trimmed := strings.TrimSpace(lines[i])
					if trimmed != "" {
						imports = append(imports, trimmed)
					}
				}
			}

			// 排序导入
			sortedImports := sortImports(imports)

			// 构建新的导入区块
			var newImportBlock string
			if len(sortedImports) == 1 {
				// 单行导入
				newImportBlock = "import " + sortedImports[0]
			} else {
				// 多行导入
				newImportBlock = "import (\n"
				for _, imp := range sortedImports {
					newImportBlock += "\t" + imp + "\n"
				}
				newImportBlock += ")"
			}

			// 替换导入区块
			edit := TextEdit{
				StartLine:   importStart,
				StartColumn: 0,
				EndLine:     importEnd,
				EndColumn:   len(lines[importEnd]),
				NewText:     newImportBlock,
			}

			return session.ApplyEdit(edit)
		},
	})

	// 重命名符号
	r.RegisterCommand(EditorCommand{
		Type: CommandRename,
		Name: "symbol",
		Args: map[string]interface{}{
			"oldName": "",
			"newName": "",
		},
		ApplyFunc: func(editor *EditorAPI, session *EditorSession, args map[string]interface{}) error {
			oldName, ok1 := args["oldName"].(string)
			newName, ok2 := args["newName"].(string)

			if !ok1 || !ok2 || oldName == "" || newName == "" {
				return fmt.Errorf("缺少必要参数: oldName 和 newName")
			}

			// 创建搜索选项
			options := SearchOptions{
				CaseSensitive: true,
				WholeWord:     true,
				RegExp:        false,
			}

			// 查找并替换
			count, err := editor.ReplaceAllWithOptions(session.filePath, oldName, newName, options)
			if err != nil {
				return err
			}

			if count == 0 {
				return fmt.Errorf("未找到符号: %s", oldName)
			}

			return nil
		},
	})
}

// sortImports 排序导入语句
func sortImports(imports []string) []string {
	// 简单实现：标准库优先，然后是第三方库
	var stdLibs []string
	var thirdParty []string

	for _, imp := range imports {
		if strings.HasPrefix(imp, "\"") && !strings.Contains(imp[1:], ".") {
			// 标准库没有点
			stdLibs = append(stdLibs, imp)
		} else {
			thirdParty = append(thirdParty, imp)
		}
	}

	// 排序
	sort.Strings(stdLibs)
	sort.Strings(thirdParty)

	// 组合结果
	sortedImports := append(stdLibs, thirdParty...)
	return sortedImports
}
