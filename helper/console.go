package helper

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// 颜色常量
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[37m"
	ColorWhite  = "\033[97m"

	ColorRedBold    = "\033[1;31m"
	ColorGreenBold  = "\033[1;32m"
	ColorYellowBold = "\033[1;33m"
	ColorBlueBold   = "\033[1;34m"
	ColorPurpleBold = "\033[1;35m"
	ColorCyanBold   = "\033[1;36m"
	ColorWhiteBold  = "\033[1;37m"
)

// ColorText 给文本添加颜色
func ColorText(text, color string) string {
	return color + text + ColorReset
}

// PrintColorText 打印彩色文本
func PrintColorText(text, color string) {
	fmt.Println(ColorText(text, color))
}

// PrintWithLabel 带标签的打印，方便调试时识别输出内容
func PrintWithLabel(label string, v ...interface{}) {
	fmt.Printf("[%s]: ", label)
	if len(v) == 0 {
		fmt.Println("nil")
		return
	}

	if len(v) == 1 {
		Print(v[0])
		return
	}

	// 处理多个参数
	fmt.Print("[ ")
	for i, item := range v {
		if i > 0 {
			fmt.Print(", ")
		}
		Print(item)
	}
	fmt.Println(" ]")
}

func Print(v interface{}) {
	switch reflect.TypeOf(v).Kind() {
	case reflect.Map, reflect.Slice, reflect.Struct, reflect.Ptr:
		formatted, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			fmt.Printf("格式化输出失败: %v\n", err)
			return
		}
		fmt.Print(string(formatted))
		fmt.Println()
	default:
		fmt.Println(v)
	}
}

// Printf 支持格式化字符串的打印
func Printf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

// Println 换行打印
func Println(v ...interface{}) {
	fmt.Println(v...)
}
