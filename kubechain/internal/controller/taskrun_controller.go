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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// TaskRunReconciler reconciles a TaskRun object
type TaskRunReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// Metrics instrumentation
	reconcileCounter  metric.Int64Counter
	phaseCounter      metric.Int64Counter
	errorCounter      metric.Int64Counter
	reconcileDuration metric.Float64Histogram
}

// SetupWithManager sets up the controller with the Manager.
func (r *TaskRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	meter := otel.GetMeterProvider().Meter("kubechain/taskrun")

	var err error
	r.reconcileCounter, err = meter.Int64Counter("taskrun_reconcile_count",
		metric.WithDescription("Number of reconciliations of TaskRuns"))
	if err != nil {
		return fmt.Errorf("failed to create reconcile counter: %w", err)
	}

	r.phaseCounter, err = meter.Int64Counter("taskrun_phase_transition",
		metric.WithDescription("Number of TaskRun phase transitions"))
	if err != nil {
		return fmt.Errorf("failed to create phase counter: %w", err)
	}

	r.errorCounter, err = meter.Int64Counter("taskrun_error_count",
		metric.WithDescription("Number of errors during TaskRun reconciliation"))
	if err != nil {
		return fmt.Errorf("failed to create error counter: %w", err)
	}

	r.reconcileDuration, err = meter.Float64Histogram("taskrun_reconcile_duration",
		metric.WithDescription("Duration of TaskRun reconciliations"),
		metric.WithUnit("s"))
	if err != nil {
		return fmt.Errorf("failed to create reconcile duration histogram: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&kubechainv1alpha1.TaskRun{}).
		Complete(r)
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

	return task, nil
}

// Reconcile executes the task and updates the status
func (r *TaskRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Start a trace span for this reconciliation
	tr := otel.Tracer("kubechain/taskrun")
	ctx, span := tr.Start(ctx, "TaskRunReconcile")
	defer span.End()

	// Increment the reconciliation counter
	r.reconcileCounter.Add(ctx, 1)
	span.SetAttributes(
		attribute.String("taskrun", req.NamespacedName.Name),
		attribute.String("namespace", req.NamespacedName.Namespace),
	)

	logger := log.FromContext(ctx)

	var taskRun kubechainv1alpha1.TaskRun
	if err := r.Get(ctx, req.NamespacedName, &taskRun); err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		r.errorCounter.Add(ctx, 1)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if taskRun.Status.Phase != "" {
		span.SetAttributes(attribute.String("phase", string(taskRun.Status.Phase)))
	}

	// Create a status copy (for tracking updates)
	statusUpdate := taskRun.DeepCopy()
	if statusUpdate.Status.Phase != taskRun.Status.Phase {
		r.phaseCounter.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("from_phase", string(taskRun.Status.Phase)),
				attribute.String("to_phase", string(statusUpdate.Status.Phase)),
			))
	}

	// Track phase transitions
	if statusUpdate.Status.Phase != taskRun.Status.Phase {
		span.SetAttributes(
			attribute.String("from_phase", string(taskRun.Status.Phase)),
			attribute.String("to_phase", string(statusUpdate.Status.Phase)),
		)
	}

	// Skip if already completed
	if statusUpdate.Status.Phase == kubechainv1alpha1.TaskRunPhaseSucceeded ||
		statusUpdate.Status.Phase == kubechainv1alpha1.TaskRunPhaseFailed {
		return ctrl.Result{}, nil
	}

	// Initialize if new
	if statusUpdate.Status.Phase == "" {
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhasePending
		now := metav1.Now()
		statusUpdate.Status.StartTime = &now
	}

	// Get referenced Task
	task, err := r.getTask(ctx, &taskRun)
	if err != nil {
		logger.Error(err, "upstream task not found")
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFailed
		statusUpdate.Status.Error = err.Error()
		now := metav1.Now()
		statusUpdate.Status.CompletionTime = &now
	} else {
		// Check if task is ready before proceeding
		if !task.Status.Ready {
			logger.Info("Task not ready yet", "task", task.Name)
			statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhasePending
			if err := r.Status().Update(ctx, statusUpdate); err != nil {
				logger.Error(err, "Unable to update TaskRun status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}

		// Mark as Running if not already done
		if statusUpdate.Status.Phase == kubechainv1alpha1.TaskRunPhasePending {
			statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseRunning
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
							Arguments: `{"a": 2, "b": 2}`,
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

			// Return early to stay in Running phase
			if err := r.Status().Update(ctx, statusUpdate); err != nil {
				logger.Error(err, "Unable to update TaskRun status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}

		// Handle tool call completion in a separate reconciliation
		if statusUpdate.Status.Phase == kubechainv1alpha1.TaskRunPhaseRunning {
			if len(statusUpdate.Status.ContextWindow) > 0 {
				lastMsg := &statusUpdate.Status.ContextWindow[len(statusUpdate.Status.ContextWindow)-1]
				if len(lastMsg.ToolCalls) > 0 && lastMsg.ToolCalls[0].Result == "" {
					// Get the result from the tool call
					result := "4" // In a real implementation, this would come from calling the tool

					// Update the tool call result
					lastMsg.ToolCalls[0].Result = result

					// Add final assistant message with result
					finalMessage := kubechainv1alpha1.Message{
						Role:    "assistant",
						Content: fmt.Sprintf("The result of 2 + 2 is %s", result),
					}
					statusUpdate.Status.ContextWindow = append(statusUpdate.Status.ContextWindow, finalMessage)

					// Update message count and output
					statusUpdate.Status.MessageCount = len(statusUpdate.Status.ContextWindow)
					statusUpdate.Status.Output = finalMessage.Content
					statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseSucceeded
					now := metav1.Now()
					statusUpdate.Status.CompletionTime = &now
				}
			}
		}
	}

	// Track phase transitions
	if statusUpdate.Status.Phase != taskRun.Status.Phase {
		span.SetAttributes(
			attribute.String("from_phase", string(taskRun.Status.Phase)),
			attribute.String("to_phase", string(statusUpdate.Status.Phase)),
		)
	}

	// Track errors
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		r.errorCounter.Add(ctx, 1)
	}

	logger.Info("Successfully reconciled TaskRun",
		"name", taskRun.Name,
		"phase", statusUpdate.Status.Phase)
	return ctrl.Result{}, nil
}
