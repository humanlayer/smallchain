// Note, may eventually move on from the client.go in this project
// in which case I would rename this file to client.go
package humanlayer

import (
	"context"
	"fmt"
	"net/url"
	"os"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	humanlayerapi "github.com/humanlayer/smallchain/kubechain/internal/humanlayerapi"
)

// NewHumanLayerClient creates a new API client using either the provided API key
// or falling back to the HUMANLAYER_API_KEY environment variable. Similarly,
// it uses the provided API base URL or falls back to HUMANLAYER_API_BASE.
func NewHumanLayerClient(optionalApiBase string) (HumanLayerClientInterface, error) {
	config := humanlayerapi.NewConfiguration()

	// Get API base from parameter or environment variable
	apiBase := os.Getenv("HUMANLAYER_API_BASE")
	if optionalApiBase != "" {
		apiBase = optionalApiBase
	}

	if apiBase == "" {
		apiBase = "https://api.humanlayer.dev"
	}

	parsedURL, err := url.Parse(apiBase)
	if err != nil {
		return nil, fmt.Errorf("failed to parse API base URL: %v", err)
	}

	config.Host = parsedURL.Host
	config.Scheme = parsedURL.Scheme
	config.Servers = humanlayerapi.ServerConfigurations{
		{
			URL:         apiBase,
			Description: "HumanLayer API server",
		},
	}

	// Create the API client with the configuration
	client := humanlayerapi.NewAPIClient(config)

	return &HumanLayerClient{client: client}, nil
}

type HumanLayerClientWrapperInterface interface {
	SetSlackConfig(slackConfig *kubechainv1alpha1.SlackChannelConfig)
	SetEmailConfig(emailConfig *kubechainv1alpha1.EmailChannelConfig)
	SetFunctionCallSpec(functionName string, args map[string]interface{})
	SetCallID(callID string)
	SetRunID(runID string)
	SetAPIKey(apiKey string)
	RequestApproval(ctx context.Context) (functionCall *humanlayerapi.FunctionCallOutput, statusCode int, err error)
}

type HumanLayerClientInterface interface {
	NewHumanLayerClient() HumanLayerClientWrapperInterface
}

type HumanLayerClientWrapper struct {
	client                *humanlayerapi.APIClient
	slackChannelInput     *humanlayerapi.SlackContactChannelInput
	emailContactChannel   *humanlayerapi.EmailContactChannel
	functionCallSpecInput *humanlayerapi.FunctionCallSpecInput
	callID                string
	runID                 string
	apiKey                string
}

type HumanLayerClient struct {
	client *humanlayerapi.APIClient
}

func (h *HumanLayerClient) NewHumanLayerClient() HumanLayerClientWrapperInterface {
	return &HumanLayerClientWrapper{
		client: h.client,
	}
}

func (h *HumanLayerClientWrapper) SetSlackConfig(slackConfig *kubechainv1alpha1.SlackChannelConfig) {
	slackChannelInput := humanlayerapi.NewSlackContactChannelInput(slackConfig.ChannelOrUserID)

	if slackConfig.ContextAboutChannelOrUser != "" {
		slackChannelInput.SetContextAboutChannelOrUser(slackConfig.ContextAboutChannelOrUser)
	}

	h.slackChannelInput = slackChannelInput
}

func (h *HumanLayerClientWrapper) SetEmailConfig(emailConfig *kubechainv1alpha1.EmailChannelConfig) {
	emailContactChannel := humanlayerapi.NewEmailContactChannel(emailConfig.Address)

	if emailConfig.ContextAboutUser != "" {
		emailContactChannel.SetContextAboutUser(emailConfig.ContextAboutUser)
	}

	h.emailContactChannel = emailContactChannel
}

func (h *HumanLayerClientWrapper) SetFunctionCallSpec(functionName string, args map[string]interface{}) {
	// Create the function call input with required parameters
	functionCallSpecInput := humanlayerapi.NewFunctionCallSpecInput(functionName, args)

	h.functionCallSpecInput = functionCallSpecInput
}

func (h *HumanLayerClientWrapper) SetCallID(callID string) {
	h.callID = callID
}

func (h *HumanLayerClientWrapper) SetRunID(runID string) {
	h.runID = runID
}

func (h *HumanLayerClientWrapper) SetAPIKey(apiKey string) {
	h.apiKey = apiKey
}

func (h *HumanLayerClientWrapper) RequestApproval(ctx context.Context) (functionCall *humanlayerapi.FunctionCallOutput, statusCode int, err error) {
	channel := humanlayerapi.NewContactChannelInput()

	if h.slackChannelInput != nil {
		channel.SetSlack(*h.slackChannelInput)
	}

	if h.emailContactChannel != nil {
		channel.SetEmail(*h.emailContactChannel)
	}

	h.functionCallSpecInput.SetChannel(*channel)
	functionCallInput := humanlayerapi.NewFunctionCallInput(h.runID, h.callID, *h.functionCallSpecInput)

	functionCall, resp, err := h.client.DefaultAPI.RequestApproval(ctx).
		Authorization("Bearer " + h.apiKey).
		FunctionCallInput(*functionCallInput).
		Execute()

	return functionCall, resp.StatusCode, err
}
