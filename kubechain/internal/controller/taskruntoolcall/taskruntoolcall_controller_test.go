package taskruntoolcall

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/humanlayer/smallchain/kubechain/test/utils"
)

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
			contactChannel := &kubechainv1alpha1.ContactChannel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-contact-channel",
					Namespace: "default",
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
		})
	})
})
