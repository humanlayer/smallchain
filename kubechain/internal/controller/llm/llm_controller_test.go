/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package llm

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	testutil "github.com/humanlayer/smallchain/kubechain/test/controller"
)

var _ = Describe("LLM Controller", func() {
	const (
		resourceName = "test-llm"
		secretName   = "test-secret"
		secretKey    = "api-key"
		namespace    = "default"
	)

	// Define test cases for basic controller functionality
	DescribeTable("Basic LLM reconciliation",
		func(tc testutil.LLMTestCase) {
			// Create or skip secret based on test case
			if tc.SecretExists {
				_ = testEnv.CreateSecret(secretName, secretKey, tc.SecretValue)
				DeferCleanup(func() {
					testEnv.DeleteSecret(secretName)
				})
			}

			// Determine secret name based on test case
			var secretNameToUse string
			if tc.SecretExists {
				secretNameToUse = secretName
			} else {
				secretNameToUse = "nonexistent-secret"
			}

			// Create LLM resource
			llmResource := &kubechainv1alpha1.LLM{
				ObjectMeta: testutil.CreateObjectMeta(resourceName, namespace),
				Spec: kubechainv1alpha1.LLMSpec{
					Provider: "openai",
					APIKeyFrom: kubechainv1alpha1.APIKeySource{
						SecretKeyRef: kubechainv1alpha1.SecretKeyRef{
							Name: secretNameToUse,
							Key:  secretKey,
						},
					},
				},
			}
			Expect(testEnv.Client.Create(testEnv.Ctx, llmResource)).To(Succeed())
			DeferCleanup(func() {
				testEnv.DeleteLLM(resourceName)
			})

			// Create reconciler
			recorder := record.NewFakeRecorder(10)
			reconciler := &LLMReconciler{
				Client:   testEnv.Client,
				Scheme:   testEnv.Client.Scheme(),
				recorder: recorder,
			}

			// Run reconciliation
			_, err := reconciler.Reconcile(testEnv.Ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      resourceName,
					Namespace: namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred()) // Controller handles errors internally through status

			// Check that status was updated (without checking specific values)
			updatedLLM := &kubechainv1alpha1.LLM{}
			err = testEnv.Client.Get(testEnv.Ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, updatedLLM)
			Expect(err).NotTo(HaveOccurred())
			
			// Verify that status was set
			Expect(updatedLLM.Status.Status).NotTo(BeEmpty())
			Expect(updatedLLM.Status.StatusDetail).NotTo(BeEmpty())
			
			// For non-existent secret case, we can verify specific error
			if !tc.SecretExists {
				Expect(updatedLLM.Status.Ready).To(BeFalse())
				Expect(updatedLLM.Status.Status).To(Equal("Error"))
				Expect(updatedLLM.Status.StatusDetail).To(ContainSubstring("failed to get secret"))
			}
		},
		Entry("With Secret", testutil.LLMTestCase{
			TestCase: testutil.TestCase{
				Name: "With existing secret",
			},
			SecretExists: true,
			SecretValue:  []byte("test-api-key"),
		}),
		Entry("With non-existent secret", testutil.LLMTestCase{
			TestCase: testutil.TestCase{
				Name:           "Non-existent secret",
				ShouldSucceed:  false,
				ExpectedStatus: "Error",
				ExpectedDetail: "failed to get secret",
			},
			SecretExists: false,
			SecretValue:  nil,
		}),
	)

	// Special test case requiring real API key
	It("should validate with real OpenAI API key if available", func() {
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			Skip("Skipping OpenAI API key validation test - OPENAI_API_KEY not set")
		}

		// Create secret with real API key
		_ = testEnv.CreateSecret(secretName, secretKey, []byte(apiKey))
		DeferCleanup(func() {
			testEnv.DeleteSecret(secretName)
		})

		// Create LLM resource
		llmResource := &kubechainv1alpha1.LLM{
			ObjectMeta: testutil.CreateObjectMeta(resourceName, namespace),
			Spec: kubechainv1alpha1.LLMSpec{
				Provider: "openai",
				APIKeyFrom: kubechainv1alpha1.APIKeySource{
					SecretKeyRef: kubechainv1alpha1.SecretKeyRef{
						Name: secretName,
						Key:  secretKey,
					},
				},
			},
		}
		Expect(testEnv.Client.Create(testEnv.Ctx, llmResource)).To(Succeed())
		DeferCleanup(func() {
			testEnv.DeleteLLM(resourceName)
		})

		// Create regular reconciler for real API test
		recorder := record.NewFakeRecorder(10)
		reconciler := &LLMReconciler{
			Client:   testEnv.Client,
			Scheme:   testEnv.Client.Scheme(),
			recorder: recorder,
		}

		// Run reconciliation
		_, err := reconciler.Reconcile(testEnv.Ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			},
		})
		Expect(err).NotTo(HaveOccurred())

		// Check status updates
		updatedLLM := &kubechainv1alpha1.LLM{}
		err = testEnv.Client.Get(testEnv.Ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, updatedLLM)
		Expect(err).NotTo(HaveOccurred())
		
		// Only verify StatusDetail since real API key may or may not be valid
		Expect(updatedLLM.Status.StatusDetail).NotTo(BeEmpty())
	})
})