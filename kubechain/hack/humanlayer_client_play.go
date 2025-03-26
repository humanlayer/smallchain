package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"

	"github.com/humanlayer/smallchain/kubechain/internal/humanlayer"
)

func main() {
	base, _ := humanlayer.NewHumanLayerClient("")

	client := base.NewHumanLayerClient()

	client.SetSlackConfig(&kubechainv1alpha1.SlackChannelConfig{
		ChannelOrUserID:           "C07HR5JL15F",
		ContextAboutChannelOrUser: "Channel for approving web fetch operations",
	})

	client.SetFunctionCallSpec("test-city", map[string]interface{}{
		"a": 1,
		"b": 2,
	})

	client.SetCallID("call-" + uuid.New().String())
	client.SetRunID("sundeep-is-testing")
	client.SetAPIKey(os.Getenv("HUMANLAYER_API_KEY"))

	functionCall, statusCode, err := client.RequestApproval(context.Background())

	fmt.Println(functionCall.GetCallId())
	fmt.Println(statusCode)
	fmt.Println(err)
}
