package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

// TaskReconciler reconciles a Task object
type TaskReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// validateAgent checks if the referenced agent exists and is ready
func (r *TaskReconciler) validateAgent(ctx context.Context, task *kubechainv1alpha1.Task) error {
	agent := &kubechainv1alpha1.Agent{}
	err := r.Get(ctx, client.ObjectKey{
		Namespace: task.Namespace,
		Name:      task.Spec.AgentRef.Name,
	}, agent)
	if err != nil {
		return fmt.Errorf("failed to get Agent %q: %w", task.Spec.AgentRef.Name, err)
	}

	if !agent.Status.Ready {
		return fmt.Errorf("Agent %q is not ready", task.Spec.AgentRef.Name)
	}

	return nil
}

// Reconcile validates the task's agent reference and parameters
func (r *TaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var task kubechainv1alpha1.Task
	if err := r.Get(ctx, req.NamespacedName, &task); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Starting reconciliation", "name", task.Name)

	// Create a copy for status update
	statusUpdate := task.DeepCopy()

	// Validate agent reference
	if err := r.validateAgent(ctx, &task); err != nil {
		logger.Error(err, "Agent validation failed")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = err.Error()
	} else {
		statusUpdate.Status.Ready = true
		statusUpdate.Status.Status = "Agent validated successfully"
	}

	// Update status
	if err := r.Status().Update(ctx, statusUpdate); err != nil {
		logger.Error(err, "Unable to update Task status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled task",
		"name", task.Name,
		"ready", statusUpdate.Status.Ready)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubechainv1alpha1.Task{}).
		Complete(r)
}
