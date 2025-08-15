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
