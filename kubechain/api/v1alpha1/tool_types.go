package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ToolSpec defines the desired state of Tool
type ToolSpec struct {
	// ToolType represents the type of tool; e.g. "function", "delegateToAgent", etc.
	// +kubebuilder:validation:Enum=function;delegateToAgent
	ToolType string `json:"toolType,omitempty"`

	// Name is used for inline/function tools (optional if the object name is used).
	Name string `json:"name,omitempty"`

	// Description provides a description of the tool.
	Description string `json:"description,omitempty"`

	// Arguments defines the JSON schema for the tool's arguments.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Type=object
	Arguments runtime.RawExtension `json:"arguments,omitempty"`

	// Execute defines how the tool should be executed.
	Execute ToolExecute `json:"execute,omitempty"`

	// AgentRef is used for delegation-type tools.
	AgentRef *AgentReference `json:"agentRef,omitempty"`
}

// AgentReference defines a reference to an agent resource.
type AgentReference struct {
	Name string `json:"name,omitempty"`
}

// ToolExecute defines execution details for the tool.
type ToolExecute struct {
	// Builtin represents an inline (builtin) tool.
	Builtin *BuiltinToolSpec `json:"builtin,omitempty"`

	// Future fields such as container or remote execution can be added here.
}

// BuiltinToolSpec defines the parameters for executing a builtin tool.
type BuiltinToolSpec struct {
	// Name is the identifier of the builtin function to run.
	Name string `json:"name,omitempty"`
}

// ToolStatus defines the observed state of Tool
type ToolStatus struct {
	// Ready indicates if the tool has been validated and is ready for use.
	Ready bool `json:"ready,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.toolType"
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"

// Tool is the Schema for the tools API
type Tool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ToolSpec   `json:"spec,omitempty"`
	Status ToolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ToolList contains a list of Tool
type ToolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tool `json:"items"`
}
