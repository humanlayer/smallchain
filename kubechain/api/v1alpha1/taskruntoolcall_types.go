package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TaskRunToolCallSpec defines the desired state of TaskRunToolCall
type TaskRunToolCallSpec struct {
	// TaskRunRef references the parent TaskRun
	// +kubebuilder:validation:Required
	TaskRunRef LocalObjectReference `json:"taskRunRef"`

	// ToolRef references the tool to execute
	// +kubebuilder:validation:Required
	ToolRef LocalObjectReference `json:"toolRef"`

	// Arguments contains the arguments for the tool call
	// +kubebuilder:validation:Required
	Arguments string `json:"arguments"`
}

// TaskRunToolCallStatus defines the observed state of TaskRunToolCall
type TaskRunToolCallStatus struct {
	// Phase indicates the current phase of the tool call
	// +optional
	Phase TaskRunToolCallPhase `json:"phase,omitempty"`

	// Result contains the result of the tool call if completed
	// +optional
	Result string `json:"result,omitempty"`

	// Error message if the tool call failed
	// +optional
	Error string `json:"error,omitempty"`

	// StartTime is when the tool call started
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is when the tool call completed
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
}

// TaskRunToolCallPhase represents the phase of a TaskRunToolCall
// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed
type TaskRunToolCallPhase string

const (
	// TaskRunToolCallPhasePending indicates the tool call is pending execution
	TaskRunToolCallPhasePending TaskRunToolCallPhase = "Pending"
	// TaskRunToolCallPhaseRunning indicates the tool call is currently executing
	TaskRunToolCallPhaseRunning TaskRunToolCallPhase = "Running"
	// TaskRunToolCallPhaseSucceeded indicates the tool call completed successfully
	TaskRunToolCallPhaseSucceeded TaskRunToolCallPhase = "Succeeded"
	// TaskRunToolCallPhaseFailed indicates the tool call failed
	TaskRunToolCallPhaseFailed TaskRunToolCallPhase = "Failed"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="TaskRun",type="string",JSONPath=".spec.taskRunRef.name"
// +kubebuilder:printcolumn:name="Tool",type="string",JSONPath=".spec.toolRef.name"
// +kubebuilder:printcolumn:name="Started",type="date",JSONPath=".status.startTime",priority=1
// +kubebuilder:printcolumn:name="Completed",type="date",JSONPath=".status.completionTime",priority=1
// +kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.error",priority=1
// +kubebuilder:resource:scope=Namespaced

// TaskRunToolCall is the Schema for the taskruntoolcalls API
type TaskRunToolCall struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TaskRunToolCallSpec   `json:"spec,omitempty"`
	Status TaskRunToolCallStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TaskRunToolCallList contains a list of TaskRunToolCall
type TaskRunToolCallList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TaskRunToolCall `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TaskRunToolCall{}, &TaskRunToolCallList{})
}
