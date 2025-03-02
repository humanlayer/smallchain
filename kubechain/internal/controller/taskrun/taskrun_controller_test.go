package taskrun

import (
	"context"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubechain "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/humanlayer/smallchain/kubechain/internal/llmclient"
	"github.com/humanlayer/smallchain/kubechain/test/utils"
)

var _ = Describe("TaskRun Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-taskrun"
		const taskName = "test-task"
		const agentName = "test-agent"
		const taskRunName = "test-taskrun"

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		ctx := context.Background()
		var reconciler *TaskRunReconciler
		var eventRecorder *record.FakeRecorder
		var mockClient *llmclient.MockRawOpenAIClient

		testSecret := &utils.TestScopedSecret{
			Name: "test-secret",
			Keys: map[string]string{
				"api-key": "test-api-key",
			},
		}

		testLLM := &utils.TestScopedLLM{
			Name:      "test-llm",
			Provider:  "openai",
			SecretRef: "test-secret",
			SecretKey: "api-key",
		}

		testTool := utils.AddTool

		testTask := &utils.TestScopedTask{
			Name:      taskName,
			AgentName: agentName,
			Message:   "What state is San Francisco in?",
		}

		testAgent := &utils.TestScopedAgent{
			Name:         agentName,
			SystemPrompt: "you are a testing assistant",
			Tools:        []string{testTool.Name},
			LLM:          testLLM.Name,
		}

		testTaskRun := &utils.TestScopedTaskRun{
			Name:     taskRunName,
			TaskName: taskName,
			Client:   k8sClient,
		}

		BeforeEach(func() {
			By("Creating mock client and event recorder")
			mockClient = &llmclient.MockRawOpenAIClient{}
			eventRecorder = record.NewFakeRecorder(30)

			By("Setting up the reconciler")
			reconciler = &TaskRunReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
				newLLMClient: func(apiKey string) (llmclient.OpenAIClient, error) {
					return mockClient, nil
				},
			}

			By("Setting up the test secret")
			testSecret.Setup(k8sClient)

			By("Setting up the test LLM")
			testLLM.Setup(k8sClient)

			By("Setting up the test Tool")
			testTool.Setup(k8sClient)

			By("Setting up the test Agent")
			testAgent.Setup(k8sClient)

			By("Setting up the test Task")
			testTask.Setup(k8sClient)

		})

		AfterEach(func() {
			// Cleanup test resources
			By("Cleanup the test secret")
			testSecret.Teardown()

			By("Cleanup the test LLM")
			testLLM.Teardown()

			By("Cleanup the test Tool")
			testTool.Teardown()

			By("Cleanup the test Agent")
			testAgent.Teardown()

			By("Cleanup the test Task")
			testTask.Teardown()

			By("Cleanup the test TaskRun")
			testTaskRun.Teardown()
		})

		It("should progress through the taskrun lifecycle for a simple task with no tools", func() {
			By("creating the taskrun")
			testTaskRun.Setup(k8sClient)
			return

			By("reconciling the taskrun")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			By("checking taskrun moves to pending after first reconciliation")
			updatedTaskRun := &kubechain.TaskRun{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechain.TaskRunPhasePending))
			Expect(updatedTaskRun.Status.Status).To(Equal("Pending"))
			Expect(updatedTaskRun.Status.StatusDetail).To(Equal("Initializing"))
			Expect(updatedTaskRun.Status.Ready).To(BeFalse())

			By("reconciling a second time")
			result, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking taskrun moves to ready after second reconciliation")
			Expect(result.Requeue).To(BeTrue())
			updatedTaskRun = &kubechain.TaskRun{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Ready).To(BeTrue())
			Expect(updatedTaskRun.Status.Status).To(Equal("Ready"))
			Expect(updatedTaskRun.Status.StatusDetail).To(Equal("Ready to send to LLM"))
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseReadyForLLM))
			Expect(updatedTaskRun.Status.Error).To(BeEmpty())
			Expect(updatedTaskRun.Status.ContextWindow).To(HaveLen(2))
			Expect(updatedTaskRun.Status.ContextWindow[0].Role).To(Equal("system"))
			Expect(updatedTaskRun.Status.ContextWindow[0].Content).To(Equal(testAgent.SystemPrompt))
			Expect(updatedTaskRun.Status.ContextWindow[1].Role).To(Equal("user"))
			Expect(updatedTaskRun.Status.ContextWindow[1].Content).To(Equal(testTask.Message))

			By("checking that validation success event was created")
			Eventually(func() bool {
				select {
				case event := <-eventRecorder.Events:
					return strings.Contains(event, "ValidationSucceeded")
				default:
					return false
				}
			}, 5*time.Second, 100*time.Millisecond).Should(BeTrue(), "Expected to find validation success event")

			By("reconciling the taskrun again")
			mockClient.Response = &kubechain.Message{
				Role:    "assistant",
				Content: "San Francisco is in California",
			}
			// Second reconciliation - should send to LLM and get response
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking post-LLM taskrun status")
			err = k8sClient.Get(ctx, types.NamespacedName{Name: taskRunName, Namespace: "default"}, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Ready).To(BeTrue())
			Expect(updatedTaskRun.Status.Status).To(Equal("Ready"))
			Expect(updatedTaskRun.Status.StatusDetail).To(Equal("LLM final response received"))
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseFinalAnswer))
			Expect(updatedTaskRun.Status.ContextWindow).To(HaveLen(3))
			Expect(updatedTaskRun.Status.ContextWindow[2].Role).To(Equal("assistant"))
			Expect(updatedTaskRun.Status.ContextWindow[2].Content).To(Equal(mockClient.Response.Content))

			By("checking that LLM event was created")
			Eventually(func() bool {
				select {
				case event := <-eventRecorder.Events:
					return strings.Contains(event, "LLMFinalAnswer")
				default:
					return false
				}
			}, 5*time.Second, 100*time.Millisecond).Should(BeTrue(), "Expected to find LLM final answer event")
		})

		It("should clear error field when entering ready state", func() {
			By("creating a taskrun with an error")
			testTaskRun.SetupWithStatus(k8sClient, kubechain.TaskRunStatus{
				Error: "previous error that should be cleared",
				Phase: kubechain.TaskRunPhasePending,
			})

			By("reconciling the taskrun")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking the taskrun status")
			updatedTaskRun := &kubechain.TaskRun{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Ready).To(BeTrue())
			Expect(updatedTaskRun.Status.Status).To(Equal("Ready"))
			Expect(updatedTaskRun.Status.StatusDetail).To(Equal("Ready to send to LLM"))
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechain.TaskRunPhaseReadyForLLM))
			Expect(updatedTaskRun.Status.Error).To(BeEmpty(), "Error field should be cleared")
		})

		It("should fail when task doesn't exist", func() {
			By("creating the taskrun with non-existent task")
			testTaskRun.SetupWithSpec(k8sClient, kubechain.TaskRunSpec{
				TaskRef: kubechain.LocalObjectReference{
					Name: "nonexistent-task",
				},
			})

			By("reconciling the taskrun")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred())

			By("checking the taskrun status")
			updatedTaskRun := &kubechain.TaskRun{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTaskRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTaskRun.Status.Ready).To(BeFalse())
			Expect(updatedTaskRun.Status.Status).To(Equal("Error"))
			Expect(updatedTaskRun.Status.StatusDetail).To(ContainSubstring("failed to get Task"))
			Expect(updatedTaskRun.Status.Error).To(ContainSubstring("failed to get Task"))

			By("checking that a failure event was created")
			Eventually(func() bool {
				select {
				case event := <-eventRecorder.Events:
					return strings.Contains(event, "ValidationFailed")
				default:
					return false
				}
			}, 5*time.Second, 100*time.Millisecond).Should(BeTrue(), "Expected to find failure event")
		})

		It("should set pending status when task exists but is not ready", func() {
			By("creating a task that is not ready")
			unreadyTask := &kubechain.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unready-task",
					Namespace: "default",
				},
				Spec: kubechain.TaskSpec{
					AgentRef: kubechain.LocalObjectReference{
						Name: agentName,
					},
					Message: "Test input",
				},
				Status: kubechain.TaskStatus{
					Ready: false,
				},
			}
			Expect(k8sClient.Create(ctx, unreadyTask)).To(Succeed())

			By("creating the taskrun referencing the unready task")
			testTaskRun.SetupWith(k8sClient, utils.TaskRunSetupInputs{
				Spec: &kubechain.TaskRunSpec{
					TaskRef: kubechain.LocalObjectReference{
						Name: "unready-task",
					},
				},
				Status: &kubechain.TaskRunStatus{
					Phase: kubechain.TaskRunPhasePending,
				},
			})

			By("reconciling the taskrun")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(5 * time.Second))

			By("checking the taskrun status")
			updatedTaskRun := &kubechain.TaskRun{}
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
					return strings.Contains(event, "Waiting for task to become ready")
				default:
					return false
				}
			}, 5*time.Second, 100*time.Millisecond).Should(BeTrue(), "Expected to find waiting event")
		})

		// 	It("should pass tools correctly to OpenAI and handle tool calls", func() {
		// 		By("creating the taskrun")
		// 		taskRun := &v1alpha1.TaskRun{
		// 			ObjectMeta: metav1.ObjectMeta{
		// 				Name:      taskRunName,
		// 				Namespace: "default",
		// 			},
		// 			Spec: v1alpha1.TaskRunSpec{
		// 				TaskRef: v1alpha1.LocalObjectReference{
		// 					Name: taskName,
		// 				},
		// 			},
		// 		}
		// 		Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())

		// 		By("creating a mock OpenAI client that validates tools and returns tool calls")
		// 		mockClient := &llmclient.MockOpenAIClient{sistant},
		// 			Response: &openai.ChatCompletionMessage{
		// 				ToolCalls: []openai.ChatCompletionMessageToolCall{
		// 					{
		// 						ID:   "call_123",
		// 						Type: openai.ChatCompletionMessageToolCallTypeFunction,
		// 						Function: openai.ChatCompletionMessageToolCallFunction{
		// 							Name:      "add",
		// 							Arguments: `{"a": 1, "b": 2}`,
		// 						},
		// 					},
		// 				},
		// 			},
		// 			ValidateTools: func(tools []openai.ChatCompletionToolParam) error {
		// 				Expect(tools).To(HaveLen(1))
		// 				Expect(tools[0].Type.Value).To(Equal(openai.ChatCompletionToolTypeFunction))
		// 				Expect(tools[0].Function.Value.Name.Value).To(Equal("add"))
		// 				Expect(tools[0].Function.Value.Description.Value).To(Equal("add two numbers"))
		// 				// Verify parameters were passed correctly
		// 				Expect(tools[0].Function.Value.Parameters.Value).To(Equal(openai.FunctionParameters{
		// 					"type": "object",
		// 					"properties": map[string]interface{}{
		// 						"a": map[string]interface{}{
		// 							"type": "number",
		// 						},
		// 						"b": map[string]interface{}{
		// 							"type": "number",
		// 						},
		// 					},
		// 					"required": []interface{}{"a", "b"},
		// 				}))
		// 				return nil
		// 			},
		// 		}

		// 		By("reconciling the taskrun")
		// 		eventRecorder := record.NewFakeRecorder(10)
		// 		reconciler := &TaskRunReconciler{
		// 			Client:   k8sClient,
		// 			Scheme:   k8sClient.Scheme(),
		// 			recorder: eventRecorder,
		// 			newLLMClient: func(apiKey string) (llmclient.OpenAIClient, error) {
		// 				return mockClient, nil
		// 			},
		// 		}

		// 		// First reconciliation - should set ReadyForLLM phase
		// 		_, err := reconciler.Reconcile(ctx, reconcile.Request{
		// 			NamespacedName: types.NamespacedName{
		// 				Name:      taskRunName,
		// 				Namespace: "default",
		// 			},
		// 		})
		// 		Expect(err).NotTo(HaveOccurred())

		// 		By("checking initial taskrun status")
		// 		updatedTaskRun := &v1alpha1.TaskRun{}
		// 		err = k8sClient.Get(ctx, types.NamespacedName{Name: taskRunName, Namespace: "default"}, updatedTaskRun)
		// 		Expect(err).NotTo(HaveOccurred())
		// 		Expect(updatedTaskRun.Status.Phase).To(Equal(v1alpha1.TaskRunPhaseReadyForLLM))
		// 		Expect(updatedTaskRun.Status.ContextWindow).To(HaveLen(2)) // System + User message

		// 		By("reconciling the taskrun again")
		// 		// Second reconciliation - should send to LLM and get tool calls
		// 		_, err = reconciler.Reconcile(ctx, reconcile.Request{
		// 			NamespacedName: types.NamespacedName{
		// 				Name:      taskRunName,
		// 				Namespace: "default",
		// 			},
		// 		})
		// 		Expect(err).NotTo(HaveOccurred())

		// 		By("checking that the taskrun status was updated correctly")
		// 		err = k8sClient.Get(ctx, types.NamespacedName{Name: taskRunName, Namespace: "default"}, updatedTaskRun)
		// 		Expect(err).NotTo(HaveOccurred())
		// 		Expect(updatedTaskRun.Status.Phase).To(Equal(v1alpha1.TaskRunPhaseToolCallsPending))
		// 		Expect(updatedTaskRun.Status.ContextWindow).To(HaveLen(3)) // System + User message + Assistant message with tool calls
		// 		Expect(updatedTaskRun.Status.ContextWindow[2].ToolCalls).To(HaveLen(1))
		// 		Expect(updatedTaskRun.Status.ContextWindow[2].ToolCalls[0].Function.Name).To(Equal("add"))
		// 		Expect(updatedTaskRun.Status.ContextWindow[2].ToolCalls[0].Function.Arguments).To(Equal(`{"a": 1, "b": 2}`))

		// 		By("checking that a TaskRunToolCall was created")
		// 		var taskRunToolCalls v1alpha1.TaskRunToolCallList
		// 		err = k8sClient.List(ctx, &taskRunToolCalls, client.InNamespace("default"))
		// 		Expect(err).NotTo(HaveOccurred())
		// 		Expect(taskRunToolCalls.Items).To(HaveLen(1))
		// 		trtc := taskRunToolCalls.Items[0]
		// 		Expect(trtc.Spec.ToolRef.Name).To(Equal("add"))
		// 		Expect(trtc.Spec.Arguments).To(Equal(`{"a": 1, "b": 2}`))
		// 	})
		// })
	})
})
