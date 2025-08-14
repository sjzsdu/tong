package rag

import (
	"context"
	"fmt"

	"github.com/sjzsdu/tong/cmdio"
	"github.com/sjzsdu/tong/lang"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

// Session 表示RAG会话
type Session struct {
	Chain   chains.Chain
	Options SessionOptions
}

// NewSession 创建新的会话
func NewSession(llm llms.Model, retriever schema.Retriever, options SessionOptions) *Session {
	// 创建基于检索的问答链
	chain := chains.NewRetrievalQAFromLLM(llm, retriever)
	chain.InputKey = "input"
	return &Session{
		Chain:   chain,
		Options: options,
	}
}

// Start 启动交互式会话
func (s *Session) Start(ctx context.Context) error {
	fmt.Println(lang.T("启动RAG会话，输入问题开始查询，输入'exit'或'quit'退出"))

	// 创建交互式会话适配器
	adapter := cmdio.CreateChatAdapter(s.Chain, s.Options.Stream)

	// 启动交互式会话
	return adapter.Start(ctx)
}

// Query 执行单次查询
func (s *Session) Query(ctx context.Context, query string) (string, error) {
	result, err := chains.Call(ctx, s.Chain, map[string]any{
		"input": query,
	})
	if err != nil {
		return "", &RagError{
			Code:    "query_failed",
			Message: "执行查询失败",
			Cause:   err,
		}
	}

	// 从结果中提取回答
	answer, ok := result["result"].(string)
	if !ok {
		return "", &RagError{
			Code:    "invalid_result",
			Message: "无效的查询结果",
		}
	}

	return answer, nil
}

// StreamingQuery 执行流式查询
func (s *Session) StreamingQuery(ctx context.Context, query string, callback func(string)) error {
	// 目前暂时没有直接的流式回调机制，将在未来版本实现
	// 临时解决方案：使用非流式查询，然后通过回调返回结果
	answer, err := s.Query(ctx, query)
	if err != nil {
		return err
	}

	// 调用回调函数返回结果
	callback(answer)
	return nil
}
