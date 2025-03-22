package task

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	testutil "github.com/humanlayer/smallchain/kubechain/test/controller"
)

// taskTestCase defines a test case for the task controller
type taskTestCase struct {
	testutil.TestCase
	AgentExists bool
	Message     string
}

var _ = Describe("Task Controller", func() {
	const (
		resourceName = "test-task"
		agentName    = "test-agent"
		namespace    = "default"
	)

	// Define test cases
	testCases := []taskTestCase{
		{
			TestCase: testutil.TestCase{
				Name:           "Valid agent reference",
				ShouldSucceed:  true,
				ExpectedStatus: "Ready",
				ExpectedDetail: "Task Run Created",
				EventType:      "TaskRunCreated",
			},
			AgentExists: true,
			Message:     "Test input",
		},
		{
			TestCase: testutil.TestCase{
				Name:           "Non-existent agent",
				ShouldSucceed:  false,
				ExpectedStatus: "Error",
				ExpectedDetail: "not found",
				EventType:      "ValidationFailed",
			},
			AgentExists: false,
			Message:     "Test input",
		},
	}

	for _, tc := range testCases {
		// Use a closure to ensure proper variable scoping
		func(tc taskTestCase) {
			It(tc.Name, func() {
				// Create Agent if the test case requires it
				var agentRef string
				if tc.AgentExists {
					agent := &kubechainv1alpha1.Agent{
						ObjectMeta: testutil.CreateObjectMeta(agentName, namespace),
						Spec: kubechainv1alpha1.AgentSpec{
							LLMRef: kubechainv1alpha1.LocalObjectReference{
								Name: "test-llm",
							},
							System: "Test agent",
						},
					}
					Expect(testEnv.Client.Create(testEnv.Ctx, agent)).To(Succeed())
					
					// Mark Agent as ready
					agent.Status.Ready = true
					agent.Status.Status = "Ready"
					agent.Status.StatusDetail = "Ready for testing"
					Expect(testEnv.Client.Status().Update(testEnv.Ctx, agent)).To(Succeed())
					
					agentRef = agentName
					
					// Clean up after test
					DeferCleanup(func() {
						agent := &kubechainv1alpha1.Agent{}
						err := testEnv.Client.Get(testEnv.Ctx, types.NamespacedName{Name: agentName, Namespace: namespace}, agent)
						if err == nil {
							Expect(testEnv.Client.Delete(testEnv.Ctx, agent)).To(Succeed())
						}
					})
				} else {
					agentRef = "nonexistent-agent"
				}
				
				// Create Task resource
				task := &kubechainv1alpha1.Task{
					ObjectMeta: testutil.CreateObjectMeta(resourceName, namespace),
					Spec: kubechainv1alpha1.TaskSpec{
						AgentRef: kubechainv1alpha1.LocalObjectReference{
							Name: agentRef,
						},
						Message: tc.Message,
					},
				}
				Expect(testEnv.Client.Create(testEnv.Ctx, task)).To(Succeed())
				
				// Clean up after test
				DeferCleanup(func() {
					task := &kubechainv1alpha1.Task{}
					err := testEnv.Client.Get(testEnv.Ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, task)
					if err == nil {
						Expect(testEnv.Client.Delete(testEnv.Ctx, task)).To(Succeed())
					}
				})
				
				// Create reconciler
				reconciler := &TaskReconciler{
					Client:   testEnv.Client,
					Scheme:   testEnv.Client.Scheme(),
					recorder: testEnv.Recorder,
				}
				
				// Reconcile the resource
				result, err := reconciler.Reconcile(testEnv.Ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      resourceName,
						Namespace: namespace,
					},
				})
				
				// Check expectations based on test case
				if !tc.ShouldSucceed {
					Expect(err).To(HaveOccurred())
					if tc.ExpectedDetail != "" {
						Expect(err.Error()).To(ContainSubstring(tc.ExpectedDetail))
					}
				} else {
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(Equal(reconcile.Result{}))
				}
				
				// Check status updates
				updatedTask := &kubechainv1alpha1.Task{}
				err = testEnv.Client.Get(testEnv.Ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, updatedTask)
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedTask.Status.Ready).To(Equal(tc.ShouldSucceed))
				Expect(updatedTask.Status.Status).To(Equal(tc.ExpectedStatus))
				Expect(updatedTask.Status.StatusDetail).To(ContainSubstring(tc.ExpectedDetail))
				
				// Verify events
				testEnv.CheckEvent(tc.EventType, 5*time.Second)
			})
		}(tc)
	}
})