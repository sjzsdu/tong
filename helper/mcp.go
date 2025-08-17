package helper

import (
	"encoding/json"
	"fmt"
)

// 时间格式常量，用于时间格式化
const TimeLayout = "2006-01-02 15:04:05"

// ToJSON pretty prints any value as JSON string.
func ToJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}
