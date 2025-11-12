package project

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sjzsdu/langchaingo-cn/llms"
	"github.com/sjzsdu/tong/config"
	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/lang"
	"github.com/sjzsdu/tong/project"
	"github.com/sjzsdu/tong/prompt"
	"github.com/sjzsdu/tong/schema"
	"github.com/sjzsdu/tong/share"
	"github.com/spf13/cobra"
	llmsPack "github.com/tmc/langchaingo/llms"
)

var (
	umlOutput        string // 输出文件路径
	umlMaxTokens     int    // 单批最大 token 数
	umlByModule      bool   // 是否按模块拆分
	umlPromptName    string // 提示词名称
	umlSignatureOnly bool   // 只包含签名，不包含实现
	umlApiMaxTokens  int    // API 的实际 token 限制
)

// UmlCommand UML 子命令
var UmlCommand = &cobra.Command{
	Use:   "uml",
	Short: lang.T("生成 UML 类图文档"),
	Long:  lang.T("分析项目代码，生成结构化的 UML 类图 Markdown 文档"),
	Run:   runUml,
}

func init() {
	UmlCommand.Flags().StringVarP(&umlOutput, "output", "o", "UML.md", lang.T("输出文件路径"))
	UmlCommand.Flags().IntVarP(&umlMaxTokens, "max-tokens", "m", 50000, lang.T("单批最大 token 数（估算）"))
	UmlCommand.Flags().BoolVarP(&umlByModule, "by-module", "b", true, lang.T("按模块/包拆分处理"))
	UmlCommand.Flags().StringVarP(&umlPromptName, "prompt", "p", "uml", lang.T("提示词名称"))
	UmlCommand.Flags().BoolVar(&umlSignatureOnly, "signature-only", false, lang.T("只包含类型定义和函数签名，不包含实现"))
	UmlCommand.Flags().IntVar(&umlApiMaxTokens, "api-limit", 28000, lang.T("API 的实际 token 限制（提示词+代码）"))
}

// CodeBatch 代码批次
type CodeBatch struct {
	Name       string   // 批次名称（如模块名）
	Files      []string // 文件路径列表
	Content    string   // 打包后的代码内容
	TokenCount int      // 估算的 token 数
}

// runUml 执行 UML 生成
func runUml(cmd *cobra.Command, args []string) {
	// 1. 获取项目实例
	if sharedProject == nil {
		log.Fatal("错误: 未找到共享的项目实例")
	}
	proj := sharedProject

	fmt.Printf("📊 开始分析项目: %s\n", proj.GetName())
	fmt.Printf("📁 项目路径: %s\n", proj.GetRootPath())
	fmt.Printf("⚙️  配置: max-tokens=%d, api-limit=%d, signature-only=%v\n",
		umlMaxTokens, umlApiMaxTokens, umlSignatureOnly)

	// 2. 加载配置（如果失败则使用默认值）
	schemaConfig, err := schema.LoadMCPConfig(proj.GetRootPath(), "")
	if err != nil {
		fmt.Println("⚠️  加载配置文件失败，将使用默认 LLM 配置:", err)
		schemaConfig = &schema.SchemaConfig{
			MasterLLM: schema.LLMConfig{
				Type:   llms.LLMType(config.GetConfigWithDefault(config.KeyMasterLLM, "deepseek")),
				Params: map[string]interface{}{},
			},
		}
	}

	// 3. 收集和分批代码文件
	batches, err := collectCodeBatches(proj)
	if err != nil {
		log.Fatal(err)
	}

	if len(batches) == 0 {
		log.Fatal("没有找到符合条件的代码文件")
	}

	fmt.Printf("📦 共找到 %d 个代码批次\n", len(batches))
	for i, batch := range batches {
		fmt.Printf("  %d. %s - %d 个文件，约 %d tokens\n",
			i+1, batch.Name, len(batch.Files), batch.TokenCount)
	}

	// 4. 初始化 LLM
	llm, err := llms.CreateLLM(schemaConfig.MasterLLM.Type, schemaConfig.MasterLLM.Params)
	if err != nil {
		log.Fatal("初始化 LLM 失败:", err)
	}

	// 5. 加载提示词模板
	systemPrompt := prompt.ShowPromptContent(umlPromptName)
	if systemPrompt == "" {
		log.Fatal("提示词模板不存在:", umlPromptName)
	}

	if share.GetDebug() {
		helper.PrintWithLabel("提示词模板:", systemPrompt)
	}

	// 6. 创建输出文件
	outputPath := filepath.Join(proj.GetRootPath(), umlOutput)
	fmt.Printf("📝 输出文件: %s\n\n", outputPath)

	// 7. 逐批次生成 UML
	ctx := context.Background()
	var fullDocument strings.Builder

	// 写入文档头部
	fullDocument.WriteString(fmt.Sprintf("# %s UML 类图文档\n\n", proj.GetName()))
	fullDocument.WriteString(fmt.Sprintf("> 自动生成于项目路径: `%s`\n\n", proj.GetRootPath()))
	fullDocument.WriteString("## 目录\n\n")
	for i, batch := range batches {
		fullDocument.WriteString(fmt.Sprintf("%d. [%s](#%d-%s)\n", i+1, batch.Name, i+1, strings.ToLower(strings.ReplaceAll(batch.Name, " ", "-"))))
	}
	fullDocument.WriteString("\n---\n\n")

	// 逐批次处理
	for i, batch := range batches {
		fmt.Printf("🔄 处理批次 %d/%d: %s\n", i+1, len(batches), batch.Name)

		// 构造该批次的提示
		batchPrompt := buildBatchPrompt(systemPrompt, proj.GetName(), batch)

		// 调用 LLM 生成 UML
		result, err := generateUMLWithLLM(ctx, llm, batchPrompt)
		if err != nil {
			log.Printf("❌ 批次 %s 生成失败: %v", batch.Name, err)
			fullDocument.WriteString(fmt.Sprintf("## %d. %s\n\n", i+1, batch.Name))
			fullDocument.WriteString(fmt.Sprintf("⚠️ 生成失败: %v\n\n", err))
			continue
		}

		// 添加到完整文档
		fullDocument.WriteString(fmt.Sprintf("## %d. %s\n\n", i+1, batch.Name))
		fullDocument.WriteString(result)
		fullDocument.WriteString("\n\n---\n\n")

		fmt.Printf("✅ 批次 %s 完成\n\n", batch.Name)
	}

	// 8. 写入文件
	fullDocument.WriteString("## 总结\n\n")
	fullDocument.WriteString("本文档由 AI 自动生成，展示了项目的主要类结构和关系。\n")

	err = os.WriteFile(outputPath, []byte(fullDocument.String()), 0644)
	if err != nil {
		log.Fatal("写入文件失败:", err)
	}

	fmt.Printf("✨ UML 文档生成成功: %s\n", outputPath)
}

// collectCodeBatches 收集并分批代码文件
func collectCodeBatches(proj *project.Project) ([]*CodeBatch, error) {
	// 收集所有符合条件的文件（sharedProject 已经根据 extensions 和 excludePatterns 过滤过了）
	var allFiles []string
	filesByModule := make(map[string][]string) // module -> files

	err := proj.Visit(func(path string, node *project.Node, depth int) error {
		// 跳过目录
		if node.IsDir {
			return nil
		}

		// 直接添加文件（已经由 sharedProject 过滤过了）
		allFiles = append(allFiles, path)

		// 如果按模块分组
		if umlByModule {
			module := extractModuleName(path)
			filesByModule[module] = append(filesByModule[module], path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 生成批次
	var batches []*CodeBatch

	if umlByModule && len(filesByModule) > 1 {
		// 按模块生成批次
		for module, files := range filesByModule {
			batch := &CodeBatch{
				Name:  module,
				Files: files,
			}
			batch.Content = packFiles(proj, files)
			batch.TokenCount = estimateTokens(batch.Content)

			// 如果单个模块超过限制，需要进一步拆分
			if batch.TokenCount > umlMaxTokens {
				fmt.Printf("  ⚠️  模块 '%s' 过大 (%d tokens)，按子目录拆分\n", module, batch.TokenCount)
				subBatches := splitBySubdirectory(proj, module, files, umlMaxTokens)
				batches = append(batches, subBatches...)
			} else {
				batches = append(batches, batch)
			}
		}
	} else {
		// 不分模块，按大小拆分
		batches = splitBatchByFileCount(proj, "全部代码", allFiles, umlMaxTokens)
	}

	return batches, nil
}

// extractModuleName 从路径提取模块名
func extractModuleName(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) > 1 {
		return parts[0] // 返回第一级目录作为模块名
	}
	return "root"
}

// packFiles 打包文件内容
func packFiles(proj *project.Project, files []string) string {
	var builder strings.Builder

	for _, filePath := range files {
		node, err := proj.FindNode(filePath)
		if err != nil {
			continue
		}

		// 读取文件内容
		content, err := node.ReadContent()
		if err != nil {
			continue
		}

		builder.WriteString(fmt.Sprintf("// FILE: %s\n", filePath))

		// 如果是签名模式，只保留签名
		if umlSignatureOnly {
			builder.WriteString(compressToSignatures(content, filePath))
		} else {
			builder.Write(content)
		}

		builder.WriteString("\n\n")
	}

	return builder.String()
}

// compressToSignatures 压缩文件内容，只保留类型定义和函数签名
func compressToSignatures(content []byte, filePath string) string {
	text := string(content)
	var result strings.Builder

	// 对于 Go 文件
	if strings.HasSuffix(filePath, ".go") {
		lines := strings.Split(text, "\n")
		inMultilineComment := false
		inStruct := false
		braceLevel := 0

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)

			// 跟踪多行注释
			if strings.Contains(trimmed, "/*") {
				inMultilineComment = true
			}
			if strings.Contains(trimmed, "*/") {
				inMultilineComment = false
				continue
			}
			if inMultilineComment {
				continue
			}

			// 跟踪大括号层级
			braceLevel += strings.Count(line, "{")
			braceLevel -= strings.Count(line, "}")

			// 保留：package, import, type, const, var 声明
			if strings.HasPrefix(trimmed, "package ") ||
				strings.HasPrefix(trimmed, "import ") ||
				strings.HasPrefix(trimmed, "type ") ||
				strings.HasPrefix(trimmed, "const ") ||
				strings.HasPrefix(trimmed, "var ") {
				result.WriteString(line + "\n")
				if strings.Contains(trimmed, "struct {") || strings.Contains(trimmed, "interface {") {
					inStruct = true
				}
				continue
			}

			// 在 struct/interface 内部，保留字段定义
			if inStruct && braceLevel > 0 {
				if trimmed != "" && !strings.HasPrefix(trimmed, "//") {
					result.WriteString(line + "\n")
				}
				if strings.Contains(trimmed, "}") && braceLevel == 0 {
					inStruct = false
				}
				continue
			}

			// 保留函数签名（但不保留实现）
			if strings.HasPrefix(trimmed, "func ") {
				// 只保留签名行
				result.WriteString(line)
				// 如果是单行函数声明（接口方法），保留完整
				if !strings.Contains(line, "{") {
					result.WriteString("\n")
				} else {
					// 多行函数，添加省略标记
					result.WriteString(" /* ... */ }\n")
					// 跳过函数体
					for braceLevel > 0 && len(lines) > 0 {
						continue
					}
				}
				continue
			}
		}
	}

	// 对于其他语言，可以扩展类似逻辑
	if result.Len() == 0 {
		return text // 如果压缩失败，返回原文
	}

	return result.String()
}

// splitBySubdirectory 按子目录拆分过大的模块
func splitBySubdirectory(proj *project.Project, moduleName string, files []string, maxTokens int) []*CodeBatch {
	// 按二级目录分组
	subDirFiles := make(map[string][]string)

	for _, file := range files {
		parts := strings.Split(strings.Trim(file, "/"), "/")
		subDir := moduleName
		if len(parts) > 2 {
			subDir = filepath.Join(parts[0], parts[1]) // 二级目录
		}
		subDirFiles[subDir] = append(subDirFiles[subDir], file)
	}

	var batches []*CodeBatch
	for subDir, subFiles := range subDirFiles {
		batch := &CodeBatch{
			Name:  subDir,
			Files: subFiles,
		}
		batch.Content = packFiles(proj, subFiles)
		batch.TokenCount = estimateTokens(batch.Content)

		// 如果子目录还是太大，按文件数量拆分
		if batch.TokenCount > maxTokens {
			fmt.Printf("    ⚠️  子目录 '%s' 仍然过大 (%d tokens)，按文件拆分\n", subDir, batch.TokenCount)
			subBatches := splitBatchByFileCount(proj, subDir, subFiles, maxTokens)
			batches = append(batches, subBatches...)
		} else {
			batches = append(batches, batch)
		}
	}

	return batches
}

// splitBatchByFileCount 按文件数量智能拆分（确保每批不超过限制）
func splitBatchByFileCount(proj *project.Project, baseName string, files []string, maxTokens int) []*CodeBatch {
	var batches []*CodeBatch
	var currentFiles []string
	var currentTokens int

	for _, filePath := range files {
		node, err := proj.FindNode(filePath)
		if err != nil {
			continue
		}

		content, err := node.ReadContent()
		if err != nil {
			continue
		}

		fileTokens := estimateTokens(string(content))

		// 如果单个文件就超过限制，截断或跳过
		if fileTokens > maxTokens {
			fmt.Printf("      ⚠️  文件 '%s' 过大 (%d tokens)，将被截断\n", filePath, fileTokens)
			truncatedContent := truncateContent(string(content), maxTokens)
			batch := &CodeBatch{
				Name:       fmt.Sprintf("%s-%s", baseName, filepath.Base(filePath)),
				Files:      []string{filePath},
				Content:    fmt.Sprintf("// FILE: %s (截断)\n%s", filePath, truncatedContent),
				TokenCount: estimateTokens(truncatedContent),
			}
			batches = append(batches, batch)
			continue
		}

		// 如果加入当前文件会超限，先保存当前批次
		if currentTokens+fileTokens > maxTokens && len(currentFiles) > 0 {
			batch := &CodeBatch{
				Name:       fmt.Sprintf("%s-Part%d", baseName, len(batches)+1),
				Files:      currentFiles,
				Content:    packFiles(proj, currentFiles),
				TokenCount: currentTokens,
			}
			batches = append(batches, batch)
			currentFiles = nil
			currentTokens = 0
		}

		currentFiles = append(currentFiles, filePath)
		currentTokens += fileTokens
	}

	// 保存最后一批
	if len(currentFiles) > 0 {
		batch := &CodeBatch{
			Name:       fmt.Sprintf("%s-Part%d", baseName, len(batches)+1),
			Files:      currentFiles,
			Content:    packFiles(proj, currentFiles),
			TokenCount: currentTokens,
		}
		batches = append(batches, batch)
	}

	return batches
}

// truncateContent 截断内容到指定 token 数
func truncateContent(content string, maxTokens int) string {
	// 简单截断：保留前 N 个字符（约 maxTokens * 4）
	maxChars := maxTokens * 4
	if len(content) <= maxChars {
		return content
	}

	truncated := content[:maxChars]
	// 尝试在完整行截断
	if lastNewline := strings.LastIndex(truncated, "\n"); lastNewline > maxChars/2 {
		truncated = truncated[:lastNewline]
	}

	return truncated + "\n\n// ... 内容被截断 ..."
}

// estimateTokens 估算 token 数量（简单估算：1 token ≈ 4 字符）
func estimateTokens(content string) int {
	return len(content) / 4
}

// buildBatchPrompt 构造批次提示词
func buildBatchPrompt(systemPrompt, projectName string, batch *CodeBatch) string {
	var builder strings.Builder

	builder.WriteString(systemPrompt)
	builder.WriteString("\n\n---\n\n")
	builder.WriteString("# 项目信息\n\n")
	builder.WriteString(fmt.Sprintf("- **项目名称**: %s\n", projectName))
	builder.WriteString(fmt.Sprintf("- **分析模块**: %s\n", batch.Name))
	builder.WriteString(fmt.Sprintf("- **文件数量**: %d\n", len(batch.Files)))
	builder.WriteString(fmt.Sprintf("- **代码规模**: 约 %d tokens\n\n", batch.TokenCount))

	// 添加签名模式提示
	if umlSignatureOnly {
		builder.WriteString("> **注意**: 当前使用签名模式，代码中只包含类型定义和函数签名。\n\n")
	}

	builder.WriteString("# 文件列表\n\n")
	for _, file := range batch.Files {
		builder.WriteString(fmt.Sprintf("- `%s`\n", file))
	}
	builder.WriteString("\n# 源代码\n\n")
	builder.WriteString("```\n")
	builder.WriteString(batch.Content)
	builder.WriteString("\n```\n\n")
	builder.WriteString("# 任务\n\n")
	builder.WriteString(fmt.Sprintf("请为 **%s** 模块生成 UML 类图文档，遵循上述格式要求。\n", batch.Name))

	promptText := builder.String()
	totalTokens := estimateTokens(promptText)

	// 检查是否超过 API 限制
	if totalTokens > umlApiMaxTokens {
		fmt.Printf("      ⚠️  警告: 提示词过大 (%d tokens，限制: %d)，可能导致 API 调用失败\n", totalTokens, umlApiMaxTokens)
		fmt.Printf("      💡 建议: 使用 --signature-only 模式或减小 --max-tokens 值\n")
	}

	return promptText
}

// generateUMLWithLLM 使用 LLM 生成 UML
func generateUMLWithLLM(ctx context.Context, llm interface{}, prompt string) (string, error) {
	// 使用 GenerateContent 方法调用 LLM
	type llmGenerator interface {
		GenerateContent(context.Context, []llmsPack.MessageContent, ...llmsPack.CallOption) (*llmsPack.ContentResponse, error)
	}

	generator, ok := llm.(llmGenerator)
	if !ok {
		return "", fmt.Errorf("LLM 类型不支持 GenerateContent 方法")
	}

	// 构造消息
	msgs := []llmsPack.MessageContent{
		{
			Role:  llmsPack.ChatMessageTypeSystem,
			Parts: []llmsPack.ContentPart{llmsPack.TextPart(prompt)},
		},
	}

	// 调用 LLM
	response, err := generator.GenerateContent(ctx, msgs)
	if err != nil {
		return "", fmt.Errorf("LLM 调用失败: %w", err)
	}

	// 提取生成的内容
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("LLM 未返回任何内容")
	}

	return response.Choices[0].Content, nil
}
