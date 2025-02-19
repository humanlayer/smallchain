package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// TaskSpec defines the desired state of Task
type TaskSpec struct {
	// AgentRef references the agent that will execute this task
	// +kubebuilder:validation:Required
	AgentRef LocalObjectReference `json:"agentRef"`

	// UserMessage is the input prompt or request for the task
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	UserMessage string `json:"userMessage"`

	// Parameters contains additional configuration for the task
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Type=object
	Parameters runtime.RawExtension `json:"parameters,omitempty"`
}

// TaskStatus defines the observed state of Task
type TaskStatus struct {
	// Ready indicates if the task is ready to be executed
	Ready bool `json:"ready,omitempty"`

	// Status provides additional information about the task's status
	// +optional
	Status string `json:"status,omitempty"`

	// Output contains the result of the task execution
	// +optional
	Output string `json:"output,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status"
// +kubebuilder:printcolumn:name="Agent",type="string",JSONPath=".spec.agentRef.name"
// +kubebuilder:printcolumn:name="Preview",type="string",JSONPath=".spec.userMessage"
// +kubebuilder:printcolumn:name="Output",type="string",JSONPath=".status.output",priority=1
// +kubebuilder:resource:scope=Namespaced

// Task is the Schema for the tasks API
type Task struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TaskSpec   `json:"spec,omitempty"`
	Status TaskStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TaskList contains a list of Task
type TaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Task `json:"items"`
}
