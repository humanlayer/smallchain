package taskrun

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	kubechain "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/humanlayer/smallchain/kubechain/internal/llmclient"
	. "github.com/humanlayer/smallchain/kubechain/test/utils"
)

var _ = Describe("TaskRun Controller", func() {
	Context("'' -> Initializing", func() {
		ctx := context.Background()
		It("moves to Initializing and sets a span context", func() {
			testTask.Setup(ctx)
			defer testTask.Teardown(ctx)

			taskRun := testTaskRun.Setup(ctx)
			defer testTaskRun.Teardown(ctx)

			By("reconciling the taskrun")
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

			By("reconciling the taskrun")
			reconciler, _ := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			// todo dont error if not found, don't requeue
			Expect(err).To(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("checking the taskrun status")
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

			By("reconciling the taskrun")
			reconciler, _ := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Second * 5))

			By("checking the taskrun status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhasePending))
			Expect(taskRun.Status.StatusDetail).To(Equal("Waiting for task \"test-task\" to become ready"))
		})
	})

	Context("Pending -> ReadyForLLM", func() {
		It("moves to ReadyForLLM when task and agent are ready", func() {
			_, _, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			taskRun := testTaskRun.SetupWithStatus(ctx, kubechain.TaskRunStatus{
				Phase: kubechain.TaskRunPhasePending,
			})
			defer testTaskRun.Teardown(ctx)

			By("reconciling the taskrun")
			reconciler, _ := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			By("checking the taskrun status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseReadyForLLM))
			Expect(taskRun.Status.ContextWindow).To(HaveLen(2))
			Expect(taskRun.Status.ContextWindow[0].Role).To(Equal("system"))
			Expect(taskRun.Status.ContextWindow[0].Content).To(Equal(testAgent.system))
			Expect(taskRun.Status.ContextWindow[1].Role).To(Equal("user"))
			Expect(taskRun.Status.ContextWindow[1].Content).To(Equal(testTask.message))
		})
	})

	Context("ReadyForLLM -> FinalAnswer", func() {
		It("moves to FinalAnswer when the LLM provides a final answer", func() {
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

			By("creating a reconciler with a mock OpenAI client")
			reconciler, _ := reconciler()
			mockClient := &MockOpenAIClient{
				SendRequestFunc: func(ctx context.Context, messages []kubechain.Message, tools []llmclient.Tool) (*kubechain.Message, error) {
					return &kubechain.Message{
						Role:    "assistant",
						Content: "The moon does not have a capital.",
					}, nil
				},
			}
			reconciler.newLLMClient = func(apiKey string) (llmclient.OpenAIClient, error) {
				return mockClient, nil
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("checking the taskrun status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseFinalAnswer))
			Expect(taskRun.Status.Output).To(Equal("The moon does not have a capital."))
			Expect(taskRun.Status.ContextWindow).To(HaveLen(3))
			Expect(taskRun.Status.ContextWindow[2].Role).To(Equal("assistant"))
			Expect(taskRun.Status.ContextWindow[2].Content).To(Equal("The moon does not have a capital."))
		})
	})

	Context("ReadyForLLM -> ToolCallsPending", func() {
		It("moves to ToolCallsPending when the LLM requests tool usage", func() {
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

			By("creating a reconciler with a mock OpenAI client that returns tools")
			reconciler, _ := reconciler()
			mockClient := &MockOpenAIClient{
				SendRequestFunc: func(ctx context.Context, messages []kubechain.Message, tools []llmclient.Tool) (*kubechain.Message, error) {
					return &kubechain.Message{
						Role: "assistant",
						ToolCalls: []kubechain.ToolCall{
							{
								ID: "1",
								Function: kubechain.ToolCallFunction{
									Name:      "fetch__fetch",
									Arguments: `{"url": "https://api.example.com/data"}`,
								},
							},
						},
					}, nil
				},
			}
			reconciler.newLLMClient = func(apiKey string) (llmclient.OpenAIClient, error) {
				return mockClient, nil
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Second * 5))

			By("checking the taskrun status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseToolCallsPending))
			Expect(taskRun.Status.ToolCallRequestId).NotTo(BeEmpty())
			Expect(taskRun.Status.ContextWindow).To(HaveLen(3))
			Expect(taskRun.Status.ContextWindow[2].Role).To(Equal("assistant"))
			Expect(taskRun.Status.ContextWindow[2].ToolCalls).To(HaveLen(1))
			Expect(taskRun.Status.ContextWindow[2].ToolCalls[0].ID).To(Equal("1"))
			Expect(taskRun.Status.ContextWindow[2].ToolCalls[0].Function.Name).To(Equal("fetch__fetch"))

			By("checking that tool calls were created")
			var toolCallList kubechain.TaskRunToolCallList
			err = k8sClient.List(ctx, &toolCallList, client.InNamespace("default"),
				client.MatchingLabels{"kubechain.humanlayer.dev/toolcallrequest": taskRun.Status.ToolCallRequestId})
			Expect(err).NotTo(HaveOccurred())
			Expect(toolCallList.Items).To(HaveLen(1))
			Expect(toolCallList.Items[0].Spec.ToolCallId).To(Equal("1"))
			Expect(toolCallList.Items[0].Spec.ToolRef.Name).To(Equal("fetch__fetch"))
		})
	})

	Context("ToolCallsPending -> ToolCallsPending", func() {
		It("Stays in ToolCallsPending if the tool calls are not completed", func() {
			_, _, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			By("setting up the taskrun with a tool call pending")
			taskRun := testTaskRun.SetupWithStatus(ctx, kubechain.TaskRunStatus{
				Phase:             kubechain.TaskRunPhaseToolCallsPending,
				ToolCallRequestId: "test123",
			})
			defer testTaskRun.Teardown(ctx)

			testTaskRunToolCall.SetupWithStatus(ctx, kubechain.TaskRunToolCallStatus{
				Phase: kubechain.TaskRunToolCallPhasePending,
			})
			defer testTaskRunToolCall.Teardown(ctx)

			By("reconciling the taskrun")
			reconciler, _ := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Second * 5))

			By("checking the taskrun status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseToolCallsPending))
		})
	})
	Context("ToolCallsPending -> ReadyForLLM", func() {
		It("moves to ReadyForLLM if all tool calls are completed", func() {
			_, _, _, _, teardown := setupSuiteObjects(ctx)
			defer teardown()

			By("setting up the taskrun with a tool call pending")
			taskRun := testTaskRun.SetupWithStatus(ctx, kubechain.TaskRunStatus{
				Phase:             kubechain.TaskRunPhaseToolCallsPending,
				ToolCallRequestId: "test123",
				ContextWindow: []kubechain.Message{
					{
						Role:    "system",
						Content: testAgent.system,
					},
					{
						Role:    "user",
						Content: testTask.message,
					},
					{
						Role: "assistant",
						ToolCalls: []kubechain.ToolCall{
							{
								ID: "1",
								Function: kubechain.ToolCallFunction{
									Name:      "fetch__fetch",
									Arguments: `{"url": "https://api.example.com/data"}`,
								},
							},
						},
					},
				},
			})
			defer testTaskRun.Teardown(ctx)

			testTaskRunToolCall.SetupWithStatus(ctx, kubechain.TaskRunToolCallStatus{
				Phase:  kubechain.TaskRunToolCallPhaseSucceeded,
				Result: `{"data": "test-data"}`,
			})
			defer testTaskRunToolCall.Teardown(ctx)

			By("reconciling the taskrun")
			reconciler, _ := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: testTaskRun.name, Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			By("checking the taskrun status")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testTaskRun.name, Namespace: "default"}, taskRun)).To(Succeed())
			Expect(taskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseReadyForLLM))
			Expect(taskRun.Status.ContextWindow).To(HaveLen(4))
			Expect(taskRun.Status.ContextWindow[3].Role).To(Equal("tool"))
			Expect(taskRun.Status.ContextWindow[3].Content).To(Equal(`{"data": "test-data"}`))
		})
	})
})

// Mock OpenAI client for testing
type MockOpenAIClient struct {
	SendRequestFunc func(ctx context.Context, messages []kubechain.Message, tools []llmclient.Tool) (*kubechain.Message, error)
}

func (m *MockOpenAIClient) SendRequest(ctx context.Context, messages []kubechain.Message, tools []llmclient.Tool) (*kubechain.Message, error) {
	return m.SendRequestFunc(ctx, messages, tools)
}

// These tests are currently disabled to focus on the current implementation
var _ = PDescribe("TaskRun Controller", func() {
	Context("When reconciling a resource", func() {
		// Placeholder tests
		It("should progress through phases correctly", func() {})
		It("should clear error field when entering ready state", func() {})
		It("should pass tools correctly to OpenAI and handle tool calls", func() {})
		It("should keep the task run in the ToolCallsPending state when tool call is pending", func() {})
		It("should correctly handle multi-message conversations with the LLM", func() {})
		It("should transition to ReadyForLLM when all tool calls are complete", func() {})
	})
})
