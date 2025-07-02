package helper

import "strings"

// UpdateOperation 定义更新操作的结构体
type UpdateOperation struct {
	Operation string // 操作类型：insert, delete, replace, replaceAll
	Target    string // 源文档中的内容块
	Content   string // 新增或更新的内容
}

// ApplyChanges 利用更新数组完成对原来文档的更新
func ApplyChanges(blogContent string, changes []UpdateOperation) string {
	for _, change := range changes {
		// 统一处理空 Target 的情况
		if change.Target == "" {
			if blogContent != "" && !strings.HasSuffix(blogContent, "\n") {
				blogContent += "\n"
			}
			blogContent += change.Content
			continue
		}

		switch change.Operation {
		case "insert":
			blogContent = strings.Replace(blogContent, change.Target, change.Target+"\n"+change.Content, 1)
		case "delete":
			blogContent = strings.Replace(blogContent, change.Target, "", 1)
		case "replace":
			blogContent = strings.Replace(blogContent, change.Target, change.Content, 1)
		case "replaceAll":
			blogContent = strings.ReplaceAll(blogContent, change.Target, change.Content)
		}
	}
	return blogContent
}
