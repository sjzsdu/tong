package display

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/guptarohit/asciigraph"
	"github.com/sjzsdu/tong/helper"
)

// AuthorsPieChart 生成作者贡献的饼状图
func AuthorsPieChart(authors map[string]int, totalLines int) {
	// 对作者按贡献排序
	type authorContribution struct {
		Name  string
		Lines int
	}
	
	contributions := make([]authorContribution, 0, len(authors))
	for author, lines := range authors {
		contributions = append(contributions, authorContribution{Name: author, Lines: lines})
	}
	
	sort.Slice(contributions, func(i, j int) bool {
		return contributions[i].Lines > contributions[j].Lines
	})
	
	// 限制显示的作者数量，太多会导致图表混乱
	maxAuthors := 5
	if len(contributions) > maxAuthors {
		// 合并其他作者
		otherLines := 0
		for i := maxAuthors; i < len(contributions); i++ {
			otherLines += contributions[i].Lines
		}
		
		if otherLines > 0 {
			contributions = append(contributions[:maxAuthors], authorContribution{Name: "其他", Lines: otherLines})
		} else {
			contributions = contributions[:maxAuthors]
		}
	}
	
	// 生成饼状图数据
	fmt.Println("\n作者贡献饼状图:")
	
	// 计算每个作者的百分比并显示
	for _, c := range contributions {
		percentage := float64(c.Lines) / float64(totalLines) * 100
		barLength := int(percentage / 2) // 每2%显示一个字符
		bar := strings.Repeat("█", barLength)
		fmt.Printf("%s: %s %.2f%% (%d行)\n", c.Name, helper.ColorText(bar, getColorForAuthor(c.Name)), percentage, c.Lines)
	}
}

// DatesLineChart 生成日期贡献的折线图
func DatesLineChart(dates map[string]int) {
	// 解析日期并排序
	type dateContribution struct {
		Date  time.Time
		Lines int
	}
	
	contributions := make([]dateContribution, 0, len(dates))
	for dateStr, lines := range dates {
		// 尝试解析日期，格式可能是 "2006-01-02"
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			// 如果解析失败，尝试其他可能的格式
			date, err = time.Parse("2006/01/02", dateStr)
			if err != nil {
				// 如果仍然失败，跳过这个日期
				continue
			}
		}
		
		contributions = append(contributions, dateContribution{Date: date, Lines: lines})
	}
	
	// 按日期排序
	sort.Slice(contributions, func(i, j int) bool {
		return contributions[i].Date.Before(contributions[j].Date)
	})
	
	// 如果没有足够的数据点，不显示图表
	if len(contributions) < 2 {
		return
	}
	
	// 准备折线图数据
	data := make([]float64, len(contributions))
	dateLabels := make([]string, len(contributions))
	
	for i, c := range contributions {
		data[i] = float64(c.Lines)
		dateLabels[i] = c.Date.Format("01-02") // 月-日格式
	}
	
	// 生成折线图
	fmt.Println("\n日期贡献折线图:")
	graph := asciigraph.Plot(data, 
		asciigraph.Height(10),
		asciigraph.Width(60),
		asciigraph.Caption("代码行数随时间变化"))
	
	fmt.Println(graph)
	
	// 显示日期标签
	fmt.Println("日期参考:")
	for i, date := range dateLabels {
		fmt.Printf("%d: %s  ", i+1, date)
		if (i+1) % 5 == 0 {
			fmt.Println()
		}
	}
	fmt.Println()
}

// getColorForAuthor 根据作者名返回一个颜色
func getColorForAuthor(author string) string {
	// 简单的哈希算法，根据作者名的字符和来选择颜色
	colors := []string{
		helper.ColorRed,
		helper.ColorGreen,
		helper.ColorYellow,
		helper.ColorBlue,
		helper.ColorPurple,
		helper.ColorCyan,
	}
	
	sum := 0
	for _, c := range author {
		sum += int(c)
	}
	
	return colors[sum%len(colors)]
}