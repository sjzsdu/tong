package helper

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/c-bata/go-prompt"
	"github.com/sjzsdu/tong/lang"
)

func ShowLoadingAnimation(done chan bool) {
	spinChars := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	i := 0
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			fmt.Print("\n") // 清除当前行
			done <- false   // 发送 false 表示动画已清理完成
			return
		case <-ticker.C:
			fmt.Printf("\r%s %s... ", spinChars[i], lang.T("Thinking"))
			i = (i + 1) % len(spinChars)
		}
	}
}

func ReadFromTerminal(promptText string) (string, error) {
	var result string
	done := make(chan struct{})
	once := &sync.Once{}

	p := prompt.New(
		func(in string) {
			result = in
			once.Do(func() { close(done) })
		},
		func(d prompt.Document) []prompt.Suggest {
			return nil
		},
		prompt.OptionPrefix(""), // 移除默认提示符
		prompt.OptionTitle("tong"),
		prompt.OptionPrefixTextColor(prompt.Blue),
		prompt.OptionInputTextColor(prompt.DefaultColor),
		prompt.OptionAddKeyBind(
			prompt.KeyBind{
				Key: prompt.ControlV,
				Fn: func(b *prompt.Buffer) {
					result = "vim"
					once.Do(func() { close(done) })
				},
			},
			prompt.KeyBind{
				Key: prompt.ControlC,
				Fn: func(b *prompt.Buffer) {
					result = "quit"
					once.Do(func() { close(done) })
				},
			},
		),
		prompt.OptionSetExitCheckerOnInput(func(in string, breakline bool) bool {
			return breakline
		}),
	)

	// 手动输出提示符
	fmt.Print(promptText)

	go p.Run()
	<-done

	return result, nil
}

func ReadPipeContent() (string, error) {
	content, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	return StripAnsiCodes(string(content)), nil
}

func ReadFromVim() (string, error) {
	// 创建一个临时文件来存储输入
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, "vim_input_"+randomString(8)+".txt")

	// 确保在函数结束时删除临时文件
	defer os.Remove(tempFile)

	// 使用 Vim 编辑临时文件，+startinsert 参数让 vim 启动后直接进入插入模式
	cmd := exec.Command("vim", "+startinsert", tempFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error running Vim: %w", err)
	}

	// 读取临时文件的内容
	content, err := os.ReadFile(tempFile)
	if err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	// 处理用户输入的内容
	userInput := strings.TrimSpace(string(content))

	return userInput, nil
}

func InputString(promptText string) (string, error) {
	input, err := ReadFromTerminal(promptText)
	if err != nil {
		return "", fmt.Errorf("error reading input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("empty input")
	}

	if input == "vim" {
		input, err = ReadFromVim()
		if err != nil {
			return "", fmt.Errorf(lang.T("Error reading vim")+": %v\n", err)
		}
		fmt.Printf(">%s\n", input)
	}

	return input, nil
}
