package git

import (
	"fmt"
	"sort"
	"time"
)

// GetTopContributors 获取文件的主要贡献者
func GetTopContributors(blameInfo *BlameInfo, limit int) []struct {
	Author string
	Lines  int
} {
	if blameInfo == nil || len(blameInfo.Authors) == 0 {
		return nil
	}
	
	// 将作者贡献转换为切片以便排序
	contributors := make([]struct {
		Author string
		Lines  int
	}, 0, len(blameInfo.Authors))
	
	for author, lines := range blameInfo.Authors {
		contributors = append(contributors, struct {
		Author string
		Lines  int
	}{
		Author: author,
		Lines:  lines,
	})
	}
	
	// 按贡献行数排序
	sort.Slice(contributors, func(i, j int) bool {
		return contributors[i].Lines > contributors[j].Lines
	})
	
	// 限制返回数量
	if limit > 0 && limit < len(contributors) {
		return contributors[:limit]
	}
	return contributors
}

// GetFileAgeInfo 获取文件的年龄信息
func GetFileAgeInfo(blameInfo *BlameInfo) struct {
	OldestLine time.Time
	NewestLine time.Time
	AvgAge     time.Duration
} {
	if blameInfo == nil || len(blameInfo.Lines) == 0 {
		return struct {
		OldestLine time.Time
		NewestLine time.Time
		AvgAge     time.Duration
	}{}
	}
	
	var oldest, newest time.Time
	var totalAge time.Duration
	now := time.Now()
	
	for i, line := range blameInfo.Lines {
		if i == 0 || line.CommitTime.Before(oldest) {
			oldest = line.CommitTime
		}
		if i == 0 || line.CommitTime.After(newest) {
			newest = line.CommitTime
		}
		
		totalAge += now.Sub(line.CommitTime)
	}
	
	return struct {
		OldestLine time.Time
		NewestLine time.Time
		AvgAge     time.Duration
	}{
		OldestLine: oldest,
		NewestLine: newest,
		AvgAge:     totalAge / time.Duration(len(blameInfo.Lines)),
	}
}

// FormatBlameOutput 格式化blame输出为易读的字符串
func FormatBlameOutput(blameInfo *BlameInfo, showEmail bool) string {
	if blameInfo == nil || len(blameInfo.Lines) == 0 {
		return "没有找到blame信息"
	}
	
	result := fmt.Sprintf("文件: %s\n总行数: %d\n\n", blameInfo.FilePath, blameInfo.TotalLines)
	result += "行号\t作者\t日期\t\t内容\n"
	result += "----\t----\t----\t\t----\n"
	
	for _, line := range blameInfo.Lines {
		author := line.Author
		if showEmail && line.Email != "" {
			author += " <" + line.Email + ">"
		}
		
		result += fmt.Sprintf("%d\t%s\t%s\t%s\n", 
			line.LineNum, 
			author, 
			line.CommitTime.Format("2006-01-02"),
			line.Content)
	}
	
	return result
}