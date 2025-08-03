package prompt

const DEFAULT_AGENT = "coder"

// 添加语言映射表
var languageMap = map[string]string{
	"zh":      "你需要用中文语言回复。",
	"cn":      "你需要用中文语言回复。",
	"zh-CN":   "你需要用中文语言回复。",
	"en":      "Please respond in English.",
	"english": "Please respond in English.",
	"jp":      "日本語で返信してください。",
	"ja":      "日本語で返信してください。",
	"kr":      "한국어로 응답해 주세요.",
	"ko":      "한국어로 응답해 주세요.",
	"fr":      "Veuillez répondre en français.",
	"de":      "Bitte antworten Sie auf Deutsch.",
	"es":      "Por favor, responda en español.",
	"ru":      "Пожалуйста, ответьте на русском языке.",
}
