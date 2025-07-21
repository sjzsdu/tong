# Display 包

`display` 包提供了一系列用于在终端中展示可视化图表的工具函数。目前支持 ASCII 饼状图和折线图的生成。

## 功能

### 饼状图

`AuthorsPieChart` 函数用于生成作者贡献的 ASCII 饼状图，可以直观地展示各个作者的贡献占比。

```go
func AuthorsPieChart(authors map[string]int, totalLines int)
```

参数说明：
- `authors`: 作者名到贡献行数的映射
- `totalLines`: 总行数

### 折线图

`DatesLineChart` 函数用于生成日期贡献的折线图，展示代码行数随时间的变化趋势。

```go
func DatesLineChart(dates map[string]int)
```

参数说明：
- `dates`: 日期字符串到贡献行数的映射

## 使用示例

```go
import (
    "github.com/sjzsdu/tong/helper/display"
)

func main() {
    // 作者贡献数据
    authors := map[string]int{
        "张三": 100,
        "李四": 50,
        "王五": 30,
    }
    totalLines := 180
    
    // 生成作者贡献饼状图
    display.AuthorsPieChart(authors, totalLines)
    
    // 日期贡献数据
    dates := map[string]int{
        "2023-01-01": 10,
        "2023-01-02": 15,
        "2023-01-03": 25,
        "2023-01-04": 20,
    }
    
    // 生成日期贡献折线图
    display.DatesLineChart(dates)
}
```

## 依赖

- [asciigraph](https://github.com/guptarohit/asciigraph): 用于生成 ASCII 折线图
- [helper](https://github.com/sjzsdu/tong/helper): 提供颜色支持