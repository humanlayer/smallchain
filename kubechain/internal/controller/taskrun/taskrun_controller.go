package taskrun

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"

	"github.com/humanlayer/smallchain/kubechain/internal/adapters"
	"github.com/humanlayer/smallchain/kubechain/internal/llmclient"
)

// TaskRunReconciler reconciles a TaskRun object
type TaskRunReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	recorder     record.EventRecorder
	newLLMClient func(apiKey string) (llmclient.OpenAIClient, error)
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
		return task, nil // Return task but indicate it's not ready
	}

	return task, nil
}

// Reconcile validates the taskrun's task reference and sends the prompt to the LLM.
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
	task, err := r.getTask(ctx, &taskRun)
	if err != nil {
		logger.Error(err, "Task validation failed")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = "Error"
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Task validation failed: %v", err)
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(&taskRun, corev1.EventTypeWarning, "ValidationFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return ctrl.Result{}, fmt.Errorf("failed to update taskrun status: %v", err)
		}
		return ctrl.Result{}, err
	}

	// Check if task exists but is not ready
	if task != nil && !task.Status.Ready {
		logger.Info("Task exists but is not ready", "task", task.Name)
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = "Pending"
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhasePending
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Waiting for task %q to become ready", task.Name)
		statusUpdate.Status.Error = "" // Clear previous error
		r.recorder.Event(&taskRun, corev1.EventTypeNormal, "Waiting", fmt.Sprintf("Waiting for task %q to become ready", task.Name))
		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Failed to update TaskRun status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	// Get the Agent referenced by the Task
	var agent kubechainv1alpha1.Agent
	if err := r.Get(ctx, client.ObjectKey{Namespace: task.Namespace, Name: task.Spec.AgentRef.Name}, &agent); err != nil {
		logger.Error(err, "Failed to get Agent")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = "Pending"
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhasePending
		statusUpdate.Status.StatusDetail = "Waiting for Agent to exist"
		statusUpdate.Status.Error = "" // Clear previous error
		r.recorder.Event(&taskRun, corev1.EventTypeNormal, "Waiting", "Waiting for Agent to exist")
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	// Check if agent is ready
	if !agent.Status.Ready {
		logger.Info("Agent exists but is not ready", "agent", agent.Name)
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = "Pending"
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhasePending
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Waiting for agent %q to become ready", agent.Name)
		statusUpdate.Status.Error = "" // Clear previous error
		r.recorder.Event(&taskRun, corev1.EventTypeNormal, "Waiting", fmt.Sprintf("Waiting for agent %q to become ready", agent.Name))
		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Failed to update TaskRun status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	// Initialize phase if not set
	if statusUpdate.Status.Phase == "" || statusUpdate.Status.Phase == kubechainv1alpha1.TaskRunPhasePending {
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseReadyForLLM
		statusUpdate.Status.Ready = true
		statusUpdate.Status.ContextWindow = []kubechainv1alpha1.Message{
			{
				Role:    "system",
				Content: agent.Spec.System,
			},
			{
				Role:    "user",
				Content: task.Spec.Message,
			},
		}
		statusUpdate.Status.Status = "Ready"
		statusUpdate.Status.StatusDetail = "Ready to send to LLM"
		statusUpdate.Status.Error = "" // Clear any previous error
		r.recorder.Event(&taskRun, corev1.EventTypeNormal, "ValidationSucceeded", "Task validated successfully")
		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Failed to update TaskRun status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Modified section of the Reconcile function to fix the workflow
	if taskRun.Status.Phase == kubechainv1alpha1.TaskRunPhaseToolCallsPending {
		// List all tool calls for this TaskRun
		toolCallList := &kubechainv1alpha1.TaskRunToolCallList{}
		logger.Info("Listing tool calls", "taskrun", taskRun.Name)
		if err := r.List(ctx, toolCallList, client.InNamespace(taskRun.Namespace),
			client.MatchingLabels{"kubechain.humanlayer.dev/taskruntoolcall": taskRun.Name}); err != nil {
			logger.Error(err, "Failed to list tool calls")
			return ctrl.Result{}, err
		}

		logger.Info("Found tool calls", "count", len(toolCallList.Items))

		// Check if all tool calls are complete
		allComplete := true
		var toolResults []kubechainv1alpha1.Message

		for _, tc := range toolCallList.Items {
			logger.Info("Checking tool call", "name", tc.Name, "phase", tc.Status.Phase)
			if tc.Status.Phase != kubechainv1alpha1.TaskRunToolCallPhaseSucceeded {
				allComplete = false
				logger.Info("Found incomplete tool call", "name", tc.Name)
				break
			}
			toolResults = append(toolResults, kubechainv1alpha1.Message{
				ToolCallId: tc.Spec.ToolCallId,
				Role:       "tool",
				Content:    tc.Status.Result,
			})
		}

		if allComplete {
			logger.Info("All tool calls complete, transitioning to ReadyForLLM")
			// All tool calls are complete, update context window and move back to ReadyForLLM phase
			// so that the LLM can process the tool results and provide a final answer
			statusUpdate := taskRun.DeepCopy()
			for _, toolResult := range toolResults {
				statusUpdate.Status.ContextWindow = append(statusUpdate.Status.ContextWindow, toolResult)
			}
			statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseReadyForLLM
			statusUpdate.Status.Ready = true
			statusUpdate.Status.Status = "Ready"
			statusUpdate.Status.StatusDetail = "All tool calls completed, ready to send tool results to LLM"

			if err := r.Status().Update(ctx, statusUpdate); err != nil {
				logger.Error(err, "Failed to update TaskRun status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}

		// Not all tool calls are complete, requeue while staying in ToolCallsPending phase
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	// at this point the only other phase is ReadyForLLM
	if taskRun.Status.Phase != kubechainv1alpha1.TaskRunPhaseReadyForLLM {
		logger.Info("TaskRun in unknown phase", "phase", taskRun.Status.Phase)
		return ctrl.Result{}, nil
	}

	// Get the LLM referenced by the Agent
	var llm kubechainv1alpha1.LLM
	if err := r.Get(ctx, client.ObjectKey{Namespace: agent.Namespace, Name: agent.Spec.LLMRef.Name}, &llm); err != nil {
		logger.Error(err, "Failed to get LLM")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = "Error"
		statusUpdate.Status.StatusDetail = "Failed to get LLM: " + err.Error()
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(&taskRun, corev1.EventTypeWarning, "LLMFetchFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Get the API key from the referenced secret
	var secret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: llm.Namespace,
		Name:      llm.Spec.APIKeyFrom.SecretKeyRef.Name,
	}, &secret); err != nil {
		logger.Error(err, "Failed to get API key secret")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = "Error"
		statusUpdate.Status.StatusDetail = "Failed to get API key secret: " + err.Error()
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(&taskRun, corev1.EventTypeWarning, "SecretFetchFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// todo we probably don't need error handling here, it should be handled by the LLM/Agent/Task validation
	// however, defense in depth
	apiKey := string(secret.Data[llm.Spec.APIKeyFrom.SecretKeyRef.Key])
	if apiKey == "" {
		err := fmt.Errorf("API key is empty in secret %s", secret.Name)
		logger.Error(err, "Empty API key")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = "Error"
		statusUpdate.Status.StatusDetail = "API key is empty"
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(&taskRun, corev1.EventTypeWarning, "EmptyAPIKey", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	llmClient, err := r.newLLMClient(apiKey)
	if err != nil {
		logger.Error(err, "Failed to create OpenAI client")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = "Error"
		statusUpdate.Status.StatusDetail = "Failed to create OpenAI client: " + err.Error()
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(&taskRun, corev1.EventTypeWarning, "OpenAIClientCreationFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	var tools []llmclient.Tool
	for _, toolName := range agent.Status.ValidTools {
		tool := &kubechainv1alpha1.Tool{}
		err := r.Get(ctx, client.ObjectKey{
			Namespace: agent.Namespace,
			Name:      toolName.Name,
		}, tool)
		if err != nil {
			logger.Error(err, "Failed to get Tool", "tool", toolName)
			continue
		}

		if converted := llmclient.FromKubechainTool(*tool); converted != nil {
			tools = append(tools, *converted)
		}
	}

	// Send the prompt to the LLM using the OpenAI client.
	output, err := llmClient.SendRequest(ctx, taskRun.Status.ContextWindow, tools)

	if err != nil {
		logger.Error(err, "LLM request failed")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = "Error"
		statusUpdate.Status.StatusDetail = fmt.Sprintf("LLM request failed: %v", err)
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(&taskRun, corev1.EventTypeWarning, "LLMRequestFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status after LLM error")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	if output.Content != "" {
		// final answer branch
		statusUpdate.Status.Output = output.Content
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFinalAnswer
		statusUpdate.Status.Ready = true
		statusUpdate.Status.ContextWindow = append(statusUpdate.Status.ContextWindow, kubechainv1alpha1.Message{
			Role:    "assistant",
			Content: output.Content,
		})
		statusUpdate.Status.Status = "Ready"
		statusUpdate.Status.StatusDetail = "LLM final response received"
		statusUpdate.Status.Error = ""
		r.recorder.Event(&taskRun, corev1.EventTypeNormal, "LLMFinalAnswer", "LLM response received successfully")
	} else {
		// tool call branch: create TaskRunToolCall objects for each tool call returned by the LLM.
		statusUpdate.Status.Output = ""
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseToolCallsPending
		statusUpdate.Status.ContextWindow = append(statusUpdate.Status.ContextWindow, kubechainv1alpha1.Message{
			Role:      "assistant",
			ToolCalls: adapters.CastOpenAIToolCallsToKubechain(output.ToolCalls),
		})
		statusUpdate.Status.Ready = true
		statusUpdate.Status.Status = "Ready"
		statusUpdate.Status.StatusDetail = "LLM response received, tool calls pending"
		statusUpdate.Status.Error = ""

		// Update the parent's status before creating tool call objects.
		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Unable to update TaskRun status")
			return ctrl.Result{}, err
		}

		// For each tool call, create a new TaskRunToolCall.
		// Using the parent's details from statusUpdate.
		statusUpdate = statusUpdate.DeepCopy()
		for i, tc := range output.ToolCalls {
			newName := fmt.Sprintf("%s-toolcall-%02d", statusUpdate.Name, i+1)
			newTRTC := &kubechainv1alpha1.TaskRunToolCall{
				ObjectMeta: metav1.ObjectMeta{
					Name:      newName,
					Namespace: statusUpdate.Namespace,
					Labels: map[string]string{
						"kubechain.humanlayer.dev/taskruntoolcall": statusUpdate.Name,
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubechain.humanlayer.dev/v1alpha1",
							Kind:       "TaskRun",
							Name:       statusUpdate.Name,
							UID:        statusUpdate.UID,
							Controller: pointer.BoolPtr(true),
						},
					},
				},
				Spec: kubechainv1alpha1.TaskRunToolCallSpec{
					ToolCallId: tc.ID,
					TaskRunRef: kubechainv1alpha1.LocalObjectReference{
						Name: statusUpdate.Name,
					},
					ToolRef: kubechainv1alpha1.LocalObjectReference{
						Name: tc.Function.Name,
					},
					Arguments: tc.Function.Arguments,
				},
			}
			if err := r.Client.Create(ctx, newTRTC); err != nil {
				logger.Error(err, "Failed to create TaskRunToolCall", "name", newName)
				return ctrl.Result{}, err
			}
			logger.Info("Created TaskRunToolCall", "name", newName)
			r.recorder.Event(&taskRun, corev1.EventTypeNormal, "ToolCallCreated", "Created TaskRunToolCall "+newName)
		}
	}
	// Update status for either branch.
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
	if r.newLLMClient == nil {
		r.newLLMClient = llmclient.NewRawOpenAIClient
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubechainv1alpha1.TaskRun{}).
		Complete(r)
}
