package helper

// 常见的程序文件扩展名
var ProgramFileExtensions = map[string]bool{
	"go":    true,
	"py":    true,
	"js":    true,
	"ts":    true,
	"jsx":   true,
	"tsx":   true,
	"java":  true,
	"cpp":   true,
	"c":     true,
	"h":     true,
	"hpp":   true,
	"rs":    true,
	"rb":    true,
	"php":   true,
	"swift": true,
	"kt":    true,
	"scala": true,
	"cs":    true,
	"vue":   true,
	"sh":    true,
	"pl":    true,
	"r":     true,
	"m":     true,
	"mm":    true,
	"lua":   true,
}

// IsProgramFile 判断是否是程序文件
func IsProgramFile(file string) bool {
	ext := GetFileExt(file)
	return ProgramFileExtensions[ext]
}
