package schema

import "github.com/sjzsdu/langchaingo-cn/llms"

// MCPServerConfig 单个 MCP 服务器的配置
type MCPServerConfig struct {
	Disabled      bool     `json:"disabled,omitempty"`
	Timeout       int      `json:"timeout,omitempty"`
	Command       string   `json:"command,omitempty"`
	Args          []string `json:"args,omitempty"`
	Env           []string `json:"env,omitempty"`
	TransportType string   `json:"transportType"`
	Url           string   `json:"url,omitempty"`
	AutoApprove   []string `json:"autoApprove,omitempty"`

	// OAuth 相关配置
	ClientID     string   `json:"clientId,omitempty"`
	ClientSecret string   `json:"clientSecret,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
}

type LLMConfig struct {
	Type   llms.LLMType           `json:"type"`
	Params map[string]interface{} `json:"params"`
}

type EmbeddingConfig struct {
	Type   llms.EmbeddingType     `json:"type"`
	Params map[string]interface{} `json:"params"`
}

// RagConfig 定义 RAG 相关的可配置项（对应 tong.json 的 rag 节）
type RagConfig struct {
	Storage struct {
		URL        string `json:"url,omitempty"`
		Collection string `json:"collection,omitempty"`
	} `json:"storage,omitempty"`
	Splitter struct {
		ChunkSize    int `json:"chunkSize,omitempty"`
		ChunkOverlap int `json:"chunkOverlap,omitempty"`
	} `json:"splitter,omitempty"`
	Retriever struct {
		TopK           int     `json:"topK,omitempty"`
		ScoreThreshold float32 `json:"scoreThreshold,omitempty"`
	} `json:"retriever,omitempty"`
	Session struct {
		Stream     *bool `json:"stream,omitempty"`
		MaxHistory int   `json:"maxHistory,omitempty"`
	} `json:"session,omitempty"`
	DocsDir string `json:"docsDir,omitempty"`
	Sync    struct {
		ForceReindex        bool `json:"forceReindex,omitempty"`
		SyncIntervalSeconds int  `json:"syncIntervalSec,omitempty"`
	} `json:"sync,omitempty"`
}

// MCPConfig MCP 配置文件结构
type SchemaConfig struct {
	MCPServers   map[string]MCPServerConfig `json:"mcpServers"`
	MasterLLM    LLMConfig                  `json:"masterLLM"`
	EmbeddingLLM EmbeddingConfig            `json:"embeddingLLM"`
	Rag          RagConfig                  `json:"rag,omitempty"`
	Agent        AgentConfig                `json:"agent,omitempty"`
}

type AgentConfig struct {
	LLMConfig LLMConfig `json:"llmConfig,omitempty"`
}
