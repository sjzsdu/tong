package helper

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShowLoadingAnimation(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
	}{
		{
			name:     "快速取消动画",
			duration: 100 * time.Millisecond,
		},
		{
			name:     "延迟取消动画",
			duration: 500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done := make(chan bool)
			go ShowLoadingAnimation(done)

			time.Sleep(tt.duration)
			done <- true
		})
	}
}

func TestReadFromVim(t *testing.T) {
	// 跳过实际执行 vim 的测试
	if os.Getenv("TEST_WITH_VIM") != "1" {
		t.Skip("跳过需要 vim 的测试。设置 TEST_WITH_VIM=1 来运行此测试")
	}

	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "基本测试",
			content: "test content",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建一个临时文件并写入测试内容
			tempDir := os.TempDir()
			tempFile := filepath.Join(tempDir, "vim_test_"+randomString(8)+".txt")
			err := os.WriteFile(tempFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("无法创建测试文件: %v", err)
			}
			defer os.Remove(tempFile)

			// 模拟 vim 的行为
			os.Setenv("EDITOR", "echo") // 使用 echo 替代 vim 用于测试

			got, err := ReadFromVim()
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadFromVim() 错误 = %v, 期望错误 %v", err, tt.wantErr)
				return
			}

			// 由于我们使用了 echo 替代 vim，这里主要测试函数是否正常运行
			// 实际内容验证可能需要手动测试或更复杂的模拟
			if err == nil && got == "" {
				t.Error("ReadFromVim() 返回空字符串")
			}
		})
	}
}
