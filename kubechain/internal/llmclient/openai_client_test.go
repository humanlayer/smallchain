package llmclient

import (
	"testing"

	"github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestFromContactChannel(t *testing.T) {
	tests := []struct {
		name     string
		channel  v1alpha1.ContactChannel
		expected Tool
	}{
		{
			name: "email contact channel",
			channel: v1alpha1.ContactChannel{
				Spec: v1alpha1.ContactChannelSpec{
					Type: v1alpha1.ContactChannelTypeEmail,
					Email: &v1alpha1.EmailChannelConfig{
						Address:          "test@example.com",
						ContextAboutUser: "A helpful test user who provides quick responses",
					},
				},
			},
			expected: Tool{
				Type: "function",
				Function: ToolFunction{
					Name:        "human_contact_email_",
					Description: "A helpful test user who provides quick responses",
					Parameters: ToolFunctionParameters{
						Type: "object",
						Properties: map[string]ToolFunctionParameter{
							"message": {Type: "string"},
						},
						Required: []string{"message"},
					},
				},
			},
		},
		{
			name: "slack contact channel",
			channel: v1alpha1.ContactChannel{
				Spec: v1alpha1.ContactChannelSpec{
					Type: v1alpha1.ContactChannelTypeSlack,
					Slack: &v1alpha1.SlackChannelConfig{
						ChannelOrUserID:           "C12345678",
						ContextAboutChannelOrUser: "A team channel for engineering discussions",
					},
				},
			},
			expected: Tool{
				Type: "function",
				Function: ToolFunction{
					Name:        "human_contact_slack_",
					Description: "A team channel for engineering discussions",
					Parameters: ToolFunctionParameters{
						Type: "object",
						Properties: map[string]ToolFunctionParameter{
							"message": {Type: "string"},
						},
						Required: []string{"message"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the name to match the expected pattern
			tt.channel.Name = ""
			result := FromContactChannel(tt.channel)

			// Assert function name contains the expected prefix
			assert.Contains(t, result.Function.Name, tt.expected.Function.Name)

			// Assert other fields match exactly
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, tt.expected.Function.Description, result.Function.Description)
			assert.Equal(t, tt.expected.Function.Parameters, result.Function.Parameters)
		})
	}
}
