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

package utils

import (
	"context"

	. "github.com/onsi/gomega" //nolint:golint,revive
	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubechain "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TestScopedAgent struct {
	Name         string
	SystemPrompt string
	Tools        []string
	LLM          string
	client       client.Client
}

func (t *TestScopedAgent) Setup(k8sClient client.Client) {
	t.client = k8sClient
	// Create a test Agent
	agent := &kubechain.Agent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
		Spec: kubechain.AgentSpec{
			LLMRef: kubechain.LocalObjectReference{
				Name: t.LLM,
			},
			System: t.SystemPrompt,
			Tools: func() []kubechain.LocalObjectReference {
				refs := make([]kubechain.LocalObjectReference, len(t.Tools))
				for i, tool := range t.Tools {
					refs[i] = kubechain.LocalObjectReference{Name: tool}
				}
				return refs
			}(),
		},
	}
	Expect(t.client.Create(context.Background(), agent)).To(Succeed())

	// Mark Agent as ready
	agent.Status.Ready = true
	agent.Status.Status = "Ready"
	agent.Status.StatusDetail = "Ready for testing"
	agent.Status.ValidTools = func() []kubechain.ResolvedTool {
		tools := make([]kubechain.ResolvedTool, len(t.Tools))
		for i, tool := range t.Tools {
			tools[i] = kubechain.ResolvedTool{
				Kind: "Tool",
				Name: tool,
			}
		}
		return tools
	}()
	Expect(t.client.Status().Update(context.Background(), agent)).To(Succeed())

}

func (t *TestScopedAgent) GetAgent() *kubechain.Agent {
	agent := &kubechain.Agent{}
	Expect(t.client.Get(context.Background(), types.NamespacedName{Name: t.Name, Namespace: "default"}, agent)).To(Succeed())
	return agent
}

func (t *TestScopedAgent) Teardown() {
	agent := &kubechain.Agent{}
	err := t.client.Get(context.Background(), types.NamespacedName{Name: t.Name, Namespace: "default"}, agent)
	if err == nil {
		Expect(t.client.Delete(context.Background(), agent)).To(Succeed())
	}
}

type TestScopedLLM struct {
	Name      string
	Provider  string
	SecretRef string
	SecretKey string
	client    client.Client
}

func (t *TestScopedLLM) Setup(k8sClient client.Client) {
	t.client = k8sClient
	// Create a test LLM
	llm := &kubechain.LLM{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.Name,
			Namespace: "default",
		},
		Spec: kubechain.LLMSpec{
			Provider: t.Provider,
			APIKeyFrom: kubechain.APIKeySource{
				SecretKeyRef: kubechain.SecretKeyRef{
					Name: t.SecretRef,
					Key:  t.SecretKey,
				},
			},
		},
	}
	Expect(t.client.Create(context.Background(), llm)).To(Succeed())

	// Mark LLM as ready
	llm.Status.Ready = true
	llm.Status.Status = "Ready"
	llm.Status.StatusDetail = "Ready for testing"
	Expect(t.client.Status().Update(context.Background(), llm)).To(Succeed())
}

func (t *TestScopedLLM) GetLLM() *kubechain.LLM {
	llm := &kubechain.LLM{}
	Expect(t.client.Get(context.Background(), types.NamespacedName{Name: t.Name, Namespace: "default"}, llm)).To(Succeed())
	return llm
}

func (t *TestScopedLLM) Teardown() {
	llm := &kubechain.LLM{}
	err := t.client.Get(context.Background(), types.NamespacedName{Name: t.Name, Namespace: "default"}, llm)
	if err == nil {
		Expect(t.client.Delete(context.Background(), llm)).To(Succeed())
	}
}

type TestScopedTool struct {
	Name          string
	Description   string
	BuiltinName   string
	ParametersRaw string
	client        client.Client
}

var AddTool = TestScopedTool{
	Name:        "add",
	Description: "add two numbers",
	BuiltinName: "add",
	ParametersRaw: `{
		"type": "object",
		"properties": {
			"a": {
				"type": "number"
			},
			"b": {
				"type": "number"
			}
		},
		"required": ["a", "b"]
	}`,
}

func (t *TestScopedTool) Setup(k8sClient client.Client) {
	t.client = k8sClient
	// Create a test Tool
	tool := &kubechain.Tool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.Name,
			Namespace: "default",
		},
		Spec: kubechain.ToolSpec{
			Name:        t.Name,
			Description: t.Description,
			Execute: kubechain.ToolExecute{
				Builtin: &kubechain.BuiltinToolSpec{
					Name: t.BuiltinName,
				},
			},
			Parameters: runtime.RawExtension{
				Raw: []byte(t.ParametersRaw),
			},
		},
	}
	Expect(t.client.Create(context.Background(), tool)).To(Succeed())
}

func (t *TestScopedTool) GetTool() *kubechain.Tool {
	tool := &kubechain.Tool{}
	Expect(t.client.Get(context.Background(), types.NamespacedName{Name: t.Name, Namespace: "default"}, tool)).To(Succeed())
	return tool
}

func (t *TestScopedTool) Teardown() {
	tool := &kubechain.Tool{}
	err := t.client.Get(context.Background(), types.NamespacedName{Name: t.Name, Namespace: "default"}, tool)
	if err == nil {
		Expect(t.client.Delete(context.Background(), tool)).To(Succeed())
	}
}

type TestScopedTask struct {
	Name      string
	AgentName string
	Message   string
	client    client.Client
}

func (t *TestScopedTask) Setup(k8sClient client.Client) {
	t.client = k8sClient
	// Create a test Task
	task := &kubechain.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.Name,
			Namespace: "default",
		},
		Spec: kubechain.TaskSpec{
			AgentRef: kubechain.LocalObjectReference{
				Name: t.AgentName,
			},
			Message: t.Message,
		},
	}
	Expect(t.client.Create(context.Background(), task)).To(Succeed())

	// Mark Task as ready
	task.Status.Ready = true
	task.Status.Status = "Ready"
	task.Status.StatusDetail = "Agent validated successfully"
	Expect(t.client.Status().Update(context.Background(), task)).To(Succeed())
}

func (t *TestScopedTask) GetTask() *kubechain.Task {
	task := &kubechain.Task{}
	Expect(t.client.Get(context.Background(), types.NamespacedName{Name: t.Name, Namespace: "default"}, task)).To(Succeed())
	return task
}

func (t *TestScopedTask) Teardown() {
	task := &kubechain.Task{}
	err := t.client.Get(context.Background(), types.NamespacedName{Name: t.Name, Namespace: "default"}, task)
	if err == nil {
		Expect(t.client.Delete(context.Background(), task)).To(Succeed())
	}
}

type TestScopedTaskRun struct {
	Name     string
	TaskName string
	Client   client.Client
}

func (t *TestScopedTaskRun) Setup(k8sClient client.Client) {
	t.Client = k8sClient
	// Create a test TaskRun
	taskRun := &kubechain.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.Name,
			Namespace: "default",
		},
		Spec: kubechain.TaskRunSpec{
			TaskRef: kubechain.LocalObjectReference{
				Name: t.TaskName,
			},
		},
	}
	Expect(t.Client.Create(context.Background(), taskRun)).To(Succeed())
}

type TaskRunSetupInputs struct {
	Spec   *kubechain.TaskRunSpec
	Status *kubechain.TaskRunStatus
}

func (t *TestScopedTaskRun) SetupWith(k8sClient client.Client, inputs TaskRunSetupInputs) {
	t.Client = k8sClient
	// Create a test TaskRun
	taskRun := &kubechain.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.Name,
			Namespace: "default",
		},
		Spec: kubechain.TaskRunSpec{
			TaskRef: kubechain.LocalObjectReference{
				Name: t.TaskName,
			},
		},
	}

	// Override spec if provided
	if inputs.Spec != nil {
		taskRun.Spec = *inputs.Spec
	}

	Expect(t.Client.Create(context.Background(), taskRun)).To(Succeed())

	// Update status if provided
	if inputs.Status != nil {
		statusUpdate := taskRun.DeepCopy()
		statusUpdate.Status = *inputs.Status
		Expect(t.Client.Status().Update(context.Background(), statusUpdate)).To(Succeed())
	}
}

func (t *TestScopedTaskRun) SetupWithSpec(k8sClient client.Client, spec kubechain.TaskRunSpec) {
	t.SetupWith(k8sClient, TaskRunSetupInputs{Spec: &spec})
}

func (t *TestScopedTaskRun) SetupWithStatus(k8sClient client.Client, status kubechain.TaskRunStatus) {
	t.SetupWith(k8sClient, TaskRunSetupInputs{Status: &status})
}

func (t *TestScopedTaskRun) GetTaskRun() *kubechain.TaskRun {
	taskRun := &kubechain.TaskRun{}
	Expect(t.Client.Get(context.Background(), types.NamespacedName{Name: t.Name, Namespace: "default"}, taskRun)).To(Succeed())
	return taskRun
}

func (t *TestScopedTaskRun) Teardown() {
	taskRun := &kubechain.TaskRun{}
	err := t.Client.Get(context.Background(), types.NamespacedName{Name: t.Name, Namespace: "default"}, taskRun)
	if err == nil {
		Expect(t.Client.Delete(context.Background(), taskRun)).To(Succeed())
	}
}

type TestScopedSecret struct {
	Name   string
	Keys   map[string]string
	client client.Client
}

func (t *TestScopedSecret) Setup(k8sClient client.Client) {
	t.client = k8sClient
	// Create a test Secret
	data := make(map[string][]byte)
	for k, v := range t.Keys {
		data[k] = []byte(v)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.Name,
			Namespace: "default",
		},
		Data: data,
	}
	Expect(t.client.Create(context.Background(), secret)).To(Succeed())
}

func (t *TestScopedSecret) GetSecret() *corev1.Secret {
	secret := &corev1.Secret{}
	Expect(t.client.Get(context.Background(), types.NamespacedName{Name: t.Name, Namespace: "default"}, secret)).To(Succeed())
	return secret
}

func (t *TestScopedSecret) Teardown() {
	secret := &corev1.Secret{}
	err := t.client.Get(context.Background(), types.NamespacedName{Name: t.Name, Namespace: "default"}, secret)
	if err == nil {
		Expect(t.client.Delete(context.Background(), secret)).To(Succeed())
	}
}
