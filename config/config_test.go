package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sjzsdu/tong/config"
)

func TestGetConfig(t *testing.T) {
	// 设置临时测试目录
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// 清除和设置测试环境变量
	os.Unsetenv("TONG_LANG")
	os.Unsetenv("TONG_DEEPSEEK_APIKEY")
	os.Unsetenv("TONG_MODEL")
	os.Unsetenv("MODEL") // 非前缀键

	// 设置测试环境变量
	os.Setenv("TONG_LANG", "zh")
	os.Setenv("TONG_DEEPSEEK_APIKEY", "test-key")
	os.Setenv("MODEL", "gpt-4") // 测试非前缀的直接环境变量

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "使用简短键获取语言",
			key:      "lang",
			expected: "zh",
		},
		{
			name:     "使用环境变量键获取语言",
			key:      "TONG_LANG",
			expected: "zh",
		},
		{
			name:     "使用简短键获取API密钥",
			key:      "deepseek_apikey",
			expected: "test-key",
		},
		{
			name:     "使用环境变量键获取API密钥",
			key:      "TONG_DEEPSEEK_APIKEY",
			expected: "test-key",
		},
		{
			name:     "获取不存在的配置",
			key:      "nonexistent",
			expected: "",
		},
		{
			name:     "直接获取非前缀环境变量",
			key:      "MODEL",
			expected: "gpt-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := config.GetConfig(tt.key); got != tt.expected {
				t.Errorf("GetConfig() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetConfigWithDefault(t *testing.T) {
	// 设置临时测试目录
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// 清除所有配置
	config.ClearAllConfig()

	// 设置测试环境变量
	os.Setenv("TONG_LANG", "zh")

	tests := []struct {
		name         string
		key          string
		defaultValue string
		expected     string
	}{
		{
			name:         "获取存在的配置",
			key:          "lang",
			defaultValue: "en",
			expected:     "zh",
		},
		{
			name:         "获取不存在的配置，返回默认值",
			key:          "nonexistent",
			defaultValue: "default-value",
			expected:     "default-value",
		},
		{
			name:         "获取空配置，返回默认值",
			key:          "empty",
			defaultValue: "default-for-empty",
			expected:     "default-for-empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := config.GetConfigWithDefault(tt.key, tt.defaultValue); got != tt.expected {
				t.Errorf("GetConfigWithDefault() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfigOperations(t *testing.T) {
	// 设置临时测试目录
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// 每次测试前清除所有配置
	config.ClearAllConfig()

	tests := []struct {
		name     string
		configs  map[string]string
		wantErr  bool
		validate func(t *testing.T)
	}{
		{
			name: "基本配置保存和加载",
			configs: map[string]string{
				"TONG_LANG":            "zh",
				"TONG_DEEPSEEK_APIKEY": "test-key",
			},
			wantErr: false,
			validate: func(t *testing.T) {
				// 重新加载配置
				if err := config.LoadConfig(); err != nil {
					t.Fatalf("加载配置失败: %v", err)
				}

				if v := config.GetConfig("lang"); v != "zh" {
					t.Errorf("lang 期望为 zh，实际为 %s", v)
				}
				if v := config.GetConfig("deepseek_apikey"); v != "test-key" {
					t.Errorf("deepseek_apikey 期望为 test-key，实际为 %s", v)
				}
			},
		},
		{
			name:    "空配置",
			configs: map[string]string{},
			wantErr: false,
			validate: func(t *testing.T) {
				if v := config.GetConfig("lang"); v != "" {
					t.Errorf("期望配置为空，实际为 %s", v)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清除所有配置
			config.ClearAllConfig()

			// 设置测试配置
			for k, v := range tt.configs {
				config.SetConfig(k, v)
			}

			// 测试保存
			if err := config.SaveConfig(); (err != nil) != tt.wantErr {
				t.Errorf("SaveConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			// 清除所有配置
			config.ClearAllConfig()

			// 测试加载
			if err := config.LoadConfig(); (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			// 验证结果
			tt.validate(t)

			// 验证文件内容
			if !tt.wantErr {
				content, err := os.ReadFile(filepath.Join(tmpDir, ".tong", "config"))
				if err != nil {
					t.Errorf("读取配置文件失败: %v", err)
				}
				if len(content) == 0 && len(tt.configs) > 0 {
					t.Error("配置文件不应为空")
				}
			}
		})
	}
}
