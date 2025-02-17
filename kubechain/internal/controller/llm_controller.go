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

package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

// LLMReconciler reconciles a LLM object
type LLMReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=llms,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=llms/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=llms/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

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
	if _, exists := secret.Data[key]; !exists {
		return fmt.Errorf("key %q not found in secret", key)
	}

	return nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *LLMReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Starting reconciliation", "namespacedName", req.NamespacedName)

	// Fetch the LLM instance
	var llm kubechainv1alpha1.LLM
	if err := r.Get(ctx, req.NamespacedName, &llm); err != nil {
		log.Error(err, "Unable to fetch LLM")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Found LLM resource", "provider", llm.Spec.Provider)

	// Create a copy for status update
	statusUpdate := llm.DeepCopy()

	// Validate secret
	if err := r.validateSecret(ctx, &llm); err != nil {
		log.Error(err, "Secret validation failed")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Message = err.Error()
	} else {
		statusUpdate.Status.Ready = true
		statusUpdate.Status.Message = "Secret validated successfully"
	}

	// Update status using SubResource client
	if err := r.Status().Patch(ctx, statusUpdate, client.MergeFrom(&llm)); err != nil {
		log.Error(err, "Unable to update LLM status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled LLM",
		"provider", llm.Spec.Provider,
		"ready", statusUpdate.Status.Ready,
		"message", statusUpdate.Status.Message)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LLMReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubechainv1alpha1.LLM{}).
		Named("llm").
		Complete(r)
}
