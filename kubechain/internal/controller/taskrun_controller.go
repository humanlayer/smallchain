package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/pkg/errors"
)

// TaskRunReconciler reconciles a TaskRun object
type TaskRunReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// getTask fetches the referenced Task
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
		return nil, fmt.Errorf("task %q is not ready", taskRun.Spec.TaskRef.Name)
	}

	return task, nil
}

// Reconcile executes the task and updates the status
func (r *TaskRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var taskRun kubechainv1alpha1.TaskRun
	if err := r.Get(ctx, req.NamespacedName, &taskRun); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Starting reconciliation", "name", taskRun.Name)

	// Create a copy for status update
	statusUpdate := taskRun.DeepCopy()

	// Skip if already completed
	if statusUpdate.Status.Phase == kubechainv1alpha1.TaskRunPhaseFinalAnswer ||
		statusUpdate.Status.Phase == kubechainv1alpha1.TaskRunPhaseErrorBackoff ||
		statusUpdate.Status.Phase == kubechainv1alpha1.TaskRunPhaseFailed {
		return ctrl.Result{}, nil
	}

	// Initialize if new
	if statusUpdate.Status.Phase == "" {
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhasePending
		now := metav1.Now()
		statusUpdate.Status.StartTime = &now

		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Unable to update TaskRun status")
			return ctrl.Result{}, err
		}

		r.recorder.Event(&taskRun, corev1.EventTypeNormal, "Started", "Started processing TaskRun")
		return ctrl.Result{Requeue: true}, nil
	}

	// Get referenced Task
	if statusUpdate.Status.Phase == kubechainv1alpha1.TaskRunPhasePending {
		_, err := r.getTask(ctx, &taskRun)
		if err != nil {
			logger.Error(err, "Task validation failed")
			statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFailed
			statusUpdate.Status.Error = err.Error()
			statusUpdate.Status.Ready = false
			now := metav1.Now()
			statusUpdate.Status.CompletionTime = &now

			if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
				logger.Error(updateErr, "Failed to update TaskRun status")
				return ctrl.Result{}, errors.Wrap(err, fmt.Sprintf("failed to update task run status: %v", err))
			}

			r.recorder.Event(&taskRun, corev1.EventTypeWarning, "TaskValidationFailed", err.Error())
			return ctrl.Result{}, err
		}

		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseReadyForLLM
		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Unable to update TaskRun status")
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	logger.Info("Successfully reconciled taskrun",
		"name", taskRun.Name,
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
