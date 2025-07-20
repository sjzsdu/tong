package streamtest

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/sjzsdu/tong/cmdio"
	"github.com/sjzsdu/tong/helper/renders"
)

// TestStreamOutput 测试流式输出功能
func TestStreamOutput() {
	// 创建处理器配置
	config := cmdio.DefaultProcessorConfig()
	config.Mode = cmdio.StreamMode

	// 创建自定义流式处理函数
	processFunc := func(ctx context.Context, input string, writer io.Writer) error {
		// 模拟流式输出
		for i := 0; i < 10; i++ {
			output := fmt.Sprintf("流式输出块 %d\n", i)
			_, err := writer.Write([]byte(output))
			if err != nil {
				return err
			}
			time.Sleep(200 * time.Millisecond) // 模拟延迟
		}
		return nil
	}

	// 创建自定义流式处理器
	processor := cmdio.NewCustomStreamProcessor(config, processFunc)

	// 创建渲染器
	renderer := renders.NewTextRenderer()

	// 设置处理器输出写入器
	cmdio.SetProcessorWriter(processor, renderer)

	// 处理输入
	ctx := context.Background()
	err := processor.ProcessInput(ctx, "测试输入")
	if err != nil {
		log.Fatalf("处理输入错误: %v", err)
	}

	// 处理输出
	err = processor.ProcessOutput(ctx)
	if err != nil {
		log.Fatalf("处理输出错误: %v", err)
	}

	// 完成输出
	renderer.Done()

	fmt.Println("\n测试完成，如果你看到了逐步输出的文本，说明流式输出正常工作")
}