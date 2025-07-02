package helper

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

func FindFont() (string, error) {
	fontPaths := []string{
		"/System/Library/Fonts/Songti.ttc", // 常用中文字体
		"/System/Library/Fonts/SimSun.ttf", // 常用中文字体
		"/System/Library/Fonts/SimHei.ttf", // 常用中文字体
		"/System/Library/Fonts/PingFang.ttc",
		"/Library/Fonts/Arial Unicode.ttf",
		"/System/Library/Fonts/STHeiti Light.ttc",
		"/System/Library/Fonts/STHeiti Medium.ttc",
	}

	for _, path := range fontPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no suitable font found")
}

//go:embed fonts/*.ttf
var embeddedFonts embed.FS

const FONT_NAME = "FangZhengFangSongJianTi-1"

func UseEmbeddedFont(fontName string) (string, error) {
	if fontName == "" {
		fontName = FONT_NAME
	}
	// 修正字体路径，需要包含 fonts 目录
	fontPath := fmt.Sprintf("fonts/%s.ttf", fontName)

	// 添加调试信息
	entries, err := embeddedFonts.ReadDir("fonts")
	if err != nil {
		return "", fmt.Errorf("无法读取字体目录: %v", err)
	}

	// 检查目录中的文件
	var foundFont bool
	for _, entry := range entries {
		fmt.Printf("Found font file: %s\n", entry.Name()) // 添加调试输出
		if entry.Name() == fmt.Sprintf("%s.ttf", fontName) {
			foundFont = true
			break
		}
	}

	if !foundFont {
		return "", fmt.Errorf("字体文件 %s.ttf 不存在于嵌入的目录中", fontName)
	}

	data, err := embeddedFonts.ReadFile(fontPath)
	if err != nil {
		return "", fmt.Errorf("读取嵌入字体失败: %v", err)
	}

	if len(data) == 0 {
		return "", fmt.Errorf("字体文件为空")
	}

	localFontPath := filepath.Join(os.TempDir(), fontName+".ttf")
	err = os.WriteFile(localFontPath, data, 0644)
	if err != nil {
		return "", fmt.Errorf("写入字体到临时目录失败: %v", err)
	}

	return localFontPath, nil
}
