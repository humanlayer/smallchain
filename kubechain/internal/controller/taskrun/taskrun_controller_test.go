package taskrun

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	kubechain "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/humanlayer/smallchain/kubechain/internal/llmclient"
	"github.com/humanlayer/smallchain/kubechain/test/utils"
	. "github.com/humanlayer/smallchain/kubechain/test/utils"
)

type TestSecret struct {
	name   string
	secret *corev1.Secret
}

func (t *TestSecret) Setup(ctx context.Context) *corev1.Secret {
	By("creating the secret")
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
		},
	}
	err := k8sClient.Create(ctx, secret)
	Expect(err).NotTo(HaveOccurred())
	t.secret = secret
	return secret
}

func (t *TestSecret) Teardown(ctx context.Context) {
	By("deleting the secret")
	Expect(k8sClient.Delete(ctx, t.secret)).To(Succeed())
}

var testSecret = &TestSecret{
	name: "test-secret",
}

type TestLLM struct {
	name string
	llm  *kubechain.LLM
}

func (t *TestLLM) Setup(ctx context.Context) *kubechain.LLM {
	By("creating the llm")
	llm := &kubechain.LLM{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
		},
		Spec: kubechain.LLMSpec{
			Provider: "openai",
			APIKeyFrom: kubechain.APIKeySource{
				SecretKeyRef: kubechain.SecretKeyRef{
					Name: testSecret.name,
					Key:  "api-key",
				},
			},
		},
	}
	err := k8sClient.Create(ctx, llm)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, llm)).To(Succeed())
	t.llm = llm
	return llm
}

func (t *TestLLM) SetupWithStatus(ctx context.Context, status kubechain.LLMStatus) *kubechain.LLM {
	llm := t.Setup(ctx)
	llm.Status = status
	Expect(k8sClient.Status().Update(ctx, llm)).To(Succeed())
	t.llm = llm
	return llm
}

func (t *TestLLM) Teardown(ctx context.Context) {
	By("deleting the llm")
	Expect(k8sClient.Delete(ctx, t.llm)).To(Succeed())
}

var testLLM = &TestLLM{
	name: "test-llm",
}

type TestAgent struct {
	name       string
	llmName    string
	system     string
	mcpServers []kubechain.LocalObjectReference
	agent      *kubechain.Agent
}

func (t *TestAgent) Setup(ctx context.Context) *kubechain.Agent {
	By("creating the agent")
	agent := &kubechain.Agent{
		ObjectMeta: metav1.ObjectMeta{
			Name: t.name,

			Namespace: "default",
		},
		Spec: kubechain.AgentSpec{
			LLMRef: kubechain.LocalObjectReference{
				Name: t.llmName,
			},
			System:     t.system,
			MCPServers: t.mcpServers,
		},
	}
	err := k8sClient.Create(ctx, agent)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, agent)).To(Succeed())
	t.agent = agent
	return agent
}

func (t *TestAgent) SetupWithStatus(ctx context.Context, status kubechain.AgentStatus) *kubechain.Agent {
	agent := t.Setup(ctx)
	agent.Status = status
	Expect(k8sClient.Status().Update(ctx, agent)).To(Succeed())
	t.agent = agent
	return agent
}

func (t *TestAgent) Teardown(ctx context.Context) {
	By("deleting the agent")
	Expect(k8sClient.Delete(ctx, t.agent)).To(Succeed())
}

var testAgent = &TestAgent{
	name:       "test-agent",
	llmName:    testLLM.name,
	system:     "you are a testing assistant",
	mcpServers: []kubechain.LocalObjectReference{},
}

type TestTask struct {
	name      string
	agentName string
	message   string
	task      *kubechain.Task
}

func (t *TestTask) Setup(ctx context.Context) *kubechain.Task {
	By("creating the task")
	task := &kubechain.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
		},
		Spec: kubechain.TaskSpec{
			AgentRef: kubechain.LocalObjectReference{
				Name: t.agentName,
			},
			Message: t.message,
		},
	}
	err := k8sClient.Create(ctx, task)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, task)).To(Succeed())
	t.task = task
	return task
}

func (t *TestTask) SetupWithStatus(ctx context.Context, status kubechain.TaskStatus) *kubechain.Task {
	task := t.Setup(ctx)
	task.Status = status
	Expect(k8sClient.Status().Update(ctx, task)).To(Succeed())
	t.task = task
	return task
}

func (t *TestTask) Teardown(ctx context.Context) {
	By("deleting the task")
	Expect(k8sClient.Delete(ctx, t.task)).To(Succeed())
}

type TestTaskRun struct {
	name     string
	taskName string
	taskRun  *kubechain.TaskRun
}

func (t *TestTaskRun) Setup(ctx context.Context) *kubechain.TaskRun {
	By("creating the taskrun")
	taskRun := &kubechain.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
		},
		Spec: kubechain.TaskRunSpec{
			TaskRef: kubechain.LocalObjectReference{
				Name: t.taskName,
			},
		},
	}
	err := k8sClient.Create(ctx, taskRun)
	Expect(err).NotTo(HaveOccurred())

	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, taskRun)).To(Succeed())
	t.taskRun = taskRun
	return taskRun
}

func (t *TestTaskRun) SetupWithStatus(ctx context.Context, status kubechain.TaskRunStatus) *kubechain.TaskRun {
	taskRun := t.Setup(ctx)
	taskRun.Status = status
	Expect(k8sClient.Status().Update(ctx, taskRun)).To(Succeed())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, taskRun)).To(Succeed())
	t.taskRun = taskRun
	return taskRun
}

func (t *TestTaskRun) Teardown(ctx context.Context) {
	By("deleting the taskrun")
	Expect(k8sClient.Delete(ctx, t.taskRun)).To(Succeed())
}

var testTask = &TestTask{
	name:      "test-task",
	agentName: "test-agent",
	message:   "what is the capital of the moon?",
}

var testTaskRun = &TestTaskRun{
	name:     "test-taskrun",
	taskName: testTask.name,
}

func setupSuiteObjects(ctx context.Context) (secret *corev1.Secret, llm *kubechain.LLM, agent *kubechain.Agent, task *kubechain.Task, teardown func()) {
	secret = testSecret.Setup(ctx)
	llm = testLLM.SetupWithStatus(ctx, kubechain.LLMStatus{
		Status: "Ready",
		Ready:  true,
	})
	agent = testAgent.SetupWithStatus(ctx, kubechain.AgentStatus{
		Status: "Ready",
		Ready:  true,
	})
	task = testTask.SetupWithStatus(ctx, kubechain.TaskStatus{
		Status: "Ready",
		Ready:  true,
	})
	teardown = func() {
		testSecret.Teardown(ctx)
		testLLM.Teardown(ctx)
		testAgent.Teardown(ctx)
		testTask.Teardown(ctx)
	}
	return secret, llm, agent, task, teardown
}

func reconciler() (*TaskRunReconciler, *record.FakeRecorder) {
	By("creating the reconciler")
	recorder := record.NewFakeRecorder(10)
	reconciler := &TaskRunReconciler{
		Client:   k8sClient,
		Scheme:   k8sClient.Scheme(),
		recorder: recorder,
	}
	return reconciler, recorder
}

var _ = Describe("TaskRun Controller", func() {
	Context("'' -> Initializing", func() {
		ctx := context.Background()
		It("moves to Initializing and sets a span context", func() {
			testTask.Setup(ctx)
			defer testTask.Teardown(ctx)

			taskRun := testTaskRun.Setup(ctx)
			defer testTaskRun.Teardown(ctx)

			reconciler, _ := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			By("checking the reconciler result")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			By("checking the taskrun status")
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseInitializing))
			Expect(taskRun.Status.SpanContext).NotTo(BeNil())
			Expect(taskRun.Status.SpanContext.TraceID).NotTo(BeEmpty())
			Expect(taskRun.Status.SpanContext.SpanID).NotTo(BeEmpty())
		})
	})
	Context("Initializing -> Error", func() {
		It("moves to error if the task is not found", func() {
			taskRun := testTaskRun.SetupWithStatus(ctx, kubechain.TaskRunStatus{
				Phase: kubechain.TaskRunPhaseInitializing,
			})
			defer testTaskRun.Teardown(ctx)

			reconciler, _ := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			// todo dont error if not found, don't requeue
			Expect(err).To(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseFailed))
			Expect(taskRun.Status.Error).To(Equal("Task \"test-task\" not found"))
		})
	})
	Context("Initializing -> Pending", func() {
		It("moves to pending if upstream task is not ready", func() {
			_ = testTask.SetupWithStatus(ctx, kubechain.TaskStatus{
				Status: kubechain.TaskStatusPending,
			})
			defer testTask.Teardown(ctx)

			taskRun := testTaskRun.SetupWithStatus(ctx, kubechain.TaskRunStatus{
				Phase: kubechain.TaskRunPhaseInitializing,
			})
			defer testTaskRun.Teardown(ctx)

			reconciler, recorder := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Second * 5))

			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhasePending))
			Expect(taskRun.Status.StatusDetail).To(ContainSubstring("Waiting for task \"test-task\" to become ready"))
			ExpectRecorder(recorder).ToEmitEventContaining("TaskNotReady")
		})
	})
	Context("Initializing -> ReadyForLLM", func() {
		It("moves to ReadyForLLM if the task is ready", func() {
			_, _, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			taskRun := testTaskRun.SetupWithStatus(ctx, kubechain.TaskRunStatus{
				Phase: kubechain.TaskRunPhaseInitializing,
			})
			defer testTaskRun.Teardown(ctx)

			reconciler, recorder := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseReadyForLLM))
			Expect(taskRun.Status.StatusDetail).To(ContainSubstring("Ready to send to LLM"))
			Expect(taskRun.Status.ContextWindow).To(HaveLen(2))
			Expect(taskRun.Status.ContextWindow[0].Role).To(Equal("system"))
			Expect(taskRun.Status.ContextWindow[0].Content).To(ContainSubstring(testAgent.system))
			Expect(taskRun.Status.ContextWindow[1].Role).To(Equal("user"))
			Expect(taskRun.Status.ContextWindow[1].Content).To(ContainSubstring(testTask.message))
			ExpectRecorder(recorder).ToEmitEventContaining("ValidationSucceeded")
		})
	})
	Context("Pending -> ReadyForLLM", func() {
		It("moves to ReadyForLLM if upstream dependencies are ready", func() {
			_, _, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			taskRun := testTaskRun.SetupWithStatus(ctx, kubechain.TaskRunStatus{
				Phase: kubechain.TaskRunPhasePending,
			})
			defer testTaskRun.Teardown(ctx)

			reconciler, recorder := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseReadyForLLM))
			Expect(taskRun.Status.StatusDetail).To(ContainSubstring("Ready to send to LLM"))
			Expect(taskRun.Status.ContextWindow).To(HaveLen(2))
			Expect(taskRun.Status.ContextWindow[0].Role).To(Equal("system"))
			Expect(taskRun.Status.ContextWindow[0].Content).To(ContainSubstring(testAgent.system))
			Expect(taskRun.Status.ContextWindow[1].Role).To(Equal("user"))
			Expect(taskRun.Status.ContextWindow[1].Content).To(ContainSubstring(testTask.message))
			ExpectRecorder(recorder).ToEmitEventContaining("ValidationSucceeded")
		})
	})
	Context("ReadyForLLM -> LLMFinalAnswer", func() {
		FIt("moves to LLMFinalAnswer after getting a response from the LLM", func() {
			_, _, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			taskRun := testTaskRun.SetupWithStatus(ctx, kubechain.TaskRunStatus{
				Phase: kubechain.TaskRunPhaseReadyForLLM,
				ContextWindow: []kubechain.Message{
					{
						Role:    "system",
						Content: testAgent.system,
					},
					{
						Role:    "user",
						Content: testTask.message,
					},
				},
			})
			defer testTaskRun.Teardown(ctx)

			reconciler, recorder := reconciler()
			mockLLMClient := &llmclient.MockRawOpenAIClient{
				Response: &v1alpha1.Message{
					Role:    "assistant",
					Content: "The moon is a natural satellite of the Earth and lacks any formal government or capital.",
				},
			}
			reconciler.newLLMClient = func(apiKey string) (llmclient.OpenAIClient, error) {
				return mockLLMClient, nil
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseFinalAnswer))
			Expect(taskRun.Status.StatusDetail).To(ContainSubstring("LLM final response received"))
			Expect(taskRun.Status.Output).To(Equal("The moon is a natural satellite of the Earth and lacks any formal government or capital."))
			Expect(taskRun.Status.ContextWindow).To(HaveLen(3))
			Expect(taskRun.Status.ContextWindow[2].Role).To(Equal("assistant"))
			Expect(taskRun.Status.ContextWindow[2].Content).To(ContainSubstring("The moon is a natural satellite of the Earth and lacks any formal government or capital."))
			ExpectRecorder(recorder).ToEmitEventContaining("LLMFinalAnswer")

			Expect(mockLLMClient.Calls).To(HaveLen(1))
			Expect(mockLLMClient.Calls[0].Messages).To(HaveLen(2))
			Expect(mockLLMClient.Calls[0].Messages[0].Role).To(Equal("system"))
			Expect(mockLLMClient.Calls[0].Messages[0].Content).To(ContainSubstring(testAgent.system))
			Expect(mockLLMClient.Calls[0].Messages[1].Role).To(Equal("user"))
			Expect(mockLLMClient.Calls[0].Messages[1].Content).To(ContainSubstring(testTask.message))
		})
	})
	Context("ReadyForLLM -> Error", func() {
		It("moves to Error if the LLM returns an error", func() {})
	})
	Context("Error -> ErrorBackoff", func() {
		It("moves to ErrorBackoff if the error is retryable", func() {})
	})
	Context("Error -> Error", func() {
		It("Stays in Error if the error is not retryable", func() {})
	})
	Context("ErrorBackoff -> ReadyForLLM", func() {
		It("moves to ReadyForLLM after the backoff period", func() {})
	})
	Context("ReadyForLLM -> ToolCallsPending", func() {
		It("moves to ToolCallsPending if the LLM returns tool calls", func() {
			// _, _, _, _, teardown := setupSuiteObjects(ctx)
			// defer teardown()

			// taskRun := testTaskRun.SetupWithStatus(ctx, kubechain.TaskRunStatus{
			// 	Phase: kubechain.TaskRunPhaseReadyForLLM,
			// })
			// defer testTaskRun.Teardown(ctx)

			// reconciler, recorder := reconciler()
			// mockLLMClient := &llmclient.MockRawOpenAIClient{
			// 	Response: &v1alpha1.Message{
			// 		Role: "assistant",
			// 		ToolCalls: []v1alpha1.ToolCall{
			// 			{
			// 				ID:       "1",
			// 				Function: v1alpha1.ToolCallFunction{Name: "fetch__fetch", Arguments: `{"url": "https://api.example.com/data"}`},
			// 			},
			// 		},
			// 	},
			// }

			// result, err := reconciler.Reconcile(ctx, reconcile.Request{
			// 	NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			// })
			// Expect(err).NotTo(HaveOccurred())
			// Expect(result.Requeue).To(BeTrue())
		})
	})
	Context("ToolCallsPending -> LLMFinalAnswer", func() {
		It("moves to LLMFinalAnswer if the LLM is ready", func() {})
	})
	Context("LLMFinalAnswer -> Completed", func() {
		It("moves to completed if the LLM final answer is received", func() {})
	})
	Context("When reconciling a resource", func() {
		const resourceName = "test-taskrun"
		const taskName = "test-task"
		const agentName = "test-agent"
		const taskRunName = "test-taskrun"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		BeforeEach(func() {
			// Create a test secret
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"api-key": []byte("test-api-key"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			// Create a test LLM
			llm := &kubechain.LLM{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-llm",
					Namespace: "default",
				},
				Spec: kubechain.LLMSpec{
					Provider: "openai",
					APIKeyFrom: kubechain.APIKeySource{
						SecretKeyRef: kubechain.SecretKeyRef{
							Name: "test-secret",
							Key:  "api-key",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, llm)).To(Succeed())

			// Mark LLM as ready
			llm.Status.Ready = true
			llm.Status.Status = StatusReady
			llm.Status.StatusDetail = "Ready for testing"
			Expect(k8sClient.Status().Update(ctx, llm)).To(Succeed())

			tool := &kubechain.Tool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "add",
					Namespace: "default",
				},
				Spec: kubechain.ToolSpec{
					Name:        "add",
					Description: "add two numbers",
					Execute: kubechain.ToolExecute{
						Builtin: &kubechain.BuiltinToolSpec{
							Name: "add",
						},
					},
					Parameters: runtime.RawExtension{
						Raw: []byte(`{
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
						}`),
					},
				},
			}
			Expect(k8sClient.Create(ctx, tool)).To(Succeed())

			// Create a test Agent
			agent := &kubechain.Agent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      agentName,
					Namespace: "default",
				},
				Spec: kubechain.AgentSpec{
					LLMRef: kubechain.LocalObjectReference{
						Name: "test-llm",
					},
					System: "you are a testing assistant",
					Tools: []kubechain.LocalObjectReference{
						{
							Name: "add",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, agent)).To(Succeed())

			// Mark Agent as ready
			agent.Status.Ready = true
			agent.Status.Status = StatusReady
			agent.Status.StatusDetail = "Ready for testing"
			agent.Status.ValidTools = []kubechain.ResolvedTool{
				{
					Kind: "Tool",
					Name: "add",
				},
			}
			Expect(k8sClient.Status().Update(ctx, agent)).To(Succeed())

			// Create a test Task
			task := &kubechain.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      taskName,
					Namespace: "default",
				},
				Spec: kubechain.TaskSpec{
					AgentRef: kubechain.LocalObjectReference{
						Name: agentName,
					},
					Message: "what is 2 + 2?",
				},
			}
			Expect(k8sClient.Create(ctx, task)).To(Succeed())

			// Mark Task as ready
			task.Status.Ready = true
			task.Status.Status = StatusReady
			task.Status.StatusDetail = "Agent validated successfully"
			Expect(k8sClient.Status().Update(ctx, task)).To(Succeed())
		})

		AfterEach(func() {
			// Cleanup test resources
			By("Cleanup the test secret")
			secret := &corev1.Secret{}
			var err error // Declare err at the start
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-secret", Namespace: "default"}, secret)
			if err == nil {
				Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
			}

			By("Cleanup the test LLM")
			llm := &kubechain.LLM{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-llm", Namespace: "default"}, llm)
			if err == nil {
				Expect(k8sClient.Delete(ctx, llm)).To(Succeed())
			}

			By("Cleanup the test Tool")
			tool := &kubechain.Tool{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "add", Namespace: "default"}, tool)
			if err == nil {
				Expect(k8sClient.Delete(ctx, tool)).To(Succeed())
			}

			By("Cleanup the test Agent")
			agent := &kubechain.Agent{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: agentName, Namespace: "default"}, agent)
			if err == nil {
				Expect(k8sClient.Delete(ctx, agent)).To(Succeed())
			}

			By("Cleanup the test Task")
			task := &kubechain.Task{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: taskName, Namespace: "default"}, task)
			if err == nil {
				Expect(k8sClient.Delete(ctx, task)).To(Succeed())
			}

			By("Cleanup the test TaskRun")
			taskRun := &kubechain.TaskRun{}
			err = k8sClient.Get(ctx, typeNamespacedName, taskRun)
			if err == nil {
				Expect(k8sClient.Delete(ctx, taskRun)).To(Succeed())
			}
		})

		It("should progress through phases correctly", func() {
			By("creating the taskrun")
			taskRun := &kubechain.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      taskRunName,
					Namespace: "default",
				},
				Spec: kubechain.TaskRunSpec{
					TaskRef: kubechain.LocalObjectReference{
						Name: taskName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())

			By("reconciling the taskrun")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := &TaskRunReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
				newLLMClient: func(apiKey string) (llmclient.OpenAIClient, error) {
					return &llmclient.MockRawOpenAIClient{}, nil
				},
			}

			// First reconciliation - should set ReadyForLLM phase
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      taskRunName,
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking initial taskrun status")
			updatedTaskRun := &kubechain.TaskRun{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: taskRunName, Namespace: "default"}, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Ready).To(BeTrue())
			Expect(updatedTaskRun.Status.Status).To(Equal(StatusReady))
			Expect(updatedTaskRun.Status.StatusDetail).To(Equal("Ready to send to LLM"))
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseReadyForLLM))

			By("checking that validation success event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationSucceeded")

			By("reconciling the taskrun again")
			// Second reconciliation - should send to LLM and get response
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      taskRunName,
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking final taskrun status")
			err = k8sClient.Get(ctx, types.NamespacedName{Name: taskRunName, Namespace: "default"}, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Ready).To(BeTrue())
			Expect(updatedTaskRun.Status.Status).To(Equal(StatusReady))
			Expect(updatedTaskRun.Status.StatusDetail).To(Equal("LLM final response received"))
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseFinalAnswer))

			By("checking that LLM final answer event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("LLMFinalAnswer")
		})

		// todo(dex) i think this is not needed anymore - check version history to restore it
		It("should clear error field when entering ready state", func() {})

		It("should pass tools correctly to OpenAI and handle tool calls", func() {
			By("creating the taskrun")
			taskRun := &kubechain.TaskRun{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kubechain/v1alpha1",
					Kind:       "TaskRun",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      taskRunName,
					Namespace: "default",
				},
				Spec: kubechain.TaskRunSpec{
					TaskRef: kubechain.LocalObjectReference{
						Name: taskName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())

			By("creating a mock OpenAI client that validates tools and returns tool calls")
			mockClient := &llmclient.MockRawOpenAIClient{
				Response: &kubechain.Message{
					Role: "assistant",
					ToolCalls: []kubechain.ToolCall{
						{
							ID:   "call_123",
							Type: "function",
							Function: kubechain.ToolCallFunction{
								Name:      "add",
								Arguments: `{"a": 1, "b": 2}`,
							},
						},
					},
				},
				ValidateTools: func(tools []llmclient.Tool) error {
					Expect(tools).To(HaveLen(1))
					Expect(tools[0].Type).To(Equal("function"))
					Expect(tools[0].Function.Name).To(Equal("add"))
					Expect(tools[0].Function.Description).To(Equal("add two numbers"))
					// Verify parameters were passed correctly
					Expect(tools[0].Function.Parameters).To(Equal(llmclient.ToolFunctionParameters{
						Type: "object",
						Properties: map[string]llmclient.ToolFunctionParameter{
							"a": {Type: "number"},
							"b": {Type: "number"},
						},
						Required: []string{"a", "b"},
					}))
					return nil
				},

				ValidateContextWindow: func(contextWindow []kubechain.Message) error {
					Expect(contextWindow).To(HaveLen(2))
					Expect(contextWindow[0].Role).To(Equal("system"))
					Expect(contextWindow[0].Content).To(Equal("you are a testing assistant"))
					Expect(contextWindow[1].Role).To(Equal("user"))
					Expect(contextWindow[1].Content).To(Equal("what is 2 + 2?"))

					return nil
				},
			}

			By("reconciling the taskrun")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := &TaskRunReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
				newLLMClient: func(apiKey string) (llmclient.OpenAIClient, error) {
					return mockClient, nil
				},
			}

			// First reconciliation - should set ReadyForLLM phase
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      taskRunName,
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking initial taskrun status")
			updatedTaskRun := &kubechain.TaskRun{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: taskRunName, Namespace: "default"}, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseReadyForLLM))
			Expect(updatedTaskRun.Status.ContextWindow).To(HaveLen(2)) // System + User message

			By("reconciling the taskrun again")
			// Second reconciliation - should send to LLM and get tool calls
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      taskRunName,
					Namespace: "default",
				},
			})

			Expect(err).NotTo(HaveOccurred())

			By("checking that the taskrun status was updated correctly")
			err = k8sClient.Get(ctx, types.NamespacedName{Name: taskRunName, Namespace: "default"}, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseToolCallsPending))
			Expect(updatedTaskRun.Status.ContextWindow).To(HaveLen(3)) // System + User message + Assistant message with tool calls
			Expect(updatedTaskRun.Status.ContextWindow[2].ToolCalls).To(HaveLen(1))
			Expect(updatedTaskRun.Status.ContextWindow[2].ToolCalls[0].Function.Name).To(Equal("add"))
			Expect(updatedTaskRun.Status.ContextWindow[2].ToolCalls[0].Function.Arguments).To(Equal(`{"a": 1, "b": 2}`))

			By("checking that TaskRunToolCalls were created")
			taskRunToolCallList := &kubechain.TaskRunToolCallList{}
			Expect(k8sClient.List(ctx, taskRunToolCallList, client.InNamespace("default"), client.MatchingLabels{"kubechain.humanlayer.dev/taskruntoolcall": taskRunName})).To(Succeed())
			Expect(taskRunToolCallList.Items).To(HaveLen(1))
			Expect(taskRunToolCallList.Items[0].ObjectMeta.Name).To(Equal("test-taskrun-toolcall-01"))
		})

		It("should keep the task run in the ToolCallsPending state when tool call is pending", func() {
			By("creating the taskrun")
			taskRun := &kubechain.TaskRun{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kubechain/v1alpha1",
					Kind:       "TaskRun",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      taskRunName,
					Namespace: "default",
				},
				Spec: kubechain.TaskRunSpec{
					TaskRef: kubechain.LocalObjectReference{
						Name: taskName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())

			By("setting up the taskrun with tool calls")
			statusUpdate := taskRun.DeepCopy()
			statusUpdate.Status.Phase = kubechain.TaskRunPhaseToolCallsPending
			Expect(k8sClient.Status().Update(ctx, statusUpdate)).To(Succeed())

			By("creating an incomplete tool call")
			taskrunToolCall := &kubechain.TaskRunToolCall{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kubechain.humanlayer.dev/v1alpha1",
					Kind:       "TaskRunToolCall",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-toolcall",
					Namespace: "default",
					Labels: map[string]string{
						"kubechain.humanlayer.dev/taskruntoolcall": taskRunName,
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubechain.humanlayer.dev/v1alpha1",
							Kind:       "TaskRun",
							Name:       taskRunName,
							UID:        taskRun.UID,
						},
					},
				},
				Spec: kubechain.TaskRunToolCallSpec{
					ToolRef: kubechain.LocalObjectReference{
						Name: "add",
					},
					TaskRunRef: kubechain.LocalObjectReference{
						Name: taskRunName,
					},
					Arguments: `{"a": 1, "b": 2}`,
				},
				Status: kubechain.TaskRunToolCallStatus{
					Status: "Running",
					Phase:  kubechain.TaskRunToolCallPhaseRunning, // Not Succeeded
				},
			}
			Expect(k8sClient.Create(ctx, taskrunToolCall)).To(Succeed())

			By("reconciling the taskrun")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := &TaskRunReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      taskRunName,
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking that the taskrun stays in ToolCallsPending phase")
			updatedTaskRun := &kubechain.TaskRun{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: taskRunName, Namespace: "default"}, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseToolCallsPending))
		})

		XIt("should set Error phase when LLM request fails", func() {
			uniqueSuffix := fmt.Sprintf("%d", time.Now().UnixNano())
			testTaskRunName := fmt.Sprintf("error-state-%s", uniqueSuffix)

			By("creating the taskrun")
			taskRun := &kubechain.TaskRun{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kubechain/v1alpha1",
					Kind:       "TaskRun",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      testTaskRunName,
					Namespace: "default",
				},
				Spec: kubechain.TaskRunSpec{
					TaskRef: kubechain.LocalObjectReference{
						Name: taskName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())

			By("setting up the taskrun with a conversation history including tool messages missing toolCallId")
			statusUpdate := taskRun.DeepCopy()
			statusUpdate.Status.Phase = kubechain.TaskRunPhaseReadyForLLM
			// Simulate conversation with system, user, assistant (with tool call), and tool response
			statusUpdate.Status.ContextWindow = []kubechain.Message{
				{
					Role:    "system",
					Content: "you are a testing assistant",
				},
				{
					Role:    "user",
					Content: "what is 2 + 2?",
				},
				{
					Role:    "assistant",
					Content: "",
					ToolCalls: []kubechain.ToolCall{
						{
							ID:   "call_123",
							Type: "function",
							Function: kubechain.ToolCallFunction{
								Name:      "add",
								Arguments: `{"a": 2, "b": 2}`,
							},
						},
					},
				},
				{
					Role:    "tool",
					Content: "4",
					// Missing ToolCallId - This should cause an error with the LLM
				},
			}
			Expect(k8sClient.Status().Update(ctx, statusUpdate)).To(Succeed())

			By("creating a mock OpenAI client that returns an error for the invalid request")
			mockClient := &llmclient.MockRawOpenAIClient{
				Error: fmt.Errorf(`request failed with status 400: {
					"error": {
						"message": "Missing parameter 'tool_call_id': messages with role 'tool' must have a 'tool_call_id'.",
						"type": "invalid_request_error",
						"param": "messages.[3].tool_call_id",
						"code": null
					}
				}`),
			}

			By("reconciling the taskrun")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := &TaskRunReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
				newLLMClient: func(apiKey string) (llmclient.OpenAIClient, error) {
					return mockClient, nil
				},
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testTaskRunName,
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking that the taskrun moved to Error phase with correct status")
			updatedTaskRun := &kubechain.TaskRun{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRunName, Namespace: "default"}, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())

			// This expectation should fail with the current code as the phase doesn't change to Failed or ErrorBackoff
			Expect(updatedTaskRun.Status.Phase).To(BeElementOf(
				kubechain.TaskRunPhaseFailed,
				kubechain.TaskRunPhaseErrorBackoff),
				"TaskRun should enter Failed or ErrorBackoff phase on LLM request failure")

			Expect(updatedTaskRun.Status.Status).To(Equal("Error"))
			Expect(updatedTaskRun.Status.Error).To(ContainSubstring("Missing parameter 'tool_call_id'"))
			Expect(updatedTaskRun.Status.StatusDetail).To(ContainSubstring("LLM request failed"))

			By("checking that an error event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("Error")
		})

		It("should correctly handle multi-message conversations with the LLM", func() {
			uniqueSuffix := fmt.Sprintf("%d", time.Now().UnixNano())
			testTaskRunName := fmt.Sprintf("multi-message-%s", uniqueSuffix)

			By("creating the taskrun")
			taskRun := &kubechain.TaskRun{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kubechain/v1alpha1",
					Kind:       "TaskRun",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      testTaskRunName,
					Namespace: "default",
				},
				Spec: kubechain.TaskRunSpec{
					TaskRef: kubechain.LocalObjectReference{
						Name: taskName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())

			By("setting up the taskrun with an existing conversation history")
			statusUpdate := taskRun.DeepCopy()
			statusUpdate.Status.Phase = kubechain.TaskRunPhaseReadyForLLM
			// Simulate an existing conversation with system, user, assistant, user
			statusUpdate.Status.ContextWindow = []kubechain.Message{
				{
					Role:    "system",
					Content: "you are a testing assistant",
				},
				{
					Role:    "user",
					Content: "what is 2 + 2?",
				},
				{
					Role:    "assistant",
					Content: "2 + 2 = 4",
				},
				{
					Role:    "user",
					Content: "what is 4 + 4?",
				},
			}
			Expect(k8sClient.Status().Update(ctx, statusUpdate)).To(Succeed())

			By("creating a mock OpenAI client that validates context window messages are passed correctly")
			mockClient := &llmclient.MockRawOpenAIClient{
				Response: &kubechain.Message{
					Role:    "assistant",
					Content: "4 + 4 = 8",
				},
				ValidateContextWindow: func(contextWindow []kubechain.Message) error {
					Expect(contextWindow).To(HaveLen(4), "All 4 messages should be sent to the LLM")

					// Verify all messages are present in the correct order
					Expect(contextWindow[0].Role).To(Equal("system"))
					Expect(contextWindow[0].Content).To(Equal("you are a testing assistant"))

					Expect(contextWindow[1].Role).To(Equal("user"))
					Expect(contextWindow[1].Content).To(Equal("what is 2 + 2?"))

					Expect(contextWindow[2].Role).To(Equal("assistant"))
					Expect(contextWindow[2].Content).To(Equal("2 + 2 = 4"))

					Expect(contextWindow[3].Role).To(Equal("user"))
					Expect(contextWindow[3].Content).To(Equal("what is 4 + 4?"))

					return nil
				},
			}

			By("reconciling the taskrun")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := &TaskRunReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
				newLLMClient: func(apiKey string) (llmclient.OpenAIClient, error) {
					return mockClient, nil
				},
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testTaskRunName,
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking that the taskrun moved to FinalAnswer phase")
			updatedTaskRun := &kubechain.TaskRun{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRunName, Namespace: "default"}, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseFinalAnswer))

			By("checking that the new assistant response was appended to the context window")
			Expect(updatedTaskRun.Status.ContextWindow).To(HaveLen(5))
			lastMessage := updatedTaskRun.Status.ContextWindow[4]
			Expect(lastMessage.Role).To(Equal("assistant"))
			Expect(lastMessage.Content).To(Equal("4 + 4 = 8"))
		})

		It("should transition to ReadyForLLM when all tool calls are complete", func() {
			uniqueSuffix := fmt.Sprintf("%d", time.Now().UnixNano())
			testTaskRunName := fmt.Sprintf("%s-%s", taskRunName, uniqueSuffix)
			testToolCallName := fmt.Sprintf("test-toolcall-%s", uniqueSuffix)

			By("creating the taskrun")
			taskRun := &kubechain.TaskRun{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kubechain/v1alpha1",
					Kind:       "TaskRun",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      testTaskRunName,
					Namespace: "default",
				},
				Spec: kubechain.TaskRunSpec{
					TaskRef: kubechain.LocalObjectReference{
						Name: taskName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())

			By("setting up the taskrun with tool calls")
			statusUpdate := taskRun.DeepCopy()
			statusUpdate.Status.Phase = kubechain.TaskRunPhaseToolCallsPending
			Expect(k8sClient.Status().Update(ctx, statusUpdate)).To(Succeed())

			By("creating a completed tool call")
			taskrunToolCall := &kubechain.TaskRunToolCall{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kubechain.humanlayer.dev/v1alpha1",
					Kind:       "TaskRunToolCall",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      testToolCallName,
					Namespace: "default",
					Labels: map[string]string{
						"kubechain.humanlayer.dev/taskruntoolcall": testTaskRunName,
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubechain.humanlayer.dev/v1alpha1",
							Kind:       "TaskRun",
							Name:       testTaskRunName,
							UID:        taskRun.UID,
						},
					},
				},
				Spec: kubechain.TaskRunToolCallSpec{
					ToolRef: kubechain.LocalObjectReference{
						Name: "add",
					},
					TaskRunRef: kubechain.LocalObjectReference{
						Name: testTaskRunName,
					},
					Arguments: `{"a": 1, "b": 2}`,
				},
				Status: kubechain.TaskRunToolCallStatus{
					Status: StatusReady,
					Result: "3",
					Phase:  kubechain.TaskRunToolCallPhaseSucceeded,
				},
			}
			Expect(k8sClient.Create(ctx, taskrunToolCall)).To(Succeed())

			By("verifying tool call was created with correct status")
			createdToolCall := &kubechain.TaskRunToolCall{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      testToolCallName,
			}, createdToolCall)).To(Succeed())

			createdToolCall.Status = kubechain.TaskRunToolCallStatus{
				Status: StatusReady,
				Result: "3",
				Phase:  kubechain.TaskRunToolCallPhaseSucceeded,
			}
			Expect(k8sClient.Status().Update(ctx, createdToolCall)).To(Succeed())

			updatedToolCall := &kubechain.TaskRunToolCall{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      testToolCallName,
			}, updatedToolCall)).To(Succeed())
			Expect(updatedToolCall.Status.Phase).To(Equal(kubechain.TaskRunToolCallPhaseSucceeded))

			By("reconciling the taskrun")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := &TaskRunReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testTaskRunName,
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking that the taskrun moved to ReadyForLLM phase")
			updatedTaskRun := &kubechain.TaskRun{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRunName, Namespace: "default"}, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseReadyForLLM))

			Expect(updatedTaskRun.Status.ContextWindow).NotTo(BeEmpty())
			toolMessage := updatedTaskRun.Status.ContextWindow[len(updatedTaskRun.Status.ContextWindow)-1]
			Expect(toolMessage.Role).To(Equal("tool"))
			Expect(toolMessage.Content).To(Equal("3"))
		})
	})
})
