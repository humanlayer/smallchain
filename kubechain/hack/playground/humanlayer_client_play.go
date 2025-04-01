package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/uuid"
	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"

	"github.com/humanlayer/smallchain/kubechain/internal/humanlayer"
	"github.com/humanlayer/smallchain/kubechain/internal/humanlayerapi"
)

func requestApproval(client humanlayer.HumanLayerClientWrapper, channelType string) *humanlayerapi.FunctionCallOutput {
	if channelType == "slack" {
		client.SetSlackConfig(&kubechainv1alpha1.SlackChannelConfig{
			ChannelOrUserID:           "C07HR5JL15F",
			ContextAboutChannelOrUser: "Channel for approving web fetch operations",
		})
	} else if channelType == "email" {
		client.SetEmailConfig(&kubechainv1alpha1.EmailChannelConfig{
			Address:          os.Getenv("HL_EXAMPLE_CONTACT_EMAIL"),
			ContextAboutUser: "Primary approver for web fetch operations",
		})
	}

	client.SetFunctionCallSpec("test-city", map[string]any{
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

func getFunctionCallStatus(client humanlayer.HumanLayerClientWrapper) *humanlayerapi.FunctionCallOutput {
	functionCall, statusCode, err := client.GetFunctionCallStatus(context.Background())

	fmt.Println(functionCall.GetCallId())
	fmt.Println(statusCode)
	fmt.Println(err)

	return functionCall
}

func main() {
	// Define command line flags
	callIDFlag := flag.String("call-id", "", "Existing call ID to check status for")
	typeFlag := flag.String("channel", "slack", "Channel type (slack or email)")
	flag.Parse()

	factory, _ := humanlayer.NewHumanLayerClientFactory("")

	client := factory.NewHumanLayerClient()
	client.SetAPIKey(os.Getenv("HUMANLAYER_API_KEY"))

	var callID string

	if *callIDFlag != "" {
		fmt.Println("Call ID provided as argument - skipping approval request")
		callID = *callIDFlag
	} else {
		fc := requestApproval(client, *typeFlag)
		callID = fc.GetCallId()
	}

	client.SetCallID(callID)

	fcStatus := getFunctionCallStatus(client)
	status := fcStatus.GetStatus()

	approved, ok := status.GetApprovedOk()

	// Check if the value was set
	if ok {
		if approved == nil {
			fmt.Println("Approval status is nil (Not responded yet)")
		} else if *approved {
			fmt.Println("Approved")
		} else {
			fmt.Println("Rejected")
		}
	} else {
		fmt.Println("Not responded yet")
	}
}
