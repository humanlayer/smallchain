package llmclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

// OpenAIClient interface for mocking in tests
type RawOpenAIClient interface {
	SendRequest(ctx context.Context, messages []v1alpha1.Message, tools []Tool) (*v1alpha1.Message, error)
}

type rawOpenAIClient struct {
	apiKey string
}

// Message represents a chat message with snake_case JSON fields
type OpenAIMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// ToolCall represents a request to call a tool with snake_case JSON fields
type ToolCall struct {
	ID       string           `json:"id"`
	Function ToolCallFunction `json:"function"`
	Type     string           `json:"type"`
}

// ToolCallFunction contains the function details with snake_case JSON fields
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolFunctionParameter struct {
	Type string `json:"type"`
}

type ToolFunctionParameters struct {
	Type       string                           `json:"type"`
	Properties map[string]ToolFunctionParameter `json:"properties"`
	Required   []string                         `json:"required"`
}

type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  ToolFunctionParameters `json:"parameters"`
}

type Tool struct {
	// Type indicates the type of tool. Currently only "function" is supported.
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

func FromKubechainTool(tool v1alpha1.Tool) *Tool {
	// Create a new Tool with function type
	clientTool := &Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        tool.Spec.Name,
			Description: tool.Spec.Description,
		},
	}

	// Parse the parameters if they exist
	if tool.Spec.Parameters.Raw != nil {
		var params ToolFunctionParameters
		if err := json.Unmarshal(tool.Spec.Parameters.Raw, &params); err != nil {
			return nil
		}
		clientTool.Function.Parameters = params
	}

	return clientTool
}

func FromKubechainMessages(messages []v1alpha1.Message) []OpenAIMessage {
	openaiMessages := make([]OpenAIMessage, len(messages))
	for i, message := range messages {
		openaiMessages[i] = *FromKubechainMessage(message)
	}
	return openaiMessages
}

func FromKubechainMessage(message v1alpha1.Message) *OpenAIMessage {
	openaiMessage := &OpenAIMessage{
		Role:       message.Role,
		Content:    message.Content,
		Name:       message.Name,
		ToolCallID: message.ToolCallID,
	}

	for _, toolCall := range message.ToolCalls {
		toolCall := ToolCall{
			ID:       toolCall.ID,
			Type:     toolCall.Type,
			Function: ToolCallFunction{Name: toolCall.Function.Name, Arguments: toolCall.Function.Arguments},
		}
		openaiMessage.ToolCalls = append(openaiMessage.ToolCalls, toolCall)
	}

	return openaiMessage
}

type chatCompletionRequest struct {
	Model    string          `json:"model"`
	Messages []OpenAIMessage `json:"messages"`
	Tools    []Tool          `json:"tools,omitempty"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message OpenAIMessage `json:"message"`
	} `json:"choices"`
}

// NewRawOpenAIClient creates a new OpenAI client
func NewRawOpenAIClient(apiKey string) (RawOpenAIClient, error) {
	return &rawOpenAIClient{apiKey: apiKey}, nil
}

func (c *rawOpenAIClient) SendRequest(ctx context.Context, messages []v1alpha1.Message, tools []Tool) (*v1alpha1.Message, error) {
	reqBody := chatCompletionRequest{
		Model:    "gpt-4",
		Messages: FromKubechainMessages(messages),
	}
	if len(tools) > 0 {
		reqBody.Tools = tools
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var completion chatCompletionResponse
	if err := json.Unmarshal(body, &completion); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(completion.Choices) == 0 {
		return nil, fmt.Errorf("no completion choices returned")
	}

	return FromOpenAIMessage(completion.Choices[0].Message), nil
}

func FromOpenAIMessage(openaiMessage OpenAIMessage) *v1alpha1.Message {
	message := &v1alpha1.Message{
		Role:       openaiMessage.Role,
		Content:    openaiMessage.Content,
		Name:       openaiMessage.Name,
		ToolCallID: openaiMessage.ToolCallID,
	}

	for _, toolCall := range openaiMessage.ToolCalls {
		toolCall := v1alpha1.ToolCall{
			ID:       toolCall.ID,
			Type:     toolCall.Type,
			Function: v1alpha1.ToolCallFunction{Name: toolCall.Function.Name, Arguments: toolCall.Function.Arguments},
		}
		message.ToolCalls = append(message.ToolCalls, toolCall)
	}

	return message
}

type MockRawOpenAIClient struct {
	Response *v1alpha1.Message
	Error    error
	Calls    []chatCompletionRequest
}

func (m *MockRawOpenAIClient) SendRequest(ctx context.Context, messages []v1alpha1.Message, tools []Tool) (*v1alpha1.Message, error) {
	m.Calls = append(m.Calls, chatCompletionRequest{
		Messages: FromKubechainMessages(messages),
		Tools:    tools,
	})
	return m.Response, m.Error
}
