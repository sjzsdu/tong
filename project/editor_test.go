package project

import (
	"strings"
	"testing"
)

func TestEditorAPI(t *testing.T) {
	// 创建测试项目
	project := NewProject("/tmp/editor_test")

	// 创建测试文件
	testContent := []byte("package main\n\nimport (\n\t\"fmt\"\n)\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n")
	testFilePath := "test.go"

	err := project.CreateFile(testFilePath, testContent, nil)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 创建编辑器API
	editor := NewEditorAPI(project)

	// 测试获取行数
	t.Run("GetLineCount", func(t *testing.T) {
		lineCount, err := editor.GetLineCount(testFilePath)
		if err != nil {
			t.Fatalf("获取行数失败: %v", err)
		}

		expectedLineCount := 10 // 包括最后一个空行
		if lineCount != expectedLineCount {
			t.Errorf("行数不匹配: 期望 %d, 实际 %d", expectedLineCount, lineCount)
		}
	})

	// 测试获取行内容
	t.Run("GetLineContent", func(t *testing.T) {
		lineContent, err := editor.GetLineContent(testFilePath, 7)
		if err != nil {
			t.Fatalf("获取行内容失败: %v", err)
		}

		expectedContent := "\tfmt.Println(\"Hello, World!\")"
		if lineContent != expectedContent {
			t.Errorf("行内容不匹配: 期望 %s, 实际 %s", expectedContent, lineContent)
		}
	})

	// 测试插入文本
	t.Run("InsertText", func(t *testing.T) {
		err := editor.InsertText(testFilePath, 7, 19, ", Go")
		if err != nil {
			t.Fatalf("插入文本失败: %v", err)
		}

		// 验证插入结果
		lineContent, err := editor.GetLineContent(testFilePath, 7)
		if err != nil {
			t.Fatalf("获取行内容失败: %v", err)
		}

		expectedContent := "\tfmt.Println(\"Hello, Go, World!\")"
		if lineContent != expectedContent {
			t.Errorf("插入后内容不匹配: 期望 %s, 实际 %s", expectedContent, lineContent)
		}
	})

	// 测试替换文本
	t.Run("ReplaceText", func(t *testing.T) {
		// 我们先读取原始行内容，确认实际字符位置
		lineContent, err := editor.GetLineContent(testFilePath, 7)
		if err != nil {
			t.Fatalf("获取行内容失败: %v", err)
		}

		// 确定 "Hello, Go, World!" 文本的精确位置
		startQuotePos := strings.Index(lineContent, "\"")
		endQuotePos := strings.LastIndex(lineContent, "\"") + 1

		// 替换整个字符串内容，包括引号
		err = editor.ReplaceText(testFilePath, 7, startQuotePos, 7, endQuotePos, "\"Tong\"")
		if err != nil {
			t.Fatalf("替换文本失败: %v", err)
		}

		// 验证替换结果
		lineContent, err = editor.GetLineContent(testFilePath, 7)
		if err != nil {
			t.Fatalf("获取行内容失败: %v", err)
		}

		expectedContent := "\tfmt.Println(\"Tong\")"
		if lineContent != expectedContent {
			t.Errorf("替换后内容不匹配: 期望 %s, 实际 %s", expectedContent, lineContent)
		}
	})

	// 测试查找文本
	t.Run("FindText", func(t *testing.T) {
		// 先在多个地方添加相同文本
		err := editor.ReplaceText(testFilePath, 7, 0, 7, 20, "\tfmt.Println(\"Tong\")\n\tfmt.Println(\"Tong\")")
		if err != nil {
			t.Fatalf("准备查找测试失败: %v", err)
		}

		// 查找所有"Tong"
		matches, err := editor.FindText(testFilePath, "Tong", true)
		if err != nil {
			t.Fatalf("查找文本失败: %v", err)
		}

		if len(matches) != 2 {
			t.Errorf("查找结果数量不匹配: 期望 2, 实际 %d", len(matches))
		}
	})

	// 测试批量替换
	t.Run("ReplaceAll", func(t *testing.T) {
		count, err := editor.ReplaceAll(testFilePath, "Tong", "编辑器", true)
		if err != nil {
			t.Fatalf("批量替换失败: %v", err)
		}

		if count != 2 {
			t.Errorf("替换次数不匹配: 期望 2, 实际 %d", count)
		}

		// 验证替换结果
		content, err := editor.project.ReadFile(testFilePath)
		if err != nil {
			t.Fatalf("读取文件失败: %v", err)
		}

		if !strings.Contains(string(content), "编辑器") || strings.Contains(string(content), "Tong") {
			t.Errorf("替换结果不正确")
		}
	})
}

func TestEditorSession(t *testing.T) {
	// 创建测试项目
	project := NewProject("/tmp/editor_session_test")

	// 创建测试文件
	testContent := []byte("package main\n\nimport (\n\t\"fmt\"\n)\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n")
	testFilePath := "session_test.go"

	err := project.CreateFile(testFilePath, testContent, nil)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 创建编辑器API和会话
	editor := NewEditorAPI(project)
	session := NewEditorSession(editor, testFilePath)

	// 测试应用编辑
	t.Run("ApplyEdit", func(t *testing.T) {
		// 先读取原始行内容，确认实际字符位置
		lineContent, err := editor.GetLineContent(testFilePath, 7)
		if err != nil {
			t.Fatalf("获取行内容失败: %v", err)
		}

		// 确定字符串的精确位置
		startQuotePos := strings.Index(lineContent, "\"")
		endQuotePos := strings.LastIndex(lineContent, "\"") + 1

		edit := TextEdit{
			StartLine:   7,
			StartColumn: startQuotePos,
			EndLine:     7,
			EndColumn:   endQuotePos,
			NewText:     "\"编辑器会话\"",
		}

		err = session.ApplyEdit(edit)
		if err != nil {
			t.Fatalf("应用编辑失败: %v", err)
		}

		// 验证编辑结果
		lineContent, err = editor.GetLineContent(testFilePath, 7)
		if err != nil {
			t.Fatalf("获取行内容失败: %v", err)
		}

		expectedContent := "\tfmt.Println(\"编辑器会话\")"
		if lineContent != expectedContent {
			t.Errorf("编辑后内容不匹配: 期望 %s, 实际 %s", expectedContent, lineContent)
		}
	})
	// 测试撤销/重做
	t.Run("UndoRedo", func(t *testing.T) {
		// 创建一个测试文件并获取其内容
		simpleTestContent := []byte("package main\n\nfunc main() {\n\tfmt.Println(\"Original Text\")\n}\n")
		simpleFilePath := "undo_redo_test.go"

		err := project.CreateFile(simpleFilePath, simpleTestContent, nil)
		if err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}

		// 创建编辑会话
		simpleEditor := NewEditorAPI(project)
		simpleSession := NewEditorSession(simpleEditor, simpleFilePath)

		// 获取行内容并找到引号位置
		lineContent, err := simpleEditor.GetLineContent(simpleFilePath, 3)
		if err != nil {
			t.Fatalf("获取行内容失败: %v", err)
		}

		// 找到引号内的文本位置
		startQuotePos := strings.Index(lineContent, "\"") + 1
		endQuotePos := strings.LastIndex(lineContent, "\"")

		// 应用第一次编辑
		edit1 := TextEdit{
			StartLine:   3,
			StartColumn: startQuotePos,
			EndLine:     3,
			EndColumn:   endQuotePos,
			NewText:     "First Edit",
		}

		err = simpleSession.ApplyEdit(edit1)
		if err != nil {
			t.Fatalf("应用第一次编辑失败: %v", err)
		}

		// 验证第一次编辑
		lineContent, err = simpleEditor.GetLineContent(simpleFilePath, 3)
		if err != nil {
			t.Fatalf("获取行内容失败: %v", err)
		}
		expectedContent := "\tfmt.Println(\"First Edit\")"
		if lineContent != expectedContent {
			t.Errorf("第一次编辑后内容不匹配: 期望 %s, 实际 %s", expectedContent, lineContent)
		}

		// 应用第二次编辑
		// 获取新的引号位置
		lineContent, err = simpleEditor.GetLineContent(simpleFilePath, 3)
		if err != nil {
			t.Fatalf("获取行内容失败: %v", err)
		}

		startQuotePos = strings.Index(lineContent, "\"") + 1
		endQuotePos = strings.LastIndex(lineContent, "\"")

		edit2 := TextEdit{
			StartLine:   3,
			StartColumn: startQuotePos,
			EndLine:     3,
			EndColumn:   endQuotePos,
			NewText:     "Second Edit",
		}

		err = simpleSession.ApplyEdit(edit2)
		if err != nil {
			t.Fatalf("应用第二次编辑失败: %v", err)
		}

		// 验证第二次编辑
		lineContent, err = simpleEditor.GetLineContent(simpleFilePath, 3)
		if err != nil {
			t.Fatalf("获取行内容失败: %v", err)
		}
		expectedContent = "\tfmt.Println(\"Second Edit\")"
		if lineContent != expectedContent {
			t.Errorf("第二次编辑后内容不匹配: 期望 %s, 实际 %s", expectedContent, lineContent)
		}

		// 撤销
		err = simpleSession.Undo()
		if err != nil {
			t.Fatalf("撤销失败: %v", err)
		}

		// 验证撤销结果
		lineContent, err = simpleEditor.GetLineContent(simpleFilePath, 3)
		if err != nil {
			t.Fatalf("获取行内容失败: %v", err)
		}
		expectedContent = "\tfmt.Println(\"First Edit\")"
		if lineContent != expectedContent {
			t.Errorf("撤销后内容不匹配: 期望 %s, 实际 %s", expectedContent, lineContent)
		}

		// 重做
		err = simpleSession.Redo()
		if err != nil {
			t.Fatalf("重做失败: %v", err)
		}

		// 验证重做结果
		lineContent, err = simpleEditor.GetLineContent(simpleFilePath, 3)
		if err != nil {
			t.Fatalf("获取行内容失败: %v", err)
		}
		expectedContent = "\tfmt.Println(\"Second Edit\")"
		if lineContent != expectedContent {
			t.Errorf("重做后内容不匹配: 期望 %s, 实际 %s", expectedContent, lineContent)
		}
	})

	// 测试智能缩进
	t.Run("SmartIndent", func(t *testing.T) {
		// 创建一个测试文件并获取其内容
		indentTestContent := []byte("package main\n\nfunc main() {\n\tif true {\n\t\t// 内部代码\n\t}\n}\n")
		indentFilePath := "indent_test.go"

		err := project.CreateFile(indentFilePath, indentTestContent, nil)
		if err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}

		// 创建编辑会话
		indentEditor := NewEditorAPI(project)
		indentSession := NewEditorSession(indentEditor, indentFilePath)

		// 添加需要缩进的行在大括号内
		edit := TextEdit{
			StartLine:   4,
			StartColumn: 0,
			EndLine:     4,
			EndColumn:   11, // "\t\t// 内部代码"
			NewText:     "fmt.Println(\"需要缩进\")",
		}

		err = indentSession.ApplyEdit(edit)
		if err != nil {
			t.Fatalf("添加需要缩进的行失败: %v", err)
		}

		// 应用智能缩进
		err = indentSession.SmartIndent(4)
		if err != nil {
			t.Fatalf("应用智能缩进失败: %v", err)
		}

		// 验证缩进结果
		lineContent, err := indentEditor.GetLineContent(indentFilePath, 4)
		if err != nil {
			t.Fatalf("获取行内容失败: %v", err)
		}

		if !strings.HasPrefix(lineContent, "\t\t") {
			t.Errorf("智能缩进不正确: %s", lineContent)
		}
	})
}

func TestEditorIntegration(t *testing.T) {
	// 创建测试项目
	project := NewProject("/tmp/editor_integration_test")

	// 创建测试文件
	testContent := []byte("package main\n\nimport (\n\t\"fmt\"\n\t\"strings\"\n)\n\nfunc main() {\n\tfmt.Println(\"编辑器集成测试\")\n}\n")
	testFilePath := "integration_test.go"

	err := project.CreateFile(testFilePath, testContent, nil)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 创建编辑器集成
	integration := NewEditorIntegration(project)

	// 测试打开文件
	t.Run("OpenFile", func(t *testing.T) {
		session, err := integration.OpenFile(testFilePath)
		if err != nil {
			t.Fatalf("打开文件失败: %v", err)
		}

		if session == nil {
			t.Errorf("未能创建有效的会话")
		}
	})

	// 测试执行命令
	t.Run("ExecuteCommand", func(t *testing.T) {
		result, err := integration.ExecuteCommand(
			CommandFormat,
			"document",
			testFilePath,
			map[string]interface{}{})

		if err != nil {
			t.Fatalf("执行命令失败: %v", err)
		}

		if !result.Success {
			t.Errorf("命令执行失败: %s", result.Message)
		}
	})

	// 测试模型集成请求
	t.Run("ProcessModelRequest", func(t *testing.T) {
		req := ModelIntegrationRequest{
			Action:      "format",
			FilePath:    testFilePath,
			QueryParams: map[string]interface{}{},
		}

		resp, err := integration.ProcessModelRequest(req)
		if err != nil {
			t.Fatalf("处理模型请求失败: %v", err)
		}

		if !resp.Success {
			t.Errorf("模型请求处理失败: %s", resp.Message)
		}
	})
}
