package controller

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

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
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
			Eventually(func() bool {
				select {
				case event := <-eventRecorder.Events:
					return strings.Contains(event, "ExecutionSucceeded")
				default:
					return false
				}
			}, 5*time.Second, 100*time.Millisecond).Should(BeTrue())
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
			Eventually(func() bool {
				select {
				case event := <-eventRecorder.Events:
					return strings.Contains(event, "ExecutionFailed")
				default:
					return false
				}
			}, 5*time.Second, 100*time.Millisecond).Should(BeTrue())
		})
	})
})
