package adapters

import (
	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/openai/openai-go"
)

// CastOpenAIToolCallsToKubechain converts OpenAI tool calls to TaskRun tool calls
func CastOpenAIToolCallsToKubechain(openaiToolCalls []openai.ChatCompletionMessageToolCall) []kubechainv1alpha1.ToolCall {
	var toolCalls []kubechainv1alpha1.ToolCall
	for _, tc := range openaiToolCalls {
		toolCall := kubechainv1alpha1.ToolCall{
			ID: tc.ID,
			Function: kubechainv1alpha1.ToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
			Type: string(tc.Type),
		}
		toolCalls = append(toolCalls, toolCall)
	}
	return toolCalls
}
