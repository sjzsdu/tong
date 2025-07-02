package project

import (
	"encoding/json"

	"github.com/sjzsdu/tong/share"
)

// Function 表示函数信息
type Function struct {
	Name       string `json:"name"`
	Parameters string `json:"parameters"`
	ReturnType string `json:"return_type"`
	Feature    string `json:"feature"`
}

// Variable 表示变量信息
type Variable struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Feature string `json:"feature"`
}

// Method 表示方法信息
type Method struct {
	Name       string `json:"name"`
	Parameters string `json:"parameters"`
	ReturnType string `json:"return_type"`
	Feature    string `json:"feature"`
}

// Class 表示类信息
type Class struct {
	Name      string     `json:"name"`
	Feature   string     `json:"feature"`
	Variables []Variable `json:"variables"`
	Methods   []Method   `json:"methods"`
}

// Interface 表示接口信息
type Interface struct {
	Name    string   `json:"name"`
	Feature string   `json:"feature"`
	Methods []Method `json:"methods"`
}

// Symbol 表示其他符号信息
type Symbol struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Feature string `json:"feature"`
}

// LLMResponse 表示 LLM 响应的完整结构
type LLMResponse struct {
	Functions    []Function  `json:"functions"`
	Classes      []Class     `json:"classes"`
	Interfaces   []Interface `json:"interfaces"`
	Variables    []Variable  `json:"variables"`
	OtherSymbols []Symbol    `json:"other_symbols"`
	Feature      string      `json:"feature"`
}

// NewLLMResponse 从 JSON 字符串创建 LLMResponse
func NewLLMResponse(jsonStr string) (*LLMResponse, error) {
	var resp LLMResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ToJSON 将 LLMResponse 转换为 JSON 字符串
func (r *LLMResponse) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// NewNotProgramResponse 创建一个表示非程序文件的特殊响应
func NewNotProgramResponse() *LLMResponse {
	return &LLMResponse{
		Feature: share.NOT_PROGRAM_TIP,
	}
}

// IsNotProgramResponse 检查是否为非程序文件的响应
func (r *LLMResponse) IsNotProgramResponse() bool {
	return r != nil && r.Feature == share.NOT_PROGRAM_TIP
}
