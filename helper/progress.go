package helper

import (
	"fmt"
	"strings"
	"sync/atomic"
)

type Progress struct {
	total   int
	current int64
	width   int
	title   string
}

func NewProgress(title string, total int) *Progress {
	return &Progress{
		total: total,
		width: 50, // 进度条宽度
		title: title,
	}
}

func (p *Progress) Increment() {
	atomic.AddInt64(&p.current, 1)
	p.render()
}

// Show 显示当前进度，不增加计数
func (p *Progress) Show() {
	p.render()
}

func (p *Progress) render() {
	percent := float64(p.current) / float64(p.total) * 100
	filled := int(percent / 2) // Each "=" represents 2%

	// Ensure filled is never negative
	if filled < 0 {
		filled = 0
	}

	// Calculate remaining, ensuring it's never negative
	remaining := 50 - filled // Total width is 50
	if remaining < 0 {
		remaining = 0
	}

	bar := strings.Repeat("=", filled) + strings.Repeat(" ", remaining)
	fmt.Printf("\r%s [%s] %.1f%% (%d/%d)",
		p.title, bar, percent, p.current, p.total)
}
