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

package controller

import (
	"net/http"
	"net/http/httptest"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestCase represents a generic controller test case
type TestCase struct {
	Name           string
	ShouldSucceed  bool
	ExpectedStatus string
	ExpectedDetail string
	EventType      string
}

// LLMTestCase represents an LLM controller test case
type LLMTestCase struct {
	TestCase
	SecretExists bool
	SecretValue  []byte
}

// NewMockOpenAIServer creates a test HTTP server that simulates the OpenAI API
func NewMockOpenAIServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer valid-key" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
}

// AgentTestCase represents an Agent controller test case
type AgentTestCase struct {
	TestCase
	LLMExists   bool
	ToolsExist  bool
	ToolsReady  bool
	ExpectError bool
}

// CreateAgentSpec creates an agent spec with the given LLM and tool references
func CreateAgentSpec(llmName string, toolNames []string) kubechainv1alpha1.AgentSpec {
	toolRefs := []kubechainv1alpha1.LocalObjectReference{}
	for _, toolName := range toolNames {
		toolRefs = append(toolRefs, kubechainv1alpha1.LocalObjectReference{Name: toolName})
	}

	return kubechainv1alpha1.AgentSpec{
		LLMRef: kubechainv1alpha1.LocalObjectReference{
			Name: llmName,
		},
		Tools:  toolRefs,
		System: "Test agent",
	}
}

// CreateLLMSpec creates an LLM spec with the given secret reference
func CreateLLMSpec(secretName, secretKey string) kubechainv1alpha1.LLMSpec {
	return kubechainv1alpha1.LLMSpec{
		Provider: "openai",
		APIKeyFrom: kubechainv1alpha1.APIKeySource{
			SecretKeyRef: kubechainv1alpha1.SecretKeyRef{
				Name: secretName,
				Key:  secretKey,
			},
		},
	}
}

// CreateToolSpec creates a tool spec with the given name
func CreateToolSpec(name string) kubechainv1alpha1.ToolSpec {
	return kubechainv1alpha1.ToolSpec{
		ToolType: "function",
		Name:     name,
	}
}

// CreateObjectMeta creates object metadata with the given name and namespace
func CreateObjectMeta(name, namespace string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}
}