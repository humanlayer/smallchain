package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	"github.com/openai/openai-go"

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
		return nil, fmt.Errorf("task %q is not ready", task.Name)
	}

	return task, nil
}

// Reconcile validates the taskrun's task reference and sends the prompt to the LLM.
func (r *TaskRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Starting TaskRunToolCall reconciliation", "request", req)

	var taskRun kubechainv1alpha1.TaskRun
	if err := r.Get(ctx, req.NamespacedName, &taskRun); err != nil {
		logger.Error(err, "Failed to get TaskRun")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Starting reconciliation", "name", taskRun.Name)
	logger.Info("Processing TaskRun",
		"phase", taskRun.Status.Phase,
		"status", taskRun.Status.Status,
		"taskRef", taskRun.Spec.TaskRef.Name,
	)

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
	if statusUpdate.Status.Phase == "" {
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

	// Handle the ToolCallsPending phase
	if taskRun.Status.Phase == kubechainv1alpha1.TaskRunPhaseToolCallsPending {
		// List all tool calls owned by this TaskRun
		var toolCalls kubechainv1alpha1.TaskRunToolCallList

		if err := r.List(ctx, &toolCalls,
			client.InNamespace(req.Namespace),
			client.MatchingFields{"spec.taskRunRef.name": taskRun.Name},
		); err != nil {
			logger.Error(err, "Failed to list TaskRunToolCalls")
			return ctrl.Result{}, err
		}

		logger.Info("Found tool calls", "count", len(toolCalls.Items))
		for i, tc := range toolCalls.Items {
			logger.Info("Tool call details",
				"index", i,
				"name", tc.Name,
				"tool", tc.Spec.ToolRef.Name,
				"phase", tc.Status.Phase,
				"result", tc.Status.Result)
		}

		allCompleted := true
		for _, tc := range toolCalls.Items {
			if tc.Status.Phase != kubechainv1alpha1.TaskRunToolCallPhaseSucceeded {
				allCompleted = false
				break
			}
		}

		if allCompleted {
			// Append tool call results to task message
			var toolResults strings.Builder
			toolResults.WriteString("\nThe following tool calls have been executed:\n")
			for _, tc := range toolCalls.Items {
				toolResults.WriteString(fmt.Sprintf("- Tool: %s, Result: %s\n", tc.Spec.ToolRef.Name, tc.Status.Result))
			}
			task.Spec.Message += toolResults.String()

			// Update context window
			statusUpdate.Status.ContextWindow = append(statusUpdate.Status.ContextWindow, kubechainv1alpha1.Message{
				Role:    "user",
				Content: toolResults.String(),
			})

			// Transition to ReadyForLLM
			statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseReadyForLLM
			statusUpdate.Status.Status = "Ready"
			statusUpdate.Status.StatusDetail = "All tool calls completed"

			if err := r.Status().Update(ctx, statusUpdate); err != nil {
				logger.Error(err, "Failed to update TaskRun status")
				return ctrl.Result{}, err
			}

			logger.Info("Updating Task message", "task", task.Name, "message", task.Spec.Message)
			if err := r.Update(ctx, task); err != nil {
				logger.Error(err, "Failed to update Task message")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}

		// Tool calls are still pending
		logger.Info("Tool calls pending, requeueing TaskRun", "taskrun", taskRun.Name, "requeueAfter", time.Second*5)
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	// Only proceed with LLM request if we're in ReadyForLLM phase
	if taskRun.Status.Phase != kubechainv1alpha1.TaskRunPhaseReadyForLLM {
		logger.Info("TaskRun not ready for LLM", "phase", taskRun.Status.Phase)
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
	logger.Info("Created LLM client", "client", llmClient)
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

	var tools []openai.ChatCompletionToolParam
	// Get the tools from the agent's ValidTools
	for _, toolName := range agent.Spec.Tools {
		tool := &kubechainv1alpha1.Tool{}
		err := r.Get(ctx, client.ObjectKey{
			Namespace: agent.Namespace,
			Name:      toolName.Name,
		}, tool)
		if err != nil {
			logger.Error(err, "Failed to get Tool", "tool", toolName)
			continue
		}

		// Convert the tool's arguments to a JSON schema
		toolParam := openai.ChatCompletionToolParam{
			Type: openai.F(openai.ChatCompletionToolTypeFunction),
			Function: openai.F(openai.FunctionDefinitionParam{
				Name:        openai.F(tool.Name),
				Description: openai.F(tool.Spec.Description),
			}),
		}

		// If the tool has arguments defined, add them to the function definition
		if tool.Spec.Parameters.Raw != nil {
			var parameters openai.FunctionParameters
			if err := json.Unmarshal(tool.Spec.Parameters.Raw, &parameters); err != nil {
				logger.Error(err, "Failed to unmarshal tool arguments", "tool", toolName)
				continue
			}
			toolParam.Function.Value.Parameters = openai.F(parameters)
		}

		tools = append(tools, toolParam)
	}

	// Send the prompt to the LLM using the OpenAI client

	// Before the SendRequest call
	logger.Info("Sending LLM request",
		"taskrun", taskRun.Name,
		"systemPrompt", agent.Spec.System,
		"userMessage", task.Spec.Message,
		"toolCount", len(tools))

	output, err := llmClient.SendRequest(ctx, agent.Spec.System, task.Spec.Message, tools)
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

	logger.Info("LLM response received",
		"taskrun", taskRun.Name,
		"hasContent", output.Content != "",
		"toolCallCount", len(output.ToolCalls))

	for i, tc := range output.ToolCalls {
		logger.Info("Tool call from LLM",
			"index", i,
			"name", tc.Function.Name,
			"arguments", tc.Function.Arguments)
	}
	logger.Info("LLM call completed", "taskrun", taskRun.Name, "output", output)

	if output.Content != "" {
		// Final answer branch
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
	} else if len(output.ToolCalls) > 0 {
		// Tool call branch: create TaskRunToolCall objects for each tool call returned by the LLM
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

		logger.Info("Updating TaskRun status", "taskrun", taskRun.Name, "status", statusUpdate.Status)

		// Update the parent's status before creating tool call objects
		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Unable to update TaskRun status")
			return ctrl.Result{}, err
		}

		// For each tool call, create a new TaskRunToolCall
		for i, tc := range output.ToolCalls {
			newName := fmt.Sprintf("%s-toolcall-%02d", statusUpdate.Name, i+1)

			// Check if a TaskRunToolCall with this name already exists
			existingTRTC := &kubechainv1alpha1.TaskRunToolCall{}
			err := r.Get(ctx, client.ObjectKey{Namespace: statusUpdate.Namespace, Name: newName}, existingTRTC)

			logger.Info("Tool call arguments", "functionArguments", tc.Function.Arguments)
			// Only create if it doesn't exist
			if err != nil && client.IgnoreNotFound(err) == nil {
				// Create new TaskRunToolCall
				newTRTC := &kubechainv1alpha1.TaskRunToolCall{
					ObjectMeta: metav1.ObjectMeta{
						Name:      newName,
						Namespace: statusUpdate.Namespace,
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: statusUpdate.APIVersion,
								Kind:       statusUpdate.Kind,
								Name:       statusUpdate.Name,
								UID:        statusUpdate.UID,
								Controller: pointer.BoolPtr(true),
							},
						},
					},
					Spec: kubechainv1alpha1.TaskRunToolCallSpec{
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
				logger.Info("Creating TaskRunToolCall", "name", newName, "toolCall", newTRTC)
				r.recorder.Event(&taskRun, corev1.EventTypeNormal, "ToolCallCreated", "Created TaskRunToolCall "+newName)
			} else if err == nil {
				// Already exists, log and continue
				logger.Info("TaskRunToolCall already exists", "name", newName, "phase", existingTRTC.Status.Phase)
			} else {
				// Handle other errors
				logger.Error(err, "Error checking if TaskRunToolCall exists", "name", newName)
				return ctrl.Result{}, err
			}
			logger.Info("Created TaskRunToolCall", "name", newName)
			r.recorder.Event(&taskRun, corev1.EventTypeNormal, "ToolCallCreated", "Created TaskRunToolCall "+newName)
		}
	} else {
		// Handle the case where no content and no tool calls were returned
		err := fmt.Errorf("LLM response contained neither content nor tool calls")
		logger.Error(err, "Invalid LLM response")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = "Error"
		statusUpdate.Status.StatusDetail = err.Error()
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(&taskRun, corev1.EventTypeWarning, "InvalidLLMResponse", err.Error())
	}

	// Update status for any branch
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
		r.newLLMClient = llmclient.NewOpenAIClient
	}

	// Add this index for looking up TaskRunToolCalls by parent TaskRun
	if err := mgr.GetFieldIndexer().IndexField(context.Background(),
		&kubechainv1alpha1.TaskRunToolCall{},
		"spec.taskRunRef.name",
		func(o client.Object) []string {
			return []string{o.(*kubechainv1alpha1.TaskRunToolCall).Spec.TaskRunRef.Name}
		}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&kubechainv1alpha1.TaskRun{}).
		Complete(r)
}
