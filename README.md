# Tong

Tong是一个功能强大的开发工具，旨在帮助开发者分析、管理和优化他们的项目。该工具提供了多种功能，包括代码统计、依赖分析、代码质量评估、项目搜索和导出等。

## 功能特点

- **项目打包**：将项目文件打包导出为不同格式（如Markdown、PDF等）
- **代码统计**：分析项目的代码行数、文件数量、目录结构等统计信息
- **依赖分析**：分析项目的依赖关系，支持多种语言（Go、JavaScript、TypeScript、Python、Java等）
- **代码质量评估**：评估代码质量，识别潜在问题
- **项目搜索**：在项目中搜索关键词
- **多语言支持**：对多种编程语言提供支持：
  - Go
  - JavaScript
  - TypeScript
  - Python
  - Java
  - Gradle

## 安装

### 从源码安装

```bash
# 克隆仓库
git clone https://github.com/sjzsdu/tong.git

# 进入项目目录
cd tong

# 安装依赖并编译
go build -o tong main.go

# 将编译好的二进制文件移动到可执行路径下
sudo mv tong /usr/local/bin/
```

### 使用预编译二进制文件

从[发布页面](https://github.com/sjzsdu/tong/releases)下载适合您操作系统的预编译二进制文件，解压后放入系统路径中。

## 使用方法

### 项目打包

将项目文件打包导出为指定格式：

```bash
tong project pack -d /path/to/project -o output.md
```

选项：
- `-d, --dir`: 指定项目目录路径
- `-o, --output`: 指定输出文件路径
- `-e, --ext`: 指定要包含的文件扩展名（如 go,js,py）
- `--exclude`: 指定要排除的文件或目录模式
- `--skip-gitignore`: 忽略.gitignore文件中的排除规则

### 代码统计

分析项目的代码统计信息：

```bash
tong project code -d /path/to/project
```

### 依赖分析

分析项目的依赖关系：

```bash
tong project deps -d /path/to/project
```

生成依赖关系的可视化图（需要安装Graphviz）：

```bash
tong project deps -d /path/to/project -o deps.dot
dot -Tpng deps.dot -o dependency.png
```

### 代码质量评估

评估项目的代码质量：

```bash
tong project quality -d /path/to/project
```

### 项目搜索

在项目中搜索关键词：

```bash
tong project search -d /path/to/project "搜索关键词"
```

## 依赖可视化

Tong提供了强大的依赖关系可视化功能：

1. **按类型分组**：依赖按类型（直接依赖、开发依赖、导入依赖等）进行分组
2. **树形结构**：以树形结构展示依赖关系
3. **彩色输出**：使用不同颜色区分不同类型的依赖
4. **DOT格式导出**：支持导出为DOT格式，可使用Graphviz等工具生成图像

## 支持的文件类型

Tong支持分析多种文件类型的依赖关系：

- `.go`: Go源文件和模块文件
- `.js`: JavaScript源文件
- `.ts`/`.tsx`: TypeScript源文件
- `.json`: JSON文件（主要用于package.json分析）
- `.py`: Python源文件
- `.java`: Java源文件
- `.gradle`: Gradle构建文件

## 配置

Tong使用命令行参数进行配置，常用的全局参数包括：

- `-d, --dir`: 指定项目目录路径
- `-o, --output`: 指定输出文件路径
- `-e, --ext`: 指定要处理的文件扩展名
- `--exclude`: 指定要排除的文件或目录模式
- `--skip-gitignore`: 忽略.gitignore文件中的排除规则
- `--debug`: 启用调试模式，输出详细日志

## 贡献

欢迎贡献代码、报告问题或提出改进建议。请遵循以下步骤：

1. Fork 仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 许可证

本项目采用 [MIT 许可证](LICENSE)。