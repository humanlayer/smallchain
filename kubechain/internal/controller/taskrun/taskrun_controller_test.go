package taskrun

import (
	"context"
	"fmt"
	"strings"
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

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/humanlayer/smallchain/kubechain/internal/llmclient"
	"github.com/humanlayer/smallchain/kubechain/test/utils"
)

type TestTask struct {
	name      string
	agentName string
	task      *kubechainv1alpha1.Task
}

func (t *TestTask) Setup(ctx context.Context) *kubechainv1alpha1.Task {
	task := &kubechainv1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
		},
		Spec: kubechainv1alpha1.TaskSpec{
			AgentRef: kubechainv1alpha1.LocalObjectReference{
				Name: t.agentName,
			},
			Message: "what is the capital of the moon?",
		},
	}
	err := k8sClient.Create(ctx, task)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, task)).To(Succeed())
	t.task = task
	return task
}

func (t *TestTask) Teardown(ctx context.Context) {
	Expect(k8sClient.Delete(ctx, t.task)).To(Succeed())
}

type TestTaskRun struct {
	name     string
	taskName string
	taskRun  *kubechainv1alpha1.TaskRun
}

func (t *TestTaskRun) Setup(ctx context.Context) *kubechainv1alpha1.TaskRun {
	taskRun := &kubechainv1alpha1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
		},
		Spec: kubechainv1alpha1.TaskRunSpec{
			TaskRef: kubechainv1alpha1.LocalObjectReference{
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

func (t *TestTaskRun) Teardown(ctx context.Context) {
	Expect(k8sClient.Delete(ctx, t.taskRun)).To(Succeed())
}

var testTask = &TestTask{
	name:      "test-task",
	agentName: "test-agent",
}

var testTaskRun = &TestTaskRun{
	name:     "test-taskrun",
	taskName: testTask.name,
}

func reconciler() (*TaskRunReconciler, *record.FakeRecorder) {
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
		FIt("moves to Initializing and sets a span context", func() {
			testTask.Setup(ctx)
			defer testTask.Teardown(ctx)

			taskRun := testTaskRun.Setup(ctx)
			defer testTaskRun.Teardown(ctx)

			reconciler, _ := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())

			Expect(taskRun.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunPhaseInitializing))
			Expect(taskRun.Status.SpanContext).NotTo(BeNil())
			Expect(taskRun.Status.SpanContext.TraceID).NotTo(BeEmpty())
			Expect(taskRun.Status.SpanContext.SpanID).NotTo(BeEmpty())
		})
	})
	Context("Initializing -> Pending", func() {
		It("moves to pending if upstream dependencies are not ready", func() {})
	})
	Context("Pending -> ReadyForLLM", func() {
		It("moves to ReadyForLLM if upstream dependencies are ready", func() {})
	})
	Context("ReadyForLLM -> LLMFinalAnswer", func() {
		It("moves to LLMFinalAnswer if the LLM is ready", func() {})
	})
	Context("ReadyForLLM -> ToolCallsPending", func() {
		It("moves to ToolCallsPending if the LLM is not ready", func() {})
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
			llm := &kubechainv1alpha1.LLM{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-llm",
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.LLMSpec{
					Provider: "openai",
					APIKeyFrom: kubechainv1alpha1.APIKeySource{
						SecretKeyRef: kubechainv1alpha1.SecretKeyRef{
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

			tool := &kubechainv1alpha1.Tool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "add",
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.ToolSpec{
					Name:        "add",
					Description: "add two numbers",
					Execute: kubechainv1alpha1.ToolExecute{
						Builtin: &kubechainv1alpha1.BuiltinToolSpec{
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
			agent := &kubechainv1alpha1.Agent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      agentName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.AgentSpec{
					LLMRef: kubechainv1alpha1.LocalObjectReference{
						Name: "test-llm",
					},
					System: "you are a testing assistant",
					Tools: []kubechainv1alpha1.LocalObjectReference{
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
			agent.Status.ValidTools = []kubechainv1alpha1.ResolvedTool{
				{
					Kind: "Tool",
					Name: "add",
				},
			}
			Expect(k8sClient.Status().Update(ctx, agent)).To(Succeed())

			// Create a test Task
			task := &kubechainv1alpha1.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      taskName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.TaskSpec{
					AgentRef: kubechainv1alpha1.LocalObjectReference{
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
			llm := &kubechainv1alpha1.LLM{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-llm", Namespace: "default"}, llm)
			if err == nil {
				Expect(k8sClient.Delete(ctx, llm)).To(Succeed())
			}

			By("Cleanup the test Tool")
			tool := &kubechainv1alpha1.Tool{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "add", Namespace: "default"}, tool)
			if err == nil {
				Expect(k8sClient.Delete(ctx, tool)).To(Succeed())
			}

			By("Cleanup the test Agent")
			agent := &kubechainv1alpha1.Agent{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: agentName, Namespace: "default"}, agent)
			if err == nil {
				Expect(k8sClient.Delete(ctx, agent)).To(Succeed())
			}

			By("Cleanup the test Task")
			task := &kubechainv1alpha1.Task{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: taskName, Namespace: "default"}, task)
			if err == nil {
				Expect(k8sClient.Delete(ctx, task)).To(Succeed())
			}

			By("Cleanup the test TaskRun")
			taskRun := &kubechainv1alpha1.TaskRun{}
			err = k8sClient.Get(ctx, typeNamespacedName, taskRun)
			if err == nil {
				Expect(k8sClient.Delete(ctx, taskRun)).To(Succeed())
			}
		})

		It("should progress through phases correctly", func() {
			By("creating the taskrun")
			taskRun := &kubechainv1alpha1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      taskRunName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.TaskRunSpec{
					TaskRef: kubechainv1alpha1.LocalObjectReference{
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
			updatedTaskRun := &kubechainv1alpha1.TaskRun{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: taskRunName, Namespace: "default"}, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Ready).To(BeTrue())
			Expect(updatedTaskRun.Status.Status).To(Equal(StatusReady))
			Expect(updatedTaskRun.Status.StatusDetail).To(Equal("Ready to send to LLM"))
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunPhaseReadyForLLM))

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
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunPhaseFinalAnswer))

			By("checking that LLM final answer event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("LLMFinalAnswer")
		})

		It("should clear error field when entering ready state", func() {
			By("creating a taskrun with an error")
			taskRun := &kubechainv1alpha1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      taskRunName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.TaskRunSpec{
					TaskRef: kubechainv1alpha1.LocalObjectReference{
						Name: taskName,
					},
				},
				Status: kubechainv1alpha1.TaskRunStatus{
					Error: "previous error that should be cleared",
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())

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

			By("checking the taskrun status")
			updatedTaskRun := &kubechainv1alpha1.TaskRun{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: taskRunName, Namespace: "default"}, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Ready).To(BeTrue())
			Expect(updatedTaskRun.Status.Status).To(Equal(StatusReady))
			Expect(updatedTaskRun.Status.StatusDetail).To(Equal("Ready to send to LLM"))
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunPhaseReadyForLLM))
			Expect(updatedTaskRun.Status.Error).To(BeEmpty(), "Error field should be cleared")
		})

		It("should fail when task doesn't exist", func() {
			By("creating the taskrun with non-existent task")
			taskRun := &kubechainv1alpha1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.TaskRunSpec{
					TaskRef: kubechainv1alpha1.LocalObjectReference{
						Name: "nonexistent-task",
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
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred())

			By("checking the taskrun status")
			updatedTaskRun := &kubechainv1alpha1.TaskRun{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Ready).To(BeFalse())
			Expect(updatedTaskRun.Status.Status).To(Equal("Error"))
			Expect(updatedTaskRun.Status.StatusDetail).To(ContainSubstring("failed to get Task"))
			Expect(updatedTaskRun.Status.Error).To(ContainSubstring("failed to get Task"))

			By("checking that a failure event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationFailed")
		})

		It("should set pending status when task exists but is not ready", func() {
			By("creating a task that is not ready")
			unreadyTask := &kubechainv1alpha1.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unready-task",
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.TaskSpec{
					AgentRef: kubechainv1alpha1.LocalObjectReference{
						Name: agentName,
					},
					Message: "Test input",
				},
				Status: kubechainv1alpha1.TaskStatus{
					Ready: false,
				},
			}
			Expect(k8sClient.Create(ctx, unreadyTask)).To(Succeed())

			By("creating the taskrun referencing the unready task")
			taskRun := &kubechainv1alpha1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.TaskRunSpec{
					TaskRef: kubechainv1alpha1.LocalObjectReference{
						Name: "unready-task",
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
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(5 * time.Second))

			By("checking the taskrun status")
			updatedTaskRun := &kubechainv1alpha1.TaskRun{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Ready).To(BeFalse())
			Expect(updatedTaskRun.Status.Status).To(Equal("Pending"))
			Expect(updatedTaskRun.Status.StatusDetail).To(ContainSubstring("Waiting for task"))
			Expect(updatedTaskRun.Status.Error).To(BeEmpty())

			By("checking that a waiting event was created")
			Eventually(func() bool {
				select {
				case event := <-eventRecorder.Events:
					return strings.Contains(event, "Waiting")
				default:
					return false
				}
			}, 5*time.Second, 100*time.Millisecond).Should(BeTrue(), "Expected to find waiting event")
			By("marking the task as ready")
			unreadyTask.Status.Ready = true
			Expect(k8sClient.Status().Update(ctx, unreadyTask)).To(Succeed())

			By("reconciling the taskrun again")
			result, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(0 * time.Second))

			By("checking the taskrun status")
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Ready).To(BeTrue())
			Expect(updatedTaskRun.Status.Status).To(Equal(StatusReady))
			Expect(updatedTaskRun.Status.StatusDetail).To(ContainSubstring("Ready to send to LLM"))
			Expect(updatedTaskRun.Status.Error).To(BeEmpty())
		})

		It("should pass tools correctly to OpenAI and handle tool calls", func() {
			By("creating the taskrun")
			taskRun := &kubechainv1alpha1.TaskRun{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kubechain/v1alpha1",
					Kind:       "TaskRun",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      taskRunName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.TaskRunSpec{
					TaskRef: kubechainv1alpha1.LocalObjectReference{
						Name: taskName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())

			By("creating a mock OpenAI client that validates tools and returns tool calls")
			mockClient := &llmclient.MockRawOpenAIClient{
				Response: &kubechainv1alpha1.Message{
					Role: "assistant",
					ToolCalls: []kubechainv1alpha1.ToolCall{
						{
							ID:   "call_123",
							Type: "function",
							Function: kubechainv1alpha1.ToolCallFunction{
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

				ValidateContextWindow: func(contextWindow []kubechainv1alpha1.Message) error {
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
			updatedTaskRun := &kubechainv1alpha1.TaskRun{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: taskRunName, Namespace: "default"}, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunPhaseReadyForLLM))
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
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunPhaseToolCallsPending))
			Expect(updatedTaskRun.Status.ContextWindow).To(HaveLen(3)) // System + User message + Assistant message with tool calls
			Expect(updatedTaskRun.Status.ContextWindow[2].ToolCalls).To(HaveLen(1))
			Expect(updatedTaskRun.Status.ContextWindow[2].ToolCalls[0].Function.Name).To(Equal("add"))
			Expect(updatedTaskRun.Status.ContextWindow[2].ToolCalls[0].Function.Arguments).To(Equal(`{"a": 1, "b": 2}`))

			By("checking that TaskRunToolCalls were created")
			taskRunToolCallList := &kubechainv1alpha1.TaskRunToolCallList{}
			Expect(k8sClient.List(ctx, taskRunToolCallList, client.InNamespace("default"), client.MatchingLabels{"kubechain.humanlayer.dev/taskruntoolcall": taskRunName})).To(Succeed())
			Expect(taskRunToolCallList.Items).To(HaveLen(1))
			Expect(taskRunToolCallList.Items[0].ObjectMeta.Name).To(Equal("test-taskrun-toolcall-01"))
		})

		It("should keep the task run in the ToolCallsPending state when tool call is pending", func() {
			By("creating the taskrun")
			taskRun := &kubechainv1alpha1.TaskRun{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kubechain/v1alpha1",
					Kind:       "TaskRun",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      taskRunName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.TaskRunSpec{
					TaskRef: kubechainv1alpha1.LocalObjectReference{
						Name: taskName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())

			By("setting up the taskrun with tool calls")
			statusUpdate := taskRun.DeepCopy()
			statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseToolCallsPending
			Expect(k8sClient.Status().Update(ctx, statusUpdate)).To(Succeed())

			By("creating an incomplete tool call")
			taskrunToolCall := &kubechainv1alpha1.TaskRunToolCall{
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
				Spec: kubechainv1alpha1.TaskRunToolCallSpec{
					ToolRef: kubechainv1alpha1.LocalObjectReference{
						Name: "add",
					},
					TaskRunRef: kubechainv1alpha1.LocalObjectReference{
						Name: taskRunName,
					},
					Arguments: `{"a": 1, "b": 2}`,
				},
				Status: kubechainv1alpha1.TaskRunToolCallStatus{
					Status: "Running",
					Phase:  kubechainv1alpha1.TaskRunToolCallPhaseRunning, // Not Succeeded
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
			updatedTaskRun := &kubechainv1alpha1.TaskRun{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: taskRunName, Namespace: "default"}, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunPhaseToolCallsPending))
		})

		XIt("should set Error phase when LLM request fails", func() {
			uniqueSuffix := fmt.Sprintf("%d", time.Now().UnixNano())
			testTaskRunName := fmt.Sprintf("error-state-%s", uniqueSuffix)

			By("creating the taskrun")
			taskRun := &kubechainv1alpha1.TaskRun{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kubechain/v1alpha1",
					Kind:       "TaskRun",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      testTaskRunName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.TaskRunSpec{
					TaskRef: kubechainv1alpha1.LocalObjectReference{
						Name: taskName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())

			By("setting up the taskrun with a conversation history including tool messages missing toolCallId")
			statusUpdate := taskRun.DeepCopy()
			statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseReadyForLLM
			// Simulate conversation with system, user, assistant (with tool call), and tool response
			statusUpdate.Status.ContextWindow = []kubechainv1alpha1.Message{
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
					ToolCalls: []kubechainv1alpha1.ToolCall{
						{
							ID:   "call_123",
							Type: "function",
							Function: kubechainv1alpha1.ToolCallFunction{
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
			updatedTaskRun := &kubechainv1alpha1.TaskRun{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRunName, Namespace: "default"}, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())

			// This expectation should fail with the current code as the phase doesn't change to Failed or ErrorBackoff
			Expect(updatedTaskRun.Status.Phase).To(BeElementOf(
				kubechainv1alpha1.TaskRunPhaseFailed,
				kubechainv1alpha1.TaskRunPhaseErrorBackoff),
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
			taskRun := &kubechainv1alpha1.TaskRun{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kubechain/v1alpha1",
					Kind:       "TaskRun",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      testTaskRunName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.TaskRunSpec{
					TaskRef: kubechainv1alpha1.LocalObjectReference{
						Name: taskName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())

			By("setting up the taskrun with an existing conversation history")
			statusUpdate := taskRun.DeepCopy()
			statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseReadyForLLM
			// Simulate an existing conversation with system, user, assistant, user
			statusUpdate.Status.ContextWindow = []kubechainv1alpha1.Message{
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
				Response: &kubechainv1alpha1.Message{
					Role:    "assistant",
					Content: "4 + 4 = 8",
				},
				ValidateContextWindow: func(contextWindow []kubechainv1alpha1.Message) error {
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
			updatedTaskRun := &kubechainv1alpha1.TaskRun{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRunName, Namespace: "default"}, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunPhaseFinalAnswer))

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
			taskRun := &kubechainv1alpha1.TaskRun{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kubechain/v1alpha1",
					Kind:       "TaskRun",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      testTaskRunName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.TaskRunSpec{
					TaskRef: kubechainv1alpha1.LocalObjectReference{
						Name: taskName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())

			By("setting up the taskrun with tool calls")
			statusUpdate := taskRun.DeepCopy()
			statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseToolCallsPending
			Expect(k8sClient.Status().Update(ctx, statusUpdate)).To(Succeed())

			By("creating a completed tool call")
			taskrunToolCall := &kubechainv1alpha1.TaskRunToolCall{
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
				Spec: kubechainv1alpha1.TaskRunToolCallSpec{
					ToolRef: kubechainv1alpha1.LocalObjectReference{
						Name: "add",
					},
					TaskRunRef: kubechainv1alpha1.LocalObjectReference{
						Name: testTaskRunName,
					},
					Arguments: `{"a": 1, "b": 2}`,
				},
				Status: kubechainv1alpha1.TaskRunToolCallStatus{
					Status: StatusReady,
					Result: "3",
					Phase:  kubechainv1alpha1.TaskRunToolCallPhaseSucceeded,
				},
			}
			Expect(k8sClient.Create(ctx, taskrunToolCall)).To(Succeed())

			By("verifying tool call was created with correct status")
			createdToolCall := &kubechainv1alpha1.TaskRunToolCall{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      testToolCallName,
			}, createdToolCall)).To(Succeed())

			createdToolCall.Status = kubechainv1alpha1.TaskRunToolCallStatus{
				Status: StatusReady,
				Result: "3",
				Phase:  kubechainv1alpha1.TaskRunToolCallPhaseSucceeded,
			}
			Expect(k8sClient.Status().Update(ctx, createdToolCall)).To(Succeed())

			updatedToolCall := &kubechainv1alpha1.TaskRunToolCall{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      testToolCallName,
			}, updatedToolCall)).To(Succeed())
			Expect(updatedToolCall.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhaseSucceeded))

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
			updatedTaskRun := &kubechainv1alpha1.TaskRun{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRunName, Namespace: "default"}, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunPhaseReadyForLLM))

			Expect(updatedTaskRun.Status.ContextWindow).NotTo(BeEmpty())
			toolMessage := updatedTaskRun.Status.ContextWindow[len(updatedTaskRun.Status.ContextWindow)-1]
			Expect(toolMessage.Role).To(Equal("tool"))
			Expect(toolMessage.Content).To(Equal("3"))
		})
	})
})
