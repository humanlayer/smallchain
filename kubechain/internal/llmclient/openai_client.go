package llmclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// LLMRequestError represents an error that occurred during an LLM request
// and includes HTTP status code information
type LLMRequestError struct {
	StatusCode int
	Message    string
	Err        error
}

func (e *LLMRequestError) Error() string {
	return fmt.Sprintf("LLM request failed with status %d: %s", e.StatusCode, e.Message)
}

func (e *LLMRequestError) Unwrap() error {
	return e.Err
}

// OpenAIClient interface for mocking in tests
type OpenAIClient interface {
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
	// KubechainToolType represents the Kubechain-specific type of tool (Standard, MCP, HumanContact)
	// This field is not sent to the LLM API but is used internally for tool identification
	KubechainToolType v1alpha1.ToolType `json:"-"`
}

func FromKubechainTool(tool v1alpha1.Tool) *Tool {
	// Create a new Tool with function type
	clientTool := &Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        tool.Spec.Name,
			Description: tool.Spec.Description,
		},
		KubechainToolType: v1alpha1.ToolTypeStandard, // Standard tool by default
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

// FromContactChannel creates a Tool from a ContactChannel resource
func FromContactChannel(channel v1alpha1.ContactChannel) *Tool {
	// Create base parameters structure for human contact tools
	params := ToolFunctionParameters{
		Type: "object",
		Properties: map[string]ToolFunctionParameter{
			"message": {Type: "string"},
		},
		Required: []string{"message"},
	}

	var description string
	var name string

	// Customize based on channel type
	switch channel.Spec.Type {
	case v1alpha1.ContactChannelTypeEmail:
		name = fmt.Sprintf("human_contact_email_%s", channel.Name)
		description = channel.Spec.Email.ContextAboutUser

	case v1alpha1.ContactChannelTypeSlack:
		name = fmt.Sprintf("human_contact_slack_%s", channel.Name)
		description = channel.Spec.Slack.ContextAboutChannelOrUser

	default:
		name = fmt.Sprintf("human_contact_%s", channel.Name)
		description = fmt.Sprintf("Contact a human via %s channel", channel.Spec.Type)
	}

	// Create the Tool
	return &Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        name,
			Description: description,
			Parameters:  params,
		},
		KubechainToolType: v1alpha1.ToolTypeHumanContact, // Set as HumanContact type
	}
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
		ToolCallID: message.ToolCallId,
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

// NewOpenAIClient creates a new OpenAI client
func NewRawOpenAIClient(apiKey string) (OpenAIClient, error) {
	return &rawOpenAIClient{apiKey: apiKey}, nil
}

func (c *rawOpenAIClient) SendRequest(ctx context.Context, messages []v1alpha1.Message, tools []Tool) (*v1alpha1.Message, error) {
	logger := log.FromContext(ctx)

	reqBody := chatCompletionRequest{
		Model:    "gpt-4o",
		Messages: FromKubechainMessages(messages),
	}
	if len(tools) > 0 {
		reqBody.Tools = tools
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	logger.Info("Sending request to OpenAI", "request", string(jsonBody))

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
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &LLMRequestError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
			Err:        fmt.Errorf("OpenAI API request failed"),
		}
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
		ToolCallId: openaiMessage.ToolCallID,
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
	Response              *v1alpha1.Message
	Error                 error
	Calls                 []chatCompletionRequest
	ValidateTools         func(tools []Tool) error
	ValidateContextWindow func(contextWindow []v1alpha1.Message) error
}

func (m *MockRawOpenAIClient) SendRequest(ctx context.Context, messages []v1alpha1.Message, tools []Tool) (*v1alpha1.Message, error) {
	m.Calls = append(m.Calls, chatCompletionRequest{
		Messages: FromKubechainMessages(messages),
		Tools:    tools,
	})

	if m.ValidateTools != nil {
		if err := m.ValidateTools(tools); err != nil {
			return nil, err
		}
	}

	if m.ValidateContextWindow != nil {
		if err := m.ValidateContextWindow(messages); err != nil {
			return nil, err
		}
	}

	if m.Error != nil {
		return m.Response, m.Error
	}

	if m.Response == nil {
		return &v1alpha1.Message{
			Role:    "assistant",
			Content: "Mock response",
		}, nil
	}

	return m.Response, m.Error
}
