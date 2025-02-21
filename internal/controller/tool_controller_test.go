package controller_test

import (
    "encoding/json"
    "testing"

    kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
    "k8s.io/apimachinery/pkg/runtime"
)

func TestToolParametersDeserialization(t *testing.T) {
    validJSON := `{"param1": "value1", "param2": 2}`
    invalidJSON := `{"param1": "value1", "param2":}`

    // Test with valid JSON:
    tool := &kubechainv1alpha1.Tool{
        Spec: kubechainv1alpha1.ToolSpec{
            Description: "A test tool",
            Parameters:  runtime.RawExtension{Raw: []byte(validJSON)},
        },
    }
    var params map[string]interface{}
    err := json.Unmarshal(tool.Spec.Parameters.Raw, &params)
    if err != nil {
        t.Errorf("expected valid JSON, but got error: %v", err)
    }
    if params["param1"] != "value1" {
        t.Errorf("expected param1 to be 'value1', got %v", params["param1"])
    }

    // Test with invalid JSON:
    toolInvalid := &kubechainv1alpha1.Tool{
        Spec: kubechainv1alpha1.ToolSpec{
            Description: "An invalid test tool",
            Parameters:  runtime.RawExtension{Raw: []byte(invalidJSON)},
        },
    }
    err = json.Unmarshal(toolInvalid.Spec.Parameters.Raw, &params)
    if err == nil {
        t.Errorf("expected error for invalid JSON, but got none")
    }
}
