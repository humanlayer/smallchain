package taskrun

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"

	"github.com/humanlayer/smallchain/kubechain/internal/adapters"
	"github.com/humanlayer/smallchain/kubechain/internal/llmclient"
	"github.com/humanlayer/smallchain/kubechain/internal/mcpmanager"
	"go.opentelemetry.io/otel"
)

const (
	StatusReady   = "Ready"
	StatusError   = "Error"
	StatusPending = "Pending"
)

// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=taskruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=taskruns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=tasks,verbs=get;list;watch
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=agents,verbs=get;list;watch
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=llms,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// TaskRunReconciler reconciles a TaskRun object
type TaskRunReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	recorder     record.EventRecorder
	newLLMClient func(apiKey string) (llmclient.OpenAIClient, error)
	MCPManager   *mcpmanager.MCPServerManager
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

// validateTaskAndAgent checks if the task and agent exist and are ready
func (r *TaskRunReconciler) validateTaskAndAgent(ctx context.Context, taskRun *kubechainv1alpha1.TaskRun, statusUpdate *kubechainv1alpha1.TaskRun) (*kubechainv1alpha1.Task, *kubechainv1alpha1.Agent, ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Get parent Task
	task, err := r.getTask(ctx, taskRun)
	if err != nil {
		r.recorder.Event(taskRun, corev1.EventTypeWarning, "TaskValidationFailed", err.Error())
		if apierrors.IsNotFound(err) {
			logger.Info("Task %q not found")
			statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFailed
			statusUpdate.Status.Error = fmt.Sprintf("Task %q not found", taskRun.Spec.TaskRef.Name)
			statusUpdate.Status.StatusDetail = fmt.Sprintf("Task %q not found", taskRun.Spec.TaskRef.Name)
			statusUpdate.Status.Status = StatusError
		} else {
			logger.Error(err, "Task validation failed")
			statusUpdate.Status.Ready = false
			statusUpdate.Status.Status = StatusError
			statusUpdate.Status.StatusDetail = fmt.Sprintf("Task validation failed: %v", err)
			statusUpdate.Status.Error = err.Error()
		}

		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return nil, nil, ctrl.Result{}, fmt.Errorf("failed to update taskrun status: %v", err)
		}
		// todo dont error if not found, don't requeue
		// (can use client.IgnoreNotFound(err), but today
		// the parent method needs err != nil to break control flow properly)
		return nil, nil, ctrl.Result{}, err
	}

	// Check if task exists but is not ready
	if task != nil && !task.Status.Ready {
		logger.Info("Task exists but is not ready", "task", task.Name)
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusPending
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhasePending
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Waiting for task %q to become ready", task.Name)
		statusUpdate.Status.Error = "" // Clear previous error
		r.recorder.Event(taskRun, corev1.EventTypeNormal, "TaskNotReady", fmt.Sprintf("Waiting for task %q to become ready", task.Name))
		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Failed to update TaskRun status")
			return nil, nil, ctrl.Result{}, err
		}
		return nil, nil, ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	// Get the Agent referenced by the Task
	var agent kubechainv1alpha1.Agent
	if err := r.Get(ctx, client.ObjectKey{Namespace: task.Namespace, Name: task.Spec.AgentRef.Name}, &agent); err != nil {
		logger.Error(err, "Failed to get Agent")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusPending
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhasePending
		statusUpdate.Status.StatusDetail = "Waiting for Agent to exist"
		statusUpdate.Status.Error = "" // Clear previous error
		r.recorder.Event(taskRun, corev1.EventTypeNormal, "Waiting", "Waiting for Agent to exist")
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return nil, nil, ctrl.Result{}, updateErr
		}
		return nil, nil, ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	// Check if agent is ready
	if !agent.Status.Ready {
		logger.Info("Agent exists but is not ready", "agent", agent.Name)
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusPending
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhasePending
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Waiting for agent %q to become ready", agent.Name)
		statusUpdate.Status.Error = "" // Clear previous error
		r.recorder.Event(taskRun, corev1.EventTypeNormal, "Waiting", fmt.Sprintf("Waiting for agent %q to become ready", agent.Name))
		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Failed to update TaskRun status")
			return nil, nil, ctrl.Result{}, err
		}
		return nil, nil, ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	return task, &agent, ctrl.Result{}, nil
}

// prepareForLLM sets up the initial state of a TaskRun with the correct context and phase
func (r *TaskRunReconciler) prepareForLLM(ctx context.Context, taskRun *kubechainv1alpha1.TaskRun, statusUpdate *kubechainv1alpha1.TaskRun, task *kubechainv1alpha1.Task, agent *kubechainv1alpha1.Agent) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if statusUpdate.Status.Phase == kubechainv1alpha1.TaskRunPhaseInitializing || statusUpdate.Status.Phase == kubechainv1alpha1.TaskRunPhasePending {
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
		statusUpdate.Status.Status = StatusReady
		statusUpdate.Status.StatusDetail = "Ready to send to LLM"
		statusUpdate.Status.Error = "" // Clear any previous error
		r.recorder.Event(taskRun, corev1.EventTypeNormal, "ValidationSucceeded", "Task validated successfully")
		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Failed to update TaskRun status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// processToolCalls handles the ToolCallsPending phase by checking tool call completion
func (r *TaskRunReconciler) processToolCalls(ctx context.Context, taskRun *kubechainv1alpha1.TaskRun) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// List all tool calls for this TaskRun
	toolCallList := &kubechainv1alpha1.TaskRunToolCallList{}
	logger.Info("Listing tool calls", "taskrun", taskRun.Name)
	if err := r.List(ctx, toolCallList, client.InNamespace(taskRun.Namespace),
		client.MatchingLabels{"kubechain.humanlayer.dev/taskruntoolcall": taskRun.Name}); err != nil {
		logger.Error(err, "Failed to list tool calls")
		return ctrl.Result{}, err
	}

	logger.Info("Found tool calls", "count", len(toolCallList.Items))
	if len(toolCallList.Items) == 0 {
		logger.Info("No tool calls found, something is very wrong")
		r.recorder.Event(taskRun, corev1.EventTypeWarning, "ToolCallsPendingWithNoToolCalls", "TaskRun is in ToolCallsPending phase but no tool calls were found")
		return ctrl.Result{}, fmt.Errorf("TaskRun %q is in ToolCallsPending phase but no tool calls were found", taskRun.Name)
	}

	// Check if all tool calls are complete
	allComplete := true
	toolResults := make([]kubechainv1alpha1.Message, 0, len(toolCallList.Items))

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
		statusUpdate.Status.ContextWindow = append(statusUpdate.Status.ContextWindow, toolResults...)
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseReadyForLLM
		statusUpdate.Status.Ready = true
		statusUpdate.Status.Status = StatusReady
		statusUpdate.Status.StatusDetail = "All tool calls completed, ready to send tool results to LLM"
		r.recorder.Event(taskRun, corev1.EventTypeNormal, "AllToolCallsCompleted", "All tool calls completed, ready to send tool results to LLM")

		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Failed to update TaskRun status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Not all tool calls are complete, requeue while staying in ToolCallsPending phase
	return ctrl.Result{RequeueAfter: time.Second * 5}, nil
}

// getLLMAndCredentials fetches the LLM and its API key from the referenced secret
func (r *TaskRunReconciler) getLLMAndCredentials(ctx context.Context, agent *kubechainv1alpha1.Agent, taskRun *kubechainv1alpha1.TaskRun, statusUpdate *kubechainv1alpha1.TaskRun) (kubechainv1alpha1.LLM, string, error) {
	logger := log.FromContext(ctx)

	// Get the LLM referenced by the Agent
	var llm kubechainv1alpha1.LLM
	if err := r.Get(ctx, client.ObjectKey{Namespace: agent.Namespace, Name: agent.Spec.LLMRef.Name}, &llm); err != nil {
		logger.Error(err, "Failed to get LLM")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.StatusDetail = "Failed to get LLM: " + err.Error()
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(taskRun, corev1.EventTypeWarning, "LLMFetchFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return llm, "", updateErr
		}
		return llm, "", err
	}

	// Get the API key from the referenced secret
	var secret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: llm.Namespace,
		Name:      llm.Spec.APIKeyFrom.SecretKeyRef.Name,
	}, &secret); err != nil {
		logger.Error(err, "Failed to get API key secret")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.StatusDetail = "Failed to get API key secret: " + err.Error()
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(taskRun, corev1.EventTypeWarning, "SecretFetchFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return llm, "", updateErr
		}
		return llm, "", err
	}

	// Validate API key
	apiKey := string(secret.Data[llm.Spec.APIKeyFrom.SecretKeyRef.Key])
	if apiKey == "" {
		err := fmt.Errorf("API key is empty in secret %s", secret.Name)
		logger.Error(err, "Empty API key")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.StatusDetail = "API key is empty"
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(taskRun, corev1.EventTypeWarning, "EmptyAPIKey", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return llm, "", updateErr
		}
		return llm, "", err
	}

	return llm, apiKey, nil
}

// collectTools gathers tools from all sources (Tool CRDs and MCP servers)
func (r *TaskRunReconciler) collectTools(ctx context.Context, agent *kubechainv1alpha1.Agent) []llmclient.Tool {
	logger := log.FromContext(ctx)
	var tools []llmclient.Tool

	// First, add tools from traditional Tool CRDs
	if len(agent.Status.ValidTools) > 0 {
		logger.Info("Adding traditional tools to LLM request", "toolCount", len(agent.Status.ValidTools))

		for _, validTool := range agent.Status.ValidTools {
			if validTool.Kind != "Tool" {
				continue
			}

			// Get the Tool resource
			tool := &kubechainv1alpha1.Tool{}
			if err := r.Get(ctx, client.ObjectKey{Namespace: agent.Namespace, Name: validTool.Name}, tool); err != nil {
				logger.Error(err, "Failed to get Tool", "name", validTool.Name)
				continue
			}

			// Convert to LLM client format
			if clientTool := llmclient.FromKubechainTool(*tool); clientTool != nil {
				tools = append(tools, *clientTool)
				logger.Info("Added traditional tool", "name", tool.Name)
			}
		}
	}

	// Then, add tools from MCP servers if available
	if r.MCPManager != nil && len(agent.Status.ValidMCPServers) > 0 {
		logger.Info("Adding MCP tools to LLM request", "mcpServerCount", len(agent.Status.ValidMCPServers))

		for _, mcpServer := range agent.Status.ValidMCPServers {
			// Get tools for this server
			mcpTools, exists := r.MCPManager.GetTools(mcpServer.Name)
			if !exists {
				logger.Error(fmt.Errorf("MCP server tools not found"), "Failed to get tools for MCP server", "server", mcpServer.Name)
				continue
			}

			// Convert MCP tools to LLM client format
			mcpClientTools := adapters.ConvertMCPToolsToLLMClientTools(mcpTools, mcpServer.Name)
			tools = append(tools, mcpClientTools...)

			logger.Info("Added MCP tools", "server", mcpServer.Name, "toolCount", len(mcpTools))
		}
	}

	return tools
}

// processLLMResponse handles the LLM's output and updates status accordingly
func (r *TaskRunReconciler) processLLMResponse(ctx context.Context, output *kubechainv1alpha1.Message, taskRun *kubechainv1alpha1.TaskRun, statusUpdate *kubechainv1alpha1.TaskRun) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if output.Content != "" {
		// final answer branch
		statusUpdate.Status.Output = output.Content
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFinalAnswer
		statusUpdate.Status.Ready = true
		statusUpdate.Status.ContextWindow = append(statusUpdate.Status.ContextWindow, kubechainv1alpha1.Message{
			Role:    "assistant",
			Content: output.Content,
		})
		statusUpdate.Status.Status = StatusReady
		statusUpdate.Status.StatusDetail = "LLM final response received"
		statusUpdate.Status.Error = ""
		r.recorder.Event(taskRun, corev1.EventTypeNormal, "LLMFinalAnswer", "LLM response received successfully")
	} else {
		// tool call branch: create TaskRunToolCall objects for each tool call returned by the LLM.
		statusUpdate.Status.Output = ""
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseToolCallsPending
		statusUpdate.Status.ContextWindow = append(statusUpdate.Status.ContextWindow, kubechainv1alpha1.Message{
			Role:      "assistant",
			ToolCalls: adapters.CastOpenAIToolCallsToKubechain(output.ToolCalls),
		})
		statusUpdate.Status.Ready = true
		statusUpdate.Status.Status = StatusReady
		statusUpdate.Status.StatusDetail = "LLM response received, tool calls pending"
		statusUpdate.Status.Error = ""
		r.recorder.Event(taskRun, corev1.EventTypeNormal, "ToolCallsPending", "LLM response received, tool calls pending")

		// Update the parent's status before creating tool call objects.
		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Unable to update TaskRun status")
			return ctrl.Result{}, err
		}

		return r.createToolCalls(ctx, taskRun, statusUpdate, output.ToolCalls)
	}
	return ctrl.Result{}, nil
}

// createToolCalls creates TaskRunToolCall objects for each tool call
func (r *TaskRunReconciler) createToolCalls(ctx context.Context, taskRun *kubechainv1alpha1.TaskRun, statusUpdate *kubechainv1alpha1.TaskRun, toolCalls []kubechainv1alpha1.ToolCall) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// For each tool call, create a new TaskRunToolCall.
	for i, tc := range toolCalls {
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
						Controller: ptr.To(true),
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
		r.recorder.Event(taskRun, corev1.EventTypeNormal, "ToolCallCreated", "Created TaskRunToolCall "+newName)
	}
	return ctrl.Result{RequeueAfter: time.Second * 5}, nil
}

// initializePhaseAndSpan initializes the TaskRun phase and starts tracing
func (r *TaskRunReconciler) initializePhaseAndSpan(ctx context.Context, statusUpdate *kubechainv1alpha1.TaskRun) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Start tracing the TaskRun
	tracer := otel.GetTracerProvider().Tracer("taskrun")
	ctx, span := tracer.Start(ctx, "TaskRun")

	// Store span context in status
	spanCtx := span.SpanContext()
	statusUpdate.Status.SpanContext = &kubechainv1alpha1.SpanContext{
		TraceID: spanCtx.TraceID().String(),
		SpanID:  spanCtx.SpanID().String(),
	}

	statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseInitializing
	statusUpdate.Status.Ready = false
	statusUpdate.Status.Status = "Initializing"
	statusUpdate.Status.StatusDetail = "Initializing"
	if err := r.Status().Update(ctx, statusUpdate); err != nil {
		logger.Error(err, "Failed to update TaskRun status")
		return ctrl.Result{}, err
	}

	// Don't end the span - it will be ended when we reach FinalAnswer
	return ctrl.Result{Requeue: true}, nil
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

	// Initialize phase if not set
	if statusUpdate.Status.Phase == "" {
		return r.initializePhaseAndSpan(ctx, statusUpdate)
	}

	// Step 1: Validate Task and Agent
	task, agent, result, err := r.validateTaskAndAgent(ctx, &taskRun, statusUpdate)
	if err != nil || !result.IsZero() {
		return result, err
	}

	// Step 2: Initialize Phase if necessary
	if result, err := r.prepareForLLM(ctx, &taskRun, statusUpdate, task, agent); err != nil || !result.IsZero() {
		return result, err
	}

	// Step 3: Handle tool calls phase
	if taskRun.Status.Phase == kubechainv1alpha1.TaskRunPhaseToolCallsPending {
		return r.processToolCalls(ctx, &taskRun)
	}

	// Step 4: Check for unexpected phase
	if taskRun.Status.Phase != kubechainv1alpha1.TaskRunPhaseReadyForLLM {
		logger.Info("TaskRun in unknown phase", "phase", taskRun.Status.Phase)
		return ctrl.Result{}, nil
	}

	// Step 5: Get API credentials (LLM is returned but not used)
	_, apiKey, err := r.getLLMAndCredentials(ctx, agent, &taskRun, statusUpdate)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Step 6: Create LLM client
	llmClient, err := r.newLLMClient(apiKey)
	if err != nil {
		logger.Error(err, "Failed to create OpenAI client")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.StatusDetail = "Failed to create OpenAI client: " + err.Error()
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(&taskRun, corev1.EventTypeWarning, "OpenAIClientCreationFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Step 7: Collect tools from all sources
	tools := r.collectTools(ctx, agent)

	r.recorder.Event(&taskRun, corev1.EventTypeNormal, "SendingContextWindowToLLM", "Sending context window to LLM")
	// Step 8: Send the prompt to the LLM
	output, err := llmClient.SendRequest(ctx, taskRun.Status.ContextWindow, tools)
	if err != nil {
		logger.Error(err, "LLM request failed")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.StatusDetail = fmt.Sprintf("LLM request failed: %v", err)
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(&taskRun, corev1.EventTypeWarning, "LLMRequestFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status after LLM error")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Step 9: Process LLM response
	if result, err := r.processLLMResponse(ctx, output, &taskRun, statusUpdate); err != nil || !result.IsZero() {
		return result, err
	}

	// Step 10: Update final status
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

	// Initialize MCPManager if not already set
	if r.MCPManager == nil {
		r.MCPManager = mcpmanager.NewMCPServerManager()
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&kubechainv1alpha1.TaskRun{}).
		Complete(r)
}
