package taskruntoolcall

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	testutil "github.com/humanlayer/smallchain/kubechain/test/controller"
)

// Define test constants
const (
	toolName     = "add"
	resourceName = "test-taskruntoolcall"
	parentName   = "parent-taskrun"
	namespace    = "default"
)

// Add a TaskRunToolCall test case
type taskrunToolCallTestCase struct {
	testutil.TestCase
	Arguments        string
	ExpectedResult   string
	ExpectedPhase    kubechainv1alpha1.TaskRunToolCallPhase
	ExpectStatusCode string
}

var _ = Describe("TaskRunToolCall Controller", func() {
	BeforeEach(func() {
		// Create test Tool for direct execution
		tool := testEnv.CreateAddTool(toolName)
		testEnv.MarkToolReady(tool)
		DeferCleanup(func() {
			testEnv.DeleteTool(toolName)
		})
	})

	// Define test cases for function execution
	testCases := []taskrunToolCallTestCase{
		{
			TestCase: testutil.TestCase{
				Name:           "Function execution with valid arguments",
				ShouldSucceed:  true,
				ExpectedStatus: "Ready",
				ExpectedDetail: "Tool executed successfully",
				EventType:      "ExecutionSucceeded",
			},
			Arguments:        `{"a": 2, "b": 3}`,
			ExpectedResult:   "5",
			ExpectedPhase:    kubechainv1alpha1.TaskRunToolCallPhaseSucceeded,
			ExpectStatusCode: "Ready",
		},
		{
			TestCase: testutil.TestCase{
				Name:           "Function execution with invalid JSON",
				ShouldSucceed:  false,
				ExpectedStatus: "Error",
				ExpectedDetail: "Invalid arguments JSON",
				EventType:      "ExecutionFailed",
			},
			Arguments:        "invalid json",
			ExpectedResult:   "",
			ExpectedPhase:    kubechainv1alpha1.TaskRunToolCallPhaseFailed,
			ExpectStatusCode: "Error",
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable
		It(tc.Name, func() {
			// Create unique test name
			testRunName := fmt.Sprintf("%s-%d", resourceName, time.Now().UnixNano())

			// Create TaskRunToolCall
			trtc := &kubechainv1alpha1.TaskRunToolCall{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testRunName,
					Namespace: namespace,
				},
				Spec: kubechainv1alpha1.TaskRunToolCallSpec{
					TaskRunRef: kubechainv1alpha1.LocalObjectReference{
						Name: parentName,
					},
					ToolRef: kubechainv1alpha1.LocalObjectReference{
						Name: toolName,
					},
					Arguments: tc.Arguments,
				},
			}
			Expect(testEnv.Client.Create(testEnv.Ctx, trtc)).To(Succeed())

			DeferCleanup(func() {
				// Delete TaskRunToolCall
				trtcToDelete := &kubechainv1alpha1.TaskRunToolCall{}
				err := testEnv.Client.Get(testEnv.Ctx, types.NamespacedName{
					Name:      testRunName, 
					Namespace: namespace,
				}, trtcToDelete)
				if err == nil {
					Expect(testEnv.Client.Delete(testEnv.Ctx, trtcToDelete)).To(Succeed())
				}
			})

			// Create reconciler
			reconciler := &TaskRunToolCallReconciler{
				Client:   testEnv.Client,
				Scheme:   testEnv.Client.Scheme(),
				recorder: testEnv.Recorder,
			}

			// First reconciliation - should initialize status
			_, err := reconciler.Reconcile(testEnv.Ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testRunName,
					Namespace: namespace,
				},
			})
			if tc.ShouldSucceed {
				Expect(err).NotTo(HaveOccurred())
			}

			// Second reconciliation - should execute function or fail
			_, err = reconciler.Reconcile(testEnv.Ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testRunName,
					Namespace: namespace,
				},
			})
			if !tc.ShouldSucceed {
				// For invalid inputs, we expect an error
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}

			// Check status updates
			updatedTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err = testEnv.Client.Get(testEnv.Ctx, types.NamespacedName{
				Name:      testRunName,
				Namespace: namespace,
			}, updatedTRTC)
			Expect(err).NotTo(HaveOccurred())

			if tc.ShouldSucceed {
				Expect(updatedTRTC.Status.Phase).To(Equal(tc.ExpectedPhase))
				Expect(updatedTRTC.Status.Result).To(Equal(tc.ExpectedResult))
			}
			Expect(updatedTRTC.Status.Status).To(Equal(tc.ExpectStatusCode))
			Expect(updatedTRTC.Status.StatusDetail).To(Equal(tc.ExpectedDetail))

			// Verify event
			testEnv.CheckEvent(tc.EventType, 5*time.Second)
		})
	}
})