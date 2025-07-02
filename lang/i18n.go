package lang

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var (
	bundle *i18n.Bundle
	loc    *i18n.Localizer
	// LocalePath 用于配置语言文件的路径
	LocalePath = "lang/locales"
)

// Init initializes the i18n system
func init() {
	SetupI18n("")
}

// SetupI18n 设置国际化配置并初始化语言
func SetupI18n(localePath string) {
	if localePath != "" {
		LocalePath = localePath
	}

	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	// 获取环境变量设置的语言
	lang := os.Getenv("TONG_LANG")
	if lang == "" {
		return
	}

	// 标准化语言代码
	switch lang {
	case "zh":
		lang = "zh-CN"
	case "cn":
		lang = "zh-CN"
	case "tw":
		lang = "zh-TW"
	}

	// 检查对应的语言文件是否存在
	langFile := filepath.Join(LocalePath, lang+".json")
	if _, err := os.Stat(langFile); err == nil {
		bundle.MustLoadMessageFile(langFile)
		loc = i18n.NewLocalizer(bundle, lang)
	}
}

// T translates a message, optionally with template data
func T(msgID string, data ...map[string]interface{}) string {
	// 如果未初始化 localizer，直接返回原始键
	if loc == nil {
		return msgID
	}

	config := &i18n.LocalizeConfig{
		MessageID: msgID,
	}

	// 如果提供了模板数据，则添加到配置中
	if len(data) > 0 {
		config.TemplateData = data[0]
	}

	msg, err := loc.Localize(config)
	if err != nil {
		// 如果翻译出错（比如键不存在），返回原始键
		return msgID
	}
	return msg
}
