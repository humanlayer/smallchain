package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

// TaskRunReconciler reconciles a TaskRun object
type TaskRunReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// getTask fetches the parent Task for this TaskRun
func (r *TaskRunReconciler) getTask(ctx context.Context, taskRun *kubechainv1alpha1.TaskRun) (*kubechainv1alpha1.Task, error) {
	task := &kubechainv1alpha1.Task{}
	err := r.Get(ctx, client.ObjectKey{
		Namespace: taskRun.Namespace,
		Name:      taskRun.Spec.TaskRef.Name,
	}, task)
	if err != nil {
		return nil, fmt.Errorf("failed to get Task %q: %w", taskRun.Spec.TaskRef.Name, err)
	}

	if !task.Status.Ready {
		return nil, fmt.Errorf("task %q is not ready", task.Name)
	}

	return task, nil
}

// Reconcile validates the taskrun's task reference and updates its phase
func (r *TaskRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var taskRun kubechainv1alpha1.TaskRun
	if err := r.Get(ctx, req.NamespacedName, &taskRun); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Starting reconciliation", "name", taskRun.Name)

	// Create a copy for status update
	statusUpdate := taskRun.DeepCopy()

	// Get parent Task
	_, err := r.getTask(ctx, &taskRun)
	if err != nil {
		logger.Error(err, "Task validation failed")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = "Error"
		statusUpdate.Status.StatusDetail = err.Error()
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(&taskRun, corev1.EventTypeWarning, "ValidationFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return ctrl.Result{}, fmt.Errorf("failed to update taskrun status: %v", err)
		}
		return ctrl.Result{}, err // requeue
	}

	// Update phase if needed
	if statusUpdate.Status.Phase == "" {
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhasePending
		statusUpdate.Status.Ready = true
		statusUpdate.Status.Status = "Ready"
		statusUpdate.Status.StatusDetail = "Task validated successfully"
		statusUpdate.Status.Error = "" // Clear any previous error when entering ready state
		r.recorder.Event(&taskRun, corev1.EventTypeNormal, "ValidationSucceeded", "Task validated successfully")
	}

	// Update status
	if err := r.Status().Update(ctx, statusUpdate); err != nil {
		logger.Error(err, "Unable to update TaskRun status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled taskrun",
		"name", taskRun.Name,
		"ready", statusUpdate.Status.Ready,
		"phase", statusUpdate.Status.Phase)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TaskRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("taskrun-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubechainv1alpha1.TaskRun{}).
		Complete(r)
}
