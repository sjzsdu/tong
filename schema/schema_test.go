package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sjzsdu/tong/share"
	"github.com/stretchr/testify/assert"
)

func TestDefaultSchemaConfig(t *testing.T) {
	// 获取默认配置
	defaultConfig := DefaultSchemaConfig()

	// 验证默认配置不为空
	assert.NotNil(t, defaultConfig)

	// 验证默认配置中的 MCPServers 不为空
	assert.NotNil(t, defaultConfig.MCPServers)

	// 验证 MasterLLM 和 EmbeddingLLM 配置不为空
	assert.NotEmpty(t, defaultConfig.MasterLLM.Type)
	assert.NotEmpty(t, defaultConfig.EmbeddingLLM.Type)
}

func TestLoadMCPConfig(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 测试无配置文件时返回默认配置
	t.Run("NoConfigFile", func(t *testing.T) {
		config, err := LoadMCPConfig(tmpDir, "")
		assert.NoError(t, err)
		assert.NotNil(t, config)
		assert.NotNil(t, config.MCPServers)
	})

	// 创建目录配置文件
	dirConfig := &SchemaConfig{
		MCPServers: map[string]MCPServerConfig{
			"dir_server": {
				Disabled:      false,
				Timeout:       120,
				Command:       "dir_command",
				TransportType: "stdio",
			},
		},
	}
	dirConfigPath := filepath.Join(tmpDir, share.SCHEMA_CONFIG_FILE)
	dirConfigData, _ := json.Marshal(dirConfig)
	err := os.WriteFile(dirConfigPath, dirConfigData, 0644)
	assert.NoError(t, err)

	// 测试加载目录配置
	t.Run("LoadDirConfig", func(t *testing.T) {
		config, err := LoadMCPConfig(tmpDir, "")
		assert.NoError(t, err)
		assert.NotNil(t, config)

		// 验证目录配置已加载
		dirServer, exists := config.MCPServers["dir_server"]
		assert.True(t, exists)
		assert.Equal(t, "dir_command", dirServer.Command)
		assert.Equal(t, 120, dirServer.Timeout)

		// 不再验证默认配置
	})

	// 创建文件配置
	fileConfig := &SchemaConfig{
		MCPServers: map[string]MCPServerConfig{
			"file_server": {
				Disabled:      false,
				Timeout:       180,
				Command:       "file_command",
				TransportType: "sse",
				Url:           "http://localhost:8080",
			},
			// 覆盖目录配置中的服务器
			"dir_server": {
				Disabled:      true,
				Timeout:       60,
				Command:       "overridden_command",
				TransportType: "stdio",
			},
		},
	}
	fileConfigPath := filepath.Join(tmpDir, "file_config.json")
	fileConfigData, _ := json.Marshal(fileConfig)
	err = os.WriteFile(fileConfigPath, fileConfigData, 0644)
	assert.NoError(t, err)

	// 测试同时加载目录和文件配置，文件配置优先级更高
	t.Run("LoadBothConfigs", func(t *testing.T) {
		config, err := LoadMCPConfig(tmpDir, fileConfigPath)
		assert.NoError(t, err)
		assert.NotNil(t, config)

		// 验证文件配置已加载
		fileServer, exists := config.MCPServers["file_server"]
		assert.True(t, exists)
		assert.Equal(t, "file_command", fileServer.Command)
		assert.Equal(t, 180, fileServer.Timeout)
		assert.Equal(t, "sse", fileServer.TransportType)
		assert.Equal(t, "http://localhost:8080", fileServer.Url)

		// 验证文件配置覆盖了目录配置
		dirServer, exists := config.MCPServers["dir_server"]
		assert.True(t, exists)
		assert.True(t, dirServer.Disabled) // 被文件配置覆盖为 true
		assert.Equal(t, "overridden_command", dirServer.Command)
		assert.Equal(t, 60, dirServer.Timeout)

		// 不再验证默认配置
	})
}

func TestMergeConfig(t *testing.T) {
	// 直接测试 MergeConfig 函数
	target := DefaultSchemaConfig()
	// 确保 target.MCPServers 已初始化
	if target.MCPServers == nil {
		target.MCPServers = make(map[string]MCPServerConfig)
	}
	// 添加一个测试服务器到 target
	target.MCPServers["test_server"] = MCPServerConfig{
		Disabled:      false,
		Timeout:       60,
		Command:       "test_command",
		TransportType: "stdio",
	}

	source := &SchemaConfig{
		MCPServers: map[string]MCPServerConfig{
			"new_server": {
				Disabled:      false,
				Timeout:       300,
				Command:       "new_command",
				TransportType: "sse",
				Url:           "http://localhost:9000",
			},
			"test_server": { // 覆盖测试服务器
				Disabled:      true,
				Timeout:       30,
				Command:       "overridden_test",
				TransportType: "stdio",
			},
		},
	}

	// 合并配置
	MergeConfig(target, source)

	// 验证新服务器已添加
	newServer, exists := target.MCPServers["new_server"]
	assert.True(t, exists)
	assert.Equal(t, "new_command", newServer.Command)
	assert.Equal(t, 300, newServer.Timeout)
	assert.Equal(t, "sse", newServer.TransportType)
	assert.Equal(t, "http://localhost:9000", newServer.Url)

	// 验证测试服务器已被覆盖
	testServer, exists := target.MCPServers["test_server"]
	assert.True(t, exists)
	assert.True(t, testServer.Disabled)
	assert.Equal(t, "overridden_test", testServer.Command)
	assert.Equal(t, 30, testServer.Timeout)
}
