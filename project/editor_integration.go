package project

import (
	"fmt"
)

// EditorIntegration 提供与大模型和自动化工具集成的编辑器接口
type EditorIntegration struct {
	editor          *EditorAPI
	commandRegistry *CommandRegistry
	openSessions    map[string]*EditorSession
}

// NewEditorIntegration 创建新的编辑器集成界面
func NewEditorIntegration(project *Project) *EditorIntegration {
	editor := NewEditorAPI(project)
	return &EditorIntegration{
		editor:          editor,
		commandRegistry: NewCommandRegistry(),
		openSessions:    make(map[string]*EditorSession),
	}
}

// OpenFile 打开文件并创建编辑会话
func (e *EditorIntegration) OpenFile(filePath string) (*EditorSession, error) {
	// 检查文件是否存在
	_, err := e.editor.project.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法打开文件: %v", err)
	}

	// 创建或获取会话
	session, exists := e.openSessions[filePath]
	if !exists {
		session = NewEditorSession(e.editor, filePath)
		e.openSessions[filePath] = session
	}

	return session, nil
}

// CloseFile 关闭文件并清理会话
func (e *EditorIntegration) CloseFile(filePath string) {
	delete(e.openSessions, filePath)
}

// GetSession 获取文件的编辑会话
func (e *EditorIntegration) GetSession(filePath string) (*EditorSession, bool) {
	session, exists := e.openSessions[filePath]
	return session, exists
}

// ExecuteCommand 执行编辑器命令
func (e *EditorIntegration) ExecuteCommand(
	cmdType CommandType,
	name string,
	filePath string,
	args map[string]interface{}) (*CommandResult, error) {

	// 获取会话
	session, exists := e.GetSession(filePath)
	if !exists {
		var err error
		session, err = e.OpenFile(filePath)
		if err != nil {
			return nil, err
		}
	}

	// 执行命令
	return e.commandRegistry.ExecuteCommand(cmdType, name, e.editor, session, args)
}

// ModelIntegrationRequest 表示与大模型集成的请求
type ModelIntegrationRequest struct {
	Action      string                 // 操作类型（edit, format, suggest等）
	FilePath    string                 // 文件路径
	Content     string                 // 文件内容（可选，如果不提供则从文件读取）
	Range       *Range                 // 操作范围（可选）
	QueryParams map[string]interface{} // 查询参数
}

// ModelIntegrationResponse 表示与大模型集成的响应
type ModelIntegrationResponse struct {
	Success     bool                   // 是否成功
	Message     string                 // 消息
	Edits       []TextEdit             // 编辑操作
	Suggestions []AutoCompleteResult   // 建议
	Actions     []CodeAction           // 代码操作
	Data        map[string]interface{} // 额外数据
}

// ProcessModelRequest 处理大模型集成请求
func (e *EditorIntegration) ProcessModelRequest(req ModelIntegrationRequest) (*ModelIntegrationResponse, error) {
	// 打开文件
	session, err := e.OpenFile(req.FilePath)
	if err != nil {
		return nil, err
	}

	// 处理不同类型的请求
	response := &ModelIntegrationResponse{
		Success:     true,
		Message:     "操作成功",
		Edits:       []TextEdit{},
		Suggestions: []AutoCompleteResult{},
		Actions:     []CodeAction{},
		Data:        make(map[string]interface{}),
	}

	switch req.Action {
	case "format":
		// 格式化代码
		lineCount, err := e.editor.GetLineCount(req.FilePath)
		if err != nil {
			return nil, err
		}

		startLine := 0
		endLine := lineCount - 1

		// 如果指定了范围，使用指定范围
		if req.Range != nil {
			startLine = req.Range.Start.Line
			endLine = req.Range.End.Line
		}

		err = session.FormatCode(startLine, endLine)
		if err != nil {
			response.Success = false
			response.Message = fmt.Sprintf("格式化失败: %v", err)
		}

	case "autocomplete":
		// 自动完成
		line, ok1 := req.QueryParams["line"].(float64)
		column, ok2 := req.QueryParams["column"].(float64)

		if !ok1 || !ok2 {
			return nil, fmt.Errorf("缺少必要参数: line 和 column")
		}

		suggestions, err := session.GetAutoComplete(int(line), int(column))
		if err != nil {
			response.Success = false
			response.Message = fmt.Sprintf("获取自动完成失败: %v", err)
		} else {
			response.Suggestions = suggestions
		}

	case "codeactions":
		// 代码操作
		line, ok1 := req.QueryParams["line"].(float64)
		column, ok2 := req.QueryParams["column"].(float64)
		context, _ := req.QueryParams["context"].(string)

		if !ok1 || !ok2 {
			return nil, fmt.Errorf("缺少必要参数: line 和 column")
		}

		actions, err := session.GetCodeActions(int(line), int(column), context)
		if err != nil {
			response.Success = false
			response.Message = fmt.Sprintf("获取代码操作失败: %v", err)
		} else {
			response.Actions = actions
		}

	case "edit":
		// 编辑文本
		edits, ok := req.QueryParams["edits"].([]TextEdit)
		if !ok || len(edits) == 0 {
			return nil, fmt.Errorf("缺少必要参数: edits")
		}

		err = e.editor.ApplyEdits(req.FilePath, edits)
		if err != nil {
			response.Success = false
			response.Message = fmt.Sprintf("应用编辑失败: %v", err)
		} else {
			response.Edits = edits
		}

	default:
		return nil, fmt.Errorf("不支持的操作类型: %s", req.Action)
	}

	return response, nil
}
