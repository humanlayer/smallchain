package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AgentSpec defines the desired state of Agent
type AgentSpec struct {
	// LLMRef references the LLM to use for this agent
	// +kubebuilder:validation:Required
	LLMRef LocalObjectReference `json:"llmRef"`

	// Tools is a list of tools this agent can use
	// +optional
	Tools []LocalObjectReference `json:"tools,omitempty"`

	// System is the system prompt for the agent
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	System string `json:"system"`
}

// LocalObjectReference contains enough information to locate the referenced resource in the same namespace
type LocalObjectReference struct {
	// Name of the referent
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// AgentStatus defines the observed state of Agent
type AgentStatus struct {
	// Ready indicates if the agent's dependencies (LLM and Tools) are valid and ready
	Ready bool `json:"ready,omitempty"`

	// Status provides additional information about the agent's status
	// +optional
	Status string `json:"status,omitempty"`

	// ValidTools is the list of tools that were successfully validated
	// +optional
	ValidTools []string `json:"validTools,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status"
// +kubebuilder:resource:scope=Namespaced

// Agent is the Schema for the agents API
type Agent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentSpec   `json:"spec,omitempty"`
	Status AgentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AgentList contains a list of Agent
type AgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Agent `json:"items"`
}
