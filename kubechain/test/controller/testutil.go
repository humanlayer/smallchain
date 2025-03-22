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
	"context"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

// TestEnv holds the common test environment for controller tests
type TestEnv struct {
	Client     client.Client
	Env        *envtest.Environment
	Config     *rest.Config
	Ctx        context.Context
	Cancel     context.CancelFunc
	Namespace  string
	Recorder   *record.FakeRecorder
}

// NewFakeRecorder creates a new fake event recorder
func (e *TestEnv) NewFakeRecorder(bufferSize int) *record.FakeRecorder {
	return record.NewFakeRecorder(bufferSize)
}

// NewTestEnv creates a new test environment for controller tests
func NewTestEnv() *TestEnv {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel := context.WithCancel(context.TODO())

	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s", "1.32.0-darwin-arm64"),
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = kubechainv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	return &TestEnv{
		Client:    k8sClient,
		Env:       testEnv,
		Config:    cfg,
		Ctx:       ctx,
		Cancel:    cancel,
		Namespace: "default",
		Recorder:  record.NewFakeRecorder(10),
	}
}

// Stop tears down the test environment
func (e *TestEnv) Stop() {
	e.Cancel()
	err := e.Env.Stop()
	Expect(err).NotTo(HaveOccurred())
}

// CreateSecret creates a test secret in the namespace
func (e *TestEnv) CreateSecret(name, key string, value []byte) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: e.Namespace,
		},
		Data: map[string][]byte{
			key: value,
		},
	}
	Expect(e.Client.Create(e.Ctx, secret)).To(Succeed())
	return secret
}

// DeleteSecret deletes a secret if it exists
func (e *TestEnv) DeleteSecret(name string) {
	secret := &corev1.Secret{}
	err := e.Client.Get(e.Ctx, types.NamespacedName{Name: name, Namespace: e.Namespace}, secret)
	if err == nil {
		Expect(e.Client.Delete(e.Ctx, secret)).To(Succeed())
	}
}

// CreateLLM creates a test LLM resource with the given name
func (e *TestEnv) CreateLLM(name, secretName, secretKey string) *kubechainv1alpha1.LLM {
	llm := &kubechainv1alpha1.LLM{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: e.Namespace,
		},
		Spec: kubechainv1alpha1.LLMSpec{
			Provider: "openai",
			APIKeyFrom: kubechainv1alpha1.APIKeySource{
				SecretKeyRef: kubechainv1alpha1.SecretKeyRef{
					Name: secretName,
					Key:  secretKey,
				},
			},
		},
	}
	Expect(e.Client.Create(e.Ctx, llm)).To(Succeed())
	return llm
}

// MarkLLMReady marks an LLM as ready
func (e *TestEnv) MarkLLMReady(llm *kubechainv1alpha1.LLM) {
	llm.Status.Status = "Ready"
	llm.Status.StatusDetail = "Ready for testing"
	llm.Status.Ready = true
	Expect(e.Client.Status().Update(e.Ctx, llm)).To(Succeed())
}

// DeleteLLM deletes an LLM if it exists
func (e *TestEnv) DeleteLLM(name string) {
	llm := &kubechainv1alpha1.LLM{}
	err := e.Client.Get(e.Ctx, types.NamespacedName{Name: name, Namespace: e.Namespace}, llm)
	if err == nil {
		Expect(e.Client.Delete(e.Ctx, llm)).To(Succeed())
	}
}

// CreateTool creates a test Tool resource with the given name
func (e *TestEnv) CreateTool(name string) *kubechainv1alpha1.Tool {
	tool := &kubechainv1alpha1.Tool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: e.Namespace,
		},
		Spec: kubechainv1alpha1.ToolSpec{
			ToolType: "function",
			Name:     "test",
		},
	}
	Expect(e.Client.Create(e.Ctx, tool)).To(Succeed())
	return tool
}

// MarkToolReady marks a Tool as ready
func (e *TestEnv) MarkToolReady(tool *kubechainv1alpha1.Tool) {
	tool.Status.Ready = true
	Expect(e.Client.Status().Update(e.Ctx, tool)).To(Succeed())
}

// DeleteTool deletes a Tool if it exists
func (e *TestEnv) DeleteTool(name string) {
	tool := &kubechainv1alpha1.Tool{}
	err := e.Client.Get(e.Ctx, types.NamespacedName{Name: name, Namespace: e.Namespace}, tool)
	if err == nil {
		Expect(e.Client.Delete(e.Ctx, tool)).To(Succeed())
	}
}

// CreateAgent creates a test Agent resource with the given name and references
func (e *TestEnv) CreateAgent(name, llmName string, toolNames []string) *kubechainv1alpha1.Agent {
	toolRefs := []kubechainv1alpha1.LocalObjectReference{}
	for _, toolName := range toolNames {
		toolRefs = append(toolRefs, kubechainv1alpha1.LocalObjectReference{Name: toolName})
	}

	agent := &kubechainv1alpha1.Agent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: e.Namespace,
		},
		Spec: kubechainv1alpha1.AgentSpec{
			LLMRef: kubechainv1alpha1.LocalObjectReference{
				Name: llmName,
			},
			Tools:  toolRefs,
			System: "Test agent",
		},
	}
	Expect(e.Client.Create(e.Ctx, agent)).To(Succeed())
	return agent
}

// DeleteAgent deletes an Agent if it exists
func (e *TestEnv) DeleteAgent(name string) {
	agent := &kubechainv1alpha1.Agent{}
	err := e.Client.Get(e.Ctx, types.NamespacedName{Name: name, Namespace: e.Namespace}, agent)
	if err == nil {
		Expect(e.Client.Delete(e.Ctx, agent)).To(Succeed())
	}
}

// CheckEvent checks if an event with the expected message was recorded
func (e *TestEnv) CheckEvent(contains string, timeout time.Duration) {
	Eventually(func() bool {
		select {
		case event := <-e.Recorder.Events:
			return strings.Contains(event, contains)
		default:
			return false
		}
	}, timeout, 100*time.Millisecond).Should(BeTrue(), "Expected to find event containing: "+contains)
}

// GetUpdatedResource gets the updated resource from the API server
func GetUpdatedResource[T client.Object](c client.Client, ctx context.Context, obj T, name, namespace string) T {
	namespacedName := types.NamespacedName{Name: name, Namespace: namespace}
	err := c.Get(ctx, namespacedName, obj)
	Expect(err).NotTo(HaveOccurred())
	return obj
}