package humanlayer

import (
	"context"
	"time"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	humanlayerapi "github.com/humanlayer/smallchain/kubechain/internal/humanlayerapi"
)

// MockHumanLayerClient implements HumanLayerClientInterface for testing
type MockHumanLayerClient struct {
	ShouldFail           bool
	StatusCode           int
	ReturnError          error
	ShouldReturnApproval bool
	LastAPIKey           string
	LastCallID           string
	LastRunID            string
	LastFunction         string
	LastArguments        map[string]interface{}
}

// MockHumanLayerClientWrapper implements HumanLayerClientWrapperInterface for testing
type MockHumanLayerClientWrapper struct {
	parent       *MockHumanLayerClient
	slackConfig  *kubechainv1alpha1.SlackChannelConfig
	emailConfig  *kubechainv1alpha1.EmailChannelConfig
	functionName string
	functionArgs map[string]interface{}
	callID       string
	runID        string
	apiKey       string
}

// NewHumanLayerClient creates a new mock client
func NewMockHumanLayerClient(shouldFail bool, statusCode int, returnError error) *MockHumanLayerClient {
	return &MockHumanLayerClient{
		ShouldFail:  shouldFail,
		StatusCode:  statusCode,
		ReturnError: returnError,
	}
}

// NewHumanLayerClient implements HumanLayerClientFactoryInterface
func (m *MockHumanLayerClient) NewHumanLayerClient() HumanLayerClientWrapperInterface {
	return &MockHumanLayerClientWrapper{
		parent: m,
	}
}

// SetSlackConfig implements HumanLayerClientWrapperInterface
func (m *MockHumanLayerClientWrapper) SetSlackConfig(slackConfig *kubechainv1alpha1.SlackChannelConfig) {
	m.slackConfig = slackConfig
}

// SetEmailConfig implements HumanLayerClientWrapperInterface
func (m *MockHumanLayerClientWrapper) SetEmailConfig(emailConfig *kubechainv1alpha1.EmailChannelConfig) {
	m.emailConfig = emailConfig
}

// SetFunctionCallSpec implements HumanLayerClientWrapperInterface
func (m *MockHumanLayerClientWrapper) SetFunctionCallSpec(functionName string, args map[string]interface{}) {
	m.functionName = functionName
	m.functionArgs = args
}

// SetCallID implements HumanLayerClientWrapperInterface
func (m *MockHumanLayerClientWrapper) SetCallID(callID string) {
	m.callID = callID
}

// SetRunID implements HumanLayerClientWrapperInterface
func (m *MockHumanLayerClientWrapper) SetRunID(runID string) {
	m.runID = runID
}

// SetAPIKey implements HumanLayerClientWrapperInterface
func (m *MockHumanLayerClientWrapper) SetAPIKey(apiKey string) {
	m.apiKey = apiKey
}

// GetFunctionCallStatus implements HumanLayerClientWrapperInterface
func (m *MockHumanLayerClientWrapper) GetFunctionCallStatus(ctx context.Context) (*humanlayerapi.FunctionCallOutput, int, error) {

	if m.parent.ShouldReturnApproval {
		now := time.Now()
		approved := true
		status := humanlayerapi.NewNullableFunctionCallStatus(&humanlayerapi.FunctionCallStatus{
			RequestedAt: *humanlayerapi.NewNullableTime(&now),
			RespondedAt: *humanlayerapi.NewNullableTime(&now),
			Approved:    *humanlayerapi.NewNullableBool(&approved),
		})
		return &humanlayerapi.FunctionCallOutput{
			Status: *status,
		}, 200, nil
	}

	return nil, m.parent.StatusCode, m.parent.ReturnError
}

// RequestApproval implements HumanLayerClientWrapperInterface
func (m *MockHumanLayerClientWrapper) RequestApproval(ctx context.Context) (*humanlayerapi.FunctionCallOutput, int, error) {
	// Store the values in the parent for test verification
	m.parent.LastAPIKey = m.apiKey
	m.parent.LastCallID = m.callID
	m.parent.LastRunID = m.runID
	m.parent.LastFunction = m.functionName
	m.parent.LastArguments = m.functionArgs

	if m.parent.ShouldFail {
		return nil, m.parent.StatusCode, m.parent.ReturnError
	}

	// Return a successful mock response
	return &humanlayerapi.FunctionCallOutput{}, m.parent.StatusCode, nil
}
