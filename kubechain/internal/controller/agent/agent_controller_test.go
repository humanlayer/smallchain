package agent

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	testutil "github.com/humanlayer/smallchain/kubechain/test/controller"
)

var _ = Describe("Agent Controller", func() {
	const (
		resourceName = "test-agent"
		llmName      = "test-llm"
		toolName     = "test-tool"
		namespace    = "default"
	)

	// Define test cases for table-driven testing
	testCases := []testutil.AgentTestCase{
		{
			TestCase: testutil.TestCase{
				Name:           "Valid dependencies",
				ShouldSucceed:  true,
				ExpectedStatus: "Ready",
				ExpectedDetail: "All dependencies validated successfully",
				EventType:      "ValidationSucceeded",
			},
			LLMExists:   true,
			ToolsExist:  true,
			ToolsReady:  true,
			ExpectError: false,
		},
		{
			TestCase: testutil.TestCase{
				Name:           "Non-existent LLM",
				ShouldSucceed:  false,
				ExpectedStatus: "Error",
				ExpectedDetail: "not found",
				EventType:      "ValidationFailed",
			},
			LLMExists:   false,
			ToolsExist:  true,
			ToolsReady:  true,
			ExpectError: true,
		},
		{
			TestCase: testutil.TestCase{
				Name:           "Non-existent Tool",
				ShouldSucceed:  false,
				ExpectedStatus: "Error",
				ExpectedDetail: "not found",
				EventType:      "ValidationFailed",
			},
			LLMExists:   true,
			ToolsExist:  false,
			ToolsReady:  true,
			ExpectError: true,
		},
	}

	for _, tc := range testCases {
		// Use a closure to ensure proper variable scoping
		func(tc testutil.AgentTestCase) {
			It(tc.Name, func() {
				// Set up test resources according to test case
				// Create LLM if the test case requires it
				if tc.LLMExists {
					llm := testEnv.CreateLLM(llmName, "test-secret", "api-key")
					testEnv.MarkLLMReady(llm)
					DeferCleanup(func() {
						testEnv.DeleteLLM(llmName)
					})
				}

				// Create Tool if the test case requires it
				toolNames := []string{}
				if tc.ToolsExist {
					tool := testEnv.CreateTool(toolName)
					if tc.ToolsReady {
						testEnv.MarkToolReady(tool)
					}
					toolNames = append(toolNames, toolName)
					DeferCleanup(func() {
						testEnv.DeleteTool(toolName)
					})
				} else {
					toolNames = append(toolNames, "nonexistent-tool")
				}

				// Prepare the Agent resource
				var llmRef string
				if tc.LLMExists {
					llmRef = llmName
				} else {
					llmRef = "nonexistent-llm"
				}
				_ = testEnv.CreateAgent(resourceName, llmRef, toolNames)
				DeferCleanup(func() {
					testEnv.DeleteAgent(resourceName)
				})

				// Create reconciler with fake recorder
				reconciler := &AgentReconciler{
					Client:   testEnv.Client,
					Scheme:   testEnv.Client.Scheme(),
					recorder: testEnv.Recorder,
				}

				// Perform reconciliation
				result, err := reconciler.Reconcile(testEnv.Ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      resourceName,
						Namespace: namespace,
					},
				})

				// Check expectations
				if tc.ExpectError {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(Equal(reconcile.Result{}))
				}

				// Verify status updates
				updatedAgent := &kubechainv1alpha1.Agent{}
				err = testEnv.Client.Get(testEnv.Ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, updatedAgent)
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedAgent.Status.Ready).To(Equal(tc.ShouldSucceed))
				Expect(updatedAgent.Status.Status).To(Equal(tc.ExpectedStatus))
				Expect(updatedAgent.Status.StatusDetail).To(ContainSubstring(tc.ExpectedDetail))

				// Verify events
				testEnv.CheckEvent(tc.EventType, 5*time.Second)

				// Additional validation for successful cases
				if tc.ShouldSucceed {
					Expect(updatedAgent.Status.ValidTools).To(ContainElement(kubechainv1alpha1.ResolvedTool{
						Kind: "Tool",
						Name: toolName,
					}))
				}
			})
		}(tc)
	}
})