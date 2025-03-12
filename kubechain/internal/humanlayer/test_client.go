package humanlayer

import (
	"context"
	"fmt"
	"log"
)

func RunTest() {
	// Create a new client with your API key
	client := NewClient("hl-a-xg52uTvDVR_XQohWAXvz4cFjVIVve-DfCBSELw3KCK4")

	// Create a context
	ctx := context.Background()

	// Example function call
	response, err := client.CallFunction(
		ctx,
		"run-123",          // Run ID
		"call-456",         // Call ID
		"example_function", // Function name
		map[string]interface{}{ // Arguments
			"param1": "value1",
			"param2": 42,
		},
	)
	if err != nil {
		log.Fatalf("Error calling function: %v", err)
	}

	fmt.Printf("Initial response: %+v\n", response)

	// Poll for approval
	finalResponse, err := client.PollForApproval(ctx, response.CallID)
	if err != nil {
		log.Fatalf("Error polling for approval: %v", err)
	}

	fmt.Printf("Final response: %+v\n", finalResponse)
}
