# Git Blame 分析工具

这个包提供了一个高效、灵活的 Git blame 分析工具，用于分析 Git 仓库中的代码贡献情况。它支持分析单个文件、目录或整个项目，并提供丰富的统计信息和格式化输出。

## 功能特点

- **多种实现**：支持两种实现方式 - 基于go-git库和基于命令行git命令
- **统一接口**：通过 `Blame` 方法提供统一的分析入口，自动识别文件、目录和项目
- **并发分析**：使用 Go 的并发特性，支持高效的并行分析
- **文件过滤**：内置文件过滤器，可自定义排除不需要分析的文件类型
- **丰富统计**：提供作者贡献、时间分布等多维度统计信息
- **格式化输出**：支持多种格式的分析结果输出
- **错误处理**：健壮的错误处理机制，确保分析过程不会因单个文件错误而中断

## 使用方法

### 创建Blamer实例

#### 使用工厂模式创建Blamer

```go
// 创建项目实例
projectPath := "/path/to/your/git/project"
p := project.NewProject(projectPath)

// 获取可用的blame分析器类型
blamerTypes := git.GetAvailableBlamerTypes()
fmt.Println("可用的blame分析器类型:")
for _, t := range blamerTypes {
    fmt.Printf("- %s: %s\n", t, git.GetBlamerTypeDescription(t))
}

// 创建基于go-git库的blame分析器
libraryBlamer, err := git.NewBlamer(p, git.LibraryBlamer)
if err != nil {
    fmt.Printf("创建基于库的blame分析器失败: %v\n", err)
    return
}

// 或者创建基于命令行git的blame分析器
cmdBlamer, err := git.NewBlamer(p, git.CommandBlamer)
if err != nil {
    fmt.Printf("创建基于命令行的blame分析器失败: %v\n", err)
    return
}
```

#### 直接创建特定类型的Blamer

```go
// 创建项目实例
projectPath := "/path/to/your/git/project"
p := project.NewProject(projectPath)

// 创建GitBlamer实例（基于go-git库）
blamer, err := git.NewGitBlamer(p)
if err != nil {
    fmt.Printf("创建GitBlamer失败: %v\n", err)
    return
}

// 或者创建CmdGitBlamer实例（基于命令行git）
cmdBlamer, err := git.NewCmdGitBlamer(p)
if err != nil {
    fmt.Printf("创建CmdGitBlamer失败: %v\n", err)
    return
}
```

### 分析单个文件

```go
// 分析单个文件
filePath := "main.go" // 相对于项目根目录的路径
blameInfo, err := blamer.Blame(filePath)
if err != nil {
    fmt.Printf("分析文件失败: %v\n", err)
    return
}

// 打印文件的blame信息
fmt.Printf("文件: %s\n", blameInfo.FilePath)
fmt.Printf("总行数: %d\n", blameInfo.TotalLines)
```

### 分析目录

```go
// 分析目录
dirPath := "cmd" // 相对于项目根目录的路径
blameInfo, err := blamer.Blame(dirPath)
if err != nil {
    fmt.Printf("分析目录失败: %v\n", err)
    return
}

// 获取目录下所有文件的详细分析结果
blameResults, err := blamer.BlameDirectory(p, dirPath)
if err != nil {
    fmt.Printf("获取详细分析结果失败: %v\n", err)
    return
}

// 打印目录分析结果摘要
fmt.Printf("目录: %s\n", dirPath)
fmt.Printf("总行数: %d\n", blameInfo.TotalLines)
fmt.Printf("分析的文件数: %d\n", len(blameResults))
```

### 分析整个项目

```go
// 创建GitBlamer实例
blamer, err := git.NewGitBlamer(p)
if err != nil {
    fmt.Printf("创建GitBlamer失败: %v\n", err)
    return
}
// 自定义文件过滤器，只分析Go文件
blamer.FileFilter = func(path string) bool {
    return filepath.Ext(path) == ".go"
}

// 分析整个项目
blameInfo, err := blamer.Blame("/")
if err != nil {
    fmt.Printf("分析项目失败: %v\n", err)
    return
}

// 获取项目所有文件的详细分析结果
blameResults, err := blamer.BlameProject(p)
if err != nil {
    fmt.Printf("获取详细分析结果失败: %v\n", err)
    return
}

// 打印项目分析结果摘要
fmt.Printf("项目: %s\n", p.GetName())
fmt.Printf("总行数: %d\n", blameInfo.TotalLines)
fmt.Printf("分析的文件数: %d\n", len(blameResults))
```

### 获取主要贡献者

```go
// 获取并打印主要贡献者
topContributors := git.GetTopContributors(blameInfo, 5)
fmt.Println("\n主要贡献者:")
for i, contributor := range topContributors {
    fmt.Printf("%d. %s: %d行 (%.2f%%)\n", 
        i+1, 
        contributor.Author, 
        contributor.Lines, 
        float64(contributor.Lines)/float64(blameInfo.TotalLines)*100)
}
```

### 获取文件年龄信息

```go
// 获取并打印文件年龄信息
ageInfo := git.GetFileAgeInfo(blameInfo)
fmt.Println("\n文件年龄信息:")
fmt.Printf("最早修改: %s\n", ageInfo.OldestLine.Format("2006-01-02"))
fmt.Printf("最近修改: %s\n", ageInfo.NewestLine.Format("2006-01-02"))
fmt.Printf("平均年龄: %.2f天\n", ageInfo.AvgAge.Hours()/24)
```

### 格式化blame输出

```go
// 打印详细的blame输出
fmt.Println("\n详细blame信息:")
fmt.Println(git.FormatBlameOutput(blameInfo, true))
```

## 数据结构

### BlameInfo

```go
type BlameInfo struct {
    Lines      []LineInfo     // 每一行的详细信息
    Authors    map[string]int // 作者贡献的行数统计
    Dates      map[string]int // 按日期统计的修改行数
    TotalLines int            // 总行数
    FilePath   string         // 文件路径
}
```

### LineInfo

```go
type LineInfo struct {
    LineNum    int       // 行号
    Author     string    // 作者
    Email      string    // 邮箱
    CommitID   string    // 提交ID
    CommitTime time.Time // 提交时间
    Content    string    // 行内容
}
```

## 高级用法

### 自定义文件过滤器

```go
// 自定义文件过滤器，只分析Go文件
blamer.FileFilter = func(path string) bool {
    return filepath.Ext(path) == ".go"
}
```

### 使用自定义的GitBlameVisitor

```go
// 创建自定义的GitBlameVisitor
visitor := git.NewGitBlameVisitor(blamer, p, 20) // 使用20个并发

// 创建遍历器
traverser := project.NewTreeTraverser(p)
traverser.SetTraverseOrder(project.PreOrder)

// 设置遍历选项，遇到错误时继续
traverser.SetOption(&project.TraverseOption{
    ContinueOnError: true,
    Errors:          make([]error, 0),
})

// 获取目录节点
node, err := p.FindNode(dirPath)
if err != nil {
    fmt.Printf("找不到目录: %v\n", err)
    return
}

// 开始遍历
err = traverser.Traverse(node, dirPath, 0, visitor)
if err != nil {
    fmt.Printf("遍历过程中发生错误: %v\n", err)
}

// 等待所有并发任务完成
visitor.WaitGroup.Wait()

// 使用结果
fmt.Printf("分析的文件数: %d\n", len(visitor.Results))
var totalLines int
for path, info := range visitor.Results {
    totalLines += info.TotalLines
    // 处理每个文件的blame信息
}
fmt.Printf("总行数: %d\n", totalLines)
```

## 实现方式比较

本工具提供了两种不同的实现方式，各有优缺点：

### 基于go-git库的实现 (LibraryBlamer)

- **优点**：
  - 无需外部依赖，可在任何环境运行
  - 与项目代码完全集成，无需额外安装
  - 可以在没有git命令的环境中工作

- **缺点**：
  - 对于大型仓库可能性能较低
  - 内存占用可能较高
  - 某些复杂的git操作支持有限

### 基于命令行git的实现 (CommandBlamer)

- **优点**：
  - 性能通常更好，特别是对于大型仓库
  - 内存占用较低
  - 支持所有git命令行支持的功能和选项

- **缺点**：
  - 依赖系统安装的git命令
  - 在某些环境中可能不可用
  - 需要处理命令行输出解析的复杂性

### 选择建议

- 如果你的应用需要在各种环境中运行，且不能保证git命令可用，选择 **LibraryBlamer**
- 如果你的应用主要在有git命令的环境中运行，且需要处理大型仓库，选择 **CommandBlamer**
- 如果性能是关键因素，选择 **CommandBlamer**
- 如果便携性和无依赖是关键因素，选择 **LibraryBlamer**

## 设计思路

本工具采用了以下设计思路：

1. **统一接口**：通过 `Blame` 方法提供统一的分析入口，根据路径类型自动选择合适的分析方法
2. **工厂模式**：使用 `NewBlamer` 工厂函数创建不同实现的Blamer实例，隐藏实现细节
3. **访问者模式**：使用 `GitBlameVisitor` 实现 `NodeVisitor` 接口，与 `TreeTraverser` 配合进行高效遍历
4. **并发处理**：利用 Go 的并发特性，通过信号量控制并发数量，平衡性能和资源使用
5. **错误处理**：采用错误收集而非中断的策略，确保分析过程的稳定性
6. **可扩展性**：通过接口和自定义过滤器，支持灵活的扩展和定制

## 示例

完整的使用示例请参考 `example.go` 文件，其中包含：

- `ExampleBlameFile`：演示如何分析单个文件
- `ExampleBlameDirectory`：演示如何分析目录
- `ExampleBlameProject`：演示如何分析整个项目
- `ExampleCustomBlameVisitor`：演示如何使用自定义的GitBlameVisitor

## 注意事项

- 确保项目路径是一个有效的Git仓库
- 大型项目的分析可能需要较长时间，建议使用并发分析
- 二进制文件和非文本文件默认会被过滤，可以通过自定义 `FileFilter` 修改过滤规则