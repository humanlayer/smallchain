package taskruntoolcall

import (
	"context"
	"time"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/humanlayer/smallchain/kubechain/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("TaskRunToolCall Controller", func() {
	Context("'' -> Pending", func() {
		It("moves to Pending:Initializing", func() {
			teardown := setupTestAddTool(ctx)
			defer teardown()

			taskRunToolCall := trtcForAddTool.Setup(ctx)

			By("reconciling the taskruntoolcall")
			reconciler, _ := reconciler()

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      taskRunToolCall.Name,
					Namespace: taskRunToolCall.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse()) // No requeue since initialization is complete

			By("checking the taskruntoolcall status was initialized")
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      taskRunToolCall.Name,
				Namespace: taskRunToolCall.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhasePending))
			Expect(updatedTRTC.Status.Status).To(Equal("Pending"))
			Expect(updatedTRTC.Status.StatusDetail).To(Equal("Initializing"))
			Expect(updatedTRTC.Status.StartTime).NotTo(BeNil())
		})
	})

	Context("Pending -> Succeeded", func() {
		It("moves to Succeeded after executing a simple function tool call", func() {
			ctx := context.Background()

			teardown := setupTestAddTool(ctx)
			defer teardown()

			// Create TaskRunToolCall with Pending status
			taskRunToolCall := trtcForAddTool.SetupWithStatus(ctx, kubechainv1alpha1.TaskRunToolCallStatus{
				Phase:        kubechainv1alpha1.TaskRunToolCallPhasePending,
				Status:       "Pending",
				StatusDetail: "Ready for execution",
				StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
			})

			By("reconciling the trtc")
			reconciler, recorder := reconciler()

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      taskRunToolCall.Name,
					Namespace: taskRunToolCall.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())

			By("checking the taskruntoolcall status has changed to Succeeded")
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      taskRunToolCall.Name,
				Namespace: taskRunToolCall.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhaseSucceeded))
			Expect(updatedTRTC.Status.Result).To(Equal("5")) // 2 + 3 = 5
			Expect(updatedTRTC.Status.Status).To(Equal("Ready"))
			Expect(updatedTRTC.Status.StatusDetail).To(Equal("Tool executed successfully"))

			By("checking that execution events were emitted")
			utils.ExpectRecorder(recorder).ToEmitEventContaining("ExecutionSucceeded")
		})
	})

	Context("Pending -> Failed", func() {
		It("fails when arguments are invalid", func() {
			teardown := setupTestAddTool(ctx)
			defer teardown()

			// Create TaskRunToolCall with Pending status but invalid arguments
			taskRunToolCall := &TestTaskRunToolCall{
				name:      "invalid-args-trtc",
				toolName:  addTool.name,
				arguments: `invalid json`, // Invalid JSON
			}
			defer taskRunToolCall.Teardown(ctx)

			trtc := taskRunToolCall.SetupWithStatus(ctx, kubechainv1alpha1.TaskRunToolCallStatus{
				Phase:        kubechainv1alpha1.TaskRunToolCallPhasePending,
				Status:       "Pending",
				StatusDetail: "Ready for execution",
				StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
			})

			By("reconciling the taskruntoolcall with invalid arguments")
			reconciler, recorder := reconciler()

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      trtc.Name,
					Namespace: trtc.Namespace,
				},
			})

			// We expect an error during reconciliation
			Expect(err).To(HaveOccurred())

			By("checking the taskruntoolcall status is set to Error")
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Status).To(Equal("Error"))
			Expect(updatedTRTC.Status.StatusDetail).To(Equal("Invalid arguments JSON"))
			Expect(updatedTRTC.Status.Error).NotTo(BeEmpty())

			By("checking that error events were emitted")
			utils.ExpectRecorder(recorder).ToEmitEventContaining("ExecutionFailed")
		})

	})

	// Tests for MCP tools without approval requirement
	Context("Pending -> Succeeded (MCP Tool)", func() {
		It("successfully executes an MCP tool without requiring approval", func() {
			// Setup MCP server without approval channel
			testSecret.Setup(ctx)
			mcpServer := &TestMCPServer{
				name:                   "test-mcp-no-approval",
				needsApproval:          false,
				approvalContactChannel: "",
			}
			mcpServer.SetupWithStatus(ctx, kubechainv1alpha1.MCPServerStatus{
				Connected: true,
				Status:    "Ready",
			})
			defer mcpServer.Teardown(ctx)

			// Setup MCP tool
			mcpTool := &TestMCPTool{
				name:        "test-mcp-no-approval-tool",
				mcpServer:   mcpServer.name,
				mcpToolName: "test-tool",
			}
			tool := mcpTool.SetupWithStatus(ctx, kubechainv1alpha1.ToolStatus{
				Ready:  true,
				Status: "Ready",
			})
			defer mcpTool.Teardown(ctx)

			// Create TaskRunToolCall with MCP tool reference
			taskRunToolCall := &TestTaskRunToolCall{
				name:      "test-mcp-no-approval-trtc",
				toolName:  tool.Spec.Name,
				arguments: `{"a": 2, "b": 3}`,
			}
			trtc := taskRunToolCall.SetupWithStatus(ctx, kubechainv1alpha1.TaskRunToolCallStatus{
				Phase:        kubechainv1alpha1.TaskRunToolCallPhasePending,
				Status:       "Pending",
				StatusDetail: "Ready for execution",
				StartTime:    &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
			})
			defer taskRunToolCall.Teardown(ctx)

			By("reconciling the taskruntoolcall that uses MCP tool without approval")
			reconciler, recorder := reconciler()

			reconciler.MCPManager = &MockMCPManager{
				NeedsApproval: false, // This test specifically doesn't want approval
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      trtc.Name,
					Namespace: trtc.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())

			By("checking the taskruntoolcall status is set to Succeeded")
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      trtc.Name,
				Namespace: trtc.Namespace,
			}, updatedTRTC)

			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhaseSucceeded))
			Expect(updatedTRTC.Status.Status).To(Equal("Ready"))
			Expect(updatedTRTC.Status.Result).To(Equal("5")) // From our mock implementation

			By("checking that appropriate events were emitted")
			utils.ExpectRecorder(recorder).ToEmitEventContaining("ExecutionSucceeded")
		})
	})
})

/*
var _ = Describe("TaskRunToolCall Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-taskruntoolcall"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		BeforeEach(func() {
			// Create test Tool for direct execution
			tool := &kubechainv1alpha1.Tool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "add",
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.ToolSpec{
					ToolType:    "function",
					Name:        "add",
					Description: "Add two numbers",
					Execute: kubechainv1alpha1.ToolExecute{
						Builtin: &kubechainv1alpha1.BuiltinToolSpec{
							Name: "add",
						},
					},
				},
			}
			_ = k8sClient.Delete(ctx, tool)
			time.Sleep(100 * time.Millisecond)
			Expect(k8sClient.Create(ctx, tool)).To(Succeed())

			// Mark Tool as ready
			tool.Status.Ready = true
			tool.Status.Status = "Ready"
			Expect(k8sClient.Status().Update(ctx, tool)).To(Succeed())
		})

		AfterEach(func() {
			// Cleanup test resources
			By("Cleanup the test Tool")
			tool := &kubechainv1alpha1.Tool{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "add", Namespace: "default"}, tool)
			if err == nil {
				Expect(k8sClient.Delete(ctx, tool)).To(Succeed())
			}

			By("Cleanup the test TaskRunToolCall")
			trtc := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, typeNamespacedName, trtc)
			if err == nil {
				Expect(k8sClient.Delete(ctx, trtc)).To(Succeed())
			}
		})

		It("should successfully execute a function tool call", func() {
			By("creating the taskruntoolcall")
			trtc := &kubechainv1alpha1.TaskRunToolCall{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.TaskRunToolCallSpec{
					TaskRunRef: kubechainv1alpha1.LocalObjectReference{
						Name: "parent-taskrun",
					},
					ToolRef: kubechainv1alpha1.LocalObjectReference{
						Name: "add",
					},
					Arguments: `{"a": 2, "b": 3}`,
				},
			}
			Expect(k8sClient.Create(ctx, trtc)).To(Succeed())

			By("reconciling the taskruntoolcall")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := &TaskRunToolCallReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			// First reconciliation - should initialize status
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Second reconciliation - should execute function
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking the taskruntoolcall status")
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTRTC)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhaseSucceeded))
			Expect(updatedTRTC.Status.Result).To(Equal("5"))
			Expect(updatedTRTC.Status.Status).To(Equal("Ready"))
			Expect(updatedTRTC.Status.StatusDetail).To(Equal("Tool executed successfully"))

			By("checking that execution events were emitted")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ExecutionSucceeded")
		})

		It("should fail with invalid arguments", func() {
			By("creating the taskruntoolcall with invalid JSON")
			trtc := &kubechainv1alpha1.TaskRunToolCall{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.TaskRunToolCallSpec{
					TaskRunRef: kubechainv1alpha1.LocalObjectReference{
						Name: "parent-taskrun",
					},
					ToolRef: kubechainv1alpha1.LocalObjectReference{
						Name: "add",
					},
					Arguments: `invalid json`,
				},
			}
			Expect(k8sClient.Create(ctx, trtc)).To(Succeed())

			By("reconciling the taskruntoolcall")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := &TaskRunToolCallReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			// First reconciliation - should initialize status
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Second reconciliation - should fail validation
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred())

			By("checking the taskruntoolcall status")
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTRTC)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Status).To(Equal("Error"))
			Expect(updatedTRTC.Status.StatusDetail).To(Equal("Invalid arguments JSON"))

			By("checking that a validation failed event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ExecutionFailed")
		})

		It("should transition to AwaitingHumanApproval when MCP tool's server has approval contact channel", func() {
			By("creating a contact channel")

			// Create a mock secret first
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"api-key": []byte("test-key"),
				},
			}
			_ = k8sClient.Delete(ctx, secret) // Delete if exists
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			contactChannel := &kubechainv1alpha1.ContactChannel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-contact-channel",
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.ContactChannelSpec{
					ChannelType: "slack",
					APIKeyFrom: kubechainv1alpha1.APIKeySource{
						SecretKeyRef: kubechainv1alpha1.SecretKeyRef{
							Name: "test-secret",
							Key:  "api-key",
						},
					},
					SlackConfig: &kubechainv1alpha1.SlackChannelConfig{
						ChannelOrUserID: "C12345678",
					},
				},
				Status: kubechainv1alpha1.ContactChannelStatus{
					Ready:  true,
					Status: "Ready",
				},
			}
			Expect(k8sClient.Create(ctx, contactChannel)).To(Succeed())

			By("creating an MCPServer with approval contact channel")
			mcpServer := &kubechainv1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-mcp-server",
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.MCPServerSpec{
					Transport: "stdio",
					ApprovalContactChannel: &kubechainv1alpha1.LocalObjectReference{
						Name: "test-contact-channel",
					},
				},
			}
			Expect(k8sClient.Create(ctx, mcpServer)).To(Succeed())

			By("creating an MCP tool")
			tool := &kubechainv1alpha1.Tool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-mcp-server-test-tool",
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.ToolSpec{
					ToolType:    "function",
					Name:        "test-mcp-server__test-tool",
					Description: "A tool that requires human approval",
					Execute: kubechainv1alpha1.ToolExecute{
						Builtin: &kubechainv1alpha1.BuiltinToolSpec{
							Name: "add",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, tool)).To(Succeed())

			By("creating the taskruntoolcall")
			trtc := &kubechainv1alpha1.TaskRunToolCall{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.TaskRunToolCallSpec{
					TaskRunRef: kubechainv1alpha1.LocalObjectReference{
						Name: "parent-taskrun",
					},
					ToolRef: kubechainv1alpha1.LocalObjectReference{
						Name: "test-mcp-server__test-tool",
					},
					Arguments: `{"a": 2, "b": 3}`,
				},
			}
			Expect(k8sClient.Create(ctx, trtc)).To(Succeed())

			By("reconciling the taskruntoolcall")
			eventRecorder := record.NewFakeRecorder(10)
			reconciler := &TaskRunToolCallReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			// First reconciliation (but not the contrition variety) - should initialize status
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Second reconciliation - should transition to AwaitingHumanApproval
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking the taskruntoolcall status")
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTRTC)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTRTC.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunToolCallPhasePending))
			Expect(updatedTRTC.Status.Status).To(Equal("AwaitingHumanApproval"))
			Expect(updatedTRTC.Status.StatusDetail).To(Equal("Waiting for human approval via contact channel test-contact-channel"))

			By("checking that appropriate events were emitted")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("AwaitingHumanApproval")

			By("Cleanup")
			Expect(k8sClient.Delete(ctx, contactChannel)).To(Succeed())
			Expect(k8sClient.Delete(ctx, mcpServer)).To(Succeed())
			Expect(k8sClient.Delete(ctx, tool)).To(Succeed())
			Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
		})
	})
})
*/
