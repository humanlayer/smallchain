package llmclient

import (
	"context"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

// NewLLMClient creates a new LLM client based on the LLM configuration
func NewLLMClient(ctx context.Context, llm kubechainv1alpha1.LLM, apiKey string) (LLMClient, error) {
	return NewLangchainClient(ctx, llm.Spec.Provider, apiKey, llm.Spec.Parameters)
}

// Legacy adapter for backward compatibility during transition
// Will be removed once all controllers are updated
// Temporary definition of OpenAIClient interface
type OpenAIClient interface {
	SendRequest(ctx context.Context, messages []kubechainv1alpha1.Message, tools []Tool) (*kubechainv1alpha1.Message, error)
}

func NewRawOpenAIClient(apiKey string) (OpenAIClient, error) {
	// Create an OpenAI client using the langchaingo client
	ctx := context.Background()
	llm := kubechainv1alpha1.LLM{
		Spec: kubechainv1alpha1.LLMSpec{
			Provider: "openai",
			Parameters: kubechainv1alpha1.BaseConfig{
				Model: "gpt-4o",
			},
		},
	}

	client, err := NewLLMClient(ctx, llm, apiKey)
	if err != nil {
		return nil, err
	}

	// Return the LLMClient directly since it already implements the OpenAIClient interface
	return client, nil
}
