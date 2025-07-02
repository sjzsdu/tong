package helper

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringSliceContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "空切片",
			slice:    []string{},
			item:     "test",
			expected: false,
		},
		{
			name:     "包含目标字符串",
			slice:    []string{"test1", "test2", "test3"},
			item:     "test2",
			expected: true,
		},
		{
			name:     "不包含目标字符串",
			slice:    []string{"test1", "test2", "test3"},
			item:     "test4",
			expected: false,
		},
		{
			name:     "空字符串测试",
			slice:    []string{"test1", "", "test3"},
			item:     "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringSliceContains(tt.slice, tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"空字符串", 0},
		{"8位字符串", 8},
		{"16位字符串", 16},
		{"32位字符串", 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := randomString(tt.length)

			// 检查长度是否符合预期
			if len(got) != tt.length {
				t.Errorf("randomString() 长度 = %v, 期望长度 %v", len(got), tt.length)
			}

			// 检查字符是否都在允许的范围内
			pattern := "^[a-zA-Z0-9]*$"
			matched, _ := regexp.MatchString(pattern, got)
			if !matched {
				t.Errorf("randomString() = %v, 包含非法字符", got)
			}

			// 生成多个字符串检查是否有重复（当长度大于0时）
			if tt.length > 0 {
				results := make(map[string]bool)
				for i := 0; i < 100; i++ {
					str := randomString(tt.length)
					results[str] = true
				}
				// 检查是否生成了不同的字符串（允许少量重复）
				if len(results) < 90 {
					t.Errorf("randomString() 生成的随机字符串重复率过高")
				}
			}
		})
	}
}
