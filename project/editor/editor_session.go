package editor

import (
	"bufio"
	"fmt"
	"strings"
)

// EditorSession 表示一个编辑会话，包含编辑历史和撤销/重做功能
type EditorSession struct {
	editor      *EditorAPI
	filePath    string
	editHistory []TextEdit
	historyPos  int
	undoStack   []TextEdit
	redoStack   []TextEdit
}

// NewEditorSession 创建一个新的编辑会话
func NewEditorSession(editor *EditorAPI, filePath string) *EditorSession {
	return &EditorSession{
		editor:      editor,
		filePath:    filePath,
		editHistory: make([]TextEdit, 0),
		historyPos:  -1,
		undoStack:   make([]TextEdit, 0),
		redoStack:   make([]TextEdit, 0),
	}
}

// ApplyEdit 应用编辑并记录历史
func (s *EditorSession) ApplyEdit(edit TextEdit) error {
	// 保存原始文本，用于撤销
	originalText, err := s.editor.GetTextInRange(
		s.filePath,
		edit.StartLine,
		edit.StartColumn,
		edit.EndLine,
		edit.EndColumn,
	)
	if err == nil {
		edit.OldText = originalText
	}

	// 应用编辑
	if err := s.editor.ApplyEdit(s.filePath, edit); err != nil {
		return err
	}

	// 记录编辑历史
	s.editHistory = append(s.editHistory[:s.historyPos+1], edit)
	s.historyPos++

	// 添加到撤销栈
	s.undoStack = append(s.undoStack, edit)
	// 清空重做栈
	s.redoStack = nil

	return nil
}

// Undo 撤销最后一次编辑
func (s *EditorSession) Undo() error {
	if len(s.undoStack) == 0 {
		return fmt.Errorf("没有可撤销的操作")
	}

	// 获取最后一次编辑
	lastEdit := s.undoStack[len(s.undoStack)-1]
	s.undoStack = s.undoStack[:len(s.undoStack)-1]

	// 创建反向编辑 - 简化版本，直接使用原始编辑中的范围
	inverseEdit := TextEdit{
		StartLine:   lastEdit.StartLine,
		StartColumn: lastEdit.StartColumn,
		EndLine:     lastEdit.StartLine,
		EndColumn:   lastEdit.StartColumn + len(lastEdit.NewText),
		NewText:     lastEdit.OldText, // 使用之前保存的原始文本
	}

	// 应用反向编辑
	if err := s.editor.ApplyEdit(s.filePath, inverseEdit); err != nil {
		return err
	}

	// 添加到重做栈
	s.redoStack = append(s.redoStack, lastEdit)

	return nil
}

// Redo 重做上次撤销的编辑
func (s *EditorSession) Redo() error {
	if len(s.redoStack) == 0 {
		return fmt.Errorf("没有可重做的操作")
	}

	// 获取最后一次撤销的编辑
	lastUndo := s.redoStack[len(s.redoStack)-1]
	s.redoStack = s.redoStack[:len(s.redoStack)-1]

	// 应用编辑
	if err := s.editor.ApplyEdit(s.filePath, lastUndo); err != nil {
		return err
	}

	// 添加到撤销栈
	s.undoStack = append(s.undoStack, lastUndo)

	return nil
}

// createInverseEdit 创建一个编辑操作的反向操作
func (s *EditorSession) createInverseEdit(edit TextEdit) TextEdit {
	// 创建反向编辑
	inverseEdit := TextEdit{
		StartLine:   edit.StartLine,
		StartColumn: edit.StartColumn,
		EndLine:     edit.StartLine + strings.Count(edit.NewText, "\n"),
		EndColumn:   0,
		NewText:     "", // 将在下面设置
	}

	// 如果新文本没有换行符，计算结束列
	if !strings.Contains(edit.NewText, "\n") {
		inverseEdit.EndLine = edit.StartLine
		inverseEdit.EndColumn = edit.StartColumn + len(edit.NewText)
	} else {
		// 如果有换行符，计算最后一行的长度
		lines := strings.Split(edit.NewText, "\n")
		inverseEdit.EndColumn = len(lines[len(lines)-1])
	}

	// 获取当前内容
	content, err := s.editor.project.ReadFile(s.filePath)
	if err != nil {
		return TextEdit{}
	}

	lines := strings.Split(string(content), "\n")

	// 获取要恢复的原始文本
	if edit.StartLine == edit.EndLine {
		// 单行编辑
		if edit.StartLine < len(lines) {
			line := lines[edit.StartLine]
			if edit.StartColumn <= len(line) && edit.EndColumn <= len(line) {
				inverseEdit.NewText = line[edit.StartColumn:edit.EndColumn]
			}
		}
	} else {
		// 多行编辑
		var builder strings.Builder

		// 处理开始行
		if edit.StartLine < len(lines) {
			startLine := lines[edit.StartLine]
			if edit.StartColumn <= len(startLine) {
				builder.WriteString(startLine[edit.StartColumn:])
				builder.WriteString("\n")
			}
		}

		// 处理中间行
		for i := edit.StartLine + 1; i < edit.EndLine && i < len(lines); i++ {
			builder.WriteString(lines[i])
			builder.WriteString("\n")
		}

		// 处理结束行
		if edit.EndLine < len(lines) {
			endLine := lines[edit.EndLine]
			if edit.EndColumn <= len(endLine) {
				builder.WriteString(endLine[:edit.EndColumn])
			}
		}

		inverseEdit.NewText = builder.String()
	}

	return inverseEdit
}

// SmartIndent 智能缩进指定行
func (s *EditorSession) SmartIndent(line int) error {
	// 获取当前行内容
	lineContent, err := s.editor.GetLineContent(s.filePath, line)
	if err != nil {
		return err
	}

	// 如果是空行，返回
	trimmedLine := strings.TrimSpace(lineContent)
	if len(trimmedLine) == 0 {
		return nil
	}

	// 获取上一行（如果存在）
	var prevIndent string
	if line > 0 {
		prevLine, err := s.editor.GetLineContent(s.filePath, line-1)
		if err == nil {
			// 提取缩进
			prevIndent = extractIndent(prevLine)

			// 检查上一行是否以大括号、冒号等结束，增加缩进
			if endsWithOpenBrace(prevLine) {
				prevIndent += "\t"
			}
		}
	}

	// 当前行缩进
	currentIndent := extractIndent(lineContent)

	// 如果缩进不同，更新行
	if prevIndent != currentIndent {
		// 创建新行
		newLine := prevIndent + trimmedLine

		// 替换整行
		edit := TextEdit{
			StartLine:   line,
			StartColumn: 0,
			EndLine:     line,
			EndColumn:   len(lineContent),
			NewText:     newLine,
		}

		return s.ApplyEdit(edit)
	}

	return nil
}

// SmartSelect 智能选择代码块或语法元素
func (s *EditorSession) SmartSelect(line, column int) (Range, error) {
	// 获取行内容
	lineContent, err := s.editor.GetLineContent(s.filePath, line)
	if err != nil {
		return Range{}, err
	}

	// 基本情况：尝试选择当前词
	wordRange := findWordAtPosition(lineContent, column)
	if wordRange.Start.Column != wordRange.End.Column {
		wordRange.Start.Line = line
		wordRange.End.Line = line
		return wordRange, nil
	}

	// 尝试智能选择括号内内容
	content, err := s.editor.GetTextInRange(s.filePath, 0, 0, line+50, 0)
	if err != nil {
		return Range{}, err
	}

	// 将内容分行
	lines := strings.Split(content, "\n")

	// 限制在实际内容范围内
	if line >= len(lines) {
		return Range{}, fmt.Errorf("行号超出范围")
	}

	// 找到匹配的括号
	bracketRange, found := findMatchingBrackets(lines, line, column)
	if found {
		return bracketRange, nil
	}

	// 如果找不到括号，尝试选择当前行
	return Range{
		Start: Position{Line: line, Column: 0},
		End:   Position{Line: line, Column: len(lineContent)},
	}, nil
}

// FormatCode 格式化指定范围的代码
func (s *EditorSession) FormatCode(startLine, endLine int) error {
	// 读取范围内的代码
	var builder strings.Builder
	for i := startLine; i <= endLine; i++ {
		line, err := s.editor.GetLineContent(s.filePath, i)
		if err != nil {
			return err
		}
		builder.WriteString(line)
		builder.WriteString("\n")
	}

	codeBlock := builder.String()

	// 简单格式化（这里只是演示，实际应调用更复杂的格式化器）
	formattedCode := formatCodeBlock(codeBlock)

	// 替换代码块
	edit := TextEdit{
		StartLine:   startLine,
		StartColumn: 0,
		EndLine:     endLine,
		EndColumn:   len(strings.Split(codeBlock, "\n")[endLine-startLine]),
		NewText:     formattedCode,
	}

	return s.ApplyEdit(edit)
}

// BatchEdit 批量编辑，对指定范围的每一行应用变换函数
func (s *EditorSession) BatchEdit(startLine, endLine int, transform func(string) string) error {
	edits := make([]TextEdit, 0, endLine-startLine+1)

	for i := startLine; i <= endLine; i++ {
		line, err := s.editor.GetLineContent(s.filePath, i)
		if err != nil {
			return err
		}

		// 应用变换
		newLine := transform(line)

		// 如果行发生变化，创建编辑
		if newLine != line {
			edits = append(edits, TextEdit{
				StartLine:   i,
				StartColumn: 0,
				EndLine:     i,
				EndColumn:   len(line),
				NewText:     newLine,
			})
		}
	}

	// 应用所有编辑
	return s.editor.ApplyEdits(s.filePath, edits)
}

// FindAndReplaceInSelection 在选定范围内查找并替换
func (s *EditorSession) FindAndReplaceInSelection(startLine, startColumn, endLine, endColumn int,
	searchText, replaceText string, caseSensitive bool) (int, error) {
	// 获取选定范围内的文本
	selectedText, err := s.editor.GetTextInRange(s.filePath,
		startLine, startColumn, endLine, endColumn)
	if err != nil {
		return 0, err
	}

	// 分割为行处理
	lines := strings.Split(selectedText, "\n")
	edits := make([]TextEdit, 0)

	replacementCount := 0

	for i, line := range lines {
		currentLine := line

		// 如果不区分大小写，转换为小写进行比较
		var searchLine, searchPattern string
		if caseSensitive {
			searchLine = currentLine
			searchPattern = searchText
		} else {
			searchLine = strings.ToLower(currentLine)
			searchPattern = strings.ToLower(searchText)
		}

		// 当前行在原文档中的行号
		docLine := startLine + i

		// 查找所有匹配
		startIndex := 0

		for {
			index := strings.Index(searchLine[startIndex:], searchPattern)
			if index == -1 {
				break
			}

			// 实际索引
			actualIndex := startIndex + index

			// 创建编辑
			var edit TextEdit
			if i == 0 && docLine == startLine {
				// 第一行需要考虑起始列偏移
				actualCol := actualIndex + startColumn
				edit = TextEdit{
					StartLine:   docLine,
					StartColumn: actualCol,
					EndLine:     docLine,
					EndColumn:   actualCol + len(searchText),
					NewText:     replaceText,
				}
			} else {
				edit = TextEdit{
					StartLine:   docLine,
					StartColumn: actualIndex,
					EndLine:     docLine,
					EndColumn:   actualIndex + len(searchText),
					NewText:     replaceText,
				}
			}

			edits = append(edits, edit)
			replacementCount++

			// 移动到下一个搜索位置
			startIndex = actualIndex + len(searchPattern)
			if startIndex >= len(searchLine) {
				break
			}
		}
	}

	// 应用所有编辑
	if len(edits) > 0 {
		if err := s.editor.ApplyEdits(s.filePath, edits); err != nil {
			return 0, err
		}
	}

	return replacementCount, nil
}

// AutoCompleteResult 表示自动完成结果
type AutoCompleteResult struct {
	Text          string // 完成的文本
	DisplayText   string // 显示的文本
	Kind          string // 类型（变量、函数、类等）
	Detail        string // 详细信息
	Documentation string // 文档
}

// GetAutoComplete 获取指定位置的自动完成建议
func (s *EditorSession) GetAutoComplete(line, column int) ([]AutoCompleteResult, error) {
	// 这个实现是示例性的，实际应该集成语言服务器或基于上下文分析

	// 获取当前单词
	currentLine, err := s.editor.GetLineContent(s.filePath, line)
	if err != nil {
		return nil, err
	}

	// 查找当前单词开始位置
	wordStart := column
	for wordStart > 0 && isWordChar(rune(currentLine[wordStart-1])) {
		wordStart--
	}

	// 提取前缀
	prefix := ""
	if wordStart < column {
		prefix = currentLine[wordStart:column]
	}

	// 基于上下文和前缀生成建议
	// 这里只是示例，实际应分析代码、使用语言服务器等
	suggestions := []AutoCompleteResult{}

	// 简单示例：基于常见关键字的建议
	keywords := []string{"func", "type", "var", "const", "package", "import", "return", "if", "else", "for", "switch", "case"}
	for _, keyword := range keywords {
		if strings.HasPrefix(keyword, prefix) && keyword != prefix {
			suggestions = append(suggestions, AutoCompleteResult{
				Text:        keyword,
				DisplayText: keyword,
				Kind:        "keyword",
				Detail:      "Go关键字",
			})
		}
	}

	return suggestions, nil
}

// Diff 表示文本差异
type Diff struct {
	Original string  // 原始文本
	Modified string  // 修改后的文本
	Changes  []Range // 变更范围
}

// GetDiff 获取文件的差异
func (s *EditorSession) GetDiff(oldText, newText string) (*Diff, error) {
	// 简单差异实现，实际应使用更高效的差异算法
	oldLines := strings.Split(oldText, "\n")
	newLines := strings.Split(newText, "\n")

	diff := &Diff{
		Original: oldText,
		Modified: newText,
		Changes:  []Range{},
	}

	// 查找差异行
	for i := 0; i < min(len(oldLines), len(newLines)); i++ {
		if oldLines[i] != newLines[i] {
			diff.Changes = append(diff.Changes, Range{
				Start: Position{Line: i, Column: 0},
				End:   Position{Line: i, Column: len(oldLines[i])},
			})
		}
	}

	// 处理额外的行
	if len(oldLines) < len(newLines) {
		// 新文本有额外的行
		diff.Changes = append(diff.Changes, Range{
			Start: Position{Line: len(oldLines), Column: 0},
			End:   Position{Line: len(newLines) - 1, Column: len(newLines[len(newLines)-1])},
		})
	} else if len(oldLines) > len(newLines) {
		// 原文本有被删除的行
		diff.Changes = append(diff.Changes, Range{
			Start: Position{Line: len(newLines), Column: 0},
			End:   Position{Line: len(oldLines) - 1, Column: len(oldLines[len(oldLines)-1])},
		})
	}

	return diff, nil
}

// CodeAction 表示可能的代码操作
type CodeAction struct {
	Title       string   // 操作标题
	Kind        string   // 操作类型（quickfix, refactor等）
	Edit        TextEdit // 相关的编辑操作
	Description string   // 操作描述
}

// GetCodeActions 获取指定位置可用的代码操作
func (s *EditorSession) GetCodeActions(line, column int, context string) ([]CodeAction, error) {
	// 这是示例实现，实际应基于代码分析和语言规则

	// 获取当前行内容
	lineContent, err := s.editor.GetLineContent(s.filePath, line)
	if err != nil {
		return nil, err
	}

	actions := []CodeAction{}

	// 检查常见的代码操作机会
	if strings.Contains(lineContent, "TODO") {
		actions = append(actions, CodeAction{
			Title: "实现TODO项",
			Kind:  "quickfix",
			Edit: TextEdit{
				StartLine:   line,
				StartColumn: strings.Index(lineContent, "TODO"),
				EndLine:     line,
				EndColumn:   strings.Index(lineContent, "TODO") + 4,
				NewText:     "",
			},
			Description: "移除TODO注释并准备实现",
		})
	}

	// 检测未使用的导入
	if strings.Contains(lineContent, "import") && strings.Contains(lineContent, "\"") && context == "import" {
		if strings.Contains(lineContent, "_") {
			// 已经是空白导入
		} else {
			actions = append(actions, CodeAction{
				Title: "转换为空白导入",
				Kind:  "refactor",
				Edit: TextEdit{
					StartLine:   line,
					StartColumn: strings.Index(lineContent, "import") + 7,
					EndLine:     line,
					EndColumn:   strings.Index(lineContent, "\""),
					NewText:     " _ ",
				},
				Description: "将未使用的包转换为空白导入",
			})
		}
	}

	// 检测可能的错误处理改进
	if strings.Contains(lineContent, "err") && strings.Contains(lineContent, "=") && !strings.Contains(lineContent, "if") {
		actions = append(actions, CodeAction{
			Title: "添加错误处理",
			Kind:  "quickfix",
			Edit: TextEdit{
				StartLine:   line,
				StartColumn: len(lineContent),
				EndLine:     line,
				EndColumn:   len(lineContent),
				NewText:     "\nif err != nil {\n\treturn fmt.Errorf(\"操作失败: %w\", err)\n}",
			},
			Description: "为错误添加检查和处理",
		})
	}

	return actions, nil
}

// SaveHistory 保存编辑历史到文件
func (s *EditorSession) SaveHistory(historyFilePath string) error {
	var builder strings.Builder

	// 将编辑历史序列化为简单格式
	for i, edit := range s.editHistory {
		builder.WriteString(fmt.Sprintf("Edit %d:\n", i+1))
		builder.WriteString(fmt.Sprintf("  Range: [%d,%d] to [%d,%d]\n",
			edit.StartLine, edit.StartColumn, edit.EndLine, edit.EndColumn))
		builder.WriteString(fmt.Sprintf("  Text: %s\n", edit.NewText))
		builder.WriteString("\n")
	}

	// 写入文件
	return s.editor.project.WriteFile(historyFilePath, []byte(builder.String()))
}

// 工具函数

// extractIndent 提取行的缩进
func extractIndent(line string) string {
	for i, char := range line {
		if !isWhitespace(char) {
			return line[:i]
		}
	}
	return line
}

// isWhitespace 检查字符是否为空白
func isWhitespace(char rune) bool {
	return char == ' ' || char == '\t'
}

// endsWithOpenBrace 检查行是否以开括号结束
func endsWithOpenBrace(line string) bool {
	trimmed := strings.TrimSpace(line)
	return len(trimmed) > 0 && (strings.HasSuffix(trimmed, "{") ||
		strings.HasSuffix(trimmed, "(") ||
		strings.HasSuffix(trimmed, "[") ||
		strings.HasSuffix(trimmed, ":"))
}

// findWordAtPosition 在指定位置查找单词
func findWordAtPosition(line string, column int) Range {
	if column >= len(line) {
		return Range{
			Start: Position{Column: column},
			End:   Position{Column: column},
		}
	}

	// 寻找单词边界
	start := column
	for start > 0 && isWordChar(rune(line[start-1])) {
		start--
	}

	end := column
	for end < len(line) && isWordChar(rune(line[end])) {
		end++
	}

	return Range{
		Start: Position{Column: start},
		End:   Position{Column: end},
	}
}

// isWordChar 检查字符是否是单词字符
func isWordChar(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == '_'
}

// findMatchingBrackets 查找匹配的括号
func findMatchingBrackets(lines []string, line, column int) (Range, bool) {
	// 简化版括号匹配
	if line >= len(lines) {
		return Range{}, false
	}

	currentLine := lines[line]
	if column >= len(currentLine) {
		return Range{}, false
	}

	// 检查当前位置是否是开括号
	char := currentLine[column]
	var openBracket, closeBracket byte

	switch char {
	case '(':
		openBracket = '('
		closeBracket = ')'
	case '{':
		openBracket = '{'
		closeBracket = '}'
	case '[':
		openBracket = '['
		closeBracket = ']'
	case ')':
		openBracket = ')'
		closeBracket = '('
	case '}':
		openBracket = '}'
		closeBracket = '{'
	case ']':
		openBracket = ']'
		closeBracket = '['
	default:
		return Range{}, false
	}

	// 在开括号的情况下向前查找闭括号
	if char == openBracket && (openBracket == '(' || openBracket == '{' || openBracket == '[') {
		depth := 1
		startPos := Position{Line: line, Column: column}

		// 搜索匹配的闭括号
		for i := line; i < len(lines); i++ {
			searchLine := lines[i]
			startCol := 0

			if i == line {
				startCol = column + 1
			}

			for j := startCol; j < len(searchLine); j++ {
				if searchLine[j] == openBracket {
					depth++
				} else if searchLine[j] == closeBracket {
					depth--
					if depth == 0 {
						return Range{
							Start: startPos,
							End:   Position{Line: i, Column: j + 1},
						}, true
					}
				}
			}
		}
	}

	// 在闭括号的情况下向后查找开括号
	if char == closeBracket && (closeBracket == ')' || closeBracket == '}' || closeBracket == ']') {
		depth := 1
		endPos := Position{Line: line, Column: column + 1}

		// 搜索匹配的开括号
		for i := line; i >= 0; i-- {
			searchLine := lines[i]
			endCol := len(searchLine) - 1

			if i == line {
				endCol = column - 1
			}

			for j := endCol; j >= 0; j-- {
				if searchLine[j] == closeBracket {
					depth++
				} else if searchLine[j] == openBracket {
					depth--
					if depth == 0 {
						return Range{
							Start: Position{Line: i, Column: j},
							End:   endPos,
						}, true
					}
				}
			}
		}
	}

	return Range{}, false
}

// formatCodeBlock 格式化代码块（简化版）
func formatCodeBlock(code string) string {
	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(code))

	indentLevel := 0

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// 检查是否减少缩进
		if strings.HasPrefix(trimmed, "}") ||
			strings.HasPrefix(trimmed, ")") ||
			strings.HasPrefix(trimmed, "]") {
			indentLevel = max(0, indentLevel-1)
		}

		// 添加适当缩进
		indent := strings.Repeat("\t", indentLevel)
		result.WriteString(indent + trimmed + "\n")

		// 检查是否增加缩进
		if strings.HasSuffix(trimmed, "{") ||
			strings.HasSuffix(trimmed, "(") ||
			strings.HasSuffix(trimmed, "[") {
			indentLevel++
		}
	}

	return result.String()
}

// max 返回两个整数中的较大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min 返回两个整数中较小的一个
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
