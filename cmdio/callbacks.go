package cmdio

import (
	"context"
	"fmt"
	"strings"

	"github.com/sjzsdu/tong/helper"
	"github.com/sjzsdu/tong/share"
	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

type CallbackSession interface {
	ProcessStreaming(content string, done bool) error
}

// LogHandler is a callback handler that prints to the standard output.
type LogHandler struct {
	session CallbackSession
}

func NewCallbackHandler(session CallbackSession) callbacks.Handler {
	return &LogHandler{session: session}
}

func (l LogHandler) HandleLLMGenerateContentStart(_ context.Context, ms []llms.MessageContent) {
	if share.GetDebug() {
		helper.PrintWithLabel("HandleLLMGenerateContentStart", ms);
	}
}

func (l LogHandler) HandleLLMGenerateContentEnd(_ context.Context, res *llms.ContentResponse) {
	if share.GetDebug() {
		helper.PrintWithLabel("HandleLLMGenerateContentEnd", res);
	}
}

func (l LogHandler) HandleStreamingFunc(_ context.Context, chunk []byte) {
	if l.session != nil {
		content := string(chunk)
		done := false
		if content == "" {
			done = true
		}
		l.session.ProcessStreaming(content, done)
	}
}

func (l LogHandler) HandleText(_ context.Context, text string) {
	if share.GetDebug() {
		helper.PrintWithLabel("HandleText", text);
	}
}

func (l LogHandler) HandleLLMStart(_ context.Context, prompts []string) {
	if share.GetDebug() {
		helper.PrintWithLabel("HandleLLMStart", prompts);
	}
}

func (l LogHandler) HandleLLMError(_ context.Context, err error) {
	if share.GetDebug() {
		helper.PrintWithLabel("HandleLLMError", err);
	}
}

func (l LogHandler) HandleChainStart(_ context.Context, inputs map[string]any) {
	if share.GetDebug() {
		helper.PrintWithLabel("HandleChainStart", formatChainValues(inputs));
	}
}

func (l LogHandler) HandleChainEnd(_ context.Context, outputs map[string]any) {
	if share.GetDebug() {
		helper.PrintWithLabel("HandleChainEnd", formatChainValues(outputs));
	}
}

func (l LogHandler) HandleChainError(_ context.Context, err error) {
	if share.GetDebug() {
		helper.PrintWithLabel("HandleChainError", err);
	}
}

func (l LogHandler) HandleToolStart(_ context.Context, input string) {
	if share.GetDebug() {
		helper.PrintWithLabel("HandleToolStart", removeNewLines(input));
	}
}

func (l LogHandler) HandleToolEnd(_ context.Context, output string) {
	if share.GetDebug() {
		helper.PrintWithLabel("HandleToolEnd", removeNewLines(output));
	}
}

func (l LogHandler) HandleToolError(_ context.Context, err error) {
	if share.GetDebug() {
		helper.PrintWithLabel("HandleToolError", err);
	}
}

func (l LogHandler) HandleAgentAction(_ context.Context, action schema.AgentAction) {
	if share.GetDebug() {
		helper.PrintWithLabel("HandleAgentAction", formatAgentAction(action));
	}
}

func (l LogHandler) HandleAgentFinish(_ context.Context, finish schema.AgentFinish) {
	if share.GetDebug() {
		helper.PrintWithLabel("HandleAgentFinish", finish);
	}
}

func (l LogHandler) HandleRetrieverStart(_ context.Context, query string) {
	if share.GetDebug() {
		helper.PrintWithLabel("HandleRetrieverStart", removeNewLines(query));
	}
}

func (l LogHandler) HandleRetrieverEnd(_ context.Context, query string, documents []schema.Document) {
	if share.GetDebug() {
		helper.PrintWithLabel("HandleRetrieverEnd", documents, query);
	}
}

func formatChainValues(values map[string]any) string {
	output := ""
	for key, value := range values {
		output += fmt.Sprintf("\"%s\" : \"%s\", ", removeNewLines(key), removeNewLines(value))
	}

	return output
}

func formatAgentAction(action schema.AgentAction) string {
	return fmt.Sprintf("\"%s\" with input \"%s\"", removeNewLines(action.Tool), removeNewLines(action.ToolInput))
}

func removeNewLines(s any) string {
	return strings.ReplaceAll(fmt.Sprint(s), "\n", " ")
}
