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
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

// LLMReconciler reconciles a LLM object
type LLMReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
}

func (r *LLMReconciler) validateOpenAIKey(apiKey string) error {
	req, err := http.NewRequest("GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid API key (status code: %d)", resp.StatusCode)
	}

	return nil
}

func (r *LLMReconciler) validateSecret(ctx context.Context, llm *kubechainv1alpha1.LLM) error {
	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      llm.Spec.APIKeyFrom.SecretKeyRef.Name,
		Namespace: llm.Namespace,
	}, secret)
	if err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}

	key := llm.Spec.APIKeyFrom.SecretKeyRef.Key
	apiKey, exists := secret.Data[key]
	if !exists {
		return fmt.Errorf("key %q not found in secret", key)
	}

	if llm.Spec.Provider == "openai" {
		if err := r.validateOpenAIKey(string(apiKey)); err != nil {
			return fmt.Errorf("OpenAI API key validation failed: %w", err)
		}
	}

	return nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *LLMReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the LLM instance
	var llm kubechainv1alpha1.LLM
	if err := r.Get(ctx, req.NamespacedName, &llm); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Starting reconciliation", "namespacedName", req.NamespacedName, "provider", llm.Spec.Provider)

	// Create a copy for status update
	statusUpdate := llm.DeepCopy()

	// Initialize status if not set
	if statusUpdate.Status.Status == "" {
		statusUpdate.Status.Status = "Pending"
		statusUpdate.Status.StatusDetail = "Validating configuration"
		r.recorder.Event(&llm, corev1.EventTypeNormal, "Initializing", "Starting validation")
	}

	// Validate secret
	if err := r.validateSecret(ctx, &llm); err != nil {
		log.Error(err, "Secret validation failed")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = "Error"
		statusUpdate.Status.StatusDetail = err.Error()
		r.recorder.Event(&llm, corev1.EventTypeWarning, "ValidationFailed", err.Error())
	} else {
		statusUpdate.Status.Ready = true
		statusUpdate.Status.Status = "Ready"
		if llm.Spec.Provider == "openai" {
			statusUpdate.Status.StatusDetail = "OpenAI API key validated successfully"
		} else {
			statusUpdate.Status.StatusDetail = "Secret validated successfully"
		}
		r.recorder.Event(&llm, corev1.EventTypeNormal, "ValidationSucceeded", statusUpdate.Status.StatusDetail)
	}

	// Update status using SubResource client
	if err := r.Status().Patch(ctx, statusUpdate, client.MergeFrom(&llm)); err != nil {
		log.Error(err, "Unable to update LLM status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled LLM",
		"provider", llm.Spec.Provider,
		"ready", statusUpdate.Status.Ready,
		"status", statusUpdate.Status.Status,
		"statusDetail", statusUpdate.Status.StatusDetail)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LLMReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("llm-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubechainv1alpha1.LLM{}).
		Named("llm").
		Complete(r)
}
