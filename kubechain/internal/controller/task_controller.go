package controller

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
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
		return fmt.Errorf("agent %q is not ready", task.Spec.AgentRef.Name)
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
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update Task status")
			return ctrl.Result{}, fmt.Errorf("failed to update task status: %v", err)
		}
		return ctrl.Result{}, err // requeue
	} else {
		statusUpdate.Status.Ready = true
		statusUpdate.Status.Status = "Task Run Created"

		// Check if we need to create a TaskRun
		taskRunList := &kubechainv1alpha1.TaskRunList{}
		if err := r.List(ctx, taskRunList, client.InNamespace(task.Namespace), client.MatchingLabels{
			"kubechain.humanlayer.dev/task": task.Name,
		}); err != nil {
			logger.Error(err, "Failed to list TaskRuns")
			return ctrl.Result{}, err
		}

		if len(taskRunList.Items) == 0 {
			// Create a new TaskRun
			taskRun := &kubechainv1alpha1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      task.Name + "-1",
					Namespace: task.Namespace,
					Labels: map[string]string{
						"kubechain.humanlayer.dev/task": task.Name,
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: kubechainv1alpha1.GroupVersion.String(),
							Kind:       "Task",
							Name:       task.Name,
							UID:        task.UID,
							Controller: pointer.Bool(true),
						},
					},
				},
				Spec: kubechainv1alpha1.TaskRunSpec{
					TaskRef: kubechainv1alpha1.LocalObjectReference{
						Name: task.Name,
					},
				},
			}

			if err := r.Create(ctx, taskRun); err != nil {
				logger.Error(err, "Failed to create TaskRun")
				return ctrl.Result{}, err
			}
			logger.Info("Created TaskRun", "name", taskRun.Name)
		}
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
