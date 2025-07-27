package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/sjzsdu/langchaingo-cn/llms"
	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/share"
)

// MCPServerConfig 单个 MCP 服务器的配置
type MCPServerConfig struct {
	Disabled      bool     `json:"disabled"`
	Timeout       int      `json:"timeout"`
	Command       string   `json:"command"`
	Args          []string `json:"args"`
	Env           []string `json:"env"`
	TransportType string   `json:"transportType"`
	Url           string   `json:"url,omitempty"`
	AutoApprove   []string `json:"autoApprove,omitempty"`
}

type LLMConfig struct {
	Type   llms.LLMType `json:"type"`
	Params map[string]interface{}
}

// MCPConfig MCP 配置文件结构
type SchemaConfig struct {
	MCPServers   map[string]MCPServerConfig `json:"mcpServers"`
	MasterLLM    LLMConfig                  `json:"masterLLM"`
	EmbeddingLLM LLMConfig                  `json:"embeddingLLM"`
}

// LoadMCPConfig 从指定目录加载 MCP 配置
// 如果配置文件不存在，则返回默认配置
// 如果同时指定了 dir 和 file，则会合并两个配置，file 的配置优先级更高
func LoadMCPConfig(dir string, file string) (*SchemaConfig, error) {
	// 获取默认配置
	config := DefaultSchemaConfig()

	// 如果没有指定目录和文件，直接返回默认配置
	if dir == "" && file == "" {
		return config, nil
	}

	// 尝试加载目录中的配置文件
	if dir != "" {
		dirConfigPath := filepath.Join(dir, share.SCHEMA_CONFIG_FILE)
		if _, err := os.Stat(dirConfigPath); err == nil {
			// 文件存在，读取并合并配置
			data, err := os.ReadFile(dirConfigPath)
			if err != nil {
				return config, err
			}

			var dirConfig SchemaConfig
			if err := json.Unmarshal(data, &dirConfig); err != nil {
				return config, err
			}

			// 合并配置
			MergeConfig(config, &dirConfig)
		}
	}

	// 如果指定了文件，则加载文件配置并合并（优先级更高）
	if file != "" {
		filePath, _ := helper.GetAbsPath(file)
		if _, err := os.Stat(filePath); err == nil {
			// 文件存在，读取并合并配置
			data, err := os.ReadFile(filePath)
			if err != nil {
				return config, err
			}

			var fileConfig SchemaConfig
			if err := json.Unmarshal(data, &fileConfig); err != nil {
				return config, err
			}

			// 合并配置，文件配置优先级更高
			MergeConfig(config, &fileConfig)
		}
	}

	return config, nil
}

// MergeConfig 合并两个配置，target 会被 source 中的非空值覆盖
func MergeConfig(target *SchemaConfig, source *SchemaConfig) {
	// 合并 MCPServers
	if source.MCPServers != nil {
		if target.MCPServers == nil {
			target.MCPServers = make(map[string]MCPServerConfig)
		}
		for name, serverConfig := range source.MCPServers {
			target.MCPServers[name] = serverConfig
		}
	}

	// 合并 MasterLLM
	if source.MasterLLM.Type != "" {
		target.MasterLLM = source.MasterLLM
	}

	// 合并 EmbeddingLLM
	if source.EmbeddingLLM.Type != "" {
		target.EmbeddingLLM = source.EmbeddingLLM
	}
}

// GetServerConfig 获取指定服务器的配置
func (c *SchemaConfig) GetServerConfig(name string) *MCPServerConfig {
	if c == nil {
		return nil
	}

	if config, exists := c.MCPServers[name]; exists && !config.Disabled {
		return &config
	}
	return nil
}

// DefaultSchemaConfig 生成默认的 SchemaConfig 配置
func DefaultSchemaConfig() *SchemaConfig {
	return &SchemaConfig{
		MCPServers: map[string]MCPServerConfig{},
		MasterLLM: LLMConfig{
			Type:   llms.LLMType(GetConfigWithDefault("MASTER_LLM", string(llms.DeepSeekLLM))),
			Params: map[string]interface{}{},
		},
		EmbeddingLLM: LLMConfig{
			Type:   llms.DeepSeekLLM,
			Params: map[string]interface{}{},
		},
	}
}
