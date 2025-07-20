# cmdio - 命令行交互式处理模块

`cmdio` 是一个高度抽象和可扩展的命令行交互式处理模块，提供了批量处理和流式处理两种模式，可以轻松构建各种类型的交互式命令行应用程序。

## 核心特性

- **多种处理模式**：支持批量处理和流式处理两种模式
- **高度抽象**：基于接口设计，易于扩展和自定义
- **灵活配置**：使用函数式选项模式进行配置
- **完善的错误处理**：支持超时、取消和错误传播
- **LLM 集成**：内置与 langchaingo 的集成支持

## 核心组件

### 处理器接口

所有处理器都实现了 `InteractiveProcessor` 接口：

```go
type InteractiveProcessor interface {
	// 处理输入
	ProcessInput(ctx context.Context, input string) error
	// 处理输出
	ProcessOutput(ctx context.Context) error
	// 设置输出写入器
	SetOutputWriter(writer io.Writer)
	// 获取处理器配置
	GetConfig() ProcessorConfig
	// 设置处理器配置
	SetConfig(config ProcessorConfig)
}
```

### 处理器类型

#### 基础处理器

- `BaseProcessor`：所有处理器的基础，提供共享功能

#### 批量处理器

- `BatchProcessor`：等待处理完成后一次性返回所有结果
- `ChainProcessor`：将 langchaingo 的 chain 适配为批量处理器
- `EchoProcessor`：简单的回显处理器，用于测试和示例

#### 流式处理器

- `StreamProcessor`：在处理过程中不断返回部分结果
- `CustomStreamProcessor`：支持自定义流式处理函数
- `StreamEchoProcessor`：流式回显处理器，用于测试和示例

### 交互式会话

- `InteractiveSession`：管理交互式会话的生命周期
- `MockInteractiveSession`：用于测试的模拟会话

## 使用示例

### 创建批量处理器

```go
// 创建一个简单的批量处理器
processor := cmdio.NewBatchProcessor(func(ctx context.Context, input string) (string, error) {
    return "处理结果: " + input, nil
})

// 创建交互式会话
session := cmdio.NewInteractiveSession(
    processor,
    cmdio.WithWelcome("欢迎使用批量处理器示例"),
    cmdio.WithTip("输入文本将被处理"),
    cmdio.WithTip("输入 'quit' 退出"),
)

// 启动会话
ctx := context.Background()
session.Start(ctx)
```

### 创建流式处理器

```go
// 创建一个流式处理器
processor := cmdio.NewStreamProcessor(func(ctx context.Context, input string) (string, error) {
    return "流式处理结果: " + input, nil
})

// 创建交互式会话
session := cmdio.NewInteractiveSession(
    processor,
    cmdio.WithWelcome("欢迎使用流式处理器示例"),
    cmdio.WithTip("输入文本将被流式处理"),
    cmdio.WithTip("输入 'quit' 退出"),
)

// 启动会话
ctx := context.Background()
session.Start(ctx)
```

### 创建自定义流式处理器

```go
// 创建一个自定义流式处理器
processor := cmdio.NewCustomStreamProcessor(func(ctx context.Context, input string, writer io.Writer) error {
    // 添加前缀
    writer.Write([]byte("开始处理: " + input + "\n"))
    
    // 逐字符输出，模拟流式响应
    for _, char := range input {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            writer.Write([]byte(string(char)))
            time.Sleep(50 * time.Millisecond)
        }
    }
    
    // 添加后缀
    writer.Write([]byte("\n处理完成！"))
    return nil
})

// 创建交互式会话
session := cmdio.NewInteractiveSession(
    processor,
    cmdio.WithWelcome("欢迎使用自定义流式处理器示例"),
    cmdio.WithTip("输入文本将被逐字符处理"),
    cmdio.WithTip("输入 'quit' 退出"),
)

// 启动会话
ctx := context.Background()
session.Start(ctx)
```

### 与 langchaingo 集成

```go
// 初始化 LLM
llm, err := llms.CreateLLM(llms.DeepSeekLLM, nil)
if err != nil {
    log.Fatal(err)
}

// 创建对话记忆
chatMemory := memory.NewConversationBuffer()

// 创建对话链
chain := chains.NewConversation(llm, chatMemory)

// 创建流式处理器
processor := cmdio.NewStreamChainProcessor(chain)

// 创建交互式会话
session := cmdio.NewInteractiveSession(
    processor,
    cmdio.WithWelcome("欢迎使用 AI 聊天助手！输入 'quit' 退出。"),
    cmdio.WithTip("提示：您可以询问任何问题，AI 将尽力回答。"),
    cmdio.WithPrompt("您: "),
    cmdio.WithExitCommands("quit", "exit", "q", "退出"),
)

// 启动会话
ctx := context.Background()
session.Start(ctx)
```

### 使用工厂函数

```go
// 使用工厂函数创建聊天适配器
session := cmdio.CreateChatAdapter(chain, true) // true 表示使用流式模式

// 启动会话
ctx := context.Background()
session.Start(ctx)
```

## 配置选项

### 处理器配置

```go
config := cmdio.DefaultProcessorConfig()
config.Mode = cmdio.StreamMode        // 设置处理模式
config.Timeout = time.Second * 30     // 设置超时时间
config.MaxWaitTime = time.Second * 60 // 设置最大等待时间
config.StreamInterval = time.Millisecond * 50 // 设置流式间隔
```

### 会话配置

```go
session := cmdio.NewInteractiveSession(
    processor,
    cmdio.WithRenderer(renderer),           // 设置渲染器
    cmdio.WithWelcome("欢迎使用"),           // 设置欢迎信息
    cmdio.WithTip("这是一个提示"),           // 添加提示信息
    cmdio.WithTips("提示1", "提示2"),       // 添加多个提示信息
    cmdio.WithPrompt("命令> "),             // 设置命令提示符
    cmdio.WithExitCommands("quit", "exit"), // 设置退出命令
)
```

## 设计理念

### 抽象与分层

- **接口抽象**：通过接口定义行为，实现解耦
- **基础组件**：提供基础功能，可被扩展和组合
- **专用实现**：针对特定场景的优化实现

### 函数式选项模式

使用函数式选项模式进行配置，提供灵活且类型安全的配置方式。

### 错误处理

- 统一的错误处理机制
- 支持上下文取消和超时
- 友好的错误提示

## 扩展

### 自定义处理器

实现 `InteractiveProcessor` 接口来创建自定义处理器。

### 自定义渲染器

实现 `renders.Renderer` 接口来创建自定义渲染器。

## 最佳实践

1. **选择合适的处理模式**：根据需求选择批量或流式处理
2. **错误处理**：妥善处理各种错误情况
3. **资源管理**：正确使用上下文进行资源管理
4. **用户体验**：提供清晰的提示和反馈
5. **测试**：使用 `MockInteractiveSession` 进行测试
