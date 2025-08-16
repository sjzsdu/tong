package config_test

import (
	"os"
	"testing"

	"github.com/sjzsdu/tong/config"
)

// TestSetConfig 测试SetConfig函数是否正确处理不同格式的键
func TestSetConfig(t *testing.T) {
	// 设置临时测试目录
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// 清除所有配置
	config.ClearAllConfig()

	tests := []struct {
		name     string
		key      string
		value    string
		checkKey string
	}{
		{
			name:     "使用简短键设置配置",
			key:      "model",
			value:    "gpt-4",
			checkKey: "TONG_MODEL",
		},
		{
			name:     "使用环境变量键设置配置",
			key:      "TONG_LANG",
			value:    "zh",
			checkKey: "TONG_LANG",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置配置
			config.SetConfig(tt.key, tt.value)

			// 检查环境变量是否被正确设置
			if got := os.Getenv(tt.checkKey); got != tt.value {
				t.Errorf("SetConfig() 环境变量 = %v, want %v", got, tt.value)
			}

			// 检查GetConfig是否返回正确的值
			if got := config.GetConfig(tt.key); got != tt.value {
				t.Errorf("GetConfig(%s) = %v, want %v", tt.key, got, tt.value)
			}

			// 检查用环境变量名获取也能正确返回
			if got := config.GetConfig(tt.checkKey); got != tt.value {
				t.Errorf("GetConfig(%s) = %v, want %v", tt.checkKey, got, tt.value)
			}
		})
	}
}

// TestClearConfig 测试ClearConfig函数是否正确处理不同格式的键
func TestClearConfig(t *testing.T) {
	// 设置临时测试目录
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// 清除所有配置
	config.ClearAllConfig()

	// 预先设置一些配置
	config.SetConfig("model", "gpt-4")
	config.SetConfig("TONG_LANG", "zh")

	tests := []struct {
		name     string
		key      string
		checkKey string
	}{
		{
			name:     "使用简短键清除配置",
			key:      "model",
			checkKey: "TONG_MODEL",
		},
		{
			name:     "使用环境变量键清除配置",
			key:      "TONG_LANG",
			checkKey: "TONG_LANG",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 确认配置存在
			if got := os.Getenv(tt.checkKey); got == "" {
				t.Fatalf("测试前配置不存在: %s", tt.checkKey)
			}

			// 清除配置
			config.ClearConfig(tt.key)

			// 检查环境变量是否被正确清除
			if got := os.Getenv(tt.checkKey); got != "" {
				t.Errorf("ClearConfig() 后环境变量仍存在 = %v", got)
			}

			// 检查GetConfig是否返回空值
			if got := config.GetConfig(tt.key); got != "" {
				t.Errorf("ClearConfig() 后 GetConfig(%s) = %v, want \"\"", tt.key, got)
			}

			// 检查用环境变量名获取也返回空值
			if got := config.GetConfig(tt.checkKey); got != "" {
				t.Errorf("ClearConfig() 后 GetConfig(%s) = %v, want \"\"", tt.checkKey, got)
			}
		})
	}
}
