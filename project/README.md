# Project 包设计文档

## 概述

Project 包是一个用于管理项目文件结构的核心组件，它提供了一个内存中的文件系统抽象，支持与实际文件系统的双向同步。该包设计用于高效地管理和操作项目文件，同时保持内存表示与磁盘状态的一致性。经过优化，该包现在具有更好的并发安全性和性能表现。

## 架构设计

### 核心数据结构

#### Node

`Node` 是表示文件系统中的文件或目录的基本单元：

```go
type Node struct {
    Name          string            // 节点名称
    Path          string            // 节点在项目中的相对路径
    IsDir         bool              // 是否为目录
    modified      bool              // 是否已修改
    Info          os.FileInfo       // 文件系统信息
    Content       []byte            // 文件内容
    ContentLoaded bool              // 标记内容是否已加载
    Children      map[string]*Node  // 子节点映射
    Parent        *Node             // 父节点引用
    mu            sync.RWMutex      // 读写锁
}
```

#### Project

`Project` 表示整个项目的文档树：

```go
type Project struct {
    root     *Node                // 根节点
    rootPath string               // 项目根路径
    inGit    bool                 // 是否在 Git 仓库中
    nodes    map[string]*Node     // 路径到节点的映射
    mu       sync.RWMutex         // 读写锁
}
```

#### 访问者模式支持

```go
type VisitorFunc func(path string, node *Node, depth int) error
```

## 设计原则

1. **线程安全**：所有操作都使用读写锁保护，支持高并发访问
2. **延迟加载**：文件内容只在需要时才从磁盘加载到内存
3. **双向同步**：支持内存结构与文件系统的双向同步
4. **路径标准化**：统一使用 Unix 风格的路径分隔符
5. **内存优化**：支持内容卸载以节省内存使用
6. **死锁预防**：优化锁策略，避免嵌套锁导致的死锁问题

## 主要功能模块

### 文件操作

- **创建文件和目录**：`CreateFile`, `CreateDir`, `CreateFileNode`
- **读写文件**：`ReadFile`, `WriteFile`
- **删除节点**：`DeleteNode`

### 路径处理

- **路径标准化**：`helper.StandardizePath`, `NormalizePath`
- **路径解析**：`resolvePath`, `GetNodePath`
- **绝对路径转换**：`GetAbsolutePath`

### 节点查找和遍历

- **查找节点**：`FindNode`, `findNodeDirect`
- **列出文件**：`ListFiles`
- **遍历节点**：`Visit`, `VisitAll`

### 文件系统同步

- **保存到文件系统**：`SaveToFS`
- **从文件系统同步**：`SyncFromFS`
- **内容加载和卸载**：`LoadFileContent`, `UnloadFileContent`

## 工作流程

### 项目初始化

1. 创建 `Project` 实例，指定根目录路径
2. 初始化根节点和节点映射表
3. 可选：从文件系统同步项目结构

### 文件操作流程

1. **创建文件/目录**：
   - 标准化路径
   - 检查节点是否已存在
   - 解析路径，获取父节点
   - 创建节点对象
   - 在文件系统中创建实际文件/目录
   - 获取文件信息并更新节点
   - 将节点添加到项目中

2. **读取文件**：
   - 查找节点
   - 检查内容是否已加载
   - 如需要，从文件系统加载内容
   - 返回文件内容

3. **写入文件**：
   - 查找节点，不存在则创建
   - 更新节点内容
   - 写入文件系统
   - 更新文件信息

## 性能考虑

1. **内存优化**：
   - 文件内容延迟加载
   - 支持卸载不需要的文件内容

2. **并发控制**：
   - 细粒度锁控制，减少锁竞争
   - 读写锁分离，提高并发读取性能

3. **缓存优化**：
   - 使用节点映射表加速路径查找
   - 避免重复计算路径

## 使用示例

```go
// 创建项目实例
proj := NewProject("/path/to/project")

// 从文件系统同步
err := proj.SyncFromFS()
if err != nil {
    log.Fatal(err)
}

// 创建目录
err = proj.CreateDir("/src/models")
if err != nil {
    log.Fatal(err)
}

// 创建文件
content := []byte("package models\n\ntype User struct {\n    ID   int\n    Name string\n}\n")
err = proj.CreateFile("/src/models/user.go", content)
if err != nil {
    log.Fatal(err)
}

// 读取文件
data, err := proj.ReadFile("/src/models/user.go")
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(data))

// 保存到文件系统
err = proj.SaveToFS()
if err != nil {
    log.Fatal(err)
}
```

## 扩展性

1. **访问者模式**：支持自定义访问者函数遍历项目结构
2. **选项模式**：使用函数选项模式配置项目行为
3. **构建器模式**：提供 `BuildProjectTree` 函数简化项目树构建

## 注意事项

1. 路径处理需要注意跨平台兼容性
2. 文件操作可能因权限问题失败
3. 大文件处理需要注意内存使用
4. 并发修改同一节点可能导致竞争条件

## 未来改进

1. 支持文件监视和变更通知
2. 增加文件元数据缓存
3. 支持更复杂的文件过滤和模式匹配
4. 优化大型项目的内存使用
5. 增加事务支持，确保操作的原子性