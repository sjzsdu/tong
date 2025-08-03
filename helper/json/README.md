# JSONStore 包

JSONStore 是一个用于管理 JSON 格式配置文件的工具包，提供了简单易用的 API 来进行 JSON 文件的增删改查操作。

## 主要功能

- 创建和管理 JSON 文件
- 读取和写入 JSON 数据
- 支持嵌套路径访问 JSON 数据 (如 "settings.theme")
- 类型安全的获取方法

## 快速开始

### 创建 JSONStore

```go
// 创建一个存储在 ~/.tong/configs 目录中的 JSONStore
store, err := json.NewJSONStore("configs")
if err != nil {
    log.Fatalf("创建 JSONStore 失败: %v", err)
}
```

### 基本操作

```go
// 保存数据
err := store.Set("app_config", map[string]interface{}{
    "name": "My App",
    "version": "1.0.0",
    "settings": map[string]interface{}{
        "theme": "dark",
        "fontSize": 14,
    },
})

// 获取数据
var config map[string]interface{}
modified, err := store.Get("app_config", &config)
fmt.Printf("配置: %v, 最后修改时间: %v\n", config, modified)

// 检查文件是否存在
if store.Exists("app_config") {
    fmt.Println("配置文件存在")
}

// 更新数据
err = store.Update("app_config", func(data map[string]interface{}) error {
    data["version"] = "1.0.1"
    return nil
})

// 删除文件
err = store.Delete("app_config")
```

### 类型安全的获取方法

```go
// 获取字符串值，如果不存在则使用默认值
name := store.GetString("app_config", "name", "Unknown App")

// 获取整数值
fontSize := store.GetInt("app_config", "settings.fontSize", 12)

// 获取布尔值
darkMode := store.GetBool("app_config", "settings.darkMode", false)
```

### 使用嵌套路径

```go
// 设置嵌套路径的值
err := store.SetValue("settings", "notifications.sound", "chime")
err = store.SetValue("settings", "display.theme", "dark")

// 获取嵌套路径的值
theme := store.GetString("settings", "display.theme", "light")

// 删除嵌套路径的值
err = store.DeleteKey("settings", "notifications.sound")
```
## 高级用法

### 搜索匹配的文件

```go
// 搜索所有以 "user_" 开头的配置文件
files, err := store.Search("user_")
if err != nil {
    log.Fatalf("搜索文件失败: %v", err)
}

for _, file := range files {
    fmt.Println(file) // 例如: user_prefs, user_history 等
}
```

### 列出所有文件

```go
// 列出所有配置文件
files, err := store.List()
if err != nil {
    log.Fatalf("列出文件失败: %v", err)
}

for _, file := range files {
    fmt.Println(file)
}
```

## 注意事项

1. 所有文件名不应包含扩展名，JSONStore 会自动添加 `.json` 扩展名
2. 路径分隔符使用点号 (.) 表示嵌套层级
3. 监控功能需要安装 `github.com/fsnotify/fsnotify` 依赖
