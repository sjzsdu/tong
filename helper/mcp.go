package helper

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// TimeLayout is the common time format layout.
const TimeLayout = "2006-01-02 15:04:05"

// GetFloatDefault returns float value from args with default.
func GetFloatDefault(args map[string]any, key string, def float64) float64 {
	if args == nil {
		return def
	}
	if v, ok := args[key]; ok {
		switch vv := v.(type) {
		case float64:
			return vv
		case int:
			return float64(vv)
		case int32:
			return float64(vv)
		case int64:
			return float64(vv)
		case string:
			if vv == "" {
				return def
			}
			if iv, err := strconv.ParseFloat(vv, 64); err == nil {
				return iv
			}
		}
	}
	return def
}

// GetArgs extracts argument map from a CallToolRequest.
func GetArgs(req mcp.CallToolRequest) map[string]any {
	if req.Params.Arguments == nil {
		return nil
	}
	switch v := req.Params.Arguments.(type) {
	case map[string]interface{}:
		return v
	default:
		return nil
	}
}

// GetStringFromRequest 从请求中获取字符串参数，支持顶层和嵌套的args结构
func GetStringFromRequest(req mcp.CallToolRequest, key string, def string) (string, bool) {
	// 首先尝试从顶层获取
	if val, err := req.RequireString(key); err == nil && val != "" {
		return val, true
	}

	// 如果顶层没有，尝试从args中获取
	args := GetArgs(req)
	if args != nil {
		// 直接从args中查找
		if val, ok := args[key].(string); ok && val != "" {
			return val, true
		}

		// 检查是否有嵌套的args结构
		if nestedArgs, ok := args["args"].(map[string]interface{}); ok && nestedArgs != nil {
			if val, ok := nestedArgs[key].(string); ok && val != "" {
				return val, true
			}
		}
	}

	return def, false
}

// GetIntFromRequest 从请求中获取整数参数，支持顶层和嵌套的args结构
func GetIntFromRequest(req mcp.CallToolRequest, key string, def int) (int, bool) {
	// 尝试从顶层获取（先尝试获取字符串然后转换）
	if valStr, err := req.RequireString(key); err == nil && valStr != "" {
		if iv, err := atoiSafe(valStr); err == nil {
			return iv, true
		}
	}

	// 如果顶层没有，尝试从args中获取
	args := GetArgs(req)
	if args != nil {
		// 直接从args中查找
		if val, ok := args[key]; ok {
			switch v := val.(type) {
			case int:
				return v, true
			case int32:
				return int(v), true
			case int64:
				return int(v), true
			case float64:
				return int(v), true
			case string:
				if v != "" {
					if iv, err := atoiSafe(v); err == nil {
						return iv, true
					}
				}
			}
		}

		// 检查是否有嵌套的args结构
		if nestedArgs, ok := args["args"].(map[string]interface{}); ok && nestedArgs != nil {
			if val, ok := nestedArgs[key]; ok {
				switch v := val.(type) {
				case int:
					return v, true
				case int32:
					return int(v), true
				case int64:
					return int(v), true
				case float64:
					return int(v), true
				case string:
					if v != "" {
						if iv, err := atoiSafe(v); err == nil {
							return iv, true
						}
					}
				}
			}
		}
	}

	return def, false
}

// GetBoolFromRequest 从请求中获取布尔参数，支持顶层和嵌套的args结构
func GetBoolFromRequest(req mcp.CallToolRequest, key string, def bool) (bool, bool) {
	// 尝试从顶层获取（先尝试获取字符串然后转换）
	if valStr, err := req.RequireString(key); err == nil && valStr != "" {
		if strings.EqualFold(valStr, "true") {
			return true, true
		} else if strings.EqualFold(valStr, "false") {
			return false, true
		}
	}

	// 如果顶层没有，尝试从args中获取
	args := GetArgs(req)
	if args != nil {
		// 直接从args中查找
		if val, ok := args[key]; ok {
			switch v := val.(type) {
			case bool:
				return v, true
			case string:
				if strings.EqualFold(v, "true") {
					return true, true
				} else if strings.EqualFold(v, "false") {
					return false, true
				}
			case float64:
				return v != 0, true
			case int:
				return v != 0, true
			}
		}

		// 检查是否有嵌套的args结构
		if nestedArgs, ok := args["args"].(map[string]interface{}); ok && nestedArgs != nil {
			if val, ok := nestedArgs[key]; ok {
				switch v := val.(type) {
				case bool:
					return v, true
				case string:
					if strings.EqualFold(v, "true") {
						return true, true
					} else if strings.EqualFold(v, "false") {
						return false, true
					}
				case float64:
					return v != 0, true
				case int:
					return v != 0, true
				}
			}
		}
	}

	return def, false
}

// GetFloatFromRequest 从请求中获取浮点数参数，支持顶层和嵌套的args结构
func GetFloatFromRequest(req mcp.CallToolRequest, key string, def float64) (float64, bool) {
	// 尝试从顶层获取（先尝试获取字符串然后转换）
	if valStr, err := req.RequireString(key); err == nil && valStr != "" {
		if fv, err := strconv.ParseFloat(valStr, 64); err == nil {
			return fv, true
		}
	}

	// 如果顶层没有，尝试从args中获取
	args := GetArgs(req)
	if args != nil {
		// 直接从args中查找
		if val, ok := args[key]; ok {
			switch v := val.(type) {
			case float64:
				return v, true
			case int:
				return float64(v), true
			case int32:
				return float64(v), true
			case int64:
				return float64(v), true
			case string:
				if v != "" {
					if fv, err := strconv.ParseFloat(v, 64); err == nil {
						return fv, true
					}
				}
			}
		}

		// 检查是否有嵌套的args结构
		if nestedArgs, ok := args["args"].(map[string]interface{}); ok && nestedArgs != nil {
			if val, ok := nestedArgs[key]; ok {
				switch v := val.(type) {
				case float64:
					return v, true
				case int:
					return float64(v), true
				case int32:
					return float64(v), true
				case int64:
					return float64(v), true
				case string:
					if v != "" {
						if fv, err := strconv.ParseFloat(v, 64); err == nil {
							return fv, true
						}
					}
				}
			}
		}
	}

	return def, false
}

// ToJSON pretty prints any value as JSON string.
func ToJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

// GetStringDefault returns string value from args with default.
func GetStringDefault(args map[string]any, key, def string) string {
	if args == nil {
		return def
	}
	if v, ok := args[key]; ok {
		if s, ok2 := v.(string); ok2 {
			return s
		}
	}
	return def
}

// GetBoolDefault returns bool value from args with default.
func GetBoolDefault(args map[string]any, key string, def bool) bool {
	if args == nil {
		return def
	}
	if v, ok := args[key]; ok {
		switch vv := v.(type) {
		case bool:
			return vv
		case string:
			return strings.EqualFold(vv, "true")
		case float64:
			return vv != 0
		}
	}
	return def
}

// GetIntDefault returns int value from args with default.
func GetIntDefault(args map[string]any, key string, def int) int {
	if args == nil {
		return def
	}
	if v, ok := args[key]; ok {
		switch vv := v.(type) {
		case int:
			return vv
		case int32:
			return int(vv)
		case int64:
			return int(vv)
		case float64:
			return int(vv)
		case string:
			if vv == "" {
				return def
			}
			if iv, err := atoiSafe(vv); err == nil {
				return iv
			}
		}
	}
	return def
}

func atoiSafe(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty")
	}
	var n int
	neg := false
	if s[0] == '-' {
		neg = true
		s = s[1:]
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid digit: %c", c)
		}
		n = n*10 + int(c-'0')
	}
	if neg {
		n = -n
	}
	return n, nil
}
