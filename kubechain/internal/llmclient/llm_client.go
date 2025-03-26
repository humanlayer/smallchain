package llmclient

import (
	"context"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

// LLMClient defines the interface for interacting with LLM providers
type LLMClient interface {
	// SendRequest sends a request to the LLM and returns the response
	SendRequest(ctx context.Context, messages []kubechainv1alpha1.Message, tools []Tool) (*kubechainv1alpha1.Message, error)
}

// Tool represents a function that can be called by the LLM
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction contains the function details
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  ToolFunctionParameters `json:"parameters"`
}

// ToolFunctionParameter defines a parameter type
type ToolFunctionParameter struct {
	Type string `json:"type"`
}

// ToolFunctionParameters defines the schema for the function parameters
type ToolFunctionParameters struct {
	Type       string                           `json:"type"`
	Properties map[string]ToolFunctionParameter `json:"properties"`
	Required   []string                         `json:"required,omitempty"`
}
