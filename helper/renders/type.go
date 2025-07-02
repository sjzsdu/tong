package renders

// Renderer 定义了通用的渲染器接口
type Renderer interface {
	// 输出文字
	WriteStream(content string) error
	// 完成输出
	Done()
}
