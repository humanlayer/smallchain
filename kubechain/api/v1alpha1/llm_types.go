/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SecretKeyRef contains the reference to a secret key
type SecretKeyRef struct {
	// Name is the name of the secret
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Key is the key in the secret
	// +kubebuilder:validation:Required
	Key string `json:"key"`
}

// APIKeySource defines how to get the API key
type APIKeySource struct {
	// SecretKeyRef references a key in a secret
	SecretKeyRef SecretKeyRef `json:"secretKeyRef"`
}

// LLMSpec defines the desired state of LLM
type LLMSpec struct {
	// Provider is the LLM provider name (ex: "openai", "anthropic")
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=openai;anthropic
	Provider string `json:"provider"`

	// APIKeyFrom references the secret containing the API key
	// +kubebuilder:validation:Required
	APIKeyFrom APIKeySource `json:"apiKeyFrom"`

	// Temperature adjusts the LLM response randomness (0.0 to 1.0)
	// +kubebuilder:validation:Pattern=^0(\.[0-9]+)?|1(\.0+)?$
	Temperature string `json:"temperature,omitempty"`

	// MaxTokens defines the maximum number of tokens for the LLM.
	// +kubebuilder:validation:Minimum=1
	MaxTokens *int `json:"maxTokens,omitempty"`
}

// LLMStatus defines the observed state of LLM
type LLMStatus struct {
	// Ready indicates if the external dependency (e.g. secret) has been validated.
	Ready bool `json:"ready,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Provider",type="string",JSONPath=".spec.provider"
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"
// +kubebuilder:resource:scope=Namespaced

// LLM is the Schema for the llms API
type LLM struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LLMSpec   `json:"spec,omitempty"`
	Status LLMStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LLMList contains a list of LLM
type LLMList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LLM `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LLM{}, &LLMList{})
}
