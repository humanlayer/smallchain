package llmclient

import (
	"context"
	"fmt"

	kubechain "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAIClient interface for mocking in tests
type OpenAIClient interface {
	SendRequest(ctx context.Context, systemPrompt string, userMessage string, tools []openai.ChatCompletionToolParam) (*openai.ChatCompletionMessage, error)
	SendContextWindow(ctx context.Context, contextWindow []kubechain.Message, tools []openai.ChatCompletionToolParam) (*openai.ChatCompletionMessage, error)
}

type realOpenAIClient struct {
	openai *openai.Client
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(apiKey string) (OpenAIClient, error) {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &realOpenAIClient{openai: client}, nil
}

func (c *realOpenAIClient) SendContextWindow(ctx context.Context, contextWindow []kubechain.Message, tools []openai.ChatCompletionToolParam) (*openai.ChatCompletionMessage, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *realOpenAIClient) SendRequest(ctx context.Context, systemPrompt string, userMessage string, tools []openai.ChatCompletionToolParam) (*openai.ChatCompletionMessage, error) {
	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userMessage),
		}),
		Model: openai.F(openai.ChatModelGPT4o),
		Tools: openai.F(tools),
	}

	// Only add tools if non-empty
	if len(tools) > 0 {
		params.Tools = openai.F(tools)
	}

	chatCompletion, err := c.openai.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat completion: %w", err)
	}

	if len(chatCompletion.Choices) == 0 {
		return nil, fmt.Errorf("no completion choices returned")
	}

	return &chatCompletion.Choices[0].Message, nil
}

// MockOpenAIClient for testing
type MockOpenAIClient struct {
	Response      *openai.ChatCompletionMessage
	Error         error
	ValidateTools func(tools []openai.ChatCompletionToolParam) error
}

func (m *MockOpenAIClient) SendRequest(ctx context.Context, systemPrompt string, userMessage string, tools []openai.ChatCompletionToolParam) (*openai.ChatCompletionMessage, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.ValidateTools != nil {
		if err := m.ValidateTools(tools); err != nil {
			return nil, err
		}
	}
	if m.Response == nil {
		return &openai.ChatCompletionMessage{
			Content: "Mock response",
		}, nil
	}
	return m.Response, nil
}
