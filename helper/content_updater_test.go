package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyChanges(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		changes  []UpdateOperation
		expected string
	}{
		{
			name:    "插入操作",
			content: "Hello World",
			changes: []UpdateOperation{
				{
					Operation: "insert",
					Target:    "Hello",
					Content:   "Beautiful",
				},
			},
			expected: "Hello\nBeautiful World",
		},
		{
			name:    "删除操作",
			content: "Hello World",
			changes: []UpdateOperation{
				{
					Operation: "delete",
					Target:    "World",
				},
			},
			expected: "Hello ",
		},
		{
			name:    "替换操作",
			content: "Hello World",
			changes: []UpdateOperation{
				{
					Operation: "replace",
					Target:    "World",
					Content:   "Golang",
				},
			},
			expected: "Hello Golang",
		},
		{
			name:    "全局替换操作",
			content: "Hello World, World!",
			changes: []UpdateOperation{
				{
					Operation: "replaceAll",
					Target:    "World",
					Content:   "Golang",
				},
			},
			expected: "Hello Golang, Golang!",
		},
		{
			name:    "多个操作组合",
			content: "Hello World",
			changes: []UpdateOperation{
				{
					Operation: "insert",
					Target:    "Hello",
					Content:   "Beautiful",
				},
				{
					Operation: "replace",
					Target:    "World",
					Content:   "Golang",
				},
			},
			expected: "Hello\nBeautiful Golang",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyChanges(tt.content, tt.changes)
			assert.Equal(t, tt.expected, result)
		})
	}
}
