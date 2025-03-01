package llmclient

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestToolDeserialization(t *testing.T) {
	jsonStr := `{
		"type": "function",
		"function": {
			"name": "add", 
			"description": "Add two numbers together.",
			"parameters": {
				"type": "object",
				"properties": {
					"x": {"type": "number"},
					"y": {"type": "number"}
				},
				"required": ["x", "y"]
			}
		}
	}`

	var tool Tool
	err := json.Unmarshal([]byte(jsonStr), &tool)
	assert.NoError(t, err)

	// Verify the deserialized fields
	assert.Equal(t, "function", tool.Type)
	assert.Equal(t, "add", tool.Function.Name)
	assert.Equal(t, "Add two numbers together.", tool.Function.Description)
	assert.Equal(t, "object", tool.Function.Parameters.Type)
	assert.Len(t, tool.Function.Parameters.Properties, 2)
	assert.Equal(t, "number", tool.Function.Parameters.Properties["x"].Type)
	assert.Equal(t, "number", tool.Function.Parameters.Properties["y"].Type)
	assert.Equal(t, []string{"x", "y"}, tool.Function.Parameters.Required)
}

func TestFromKubechainTool(t *testing.T) {
	// Create a kubechain Tool with parameters directly
	kubechainTool := v1alpha1.Tool{
		Spec: v1alpha1.ToolSpec{
			Name:        "add",
			Description: "Add two numbers together.",
			Parameters: runtime.RawExtension{
				Raw: []byte(`{
					"type": "object",
					"properties": {
						"x": {"type": "number"},
						"y": {"type": "number"}
					},
					"required": ["x", "y"]
				}`),
			},
		},
	}

	// Convert to client Tool
	clientTool := FromKubechainTool(kubechainTool)
	assert.NotNil(t, clientTool)

	// Verify the converted fields
	assert.Equal(t, "function", clientTool.Type)
	assert.Equal(t, "add", clientTool.Function.Name)
	assert.Equal(t, "Add two numbers together.", clientTool.Function.Description)
	assert.Equal(t, "object", clientTool.Function.Parameters.Type)
	assert.Len(t, clientTool.Function.Parameters.Properties, 2)
	assert.Equal(t, "number", clientTool.Function.Parameters.Properties["x"].Type)
	assert.Equal(t, "number", clientTool.Function.Parameters.Properties["y"].Type)
	assert.Equal(t, []string{"x", "y"}, clientTool.Function.Parameters.Required)
}

func TestSendRequest(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY is not set")
	}

	client, err := NewRawOpenAIClient(apiKey)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	ctx := context.Background()
	tools := []Tool{
		{
			Type: "function",
			Function: ToolFunction{
				Name: "add",
				Parameters: ToolFunctionParameters{
					Type: "object",
					Properties: map[string]ToolFunctionParameter{
						"x": {Type: "number"},
						"y": {Type: "number"},
					},
					Required: []string{"x", "y"},
				},
			},
		},
	}

	messages := []v1alpha1.Message{
		{
			Role:    "system",
			Content: "You are a helpful assistant that can add two numbers together.",
		},
		{
			Role:    "user",
			Content: "What is 2 + 2?",
		},
	}

	response, err := client.SendRequest(ctx, messages, tools)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "add", response.ToolCalls[0].Function.Name)

	expectedArgs := map[string]interface{}{"x": 2.0, "y": 2.0}
	actualArgs := map[string]interface{}{}
	err = json.Unmarshal([]byte(response.ToolCalls[0].Function.Arguments), &actualArgs)
	assert.NoError(t, err)
	assert.Equal(t, expectedArgs, actualArgs)

	messages = append(messages, *response)

	messages = append(messages, v1alpha1.Message{
		Role:       "tool",
		ToolCallID: response.ToolCalls[0].ID,
		Name:       response.ToolCalls[0].Function.Name,
		Content:    "4",
	})

	response, err = client.SendRequest(ctx, messages, tools)
	assert.NoError(t, err)
	assert.NotNil(t, response)

	assert.NotEmpty(t, response.Content)
	assert.Contains(t, response.Content, "4")
}
