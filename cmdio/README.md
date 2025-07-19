# 通用命令行交互模块

这个模块提供了一套通用的命令行交互功能，可以快速构建各种类型的交互式命令行程序。

## 主要特性

- **输入处理**: 支持自定义输入预处理逻辑
- **输出处理**: 支持普通输出和流式输出
- **命令管理**: 支持特殊命令和退出命令的自定义
- **会话管理**: 统一的会话启动和生命周期管理
- **错误处理**: 完善的错误处理和上下文取消支持
- **可扩展性**: 基于接口设计，易于扩展和自定义

## 核心接口

### CommandProcessor 接口

所有命令处理器都需要实现这个接口：

```go
type CommandProcessor interface {
    ProcessCommand(ctx context.Context, input string) error
    GetPrompt() string
    IsExitCommand(input string) bool
    HandleSpecialCommand(ctx context.Context, input string) (bool, error)
}
```

### StreamProcessor 接口

用于流式处理：

```go
type StreamProcessor interface {
    ProcessStream(ctx context.Context, callback func(content string, done bool) error) error
}
```

## 内置处理器

### 1. InputOutputProcessor

最灵活的处理器，支持自定义输入和输出处理逻辑：

```go
processor := NewInputOutputProcessor().
    SetPrompt("myapp> ").
    SetInputHandler(func(input string) (string, error) {
        // 输入预处理
        return strings.ToUpper(input), nil
    }).
    SetOutputHandler(func(ctx context.Context, input string) (string, error) {
        // 处理命令并返回输出
        return fmt.Sprintf("处理结果: %s", input), nil
    }).
    AddSpecialCommand("help", func(ctx context.Context) error {
        fmt.Println("帮助信息")
        return nil
    })
```

### 2. SimpleCommandProcessor

基于命令的处理器，适合构建命令行工具：

```go
processor := NewSimpleCommandProcessor().
    AddCommand("greet", func(ctx context.Context, args string) error {
        fmt.Printf("Hello, %s!\n", args)
        return nil
    }).
    AddCommand("calc", func(ctx context.Context, args string) error {
        // 计算逻辑
        return nil
    })
```

### 3. 自定义处理器

可以实现 `CommandProcessor` 接口来创建完全自定义的处理器。

## 使用方法

### 基本使用

```go
// 1. 创建命令处理器
processor := NewInputOutputProcessor().
    SetOutputHandler(func(ctx context.Context, input string) (string, error) {
        return "Echo: " + input, nil
    })

// 2. 创建交互会话
session := NewInteractiveSession(processor, nil).
    SetWelcome("欢迎使用我的程序!").
    AddTip("输入 'quit' 退出")

// 3. 启动会话
ctx := context.Background()
session.Start(ctx)
```

### 流式处理

```go
processor := NewInputOutputProcessor().
    SetStreamHandler(func(ctx context.Context, input string, callback func(content string, done bool) error) error {
        words := strings.Fields(input)
        for _, word := range words {
            callback(word+" ", false) // 流式输出
            time.Sleep(100 * time.Millisecond)
        }
        return callback("", true) // 标记完成
    })
```

### 带加载动画的流式处理

```go
// 使用内置的带加载动画的流式处理
err := ProcessStreamWithLoading(ctx, streamProcessor, renderer)
```

## 配置选项

### InteractiveSession 配置

- `SetWelcome(string)`: 设置欢迎信息
- `AddTip(string)`: 添加提示信息
- `SetDebug(bool)`: 设置调试模式

### InputOutputProcessor 配置

- `SetPrompt(string)`: 设置命令提示符
- `SetInputHandler(func)`: 设置输入预处理器
- `SetOutputHandler(func)`: 设置输出处理器
- `SetStreamHandler(func)`: 设置流式处理器
- `AddSpecialCommand(string, func)`: 添加特殊命令
- `SetExitCommands([]string)`: 设置退出命令列表

## 错误处理

模块内置了完善的错误处理机制：

- 自动处理上下文取消 (`context.Canceled`)
- 超时处理 (`context.DeadlineExceeded`)
- 流式错误处理
- 用户友好的错误消息

## 示例

详细的使用示例请参考 `examples.go` 文件，包含：

1. 简单回显程序
2. 命令行工具
3. 流式处理器
4. 自定义处理器（数据管理工具）

## 扩展

### 自定义渲染器

实现 `renders.Renderer` 接口：

```go
type Renderer interface {
    WriteStream(content string) error
    Done()
}
```

### 自定义命令处理器

实现 `CommandProcessor` 接口来创建完全自定义的处理逻辑。

## 最佳实践

1. **输入验证**: 在 `InputHandler` 中进行输入验证和预处理
2. **错误处理**: 妥善处理命令执行中的错误
3. **上下文管理**: 正确使用 `context.Context` 进行取消和超时控制
4. **用户体验**: 提供清晰的帮助信息和错误提示
5. **性能考虑**: 对于长时间运行的操作使用流式处理

## 与原有代码的兼容性

这个通用模块完全兼容原有的 Chat 功能，原有的 `StartInteractiveSession` 方法仍然可以正常使用。新的通用功能可以独立使用，也可以与现有功能结合使用。
