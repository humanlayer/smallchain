package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TaskRunSpec defines the desired state of TaskRun
type TaskRunSpec struct {
	// TaskRef references the task to run
	// +kubebuilder:validation:Required
	TaskRef LocalObjectReference `json:"taskRef"`

	// TaskRunToolCallRef is used when the TaskRun is created for a tool call delegation.
	// +optional
	TaskRunToolCallRef *LocalObjectReference `json:"taskRunToolCallRef,omitempty"`

	// AgentRef is used when the TaskRun is created for a tool call delegation.
	// +optional
	AgentRef *LocalObjectReference `json:"agentRef,omitempty"`
}

// Message represents a single message in the conversation
type Message struct {
	// Role is the role of the message sender (system, user, assistant, tool)
	// +kubebuilder:validation:Enum=system;user;assistant;tool
	Role string `json:"role"`

	// Content is the message content
	Content string `json:"content"`

	// ToolCalls contains any tool calls requested by this message
	// +optional
	ToolCalls []ToolCall `json:"toolCalls,omitempty"`
}

// ToolCall represents a request to call a tool
type ToolCall struct {
	// ID is the unique identifier for this tool call
	ID string `json:"id"`

	// Function contains the details of the function to call
	Function ToolCallFunction `json:"function"`

	// Type indicates the type of tool call. Currently only "function" is supported.
	Type string `json:"type"`
}

// ToolCallFunction contains the function details for a tool call
type ToolCallFunction struct {
	// Name is the name of the function to call
	Name string `json:"name"`

	// Arguments contains the arguments to pass to the function in JSON format
	Arguments string `json:"arguments"`
}

// TaskRunStatus defines the observed state of TaskRun
type TaskRunStatus struct {
	// Phase indicates the current phase of the TaskRun
	// +optional
	Phase TaskRunPhase `json:"phase,omitempty"`

	// Ready indicates if the TaskRun is ready to be executed
	// +optional
	Ready bool `json:"ready,omitempty"`

	// Status indicates the current status of the taskrun
	// +kubebuilder:validation:Enum=Ready;Error;Pending
	Status string `json:"status,omitempty"`

	// StatusDetail provides additional details about the current status
	StatusDetail string `json:"statusDetail,omitempty"`

	// StartTime is when the TaskRun started
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is when the TaskRun completed
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Output contains the result of the task execution
	// +optional
	Output string `json:"output,omitempty"`

	// ContextWindow maintains the conversation history as a sequence of messages
	// +optional
	ContextWindow []Message `json:"contextWindow,omitempty"`

	// MessageCount contains the number of messages in the context window
	// +optional
	MessageCount int `json:"messageCount,omitempty"`

	// UserMsgPreview stores the first 50 characters of the user's message
	// +optional
	UserMsgPreview string `json:"userMsgPreview,omitempty"`

	// Error message if the task failed
	// +optional
	Error string `json:"error,omitempty"`
}

// TaskRunPhase represents the phase of a TaskRun
// +kubebuilder:validation:Enum=Pending;ReadyForLLM;SendContextWindowToLLM;ToolCallsPending;FinalAnswer;ErrorBackoff;Failed
type TaskRunPhase string

const (
	// TaskRunPhasePending indicates the TaskRun is pending execution
	TaskRunPhasePending TaskRunPhase = "Pending"
	// TaskRunPhaseReadyForLLM indicates the TaskRun is ready for context to be sent to LLM
	TaskRunPhaseReadyForLLM TaskRunPhase = "ReadyForLLM"
	// TaskRunPhaseSendContextWindowToLLM indicates the TaskRun is sending context to LLM
	TaskRunPhaseSendContextWindowToLLM TaskRunPhase = "SendContextWindowToLLM"
	// TaskRunPhaseToolCallsPending indicates the TaskRun has pending tool calls
	TaskRunPhaseToolCallsPending TaskRunPhase = "ToolCallsPending"
	// TaskRunPhaseFinalAnswer indicates the TaskRun has received final answer
	TaskRunPhaseFinalAnswer TaskRunPhase = "FinalAnswer"
	// TaskRunPhaseErrorBackoff indicates the TaskRun has failed and is in error backoff
	TaskRunPhaseErrorBackoff TaskRunPhase = "ErrorBackoff"
	// TaskRunPhaseFailed indicates the TaskRun has failed
	TaskRunPhaseFailed TaskRunPhase = "Failed"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status"
// +kubebuilder:printcolumn:name="Detail",type="string",JSONPath=".status.statusDetail",priority=1
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Task",type="string",JSONPath=".spec.taskRef.name"
// +kubebuilder:printcolumn:name="Preview",type="string",JSONPath=".status.userMsgPreview"
// +kubebuilder:printcolumn:name="Output",type="string",JSONPath=".status.output"
// +kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.error",priority=1
// +kubebuilder:printcolumn:name="Started",type="date",JSONPath=".status.startTime",priority=1
// +kubebuilder:printcolumn:name="Completed",type="date",JSONPath=".status.completionTime",priority=1
// +kubebuilder:resource:scope=Namespaced

// TaskRun is the Schema for the taskruns API
type TaskRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TaskRunSpec   `json:"spec,omitempty"`
	Status TaskRunStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TaskRunList contains a list of TaskRun
type TaskRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TaskRun `json:"items"`
}
