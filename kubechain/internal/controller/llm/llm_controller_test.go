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
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

var _ = Describe("LLM Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		const secretName = "test-secret"
		const secretKey = "api-key"

		ctx := context.Background()
		var mockOpenAIServer *httptest.Server

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		BeforeEach(func() {
			// Set up mock OpenAI server for local testing when no API key is available
			mockOpenAIServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") == "Bearer valid-key" {
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(http.StatusUnauthorized)
				}
			}))
		})

		AfterEach(func() {
			mockOpenAIServer.Close()

			By("Cleanup the test secret")
			secret := &corev1.Secret{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: "default"}, secret)
			if err == nil {
				Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
			}

			By("Cleanup the specific resource instance LLM")
			resource := &kubechainv1alpha1.LLM{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should successfully validate OpenAI API key", func() {
			apiKey := os.Getenv("OPENAI_API_KEY")
			if apiKey == "" {
				Skip("Skipping OpenAI API key validation test - OPENAI_API_KEY not set")
			}

			By("creating the test secret with real API key")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					secretKey: []byte(apiKey),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			By("creating the custom resource for the Kind LLM")
			resource := &kubechainv1alpha1.LLM{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
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
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			By("Reconciling the created resource")
			eventRecorder := record.NewFakeRecorder(10)
			controllerReconciler := &LLMReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedLLM := &kubechainv1alpha1.LLM{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedLLM)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedLLM.Status.Ready).To(BeTrue())
			Expect(updatedLLM.Status.Status).To(Equal("Ready"))
			Expect(updatedLLM.Status.StatusDetail).To(Equal("OpenAI API key validated successfully"))

			By("checking that a success event was created")
			Eventually(func() bool {
				select {
				case event := <-eventRecorder.Events:
					return strings.Contains(event, "ValidationSucceeded")
				default:
					return false
				}
			}, 5*time.Second, 100*time.Millisecond).Should(BeTrue(), "Expected to find success event")
		})

		It("should fail reconciliation with invalid API key", func() {
			By("Creating a secret with invalid API key")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					secretKey: []byte("invalid-key"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			By("creating the custom resource for the Kind LLM")
			resource := &kubechainv1alpha1.LLM{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
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
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			By("Reconciling the resource")
			eventRecorder := record.NewFakeRecorder(10)
			controllerReconciler := &LLMReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedLLM := &kubechainv1alpha1.LLM{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedLLM)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedLLM.Status.Ready).To(BeFalse())
			Expect(updatedLLM.Status.Status).To(Equal("Error"))
			Expect(updatedLLM.Status.StatusDetail).To(ContainSubstring("OpenAI API key validation failed"))

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

		It("should fail reconciliation with non-existent secret", func() {
			By("Creating the LLM resource with non-existent secret")
			resource := &kubechainv1alpha1.LLM{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.LLMSpec{
					Provider: "openai",
					APIKeyFrom: kubechainv1alpha1.APIKeySource{
						SecretKeyRef: kubechainv1alpha1.SecretKeyRef{
							Name: "nonexistent-secret",
							Key:  secretKey,
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			By("Reconciling the resource")
			eventRecorder := record.NewFakeRecorder(10)
			controllerReconciler := &LLMReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedLLM := &kubechainv1alpha1.LLM{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedLLM)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedLLM.Status.Ready).To(BeFalse())
			Expect(updatedLLM.Status.Status).To(Equal("Error"))
			Expect(updatedLLM.Status.StatusDetail).To(ContainSubstring("failed to get secret"))

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
	})
})
