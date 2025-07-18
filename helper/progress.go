package helper

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Progress struct {
	total       int
	current     int64
	width       int
	title       string
	startTime   time.Time
	mu          sync.Mutex
	finished    bool
	showETA     bool
	showPercent bool
}

// ProgressOption 进度条选项
type ProgressOption func(*Progress)

// WithETA 显示预计完成时间
func WithETA() ProgressOption {
	return func(p *Progress) {
		p.showETA = true
	}
}

// WithPercent 显示百分比
func WithPercent() ProgressOption {
	return func(p *Progress) {
		p.showPercent = true
	}
}

// WithWidth 设置进度条宽度
func WithWidth(width int) ProgressOption {
	return func(p *Progress) {
		p.width = width
	}
}

func NewProgress(title string, total int, opts ...ProgressOption) *Progress {
	p := &Progress{
		total:       total,
		width:       50, // 进度条宽度
		title:       title,
		startTime:   time.Now(),
		showETA:     true,
		showPercent: true,
	}

	// 应用选项
	for _, opt := range opts {
		opt(p)
	}

	return p
}

func (p *Progress) Increment() {
	atomic.AddInt64(&p.current, 1)
	p.render()
}

// Update 更新进度到指定值
func (p *Progress) Update(current int) {
	atomic.StoreInt64(&p.current, int64(current))
	p.render()
}

// Show 显示当前进度，不增加计数
func (p *Progress) Show() {
	p.render()
}

// Finish 完成进度条
func (p *Progress) Finish() {
	p.mu.Lock()
	defer p.mu.Unlock()

	atomic.StoreInt64(&p.current, int64(p.total))
	p.finished = true
	p.render()
	fmt.Println() // 换行
}

func (p *Progress) render() {
	if p.total == 0 {
		return
	}

	current := atomic.LoadInt64(&p.current)
	percent := float64(current) / float64(p.total) * 100
	if percent > 100 {
		percent = 100
	}

	filled := int(percent * float64(p.width) / 100)
	if filled > p.width {
		filled = p.width
	}

	// 计算剩余部分
	remaining := p.width - filled
	if remaining < 0 {
		remaining = 0
	}

	// 构建进度条
	bar := strings.Repeat("█", filled) + strings.Repeat("░", remaining)

	// 构建显示字符串
	var display strings.Builder
	display.WriteString(fmt.Sprintf("\r%s [%s]", p.title, bar))

	// 显示百分比
	if p.showPercent {
		display.WriteString(fmt.Sprintf(" %.1f%%", percent))
	}

	// 显示当前进度/总数
	display.WriteString(fmt.Sprintf(" (%d/%d)", current, p.total))

	// 显示ETA
	if p.showETA && current > 0 {
		elapsed := time.Since(p.startTime)
		if current < int64(p.total) {
			// 计算预计剩余时间
			rate := float64(current) / elapsed.Seconds()
			if rate > 0 {
				remaining := float64(p.total-int(current)) / rate
				eta := time.Duration(remaining) * time.Second
				display.WriteString(fmt.Sprintf(" ETA: %s", formatDuration(eta)))
			}
		} else {
			display.WriteString(fmt.Sprintf(" 用时: %s", formatDuration(elapsed)))
		}
	}

	// 输出进度条
	fmt.Print(display.String())
}

// formatDuration 格式化时间显示
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) - minutes*60
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	} else {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) - hours*60
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
}
