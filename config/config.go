package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/share"
)

var configMap map[string]string

func init() {
	configMap = make(map[string]string)
	if err := LoadConfig(); err == nil {
		for key, value := range configMap {
			os.Setenv(key, value)
		}
	}
}

func GetConfig(key string) string {
	envKey := key
	if !strings.HasPrefix(key, share.PREFIX) {
		envKey = GetEnvKey(key)
	}
	return os.Getenv(envKey)
}

func LoadConfig() error {
	configFile := helper.GetPath("config")
	file, err := os.Open(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	// 清空现有配置
	configMap = make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			configMap[parts[0]] = parts[1]
			os.Setenv(parts[0], parts[1])
		}
	}
	return scanner.Err()
}

func SaveConfig() error {
	configDir := helper.GetPath("")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configFile := filepath.Join(configDir, "config")
	file, err := os.Create(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// 确保写入所有配置项
	for key, value := range configMap {
		if _, err := fmt.Fprintf(file, "%s=%s\n", key, value); err != nil {
			return err
		}
	}
	return file.Sync() // 确保数据写入磁盘
}

func GetEnvKey(flagKey string) string {
	return share.PREFIX + strings.ToUpper(flagKey)
}

// SetConfig 设置配置值并更新环境变量
func SetConfig(key, value string) {
	envKey := key
	if !strings.HasPrefix(key, share.PREFIX) {
		envKey = GetEnvKey(key)
	}
	configMap[envKey] = value
	os.Setenv(envKey, value)
}

// ClearConfig 清除指定配置
func ClearConfig(key string) {
	envKey := key
	if !strings.HasPrefix(key, share.PREFIX) {
		envKey = GetEnvKey(key)
	}
	delete(configMap, envKey)
	os.Unsetenv(envKey)
}

// ClearAllConfig 清除所有配置
func ClearAllConfig() {
	for key := range configMap {
		os.Unsetenv(key)
	}
	configMap = make(map[string]string)
}

// ClearAllConfig 清除所有配置
func GetConfigMap() map[string]string {
	return configMap
}
