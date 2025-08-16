package config

// ConfigKeyInfo 存储配置键的相关信息
type ConfigKeyInfo struct {
	Description string   // 配置项描述
	Options     []string // 可选值，如果为空则表示没有限制
	Type        string   // 配置项类型，默认为 "string"，可以是 "json", "secret" 等
}

// 配置键常量定义
const (
	KeyLang                 = "lang"
	KeyRenderer             = "renderer"
	KeyMasterLLM            = "master_llm"
	KeyMasterLLMParams      = "master_llm_params"
	KeyEmbeddingLLM         = "embedding_llm"
	KeyEmbeddingLLMParams   = "embedding_llm_params"
	KeySearchEngines        = "search_engines"
	KeyGoogleAPIKey         = "google_api_key"
	KeyGoogleSearchEngineID = "google_search_engine_id"
	KeyBingAPIKey           = "bing_api_key"
)

// ConfigKeys 存储所有配置键及其信息
var ConfigKeys = map[string]ConfigKeyInfo{
	KeyLang: {
		Description: "Set language",
		Options:     []string{"en", "zh-CN", "zh-TW"},
		Type:        "string",
	},
	KeyRenderer: {
		Description: "Set llm response render type",
		Options:     []string{"text", "markdown"},
		Type:        "string",
	},
	KeyMasterLLM: {
		Description: "Set master llm",
		Options:     []string{"deepseek", "qwen", "kimi"},
		Type:        "string",
	},
	KeyMasterLLMParams: {
		Description: "Set master llm params",
		Options:     []string{},
		Type:        "json",
	},
	KeyEmbeddingLLM: {
		Description: "Set embedding model",
		Options:     []string{"deepseek", "qwen", "kimi"},
		Type:        "string",
	},
	KeyEmbeddingLLMParams: {
		Description: "Set embedding model params",
		Options:     []string{},
		Type:        "json",
	},
	KeySearchEngines: {
		Description: "Set search engines priority order (comma-separated list)",
		Options:     []string{},
		Type:        "csv",
	},
	KeyGoogleAPIKey: {
		Description: "Set Google API key for web search",
		Options:     []string{},
		Type:        "secret",
	},
	KeyGoogleSearchEngineID: {
		Description: "Set Google Search Engine ID for web search",
		Options:     []string{},
		Type:        "string",
	},
	KeyBingAPIKey: {
		Description: "Set Bing API key for web search",
		Options:     []string{},
		Type:        "secret",
	},
}

// GetConfigDescription 获取配置键的描述
func GetConfigDescription(key string) string {
	if info, exists := ConfigKeys[key]; exists {
		return info.Description
	}
	return ""
}

// GetConfigOptions 获取配置键的可选值
func GetConfigOptions(key string) []string {
	if info, exists := ConfigKeys[key]; exists {
		return info.Options
	}
	return nil
}

// GetConfigType 获取配置键的类型
func GetConfigType(key string) string {
	if info, exists := ConfigKeys[key]; exists {
		return info.Type
	}
	return "string" // 默认类型为字符串
}

// IsValidConfigOption 检查给定的值是否是配置键的有效选项
func IsValidConfigOption(key, value string) bool {
	options := GetConfigOptions(key)
	if len(options) == 0 {
		// 如果没有定义选项，则认为所有值都有效
		return true
	}

	// 检查配置项类型
	configType := GetConfigType(key)

	// 对于CSV类型的配置项，验证每个逗号分隔的值
	if configType == "csv" {
		// 允许 CSV 类型的空选项
		return true
	}

	// 对于普通类型，验证值是否在选项列表中
	for _, option := range options {
		if option == value {
			return true
		}
	}
	return false
}

// GetAllConfigKeys 获取所有配置键
func GetAllConfigKeys() []string {
	keys := make([]string, 0, len(ConfigKeys))
	for key := range ConfigKeys {
		keys = append(keys, key)
	}
	return keys
}
