// Note, may eventually move on from the client.go in this project
// in which case I would rename this file to client.go
package humanlayer

import (
	"fmt"
	"net/url"
	"os"

	humanlayerapi "github.com/humanlayer/smallchain/kubechain/internal/humanlayerapi"
)

// NewHumanLayerClient creates a new API client using either the provided API key
// or falling back to the HUMANLAYER_API_KEY environment variable. Similarly,
// it uses the provided API base URL or falls back to HUMANLAYER_API_BASE.
func NewHumanLayerClient(optionalApiBase string) (*humanlayerapi.APIClient, error) {
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

	return client, nil
}
