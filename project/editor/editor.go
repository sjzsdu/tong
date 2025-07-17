package editor

import (
	"fmt"
	"strings"

	"github.com/sjzsdu/tong/project"
)

// TextEdit 表示对文本的单个编辑操作
type TextEdit struct {
	// 编辑开始的位置 (行号和列号，均从0开始)
	StartLine   int
	StartColumn int
	// 编辑结束的位置
	EndLine   int
	EndColumn int
	// 替换的新文本
	NewText string
	// 原始文本（用于撤销）
	OldText string
}

// Position 表示文本中的位置
type Position struct {
	Line   int // 从0开始
	Column int // 从0开始
}

// Range 表示文本中的范围
type Range struct {
	Start Position
	End   Position
}

// SearchOptions 搜索选项
type SearchOptions struct {
	CaseSensitive bool // 是否区分大小写
	WholeWord     bool // 是否匹配整个单词
	RegExp        bool // 是否使用正则表达式
}

// LineEndingType 表示行尾类型
type LineEndingType int

const (
	LineEndingLF   LineEndingType = iota // \n (Unix/Linux/macOS)
	LineEndingCRLF                       // \r\n (Windows)
	LineEndingCR                         // \r (旧版 macOS)
)

// ChangeSet 表示一组编辑操作
type ChangeSet struct {
	Edits []TextEdit
}

// EditorAPI 提供类似编辑器的API，用于高效编辑文本
type EditorAPI struct {
	project *project.Project
}

// NewEditorAPI 创建一个新的编辑器API
func NewEditorAPI(project *project.Project) *EditorAPI {
	return &EditorAPI{
		project: project,
	}
}

// ApplyEdit 应用单个编辑操作到文件
func (e *EditorAPI) ApplyEdit(filePath string, edit TextEdit) error {
	// 读取文件内容
	content, err := e.project.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %v", err)
	}

	// 将内容分割为行
	lines := strings.Split(string(content), "\n")

	// 检查范围是否有效
	if edit.StartLine < 0 || edit.StartLine >= len(lines) ||
		edit.EndLine < 0 || edit.EndLine >= len(lines) {
		return fmt.Errorf("编辑范围超出文件边界: [%d,%d] - [%d,%d]",
			edit.StartLine, edit.StartColumn, edit.EndLine, edit.EndColumn)
	}

	// 应用编辑
	newLines := make([]string, 0, len(lines))

	// 添加编辑前的行
	for i := 0; i < edit.StartLine; i++ {
		newLines = append(newLines, lines[i])
	}

	// 处理编辑开始行
	if edit.StartLine == edit.EndLine {
		// 单行编辑
		line := lines[edit.StartLine]
		if edit.StartColumn > len(line) || edit.EndColumn > len(line) {
			return fmt.Errorf("编辑列超出行边界: [%d,%d]", edit.StartColumn, edit.EndColumn)
		}

		newLine := line[:edit.StartColumn] + edit.NewText + line[edit.EndColumn:]
		newLines = append(newLines, newLine)
	} else {
		// 多行编辑
		startLine := lines[edit.StartLine]
		if edit.StartColumn > len(startLine) {
			return fmt.Errorf("开始列超出行边界: [%d]", edit.StartColumn)
		}

		// 处理开始行
		firstPart := startLine[:edit.StartColumn]

		// 处理结束行
		endLine := lines[edit.EndLine]
		if edit.EndColumn > len(endLine) {
			return fmt.Errorf("结束列超出行边界: [%d]", edit.EndColumn)
		}
		lastPart := endLine[edit.EndColumn:]

		// 构建新文本
		editedText := firstPart + edit.NewText + lastPart

		// 分割新文本为行并添加
		editedLines := strings.Split(editedText, "\n")
		newLines = append(newLines, editedLines...)
	}

	// 添加编辑后的行
	for i := edit.EndLine + 1; i < len(lines); i++ {
		newLines = append(newLines, lines[i])
	}

	// 更新文件内容
	newContent := []byte(strings.Join(newLines, "\n"))
	return e.project.WriteFile(filePath, newContent)
}

// ApplyEdits 应用多个编辑操作到文件
// 注意：编辑操作按照从后向前的顺序应用，以避免位置变化影响后续编辑
func (e *EditorAPI) ApplyEdits(filePath string, edits []TextEdit) error {
	// 按编辑起始位置排序（从后向前）
	sortEdits(edits)

	// 逐个应用编辑
	for _, edit := range edits {
		if err := e.ApplyEdit(filePath, edit); err != nil {
			return err
		}
	}

	return nil
}

// ApplyChangeSet 应用变更集
func (e *EditorAPI) ApplyChangeSet(filePath string, changeSet ChangeSet) error {
	return e.ApplyEdits(filePath, changeSet.Edits)
}

// GetLineContent 获取指定行的内容
func (e *EditorAPI) GetLineContent(filePath string, line int) (string, error) {
	content, err := e.project.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	if line < 0 || line >= len(lines) {
		return "", fmt.Errorf("行号超出范围: %d", line)
	}

	return lines[line], nil
}

// GetLineCount 获取文件的行数
func (e *EditorAPI) GetLineCount(filePath string) (int, error) {
	content, err := e.project.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("读取文件失败: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	return len(lines), nil
}

// GetTextInRange 获取指定范围内的文本
func (e *EditorAPI) GetTextInRange(filePath string, startLine, startColumn, endLine, endColumn int) (string, error) {
	content, err := e.project.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	if startLine < 0 || startLine >= len(lines) ||
		endLine < 0 || endLine >= len(lines) ||
		startLine > endLine {
		return "", fmt.Errorf("范围无效: [%d,%d] - [%d,%d]",
			startLine, startColumn, endLine, endColumn)
	}

	if startLine == endLine {
		// 单行
		line := lines[startLine]
		if startColumn > len(line) || endColumn > len(line) || startColumn > endColumn {
			return "", fmt.Errorf("列范围无效: [%d,%d]", startColumn, endColumn)
		}
		return line[startColumn:endColumn], nil
	}

	// 多行
	var result strings.Builder

	// 第一行
	firstLine := lines[startLine]
	if startColumn > len(firstLine) {
		return "", fmt.Errorf("开始列超出行边界: [%d]", startColumn)
	}
	result.WriteString(firstLine[startColumn:])
	result.WriteString("\n")

	// 中间行
	for i := startLine + 1; i < endLine; i++ {
		result.WriteString(lines[i])
		result.WriteString("\n")
	}

	// 最后一行
	lastLine := lines[endLine]
	if endColumn > len(lastLine) {
		return "", fmt.Errorf("结束列超出行边界: [%d]", endColumn)
	}
	result.WriteString(lastLine[:endColumn])

	return result.String(), nil
}

// InsertText 在指定位置插入文本
func (e *EditorAPI) InsertText(filePath string, line, column int, text string) error {
	edit := TextEdit{
		StartLine:   line,
		StartColumn: column,
		EndLine:     line,
		EndColumn:   column,
		NewText:     text,
	}
	return e.ApplyEdit(filePath, edit)
}

// ReplaceText 替换指定范围的文本
func (e *EditorAPI) ReplaceText(filePath string, startLine, startColumn, endLine, endColumn int, newText string) error {
	edit := TextEdit{
		StartLine:   startLine,
		StartColumn: startColumn,
		EndLine:     endLine,
		EndColumn:   endColumn,
		NewText:     newText,
	}
	return e.ApplyEdit(filePath, edit)
}

// DeleteText 删除指定范围的文本
func (e *EditorAPI) DeleteText(filePath string, startLine, startColumn, endLine, endColumn int) error {
	edit := TextEdit{
		StartLine:   startLine,
		StartColumn: startColumn,
		EndLine:     endLine,
		EndColumn:   endColumn,
		NewText:     "",
	}
	return e.ApplyEdit(filePath, edit)
}

// FindText 在文件中查找文本，返回所有匹配的位置
func (e *EditorAPI) FindText(filePath string, searchText string, caseSensitive bool) ([]Range, error) {
	content, err := e.project.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}

	contentStr := string(content)
	if !caseSensitive {
		contentStr = strings.ToLower(contentStr)
		searchText = strings.ToLower(searchText)
	}

	lines := strings.Split(string(content), "\n")
	var results []Range

	// 遍历每一行查找匹配
	for lineIdx, line := range lines {
		lineToSearch := line
		if !caseSensitive {
			lineToSearch = strings.ToLower(line)
		}

		startIdx := 0
		for {
			columnIdx := strings.Index(lineToSearch[startIdx:], searchText)
			if columnIdx == -1 {
				break
			}

			columnIdx += startIdx
			results = append(results, Range{
				Start: Position{Line: lineIdx, Column: columnIdx},
				End:   Position{Line: lineIdx, Column: columnIdx + len(searchText)},
			})

			startIdx = columnIdx + 1
		}
	}

	return results, nil
}

// ReplaceAll 替换文件中所有匹配的文本
func (e *EditorAPI) ReplaceAll(filePath string, searchText, replaceText string, caseSensitive bool) (int, error) {
	// 找到所有匹配位置
	ranges, err := e.FindText(filePath, searchText, caseSensitive)
	if err != nil {
		return 0, err
	}

	// 按从后往前的顺序替换
	edits := make([]TextEdit, len(ranges))
	for i, r := range ranges {
		edits[i] = TextEdit{
			StartLine:   r.Start.Line,
			StartColumn: r.Start.Column,
			EndLine:     r.End.Line,
			EndColumn:   r.End.Column,
			NewText:     replaceText,
		}
	}

	// 应用所有编辑
	if err := e.ApplyEdits(filePath, edits); err != nil {
		return 0, err
	}

	return len(ranges), nil
}

// GetLineEndings 检测文件的行尾类型
func (e *EditorAPI) GetLineEndings(filePath string) (LineEndingType, error) {
	content, err := e.project.ReadFile(filePath)
	if err != nil {
		return LineEndingLF, err
	}

	text := string(content)

	// 检测行尾类型
	if strings.Contains(text, "\r\n") {
		return LineEndingCRLF, nil
	} else if strings.Contains(text, "\r") {
		return LineEndingCR, nil
	}
	return LineEndingLF, nil
}

// SetLineEndings 设置文件的行尾类型
func (e *EditorAPI) SetLineEndings(filePath string, endingType LineEndingType) error {
	content, err := e.project.ReadFile(filePath)
	if err != nil {
		return err
	}

	text := string(content)

	// 先标准化为LF
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// 转换为目标格式
	var newText string
	switch endingType {
	case LineEndingCRLF:
		newText = strings.ReplaceAll(text, "\n", "\r\n")
	case LineEndingCR:
		newText = strings.ReplaceAll(text, "\n", "\r")
	default:
		newText = text
	}

	return e.project.WriteFile(filePath, []byte(newText))
}

// FindTextWithOptions 在文件中查找文本，支持高级搜索选项
func (e *EditorAPI) FindTextWithOptions(filePath string, searchText string, options SearchOptions) ([]Range, error) {
	content, err := e.project.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}

	var searchTextLower string
	if !options.CaseSensitive {
		searchTextLower = strings.ToLower(searchText)
	} else {
		searchTextLower = searchText
	}

	lines := strings.Split(string(content), "\n")
	var results []Range

	// 遍历每一行查找匹配
	for lineIdx, line := range lines {
		lineToSearch := line
		if !options.CaseSensitive {
			lineToSearch = strings.ToLower(line)
		}

		startIdx := 0
		for {
			columnIdx := strings.Index(lineToSearch[startIdx:], searchTextLower)
			if columnIdx == -1 {
				break
			}

			columnIdx += startIdx

			// 如果需要匹配整词，检查边界
			if options.WholeWord {
				// 检查左边界
				leftBoundary := columnIdx == 0 || !isWordChar(rune(lineToSearch[columnIdx-1]))
				// 检查右边界
				rightBoundary := columnIdx+len(searchTextLower) >= len(lineToSearch) ||
					!isWordChar(rune(lineToSearch[columnIdx+len(searchTextLower)]))

				if !leftBoundary || !rightBoundary {
					// 不是完整词，继续查找
					startIdx = columnIdx + 1
					continue
				}
			}

			results = append(results, Range{
				Start: Position{Line: lineIdx, Column: columnIdx},
				End:   Position{Line: lineIdx, Column: columnIdx + len(searchText)},
			})

			startIdx = columnIdx + 1
		}
	}

	return results, nil
}

// ReplaceAllWithOptions 替换文件中所有匹配的文本，支持高级搜索选项
func (e *EditorAPI) ReplaceAllWithOptions(filePath string, searchText, replaceText string, options SearchOptions) (int, error) {
	// 找到所有匹配位置
	ranges, err := e.FindTextWithOptions(filePath, searchText, options)
	if err != nil {
		return 0, err
	}

	// 按从后往前的顺序替换
	edits := make([]TextEdit, len(ranges))
	for i, r := range ranges {
		edits[i] = TextEdit{
			StartLine:   r.Start.Line,
			StartColumn: r.Start.Column,
			EndLine:     r.End.Line,
			EndColumn:   r.End.Column,
			NewText:     replaceText,
		}
	}

	// 应用所有编辑
	if err := e.ApplyEdits(filePath, edits); err != nil {
		return 0, err
	}

	return len(ranges), nil
}

// sortEdits 对编辑操作进行排序，以便从后向前应用
func sortEdits(edits []TextEdit) {
	// 使用冒泡排序（对于小数量的编辑操作足够高效）
	n := len(edits)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if comparePositions(edits[j].StartLine, edits[j].StartColumn,
				edits[j+1].StartLine, edits[j+1].StartColumn) < 0 {
				edits[j], edits[j+1] = edits[j+1], edits[j]
			}
		}
	}
}

// comparePositions 比较两个位置，返回:
// 1: pos1在pos2之后
// 0: 位置相同
// -1: pos1在pos2之前
func comparePositions(line1, col1, line2, col2 int) int {
	if line1 > line2 {
		return 1
	}
	if line1 < line2 {
		return -1
	}
	// 行相同，比较列
	if col1 > col2 {
		return 1
	}
	if col1 < col2 {
		return -1
	}
	return 0
}
