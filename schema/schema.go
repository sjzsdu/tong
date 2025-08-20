package schema

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/sjzsdu/langchaingo-cn/llms"
	"github.com/sjzsdu/tong/config"
	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/share"
)

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

	// 合并 Rag（整体覆盖，按 dir 与 file 的优先级）
	if !isZeroRagConfig(source.Rag) {
		target.Rag = source.Rag
	}
}

// 判断 RagConfig 是否为零值（用于决定是否覆盖）
func isZeroRagConfig(r RagConfig) bool {
	if r.Storage.URL != "" || r.Storage.Collection != "" {
		return false
	}
	if r.Splitter.ChunkSize > 0 || r.Splitter.ChunkOverlap > 0 {
		return false
	}
	if r.Retriever.TopK > 0 || r.Retriever.ScoreThreshold > 0 {
		return false
	}
	if r.Session.Stream != nil || r.Session.MaxHistory > 0 {
		return false
	}
	if r.DocsDir != "" {
		return false
	}
	if r.Sync.ForceReindex || r.Sync.SyncIntervalSeconds > 0 {
		return false
	}
	return true
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

// ToJSON 将配置序列化为 JSON 并写入指定路径
func (c *SchemaConfig) ToJSON(filePath string) error {
	// 将配置转换为 JSON
	configJSON, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	// 写入文件
	return os.WriteFile(filePath, configJSON, 0644)
}

// DefaultSchemaConfig 生成默认的 SchemaConfig 配置
func DefaultSchemaConfig() *SchemaConfig {
	// 默认 RAG 配置
	rc := RagConfig{}
	rc.Storage.URL = share.RAG_VECTOR_URL
	rc.Storage.Collection = share.RAG_COLLECTION
	rc.Splitter.ChunkSize = 1000
	rc.Splitter.ChunkOverlap = 200
	rc.Retriever.TopK = 4
	streamDefault := true
	rc.Session.Stream = &streamDefault
	// DocsDir 留空，运行时取项目根或命令行
	rc.Sync.SyncIntervalSeconds = 300

	return &SchemaConfig{
		MCPServers: map[string]MCPServerConfig{
			"tong": {
				Command:       "tong",
				Args:          []string{"mcp", "server", "--transport", "stdio"},
				TransportType: "stdio",
			},
		},
		MasterLLM: LLMConfig{
			Type:   llms.LLMType(config.GetConfigWithDefault("MASTER_LLM", string(llms.DeepSeekLLM))),
			Params: map[string]interface{}{},
		},
		EmbeddingLLM: EmbeddingConfig{
			Type:   llms.EmbeddingType(config.GetConfigWithDefault("EMBEDDING_LLM", string(llms.OllamaEmbedding))),
			Params: map[string]interface{}{},
		},
		Rag: rc,
	}
}
