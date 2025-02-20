package controller

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
)

// TaskRunReconciler reconciles a TaskRun object
type TaskRunReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	recorder ctrl.EventRecorder
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

	// todo handle backoff resolution

	// Initialize if new
	if statusUpdate.Status.Phase == "" {
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhasePending
		now := metav1.Now()
		statusUpdate.Status.StartTime = &now

		// Update status immediately to ensure Pending state is visible
		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Unable to update TaskRun status")
			return ctrl.Result{}, err
		}

		// Requeue to continue processing
		return ctrl.Result{Requeue: true}, nil
	}

	// Get referenced Task
	task, err := r.getTask(ctx, &taskRun)
	if err != nil {
		logger.Error(err, "Task validation failed")
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFailed
		statusUpdate.Status.Error = err.Error()
		statusUpdate.Status.Ready = false
		now := metav1.Now()
		statusUpdate.Status.CompletionTime = &now

		// Record failure event
		r.recorder.Event(&taskRun, corev1.EventTypeWarning, "TaskValidationFailed", err.Error())

		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return ctrl.Result{}, errors.Wrap(err, fmt.Sprintf("failed to update task run status: %v", err))
		}

		return ctrl.Result{}, err // requeue
	} else {
		// Mark as Running if not already done
		if statusUpdate.Status.Phase == kubechainv1alpha1.TaskRunPhasePending {
			statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseReadyForLLM
			now := metav1.Now()
			statusUpdate.Status.StartTime = &now

			// Initialize context window with system message and user input
			agent := &kubechainv1alpha1.Agent{}
			if err := r.Get(ctx, client.ObjectKey{
				Namespace: task.Namespace,
				Name:      task.Spec.AgentRef.Name,
			}, agent); err != nil {
				logger.Error(err, "Failed to get agent")
				statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFailed
				statusUpdate.Status.Error = fmt.Sprintf("Failed to get agent: %v", err)
				now := metav1.Now()
				statusUpdate.Status.CompletionTime = &now
				return ctrl.Result{}, nil
			}

			statusUpdate.Status.ContextWindow = []kubechainv1alpha1.Message{
				{
					Role:    "system",
					Content: agent.Spec.System,
				},
				{
					Role:    "user",
					Content: task.Spec.Message,
				},
				{
					Role:    "assistant",
					Content: "I'll help calculate that for you.",
					ToolCalls: []kubechainv1alpha1.ToolCall{
						{
							Name:      "add",
							Arguments: `{"a": 20, "b": 30}`,
						},
					},
				},
			}

			// Update message count and user preview
			statusUpdate.Status.MessageCount = len(statusUpdate.Status.ContextWindow)
			for _, msg := range statusUpdate.Status.ContextWindow {
				if msg.Role == "user" {
					if len(msg.Content) > 50 {
						statusUpdate.Status.UserMsgPreview = msg.Content[:50]
					} else {
						statusUpdate.Status.UserMsgPreview = msg.Content
					}
					break
				}
			}
		}

		// For now, simulate tool call completion
		if len(statusUpdate.Status.ContextWindow) > 0 {
			lastMsg := &statusUpdate.Status.ContextWindow[len(statusUpdate.Status.ContextWindow)-1]
			if len(lastMsg.ToolCalls) > 0 && lastMsg.ToolCalls[0].Result == "" {
				lastMsg.ToolCalls[0].Result = "50"

				// Add final assistant message with result
				statusUpdate.Status.ContextWindow = append(statusUpdate.Status.ContextWindow, kubechainv1alpha1.Message{
					Role:    "assistant",
					Content: "The result of 20 + 30 is 50",
				})

				// Update message count after adding new message
				statusUpdate.Status.MessageCount = len(statusUpdate.Status.ContextWindow)
				statusUpdate.Status.Output = "The result of 20 + 30 is 50"
				statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFinalAnswer
				statusUpdate.Status.Ready = true
				now := metav1.Now()
				statusUpdate.Status.CompletionTime = &now
			}
		}
	}

	// Update status
	if err := r.Status().Update(ctx, statusUpdate); err != nil {
		logger.Error(err, "Unable to update TaskRun status")
		return ctrl.Result{}, err
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
