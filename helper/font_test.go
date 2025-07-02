package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindFont(t *testing.T) {
	fontPath, err := FindFont()
	
	// 在 MacOS 系统上应该能找到至少一个字体
	if assert.NoError(t, err) {
		assert.NotEmpty(t, fontPath)
		assert.Contains(t, []string{
			"/System/Library/Fonts/PingFang.ttc",
			"/Library/Fonts/Arial Unicode.ttf",
			"/System/Library/Fonts/STHeiti Light.ttc",
			"/System/Library/Fonts/STHeiti Medium.ttc",
		}, fontPath)
	}
}