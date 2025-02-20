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

var _ = Describe("TaskRun Controller", func() {
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
					System: "Test agent",
				},
			}
			Expect(k8sClient.Create(ctx, agent)).To(Succeed())

			// Mark Agent as ready
			agent.Status.Ready = true
			agent.Status.Status = "Ready"
			agent.Status.StatusDetail = "Ready for testing"
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
					Message: "Test input",
				},
			}
			Expect(k8sClient.Create(ctx, task)).To(Succeed())

			// Mark Task as ready
			task.Status.Ready = true
			task.Status.Status = "Ready"
			task.Status.StatusDetail = "Agent validated successfully"
			Expect(k8sClient.Status().Update(ctx, task)).To(Succeed())
		})

		AfterEach(func() {
			// Cleanup test resources
			By("Cleanup the test Agent")
			agent := &kubechainv1alpha1.Agent{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: agentName, Namespace: "default"}, agent)
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
			Expect(updatedTaskRun.Status.Status).To(Equal("Ready"))
			Expect(updatedTaskRun.Status.StatusDetail).To(Equal("Task validated successfully"))
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunPhasePending))

			By("checking that validation success event was created")
			Eventually(func() bool {
				select {
				case event := <-eventRecorder.Events:
					return strings.Contains(event, "ValidationSucceeded")
				default:
					return false
				}
			}, 5*time.Second, 100*time.Millisecond).Should(BeTrue(), "Expected to find validation success event")
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
			Eventually(func() bool {
				select {
				case event := <-eventRecorder.Events:
					return strings.Contains(event, "ValidationFailed")
				default:
					return false
				}
			}, 5*time.Second, 100*time.Millisecond).Should(BeTrue(), "Expected to find failure event")
		})

		It("should fail when task exists but is not ready", func() {
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
			Expect(updatedTaskRun.Status.StatusDetail).To(ContainSubstring("is not ready"))
			Expect(updatedTaskRun.Status.Error).To(ContainSubstring("is not ready"))

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
			Expect(updatedTaskRun.Status.Status).To(Equal("Ready"))
			Expect(updatedTaskRun.Status.StatusDetail).To(Equal("Task validated successfully"))
			Expect(updatedTaskRun.Status.Phase).To(Equal(kubechainv1alpha1.TaskRunPhasePending))
			Expect(updatedTaskRun.Status.Error).To(BeEmpty(), "Error field should be cleared")
		})
	})
})
