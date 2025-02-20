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

// AgentReconciler reconciles a Agent object
type AgentReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// validateLLM checks if the referenced LLM exists and is ready
func (r *AgentReconciler) validateLLM(ctx context.Context, agent *kubechainv1alpha1.Agent) error {
	llm := &kubechainv1alpha1.LLM{}
	err := r.Get(ctx, client.ObjectKey{
		Namespace: agent.Namespace,
		Name:      agent.Spec.LLMRef.Name,
	}, llm)
	if err != nil {
		return fmt.Errorf("failed to get LLM %q: %w", agent.Spec.LLMRef.Name, err)
	}

	if llm.Status.Status != "Ready" {
		return fmt.Errorf("LLM %q is not ready", agent.Spec.LLMRef.Name)
	}

	return nil
}

// validateTools checks if all referenced tools exist and are ready
func (r *AgentReconciler) validateTools(ctx context.Context, agent *kubechainv1alpha1.Agent) ([]string, error) {
	validTools := make([]string, 0, len(agent.Spec.Tools))

	for _, toolRef := range agent.Spec.Tools {
		tool := &kubechainv1alpha1.Tool{}
		err := r.Get(ctx, client.ObjectKey{
			Namespace: agent.Namespace,
			Name:      toolRef.Name,
		}, tool)
		if err != nil {
			return validTools, fmt.Errorf("failed to get Tool %q: %w", toolRef.Name, err)
		}

		if !tool.Status.Ready {
			return validTools, fmt.Errorf("tool %q is not ready", toolRef.Name)
		}

		validTools = append(validTools, toolRef.Name)
	}

	return validTools, nil
}

// Reconcile validates the agent's LLM and Tool references
func (r *AgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var agent kubechainv1alpha1.Agent
	if err := r.Get(ctx, req.NamespacedName, &agent); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Starting reconciliation", "name", agent.Name)

	// Create a copy for status update
	statusUpdate := agent.DeepCopy()

	// Initialize status if not set
	if statusUpdate.Status.Status == "" {
		statusUpdate.Status.Status = "Pending"
		statusUpdate.Status.StatusDetail = "Validating dependencies"
		r.recorder.Event(&agent, corev1.EventTypeNormal, "Initializing", "Starting validation")
	}

	// Initialize empty valid tools slice
	validTools := make([]string, 0)

	// Validate LLM reference
	if err := r.validateLLM(ctx, &agent); err != nil {
		logger.Error(err, "LLM validation failed")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = "Error"
		statusUpdate.Status.StatusDetail = err.Error()
		statusUpdate.Status.ValidTools = validTools
		r.recorder.Event(&agent, corev1.EventTypeWarning, "ValidationFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update Agent status")
			return ctrl.Result{}, fmt.Errorf("failed to update agent status: %v", err)
		}
		return ctrl.Result{}, err // requeue
	}

	// Validate Tool references
	validTools, err := r.validateTools(ctx, &agent)
	if err != nil {
		logger.Error(err, "Tool validation failed")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = "Error"
		statusUpdate.Status.StatusDetail = err.Error()
		statusUpdate.Status.ValidTools = validTools
		r.recorder.Event(&agent, corev1.EventTypeWarning, "ValidationFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update Agent status")
			return ctrl.Result{}, fmt.Errorf("failed to update agent status: %v", err)
		}
		return ctrl.Result{}, err // requeue
	}

	// All validations passed
	statusUpdate.Status.Ready = true
	statusUpdate.Status.Status = "Ready"
	statusUpdate.Status.StatusDetail = "All dependencies validated successfully"
	statusUpdate.Status.ValidTools = validTools
	r.recorder.Event(&agent, corev1.EventTypeNormal, "ValidationSucceeded", "All dependencies validated successfully")

	// Update status
	if err := r.Status().Update(ctx, statusUpdate); err != nil {
		logger.Error(err, "Unable to update Agent status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled agent",
		"name", agent.Name,
		"ready", statusUpdate.Status.Ready,
		"status", statusUpdate.Status.Status,
		"validTools", statusUpdate.Status.ValidTools)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("agent-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubechainv1alpha1.Agent{}).
		Complete(r)
}
