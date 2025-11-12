package project

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/sjzsdu/langchaingo-cn/llms"
	"github.com/sjzsdu/tong/config"
	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/project"
	"github.com/sjzsdu/tong/project/tree"
	"github.com/sjzsdu/tong/prompt"
	"github.com/sjzsdu/tong/schema"
	"github.com/spf13/cobra"
	llmsPack "github.com/tmc/langchaingo/llms"
)

var (
	umlOutput        string // 输出文件路径
	umlMaxTokens     int    // 每批代码的最大 token 数
	umlSignatureOnly bool   // 只包含签名，不包含实现
	umlConcurrency   int    // 并发数
)

// UmlCommand UML 子命令
var UmlCommand = &cobra.Command{
	Use:   "uml",
	Short: lang.T("生成 UML 类图文档（智能架构分析）"),
	Long:  lang.T("两阶段生成：1) 分析项目生成主题大纲 2) 并发生成各主题的UML类图"),
	Run:   runUml,
}

func init() {
	UmlCommand.Flags().StringVarP(&umlOutput, "output", "o", "UML.md", lang.T("输出文件路径"))
	UmlCommand.Flags().IntVarP(&umlMaxTokens, "max-tokens", "m", 30000, lang.T("每批代码的最大 token 数"))
	UmlCommand.Flags().BoolVar(&umlSignatureOnly, "signature-only", false, lang.T("只包含类型定义和函数签名"))
	UmlCommand.Flags().IntVar(&umlConcurrency, "concurrency", 3, lang.T("并发处理主题的数量"))
}

// CodeBatch 代码批次
type CodeBatch struct {
	Name       string          // 批次名称
	Files      []string        // 文件路径列表
	Nodes      []*project.Node // 文件节点列表
	Content    string          // 打包后的代码内容
	TokenCount int             // 估算的 token 数
}

// ProjectOutline 项目大纲
type ProjectOutline struct {
	Topics []TopicOutline `json:"topics"`
}

// TopicOutline 主题大纲
type TopicOutline struct {
	ID          string   `json:"id"`          // 主题ID
	Title       string   `json:"title"`       // 主题标题
	Description string   `json:"description"` // 主题描述
	Modules     []string `json:"modules"`     // 包含的模块（批次名）
}

// TopicContent 主题内容
type TopicContent struct {
	Topic   TopicOutline
	Content string // UML 章节内容
	Error   error
}

// DocumentTools 文档工具集（用于LLM工具调用）
type DocumentTools struct {
	mu       sync.Mutex
	sections map[string]string // title -> content
}

// NewDocumentTools 创建文档工具
func NewDocumentTools() *DocumentTools {
	return &DocumentTools{
		sections: make(map[string]string),
	}
}

// AppendSection 添加新章节
func (dt *DocumentTools) AppendSection(title, content string) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.sections[title] = content
}

// UpdateSection 更新已有章节
func (dt *DocumentTools) UpdateSection(title, content string) bool {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if _, exists := dt.sections[title]; exists {
		dt.sections[title] = content
		return true
	}
	return false
}

// GetDocumentStructure 获取文档结构
func (dt *DocumentTools) GetDocumentStructure() string {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if len(dt.sections) == 0 {
		return "当前文档为空"
	}

	var result strings.Builder
	result.WriteString("当前文档结构：\n")
	for title := range dt.sections {
		result.WriteString(fmt.Sprintf("- %s\n", title))
	}
	return result.String()
}

// BuildDocument 构建完整文档
func (dt *DocumentTools) BuildDocument() string {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	// 按标题排序
	titles := make([]string, 0, len(dt.sections))
	for title := range dt.sections {
		titles = append(titles, title)
	}
	sort.Strings(titles)

	var doc strings.Builder
	for _, title := range titles {
		doc.WriteString(fmt.Sprintf("\n## %s\n\n", title))
		doc.WriteString(dt.sections[title])
		doc.WriteString("\n\n---\n")
	}
	return doc.String()
}

// runUml 执行 UML 生成（两阶段）
func runUml(cmd *cobra.Command, args []string) {
	if sharedProject == nil {
		log.Fatal("错误: 未找到共享的项目实例")
	}
	proj := sharedProject

	fmt.Printf("\n🎯 UML 智能生成器\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📊 项目: %s\n", proj.GetName())
	fmt.Printf("📁 路径: %s\n", proj.GetRootPath())
	fmt.Printf("⚙️  并发: %d\n\n", umlConcurrency)

	// 加载配置
	schemaConfig, err := schema.LoadMCPConfig(proj.GetRootPath(), "")
	if err != nil {
		fmt.Println("⚠️  使用默认配置")
		schemaConfig = &schema.SchemaConfig{
			MasterLLM: schema.LLMConfig{
				Type:   llms.LLMType(config.GetConfigWithDefault(config.KeyMasterLLM, "deepseek")),
				Params: map[string]interface{}{},
			},
		}
	}

	// 初始化 LLM
	ctx := context.Background()
	llm, err := llms.CreateLLM(schemaConfig.MasterLLM.Type, schemaConfig.MasterLLM.Params)
	if err != nil {
		log.Fatal("初始化 LLM 失败:", err)
	}

	// ==================== 阶段 1: 分析项目，生成主题大纲 ====================
	fmt.Printf("📋 阶段 1: 分析项目结构，生成主题大纲\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	batches, err := collectCodeBatches(proj)
	if err != nil {
		log.Fatal("收集代码失败:", err)
	}

	if len(batches) == 0 {
		log.Fatal("没有找到符合条件的代码文件")
	}

	fmt.Printf("📦 收集到 %d 个代码模块\n", len(batches))

	// 生成项目大纲
	outline, err := generateProjectOutline(ctx, llm, proj.GetName(), proj, batches)
	if err != nil {
		log.Fatal("生成项目大纲失败:", err)
	}

	fmt.Printf("\n✅ 大纲生成成功，共 %d 个主题:\n", len(outline.Topics))
	for i, topic := range outline.Topics {
		fmt.Printf("   %d. %s (%d 个模块)\n", i+1, topic.Title, len(topic.Modules))
	}

	// ==================== 阶段 2: 并发生成各主题的 UML ====================
	fmt.Printf("\n🚀 阶段 2: 并发生成 UML 类图\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// 创建文档工具
	docTools := NewDocumentTools()
	results := generateTopicsParallel(ctx, llm, proj.GetName(), outline, batches, docTools)

	// ==================== 阶段 3: 组装最终文档 ====================
	fmt.Printf("\n📝 阶段 3: 组装文档\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	finalDoc := assembleFinalDocument(proj.GetName(), proj.GetRootPath(), results)

	// 写入文件
	outputPath := filepath.Join(proj.GetRootPath(), umlOutput)
	err = os.WriteFile(outputPath, []byte(finalDoc), 0644)
	if err != nil {
		log.Fatal("写入文件失败:", err)
	}

	// 统计
	successCount := 0
	for _, r := range results {
		if r.Error == nil {
			successCount++
		}
	}

	fmt.Printf("\n✨ UML 文档生成完成!\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📄 输出: %s\n", outputPath)
	fmt.Printf("📊 成功: %d/%d 个主题\n", successCount, len(results))
	if successCount < len(results) {
		fmt.Printf("⚠️  失败: %d 个主题\n", len(results)-successCount)
	}
	fmt.Println()
}

// generateProjectOutline 生成项目大纲（阶段1）
func generateProjectOutline(ctx context.Context, llm interface{}, projectName string, proj *project.Project, batches []*CodeBatch) (*ProjectOutline, error) {
	var promptBuilder strings.Builder

	// ==================== 1. 添加目录树结构 ====================
	promptBuilder.WriteString("## 项目目录结构\n\n```\n")
	treeOutput := tree.TreeWithOptions(proj.Root(), false, false, 4) // 只显示目录，深度4
	promptBuilder.WriteString(treeOutput)
	promptBuilder.WriteString("```\n\n")

	// ==================== 2. 添加项目管理文件 ====================
	projectFiles := []string{"go.mod", "package.json", "pyproject.toml", "Cargo.toml", "pom.xml", "requirements.txt"}
	hasProjectFile := false

	for _, filename := range projectFiles {
		// 在根目录查找文件
		filePath := filepath.Join(proj.GetRootPath(), filename)
		if node := findNodeByPath(proj.Root(), filePath); node != nil {
			if !hasProjectFile {
				promptBuilder.WriteString("## 项目配置文件\n\n")
				hasProjectFile = true
			}
			content, err := node.ReadContent()
			if err == nil {
				promptBuilder.WriteString(fmt.Sprintf("### %s\n\n```\n%s\n```\n\n", filename, string(content)))
			}
		}
	}

	// ==================== 3. 添加模块摘要 ====================
	promptBuilder.WriteString("## 代码模块概览\n\n")
	for i, batch := range batches {
		if i >= 30 { // 限制数量，避免prompt过大
			promptBuilder.WriteString(fmt.Sprintf("...（还有 %d 个模块）\n", len(batches)-i))
			break
		}
		promptBuilder.WriteString(fmt.Sprintf("### %s (%d 文件, ~%d tokens)\n",
			batch.Name, len(batch.Files), batch.TokenCount))

		// 列出文件
		for j, file := range batch.Files {
			if j >= 5 {
				promptBuilder.WriteString(fmt.Sprintf("  ... 还有 %d 个文件\n", len(batch.Files)-j))
				break
			}
			promptBuilder.WriteString(fmt.Sprintf("  - %s\n", file))
		}
		promptBuilder.WriteString("\n")
	}

	// 构建大纲生成提示词
	outlinePrompt := buildOutlinePrompt(projectName, promptBuilder.String())

	// 调用 LLM
	generator, ok := llm.(interface {
		GenerateContent(context.Context, []llmsPack.MessageContent, ...llmsPack.CallOption) (*llmsPack.ContentResponse, error)
	})
	if !ok {
		return nil, fmt.Errorf("LLM 不支持 GenerateContent 方法")
	}

	msgs := []llmsPack.MessageContent{
		{
			Role:  llmsPack.ChatMessageTypeSystem,
			Parts: []llmsPack.ContentPart{llmsPack.TextPart(outlinePrompt)},
		},
	}

	fmt.Printf("🤖 正在分析项目结构...\n")
	response, err := generator.GenerateContent(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("LLM 调用失败: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("LLM 未返回内容")
	}

	content := response.Choices[0].Content

	// 提取 JSON
	jsonStr := extractJSON(content)
	if jsonStr == "" {
		return nil, fmt.Errorf("未找到有效的 JSON 大纲")
	}

	var outline ProjectOutline
	if err := json.Unmarshal([]byte(jsonStr), &outline); err != nil {
		return nil, fmt.Errorf("解析大纲 JSON 失败: %w\n内容: %s", err, jsonStr)
	}

	// 验证大纲
	if len(outline.Topics) == 0 {
		return nil, fmt.Errorf("大纲为空")
	}

	return &outline, nil
}

// generateTopicsParallel 并发生成各主题的 UML（阶段2）
func generateTopicsParallel(ctx context.Context, llm interface{}, projectName string, outline *ProjectOutline, batches []*CodeBatch, docTools *DocumentTools) []TopicContent {
	results := make([]TopicContent, len(outline.Topics))

	// 创建批次映射
	batchMap := make(map[string]*CodeBatch)
	for _, batch := range batches {
		batchMap[batch.Name] = batch
	}

	// 使用信号量控制并发数
	semaphore := make(chan struct{}, umlConcurrency)
	var wg sync.WaitGroup

	for i, topic := range outline.Topics {
		wg.Add(1)
		go func(idx int, t TopicOutline) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			fmt.Printf("🔄 [%d/%d] 生成主题: %s\n", idx+1, len(outline.Topics), t.Title)

			content, err := generateTopicUMLWithTools(ctx, llm, projectName, t, batchMap, docTools)
			results[idx] = TopicContent{
				Topic:   t,
				Content: content,
				Error:   err,
			}

			if err != nil {
				fmt.Printf("❌ [%d/%d] 失败: %s - %v\n", idx+1, len(outline.Topics), t.Title, err)
			} else {
				fmt.Printf("✅ [%d/%d] 完成: %s\n", idx+1, len(outline.Topics), t.Title)
			}
		}(i, topic)
	}

	wg.Wait()
	return results
}

// generateTopicUMLWithTools 使用工具生成单个主题的 UML
func generateTopicUMLWithTools(ctx context.Context, llm interface{}, projectName string, topic TopicOutline, batchMap map[string]*CodeBatch, docTools *DocumentTools) (string, error) {
	// 收集该主题相关的代码
	var codeBuilder strings.Builder
	var allFiles []string

	// 合并所有模块的类型和函数信息
	allTypes := make(map[string]*TypeInfo)
	allFunctions := make(map[string]*FuncInfo)

	for _, moduleName := range topic.Modules {
		batch, ok := batchMap[moduleName]
		if !ok {
			continue
		}

		codeBuilder.WriteString(fmt.Sprintf("\n### 模块: %s\n\n", moduleName))
		codeBuilder.WriteString(batch.Content)
		codeBuilder.WriteString("\n")

		allFiles = append(allFiles, batch.Files...)

		// 提取每个文件的AST信息
		for _, node := range batch.Nodes {
			content, err := node.ReadContent()
			if err != nil {
				continue
			}

			types, funcs := extractCodeIndexWithAST(string(content))

			// 合并类型信息
			for name, typeInfo := range types {
				if existing, exists := allTypes[name]; exists {
					// 如果类型已存在，合并字段
					existing.Fields = append(existing.Fields, typeInfo.Fields...)
				} else {
					allTypes[name] = typeInfo
				}
			}

			// 合并函数信息
			for name, funcInfo := range funcs {
				allFunctions[name] = funcInfo
			}
		}
	}

	// 构建提示词（传入完整代码和索引）
	topicPrompt := buildTopicPromptWithIndex(projectName, topic, codeBuilder.String(), allTypes, allFunctions)

	// 调用 LLM
	generator, ok := llm.(interface {
		GenerateContent(context.Context, []llmsPack.MessageContent, ...llmsPack.CallOption) (*llmsPack.ContentResponse, error)
	})
	if !ok {
		return "", fmt.Errorf("LLM 不支持 GenerateContent 方法")
	}

	msgs := []llmsPack.MessageContent{
		{
			Role:  llmsPack.ChatMessageTypeSystem,
			Parts: []llmsPack.ContentPart{llmsPack.TextPart(topicPrompt)},
		},
	}

	response, err := generator.GenerateContent(ctx, msgs)
	if err != nil {
		return "", fmt.Errorf("LLM 调用失败: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("LLM 未返回内容")
	}

	content := response.Choices[0].Content

	// 组装章节
	var section strings.Builder
	section.WriteString(fmt.Sprintf("## %s\n\n", topic.Title))
	section.WriteString(content)
	section.WriteString("\n\n**相关文件**:\n\n")
	for _, file := range allFiles {
		section.WriteString(fmt.Sprintf("- `%s`\n", file))
	}
	section.WriteString("\n---\n\n")

	return section.String(), nil
}

// assembleFinalDocument 组装最终文档（阶段3）
func assembleFinalDocument(projectName, projectPath string, results []TopicContent) string {
	var doc strings.Builder

	// 文档头部
	doc.WriteString(fmt.Sprintf("# %s - UML 架构文档\n\n", projectName))
	doc.WriteString(fmt.Sprintf("> 📁 项目路径: `%s`\n", projectPath))
	doc.WriteString(fmt.Sprintf("> 🤖 AI 智能生成 | 主题驱动架构分析\n\n"))

	// 目录
	doc.WriteString("## 📋 目录\n\n")
	for i, result := range results {
		if result.Error == nil {
			doc.WriteString(fmt.Sprintf("%d. [%s](#%d-%s)\n",
				i+1, result.Topic.Title, i+1, slugify(result.Topic.Title)))
		}
	}
	doc.WriteString("\n---\n\n")

	// 主题内容
	for i, result := range results {
		doc.WriteString(fmt.Sprintf("## %d. %s\n\n", i+1, result.Topic.Title))
		if result.Error != nil {
			doc.WriteString(fmt.Sprintf("⚠️ 生成失败: %v\n\n", result.Error))
		} else {
			doc.WriteString(result.Content)
		}
		doc.WriteString("\n---\n\n")
	}

	// 总结
	successCount := 0
	for _, r := range results {
		if r.Error == nil {
			successCount++
		}
	}

	doc.WriteString("## 📊 生成总结\n\n")
	doc.WriteString(fmt.Sprintf("- **总主题数**: %d\n", len(results)))
	doc.WriteString(fmt.Sprintf("- **成功生成**: %d\n", successCount))
	if successCount < len(results) {
		doc.WriteString(fmt.Sprintf("- **失败数量**: %d\n", len(results)-successCount))
	}
	doc.WriteString("\n---\n\n")
	doc.WriteString("*本文档由 AI 自动生成，采用主题驱动的架构分析方法*\n")

	return doc.String()
}

// TypeInfo 类型信息
type TypeInfo struct {
	Name   string
	Kind   string   // struct, interface, alias, etc.
	Fields []string // 字段列表
}

// FuncInfo 函数信息
type FuncInfo struct {
	Name     string
	Receiver string // 接收者类型，如果是方法
	IsMethod bool
}

// extractCodeIndexWithAST 使用 AST 提取代码索引
func extractCodeIndexWithAST(code string) (map[string]*TypeInfo, map[string]*FuncInfo) {
	types := make(map[string]*TypeInfo)
	functions := make(map[string]*FuncInfo)

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", code, parser.ParseComments)
	if err != nil {
		return types, functions
	}

	// 遍历声明
	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			// 处理类型声明
			if d.Tok == token.TYPE {
				for _, spec := range d.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						typeName := typeSpec.Name.Name
						typeInfo := &TypeInfo{
							Name:   typeName,
							Fields: []string{},
						}

						switch t := typeSpec.Type.(type) {
						case *ast.StructType:
							typeInfo.Kind = "struct"
							if t.Fields != nil {
								for _, field := range t.Fields.List {
									for _, name := range field.Names {
										typeInfo.Fields = append(typeInfo.Fields, name.Name)
									}
								}
							}
						case *ast.InterfaceType:
							typeInfo.Kind = "interface"
							if t.Methods != nil {
								for _, method := range t.Methods.List {
									for _, name := range method.Names {
										typeInfo.Fields = append(typeInfo.Fields, name.Name)
									}
								}
							}
						default:
							typeInfo.Kind = "alias"
						}

						types[typeName] = typeInfo
					}
				}
			}

		case *ast.FuncDecl:
			// 处理函数/方法声明
			funcName := d.Name.Name
			funcInfo := &FuncInfo{
				Name: funcName,
			}

			if d.Recv != nil && len(d.Recv.List) > 0 {
				// 这是一个方法
				funcInfo.IsMethod = true
				// 提取接收者类型
				recvType := d.Recv.List[0].Type
				funcInfo.Receiver = formatType(recvType)
			}

			functions[funcName] = funcInfo
		}
	}

	return types, functions
}

// formatType 格式化类型表达式
func formatType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return formatType(t.X)
	case *ast.SelectorExpr:
		return formatType(t.X) + "." + t.Sel.Name
	default:
		return "unknown"
	}
}

// buildCodeIndex 构建代码索引字符串
func buildCodeIndex(types map[string]*TypeInfo, functions map[string]*FuncInfo) string {
	var index strings.Builder

	if len(types) > 0 {
		index.WriteString("## 📋 代码中实际存在的类型（只能使用这些）\n\n")

		// 按名称排序
		typeNames := make([]string, 0, len(types))
		for name := range types {
			typeNames = append(typeNames, name)
		}
		sort.Strings(typeNames)

		for _, name := range typeNames {
			typeInfo := types[name]
			index.WriteString(fmt.Sprintf("### `%s` (%s)\n", name, typeInfo.Kind))
			if len(typeInfo.Fields) > 0 {
				index.WriteString("字段/方法:\n")
				for _, field := range typeInfo.Fields {
					index.WriteString(fmt.Sprintf("- `%s`\n", field))
				}
			}
			index.WriteString("\n")
		}
	}

	if len(functions) > 0 {
		index.WriteString("## 📋 代码中实际存在的函数/方法（只能使用这些）\n\n")

		// 按名称排序
		funcNames := make([]string, 0, len(functions))
		for name := range functions {
			funcNames = append(funcNames, name)
		}
		sort.Strings(funcNames)

		for _, name := range funcNames {
			funcInfo := functions[name]
			if funcInfo.IsMethod {
				index.WriteString(fmt.Sprintf("- `%s` (方法，接收者: `%s`)\n", name, funcInfo.Receiver))
			} else {
				index.WriteString(fmt.Sprintf("- `%s` (函数)\n", name))
			}
		}
		index.WriteString("\n")
	}

	return index.String()
}

// buildTopicPromptWithIndex 构建主题UML生成提示词（使用预解析的索引）
func buildTopicPromptWithIndex(projectName string, topic TopicOutline, code string, types map[string]*TypeInfo, functions map[string]*FuncInfo) string {
	// 加载系统提示词
	systemPrompt := prompt.ShowPromptContent("uml")
	if systemPrompt == "" {
		systemPrompt = "你是 UML 类图生成专家。"
	}

	var promptBuilder strings.Builder

	promptBuilder.WriteString(systemPrompt)
	promptBuilder.WriteString("\n\n---\n\n")

	fmt.Fprintf(&promptBuilder, "## 当前主题: %s\n\n", topic.Title)
	fmt.Fprintf(&promptBuilder, "**描述**: %s\n\n", topic.Description)
	fmt.Fprintf(&promptBuilder, "**包含模块**: %s\n\n", strings.Join(topic.Modules, ", "))

	// 使用预解析的代码索引
	codeIndex := buildCodeIndex(types, functions)

	if codeIndex != "" {
		promptBuilder.WriteString(codeIndex)
		promptBuilder.WriteString("---\n\n")

		// 生成类型白名单
		typeList := make([]string, 0, len(types))
		for name := range types {
			typeList = append(typeList, name)
		}
		sort.Strings(typeList)

		// 类型白名单
		fmt.Fprintf(&promptBuilder, `## 🚨 强制约束：类型白名单

**你只能使用以下类型作为UML类名**（一个都不能多）：

`)
		for _, name := range typeList {
			fmt.Fprintf(&promptBuilder, "- `%s` ✅\n", name)
		}

		fmt.Fprintf(&promptBuilder, `
**任何不在此列表中的类型名都是禁止的！**

## 🚨 错误示例（这些都是幻觉，绝对不能出现）

❌ class ProjectController - 代码中没有这个类型
❌ class ProjectService - 代码中没有这个类型
❌ class ProjectRepository - 代码中没有这个类型
❌ class SearchEngineClient - 代码中没有这个类型
❌ class FileProcessor - 代码中没有这个类型

## ✅ 正确做法

**第1步**：查看上面的类型白名单
**第2步**：只使用白名单中的类型名
**第3步**：如果代码只有2-3个类型，那就只生成2-3个类的UML，不要臆造其他类

## 📝 提醒

- 如果白名单中只有很少的类型（比如只有2个），说明这个模块很简单
- 不要试图"美化"或"完善"架构，只描述实际存在的代码
- 宁可生成一个简单但准确的UML，也不要生成一个复杂但充满幻觉的UML

`)
	}

	promptBuilder.WriteString("## 代码内容\n\n")
	promptBuilder.WriteString("```\n")
	promptBuilder.WriteString(code)
	promptBuilder.WriteString("\n```\n\n")

	promptBuilder.WriteString("## 你的任务\n\n")
	promptBuilder.WriteString(fmt.Sprintf("请为 **%s** 主题生成 UML 类图，严格遵守上述所有约束。\n", topic.Title))

	return promptBuilder.String()
}

// buildOutlinePrompt 构建大纲生成提示词
func buildOutlinePrompt(projectName, projectInfo string) string {
	systemPrompt := prompt.ShowPromptContent("uml-outline")
	if systemPrompt == "" {
		systemPrompt = "你是软件架构分析专家，负责生成 UML 大纲。"
	}

	var promptBuilder strings.Builder
	promptBuilder.WriteString(systemPrompt)
	promptBuilder.WriteString("\n\n")
	fmt.Fprintf(&promptBuilder, "## 项目名称: %s\n\n", projectName)
	promptBuilder.WriteString(projectInfo)
	return promptBuilder.String()
}

// extractJSON 从文本中提取 JSON
func extractJSON(text string) string {
	// 尝试找到 JSON 代码块
	if strings.Contains(text, "```json") {
		start := strings.Index(text, "```json")
		end := strings.Index(text[start+7:], "```")
		if end > 0 {
			return strings.TrimSpace(text[start+7 : start+7+end])
		}
	}

	// 尝试找到纯 JSON（以 { 开头，} 结尾）
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		return strings.TrimSpace(text[start : end+1])
	}

	return ""
}

// slugify 将标题转换为 slug
func slugify(title string) string {
	s := strings.ToLower(title)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	return s
}

// FileNode 文件节点信息
type FileNode struct {
	Path string
	Node *project.Node
}

// collectCodeBatches 收集代码批次
func collectCodeBatches(proj *project.Project) ([]*CodeBatch, error) {
	// 按目录组织文件节点
	dirMap := make(map[string][]FileNode)

	err := proj.Visit(func(path string, node *project.Node, depth int) error {
		if node.IsDir {
			return nil
		}

		// 只处理 .go 文件
		if !strings.HasSuffix(node.Name, ".go") {
			return nil
		}

		// 获取目录路径
		dir := filepath.Dir(path)
		if dir == "." {
			dir = "root"
		}

		// 去掉前导斜杠，统一格式（LLM 返回的模块名不带斜杠）
		dir = strings.TrimPrefix(dir, "/")
		if dir == "" {
			dir = "root"
		}

		// 跳过某些目录
		if shouldSkipDirectory(dir) {
			return nil
		}

		dirMap[dir] = append(dirMap[dir], FileNode{Path: path, Node: node})
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 为每个目录创建批次
	var batches []*CodeBatch

	// 排序目录以保证稳定性
	var dirs []string
	for dir := range dirMap {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	for _, dir := range dirs {
		fileNodes := dirMap[dir]

		// 打包文件
		content, files, nodes, err := packFileNodes(fileNodes)
		if err != nil {
			log.Printf("警告: 打包 %s 失败: %v", dir, err)
			continue
		}

		// 估算 token 数
		tokenCount := estimateTokens(content)

		// 如果超过限制，尝试拆分
		if tokenCount > umlMaxTokens {
			subBatches := splitByFileNodes(fileNodes, umlMaxTokens)
			for _, subBatch := range subBatches {
				if !shouldSkipBatch(subBatch) {
					batches = append(batches, subBatch)
				} else {
					fmt.Printf("⏭️  跳过纯函数模块: %s (只有函数，没有类型定义)\n", subBatch.Name)
				}
			}
		} else {
			batch := &CodeBatch{
				Name:       dir,
				Files:      files,
				Nodes:      nodes,
				Content:    content,
				TokenCount: tokenCount,
			}

			// 检查是否应该跳过
			if !shouldSkipBatch(batch) {
				batches = append(batches, batch)
			} else {
				fmt.Printf("⏭️  跳过纯函数模块: %s (只有函数，没有类型定义)\n", dir)
			}
		}
	}

	return batches, nil
}

// packFileNodes 打包文件节点内容
func packFileNodes(fileNodes []FileNode) (string, []string, []*project.Node, error) {
	var builder strings.Builder
	var files []string
	var nodes []*project.Node

	for _, fn := range fileNodes {
		content, err := fn.Node.ReadContent()
		if err != nil {
			log.Printf("警告: 读取 %s 失败: %v", fn.Path, err)
			continue
		}

		builder.WriteString(fmt.Sprintf("// FILE: %s\n", fn.Path))

		// 如果启用签名模式，提取签名
		if umlSignatureOnly {
			signature := compressToSignatures(string(content))
			builder.WriteString(signature)
		} else {
			builder.WriteString(string(content))
		}

		builder.WriteString("\n\n")

		files = append(files, fn.Path)
		nodes = append(nodes, fn.Node)
	}

	return builder.String(), files, nodes, nil
}

// compressToSignatures 压缩到签名（简化版）
func compressToSignatures(code string) string {
	var result strings.Builder
	lines := strings.Split(code, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 保留 package, import, type, const, var, func 声明
		if strings.HasPrefix(trimmed, "package ") ||
			strings.HasPrefix(trimmed, "import ") ||
			strings.HasPrefix(trimmed, "type ") ||
			strings.HasPrefix(trimmed, "const ") ||
			strings.HasPrefix(trimmed, "var ") ||
			strings.HasPrefix(trimmed, "func ") {
			result.WriteString(line + "\n")
		}
	}

	return result.String()
}

// splitByFileNodes 按文件数拆分
func splitByFileNodes(fileNodes []FileNode, maxTokens int) []*CodeBatch {
	var batches []*CodeBatch
	var currentBatch *CodeBatch
	var currentFiles []string
	var currentNodes []*project.Node

	for _, fn := range fileNodes {
		content, err := fn.Node.ReadContent()
		if err != nil {
			continue
		}

		fileContent := fmt.Sprintf("// FILE: %s\n%s\n\n", fn.Path, string(content))
		fileTokens := estimateTokens(fileContent)

		// 如果当前批次为空或者加入会超限
		if currentBatch == nil || currentBatch.TokenCount+fileTokens > maxTokens {
			if currentBatch != nil {
				batches = append(batches, currentBatch)
			}
			// 去掉前导斜杠，统一格式
			dirName := strings.TrimPrefix(filepath.Dir(fn.Path), "/")
			currentBatch = &CodeBatch{
				Name:       fmt.Sprintf("%s-part-%d", dirName, len(batches)+1),
				Files:      []string{},
				Nodes:      []*project.Node{},
				Content:    "",
				TokenCount: 0,
			}
			currentFiles = []string{}
			currentNodes = []*project.Node{}
		}

		currentFiles = append(currentFiles, fn.Path)
		currentNodes = append(currentNodes, fn.Node)
		currentBatch.Files = currentFiles
		currentBatch.Nodes = currentNodes
		currentBatch.Content += fileContent
		currentBatch.TokenCount += fileTokens
	}

	if currentBatch != nil && len(currentBatch.Files) > 0 {
		batches = append(batches, currentBatch)
	}

	return batches
}

// estimateTokens 估算 token 数
func estimateTokens(text string) int {
	// 简单估算：1 token ≈ 4 字符
	return len(text) / 4
}

// findNodeByPath 根据路径查找节点
func findNodeByPath(root *project.Node, targetPath string) *project.Node {
	if root == nil {
		return nil
	}

	// 如果当前节点就是目标
	if root.Path == targetPath {
		return root
	}

	// 递归查找子节点
	if root.IsDir {
		for _, child := range root.Children {
			if result := findNodeByPath(child, targetPath); result != nil {
				return result
			}
		}
	}

	return nil
}

// shouldSkipDirectory 判断是否跳过目录
func shouldSkipDirectory(dir string) bool {
	skipDirs := []string{
		"helper/coroutine", "helper/display", "helper/json", "helper/renders", "helper/fonts",
	}

	for _, skip := range skipDirs {
		if strings.Contains(dir, skip) {
			return true
		}
	}
	return false
}

// shouldSkipBatch 判断是否跳过批次（纯函数模块）
func shouldSkipBatch(batch *CodeBatch) bool {
	// 使用 AST 分析代码
	typeCount := 0
	funcCount := 0

	for _, node := range batch.Nodes {
		content, err := node.ReadContent()
		if err != nil {
			continue
		}

		types, funcs := extractCodeIndexWithAST(string(content))
		typeCount += len(types)
		funcCount += len(funcs)
	}

	// 如果只有函数，没有类型定义，跳过
	if typeCount == 0 && funcCount > 0 {
		return true
	}

	// 如果函数数量远大于类型数量（比如 >20 个函数但 <3 个类型），也跳过
	if funcCount > 20 && typeCount < 3 {
		return true
	}

	return false
}
