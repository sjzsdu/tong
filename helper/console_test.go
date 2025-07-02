package helper

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestPrintWithLabel(t *testing.T) {
	tests := []struct {
		name     string
		label    string
		input    interface{}
		expected string
	}{
		{
			name:     "打印字符串",
			label:    "测试",
			input:    "Hello World",
			expected: "[测试]: Hello World\n",
		},
		{
			name:     "打印整数",
			label:    "数字",
			input:    42,
			expected: "[数字]: 42\n",
		},
		{
			name:  "打印结构体",
			label: "用户信息",
			input: struct {
				Name string
				Age  int
			}{Name: "张三", Age: 25},
			expected: `[用户信息]: {
  "Name": "张三",
  "Age": 25
}
`,
		},
		{
			name:  "打印map",
			label: "配置",
			input: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
			},
			expected: `[配置]: {
  "key1": "value1",
  "key2": 123
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 使用bytes.Buffer捕获输出
			var buf bytes.Buffer
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			PrintWithLabel(tt.label, tt.input)

			w.Close()
			os.Stdout = old
			io.Copy(&buf, r)

			got := buf.String()
			if got != tt.expected {
				t.Errorf("PrintWithLabel() = %q, want %q", got, tt.expected)
			}
		})
	}
}
