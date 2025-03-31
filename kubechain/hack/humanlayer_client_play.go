package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"

	"github.com/humanlayer/smallchain/kubechain/internal/humanlayer"
	"github.com/humanlayer/smallchain/kubechain/internal/humanlayerapi"
)

func requestApproval(client humanlayer.HumanLayerClientWrapperInterface) *humanlayerapi.FunctionCallOutput {
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

	functionCall, statusCode, err := client.RequestApproval(context.Background())

	fmt.Println(functionCall.GetCallId())
	fmt.Println(statusCode)
	fmt.Println(err)

	return functionCall
}

func getFunctionCallStatus(client humanlayer.HumanLayerClientWrapperInterface) *humanlayerapi.FunctionCallOutput {
	functionCall, statusCode, err := client.GetFunctionCallStatus(context.Background())

	fmt.Println(functionCall.GetCallId())
	fmt.Println(statusCode)
	fmt.Println(err)

	return functionCall
}

func main() {
	factory, _ := humanlayer.NewHumanLayerClientFactory("")

	client := factory.NewHumanLayerClient()
	client.SetAPIKey(os.Getenv("HUMANLAYER_API_KEY"))

	var callID string

	if len(os.Args) > 1 {
		fmt.Println("Call ID provided as argument - skipping approval request")
		callID = os.Args[1]
	} else {
		fc := requestApproval(client)
		callID = fc.GetCallId()
	}

	client.SetCallID(callID)

	fcStatus := getFunctionCallStatus(client)
	status := fcStatus.GetStatus()

	respondedAt := status.RespondedAt.Get()

	fmt.Println("Approved: ", status.GetApproved())
	fmt.Println("Responded at", respondedAt)
}
