package helper

import (
	"encoding/json"
	"fmt"
	"reflect"
)

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
