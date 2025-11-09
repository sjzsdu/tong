package pack

// PackOptions 打包配置选项
type PackOptions struct {
	Formatter     Formatter
	IncludeExts   []string
	ExcludeExts   []string
	Recursive     bool
	IncludeHidden bool
	// IncludedFiles 打包过程中实际被包含的文件(相对路径)
	IncludedFiles []string
}

// DefaultOptions 返回默认的打包选项
func DefaultOptions() *PackOptions {
	return &PackOptions{
		Formatter:     &MarkdownFormatter{},
		Recursive:     true,
		IncludeHidden: false,
		IncludedFiles: []string{},
	}
}
