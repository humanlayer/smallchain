package tool

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	testutil "github.com/humanlayer/smallchain/kubechain/test/controller"
)

// toolTestCase defines a test case for the tool controller
type toolTestCase struct {
	testutil.TestCase
	ToolType    string
	Name        string
	Description string
	Parameters  string
	BuiltinName string
}

var _ = Describe("Tool Controller", func() {
	const (
		resourceName = "test-tool"
		namespace    = "default"
	)

	// Define test cases
	testCases := []toolTestCase{
		{
			TestCase: testutil.TestCase{
				Name:           "Function tool",
				ShouldSucceed:  true,
				ExpectedStatus: "Ready",
				ExpectedDetail: "Tool validation successful",
				EventType:      "ValidationSucceeded",
			},
			ToolType:    "function",
			Name:        "add",
			Description: "Add two numbers",
			Parameters: `{
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
			}`,
			BuiltinName: "add",
		},
	}

	for _, tc := range testCases {
		// Use a closure to ensure proper variable scoping
		func(tc toolTestCase) {
			It(tc.Name, func() {
				// Create the tool resource
				resource := &kubechainv1alpha1.Tool{
					ObjectMeta: testutil.CreateObjectMeta(resourceName, namespace),
					Spec: kubechainv1alpha1.ToolSpec{
						ToolType:    tc.ToolType,
						Name:        tc.Name,
						Description: tc.Description,
						Parameters: runtime.RawExtension{
							Raw: []byte(tc.Parameters),
						},
						Execute: kubechainv1alpha1.ToolExecute{
							Builtin: &kubechainv1alpha1.BuiltinToolSpec{
								Name: tc.BuiltinName,
							},
						},
					},
				}
				Expect(testEnv.Client.Create(testEnv.Ctx, resource)).To(Succeed())
				
				// Clean up after test
				DeferCleanup(func() {
					// Delete the tool
					tool := &kubechainv1alpha1.Tool{}
					err := testEnv.Client.Get(testEnv.Ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, tool)
					if err == nil {
						Expect(testEnv.Client.Delete(testEnv.Ctx, tool)).To(Succeed())
					}
				})

				// Create reconciler for test
				reconciler := &ToolReconciler{
					Client:   testEnv.Client,
					Scheme:   testEnv.Client.Scheme(),
					recorder: testEnv.Recorder,
				}

				// Reconcile the resource
				_, err := reconciler.Reconcile(testEnv.Ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      resourceName,
						Namespace: namespace,
					},
				})
				Expect(err).NotTo(HaveOccurred())

				// Check status updates
				updatedTool := &kubechainv1alpha1.Tool{}
				err = testEnv.Client.Get(testEnv.Ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, updatedTool)
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedTool.Status.Ready).To(Equal(tc.ShouldSucceed))
				Expect(updatedTool.Status.Status).To(Equal(tc.ExpectedStatus))
				Expect(updatedTool.Status.StatusDetail).To(Equal(tc.ExpectedDetail))

				// Verify events
				testEnv.CheckEvent(tc.EventType, 5*time.Second)
			})
		}(tc)
	}
})