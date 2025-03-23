/*
Copyright 2025 the Kubechain Authors.

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

package controller

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

// Mock functions for testing
var validateHumanLayerAPIKey func(string) error
var validateSlackToken func(string) error

var _ = Describe("ContactChannel Controller", func() {

	Context("When reconciling a resource", func() {
		const resourceName = "test-contactchannel"
		const secretName = "test-contactchannel-secret"
		const secretKey = "api-key"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		AfterEach(func() {
			By("Cleanup the test secret")
			secret := &corev1.Secret{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: "default"}, secret)
			if err == nil {
				Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
			}

			By("Cleanup the ContactChannel resource")
			resource := &kubechainv1alpha1.ContactChannel{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		// Mock the validation function in the controller to avoid real HTTP requests
		var originalValidateHumanLayerAPIKey func(string) error
		var originalValidateSlackToken func(string) error

		BeforeEach(func() {
			// Save original functions
			originalValidateHumanLayerAPIKey = validateHumanLayerAPIKey
			originalValidateSlackToken = validateSlackToken

			// Mock for testing
			validateHumanLayerAPIKey = func(apiKey string) error {
				if apiKey == "valid-humanlayer-key" {
					return nil
				}
				return fmt.Errorf("invalid HumanLayer API key")
			}

			validateSlackToken = func(token string) error {
				if token == "valid-slack-token" {
					return nil
				}
				return fmt.Errorf("invalid Slack token")
			}
		})

		AfterEach(func() {
			// Restore original functions
			validateHumanLayerAPIKey = originalValidateHumanLayerAPIKey
			validateSlackToken = originalValidateSlackToken
		})

		It("should successfully validate a Slack channel with valid config", func() {
			By("Creating a secret with valid API key")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					secretKey: []byte("valid-humanlayer-key"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			By("Creating a ContactChannel resource for Slack")
			channel := &kubechainv1alpha1.ContactChannel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.ContactChannelSpec{
					ChannelType: "slack",
					APIKeyFrom: kubechainv1alpha1.APIKeySource{
						SecretKeyRef: kubechainv1alpha1.SecretKeyRef{
							Name: secretName,
							Key:  secretKey,
						},
					},
					SlackConfig: &kubechainv1alpha1.SlackChannelConfig{
						ChannelOrUserID: "C12345678",
						ContextAboutChannelOrUser: "A test channel",
					},
				},
			}
			Expect(k8sClient.Create(ctx, channel)).To(Succeed())

			By("Reconciling the resource")
			eventRecorder := record.NewFakeRecorder(10)
			controllerReconciler := &ContactChannelReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedChannel := &kubechainv1alpha1.ContactChannel{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedChannel)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedChannel.Status.Ready).To(BeTrue())
			Expect(updatedChannel.Status.Status).To(Equal(statusReady))
			Expect(updatedChannel.Status.StatusDetail).To(ContainSubstring("validated successfully"))
		})

		It("should successfully validate an Email channel with valid config", func() {
			By("Creating a secret with valid API key")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					secretKey: []byte("valid-humanlayer-key"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			By("Creating a ContactChannel resource for Email")
			channel := &kubechainv1alpha1.ContactChannel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.ContactChannelSpec{
					ChannelType: "email",
					APIKeyFrom: kubechainv1alpha1.APIKeySource{
						SecretKeyRef: kubechainv1alpha1.SecretKeyRef{
							Name: secretName,
							Key:  secretKey,
						},
					},
					EmailConfig: &kubechainv1alpha1.EmailChannelConfig{
						Address:         "test@example.com",
						ContextAboutUser: "Test user",
						Subject:         "Test notification",
					},
				},
			}
			Expect(k8sClient.Create(ctx, channel)).To(Succeed())

			By("Reconciling the resource")
			eventRecorder := record.NewFakeRecorder(10)
			controllerReconciler := &ContactChannelReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedChannel := &kubechainv1alpha1.ContactChannel{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedChannel)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedChannel.Status.Ready).To(BeTrue())
			Expect(updatedChannel.Status.Status).To(Equal(statusReady))
			Expect(updatedChannel.Status.StatusDetail).To(ContainSubstring("validated successfully"))
		})

		It("should fail validation with invalid configuration", func() {
			By("Creating a secret with valid API key")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					secretKey: []byte("valid-humanlayer-key"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			By("Creating a ContactChannel resource with missing config")
			channel := &kubechainv1alpha1.ContactChannel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.ContactChannelSpec{
					ChannelType: "slack",
					APIKeyFrom: kubechainv1alpha1.APIKeySource{
						SecretKeyRef: kubechainv1alpha1.SecretKeyRef{
							Name: secretName,
							Key:  secretKey,
						},
					},
					// Missing SlackConfig
				},
			}
			Expect(k8sClient.Create(ctx, channel)).To(Succeed())

			By("Reconciling the resource")
			eventRecorder := record.NewFakeRecorder(10)
			controllerReconciler := &ContactChannelReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedChannel := &kubechainv1alpha1.ContactChannel{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedChannel)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedChannel.Status.Ready).To(BeFalse())
			Expect(updatedChannel.Status.Status).To(Equal(statusError))
			Expect(updatedChannel.Status.StatusDetail).To(ContainSubstring("slackConfig"))
		})

		It("should fail validation with invalid API key", func() {
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

			By("Creating a ContactChannel resource")
			channel := &kubechainv1alpha1.ContactChannel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: kubechainv1alpha1.ContactChannelSpec{
					ChannelType: "slack",
					APIKeyFrom: kubechainv1alpha1.APIKeySource{
						SecretKeyRef: kubechainv1alpha1.SecretKeyRef{
							Name: secretName,
							Key:  secretKey,
						},
					},
					SlackConfig: &kubechainv1alpha1.SlackChannelConfig{
						ChannelOrUserID: "C12345678",
					},
				},
			}
			Expect(k8sClient.Create(ctx, channel)).To(Succeed())

			By("Reconciling the resource")
			eventRecorder := record.NewFakeRecorder(10)
			controllerReconciler := &ContactChannelReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedChannel := &kubechainv1alpha1.ContactChannel{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedChannel)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedChannel.Status.Ready).To(BeFalse())
			Expect(updatedChannel.Status.Status).To(Equal(statusError))
			Expect(updatedChannel.Status.StatusDetail).To(ContainSubstring("invalid"))
		})
	})
})
