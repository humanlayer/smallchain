package taskrun

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	// These constants are kept for backward compatibility during refactoring
	StatusReady   = kubechainv1alpha1.TaskRunStatusStatusReady
	StatusError   = kubechainv1alpha1.TaskRunStatusStatusError
	StatusPending = kubechainv1alpha1.TaskRunStatusStatusPending
)

// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=taskruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=taskruns/status,verbs=get;update;patch
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
	Tracer       trace.Tracer
}

// initializePhaseAndSpan initializes the phase and span context for a new TaskRun
func (r *TaskRunReconciler) initializePhaseAndSpan(ctx context.Context, taskRun *kubechainv1alpha1.TaskRun) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Create a new span for the TaskRun
	spanCtx, span := r.Tracer.Start(ctx, "TaskRun")
	defer span.End()

	// Set initial phase
	taskRun.Status.Phase = kubechainv1alpha1.TaskRunPhaseInitializing
	taskRun.Status.Status = StatusPending
	taskRun.Status.StatusDetail = "Initializing TaskRun"

	// Save span context for future use
	taskRun.Status.SpanContext = &kubechainv1alpha1.SpanContext{
		TraceID: span.SpanContext().TraceID().String(),
		SpanID:  span.SpanContext().SpanID().String(),
	}

	if err := r.Status().Update(spanCtx, taskRun); err != nil {
		logger.Error(err, "Failed to update TaskRun status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{Requeue: true}, nil
}

// validateTaskAndAgent checks if the agent exists and is ready
func (r *TaskRunReconciler) validateTaskAndAgent(ctx context.Context, taskRun *kubechainv1alpha1.TaskRun, statusUpdate *kubechainv1alpha1.TaskRun) (*kubechainv1alpha1.Agent, ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var agent kubechainv1alpha1.Agent
	if err := r.Get(ctx, client.ObjectKey{Namespace: taskRun.Namespace, Name: taskRun.Spec.AgentRef.Name}, &agent); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Agent not found, waiting for it to exist")
			statusUpdate.Status.Ready = false
			statusUpdate.Status.Status = StatusPending
			statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhasePending
			statusUpdate.Status.StatusDetail = "Waiting for Agent to exist"
			statusUpdate.Status.Error = "" // Clear previous error
			r.recorder.Event(taskRun, corev1.EventTypeNormal, "Waiting", "Waiting for Agent to exist")
		} else {
			logger.Error(err, "Failed to get Agent")
			statusUpdate.Status.Ready = false
			statusUpdate.Status.Status = StatusError
			statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFailed
			statusUpdate.Status.Error = err.Error()
			r.recorder.Event(taskRun, corev1.EventTypeWarning, "AgentFetchFailed", err.Error())
		}
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return nil, ctrl.Result{}, updateErr
		}
		return nil, ctrl.Result{RequeueAfter: time.Second * 5}, nil
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
			return nil, ctrl.Result{}, err
		}
		return nil, ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	return &agent, ctrl.Result{}, nil
}

// prepareForLLM sets up the initial state of a TaskRun with the correct context and phase
func (r *TaskRunReconciler) prepareForLLM(ctx context.Context, taskRun *kubechainv1alpha1.TaskRun, statusUpdate *kubechainv1alpha1.TaskRun, agent *kubechainv1alpha1.Agent) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// If we're in Initializing or Pending phase, transition to ReadyForLLM
	if statusUpdate.Status.Phase == kubechainv1alpha1.TaskRunPhaseInitializing || statusUpdate.Status.Phase == kubechainv1alpha1.TaskRunPhasePending {
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseReadyForLLM
		statusUpdate.Status.Ready = true

		if taskRun.Spec.UserMessage == "" {
			err := fmt.Errorf("userMessage is required")
			logger.Error(err, "Missing message")
			statusUpdate.Status.Ready = false
			statusUpdate.Status.Status = StatusError
			statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFailed
			statusUpdate.Status.StatusDetail = err.Error()
			statusUpdate.Status.Error = err.Error()
			r.recorder.Event(taskRun, corev1.EventTypeWarning, "ValidationFailed", err.Error())
			if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
				logger.Error(updateErr, "Failed to update TaskRun status")
				return ctrl.Result{}, updateErr
			}
			return ctrl.Result{}, err
		}

		// Set up the context window
		statusUpdate.Status.ContextWindow = []kubechainv1alpha1.Message{
			{
				Role:    "system",
				Content: agent.Spec.System,
			},
			{
				Role:    "user",
				Content: taskRun.Spec.UserMessage,
			},
		}
		statusUpdate.Status.Status = StatusReady
		statusUpdate.Status.StatusDetail = "Ready to send to LLM"
		statusUpdate.Status.Error = "" // Clear previous error
		r.recorder.Event(taskRun, corev1.EventTypeNormal, "ValidationSucceeded", "TaskRun validation succeeded")

		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Failed to update TaskRun status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// processToolCalls handles the tool calls phase
func (r *TaskRunReconciler) processToolCalls(ctx context.Context, taskRun *kubechainv1alpha1.TaskRun) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// List all tool calls for this TaskRun
	toolCalls := &kubechainv1alpha1.TaskRunToolCallList{}
	if err := r.List(ctx, toolCalls, client.InNamespace(taskRun.Namespace), client.MatchingLabels{
		"kubechain.humanlayer.dev/toolcallrequest": taskRun.Status.ToolCallRequestID,
	}); err != nil {
		logger.Error(err, "Failed to list tool calls")
		return ctrl.Result{}, err
	}

	// Check if all tool calls are completed
	allCompleted := true
	for _, tc := range toolCalls.Items {
		if tc.Status.Phase != kubechainv1alpha1.TaskRunToolCallPhaseSucceeded {
			allCompleted = false
			break
		}
	}

	if !allCompleted {
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	// All tool calls are completed, append results to context window
	for _, tc := range toolCalls.Items {
		taskRun.Status.ContextWindow = append(taskRun.Status.ContextWindow, kubechainv1alpha1.Message{
			Role:    "tool",
			Content: tc.Status.Result,
		})
	}

	// Update status
	taskRun.Status.Phase = kubechainv1alpha1.TaskRunPhaseReadyForLLM
	taskRun.Status.Status = StatusReady
	taskRun.Status.StatusDetail = "All tool calls completed, ready to send tool results to LLM"
	taskRun.Status.Error = "" // Clear previous error
	r.recorder.Event(taskRun, corev1.EventTypeNormal, "AllToolCallsCompleted", "All tool calls completed")

	if err := r.Status().Update(ctx, taskRun); err != nil {
		logger.Error(err, "Failed to update TaskRun status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{Requeue: true}, nil
}

// getLLMAndCredentials gets the LLM and API key for the agent
func (r *TaskRunReconciler) getLLMAndCredentials(ctx context.Context, agent *kubechainv1alpha1.Agent, taskRun *kubechainv1alpha1.TaskRun, statusUpdate *kubechainv1alpha1.TaskRun) (*kubechainv1alpha1.LLM, string, error) {
	logger := log.FromContext(ctx)

	// Get the LLM
	var llm kubechainv1alpha1.LLM
	if err := r.Get(ctx, client.ObjectKey{Namespace: taskRun.Namespace, Name: agent.Spec.LLMRef.Name}, &llm); err != nil {
		logger.Error(err, "Failed to get LLM")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFailed
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Failed to get LLM: %v", err)
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(taskRun, corev1.EventTypeWarning, "LLMFetchFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return nil, "", updateErr
		}
		return nil, "", err
	}

	// Get the API key from the secret
	var secret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: taskRun.Namespace,
		Name:      llm.Spec.APIKeyFrom.SecretKeyRef.Name,
	}, &secret); err != nil {
		logger.Error(err, "Failed to get API key secret")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFailed
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Failed to get API key secret: %v", err)
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(taskRun, corev1.EventTypeWarning, "APIKeySecretFetchFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return nil, "", updateErr
		}
		return nil, "", err
	}

	apiKey := string(secret.Data[llm.Spec.APIKeyFrom.SecretKeyRef.Key])
	if apiKey == "" {
		err := fmt.Errorf("API key is empty")
		logger.Error(err, "Empty API key")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFailed
		statusUpdate.Status.StatusDetail = "API key is empty"
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(taskRun, corev1.EventTypeWarning, "EmptyAPIKey", "API key is empty")
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return nil, "", updateErr
		}
		return nil, "", err
	}

	return &llm, apiKey, nil
}

// endTaskRunSpan ends the TaskRun span with the given status
func (r *TaskRunReconciler) endTaskRunSpan(ctx context.Context, taskRun *kubechainv1alpha1.TaskRun, code codes.Code, message string) {
	if taskRun.Status.SpanContext == nil {
		return
	}

	traceID, err := trace.TraceIDFromHex(taskRun.Status.SpanContext.TraceID)
	if err != nil {
		return
	}
	spanID, err := trace.SpanIDFromHex(taskRun.Status.SpanContext.SpanID)
	if err != nil {
		return
	}

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	})

	ctx = trace.ContextWithSpanContext(ctx, spanCtx)
	_, span := r.Tracer.Start(ctx, "TaskRun")
	defer span.End()

	span.SetStatus(code, message)
}

// collectTools collects all tools from the agent's MCP servers
func (r *TaskRunReconciler) collectTools(ctx context.Context, agent *kubechainv1alpha1.Agent) []llmclient.Tool {
	var tools []llmclient.Tool

	// Get tools from MCP manager
	mcpTools := r.MCPManager.GetToolsForAgent(agent)

	// Convert MCP tools to LLM tools
	for _, mcpTool := range mcpTools {
		tools = append(tools, adapters.ConvertMCPToolsToLLMClientTools([]kubechainv1alpha1.MCPTool{mcpTool}, mcpTool.Name)...)
	}

	return tools
}

// createLLMRequestSpan creates a child span for the LLM request
func (r *TaskRunReconciler) createLLMRequestSpan(ctx context.Context, taskRun *kubechainv1alpha1.TaskRun, numMessages int, numTools int) (context.Context, trace.Span) {
	if taskRun.Status.SpanContext == nil {
		return ctx, nil
	}

	traceID, err := trace.TraceIDFromHex(taskRun.Status.SpanContext.TraceID)
	if err != nil {
		return ctx, nil
	}
	spanID, err := trace.SpanIDFromHex(taskRun.Status.SpanContext.SpanID)
	if err != nil {
		return ctx, nil
	}

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	})

	ctx = trace.ContextWithSpanContext(ctx, spanCtx)
	childCtx, span := r.Tracer.Start(ctx, "LLMRequest")

	span.SetAttributes(
		attribute.Int("messages", numMessages),
		attribute.Int("tools", numTools),
	)

	return childCtx, span
}

// processLLMResponse processes the LLM response and updates the TaskRun status
func (r *TaskRunReconciler) processLLMResponse(ctx context.Context, output *kubechainv1alpha1.Message, taskRun *kubechainv1alpha1.TaskRun, statusUpdate *kubechainv1alpha1.TaskRun) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Append the LLM response to the context window
	statusUpdate.Status.ContextWindow = append(statusUpdate.Status.ContextWindow, *output)

	// Check if the response contains tool calls
	if len(output.ToolCalls) > 0 {
		// Generate a unique ID for this set of tool calls
		toolCallRequestID := uuid.New().String()
		statusUpdate.Status.ToolCallRequestID = toolCallRequestID

		// Create TaskRunToolCall objects for each tool call
		for _, tc := range output.ToolCalls {
			toolCall := &kubechainv1alpha1.TaskRunToolCall{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%s", taskRun.Name, tc.ID),
					Namespace: taskRun.Namespace,
					Labels: map[string]string{
						"kubechain.humanlayer.dev/taskruntoolcall": taskRun.Name,
						"kubechain.humanlayer.dev/toolcallrequest": toolCallRequestID,
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: taskRun.APIVersion,
							Kind:       taskRun.Kind,
							Name:       taskRun.Name,
							UID:        taskRun.UID,
							Controller: ptr.To(true),
						},
					},
				},
				Spec: kubechainv1alpha1.TaskRunToolCallSpec{
					TaskRunRef: kubechainv1alpha1.LocalObjectReference{
						Name: taskRun.Name,
					},
					ToolRef: kubechainv1alpha1.LocalObjectReference{
						Name: tc.Function.Name,
					},
					Arguments: tc.Function.Arguments,
				},
			}

			if err := r.Create(ctx, toolCall); err != nil {
				logger.Error(err, "Failed to create TaskRunToolCall")
				return ctrl.Result{}, err
			}
		}

		// Update status
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseToolCallsPending
		statusUpdate.Status.Status = StatusPending
		statusUpdate.Status.StatusDetail = "LLM response received, tool calls pending"
		statusUpdate.Status.Error = "" // Clear previous error
		r.recorder.Event(taskRun, corev1.EventTypeNormal, "ToolCallsPending", "Tool calls created and pending execution")

		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			logger.Error(err, "Failed to update TaskRun status")
			return ctrl.Result{}, err
		}

		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	// No tool calls, this is a final answer
	statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFinalAnswer
	statusUpdate.Status.Status = StatusReady
	statusUpdate.Status.StatusDetail = "LLM final response received"
	statusUpdate.Status.Output = output.Content
	statusUpdate.Status.Error = "" // Clear previous error
	r.recorder.Event(taskRun, corev1.EventTypeNormal, "LLMFinalAnswer", "LLM final response received")

	if err := r.Status().Update(ctx, statusUpdate); err != nil {
		logger.Error(err, "Failed to update TaskRun status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// Reconcile validates the taskrun's agent reference and sends the prompt to the LLM.
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

	// Skip reconciliation for terminal states
	if statusUpdate.Status.Phase == kubechainv1alpha1.TaskRunPhaseFinalAnswer || statusUpdate.Status.Phase == kubechainv1alpha1.TaskRunPhaseFailed {
		logger.V(1).Info("TaskRun in terminal state, skipping reconciliation", "phase", statusUpdate.Status.Phase)
		return ctrl.Result{}, nil
	}

	// Step 1: Validate Agent
	logger.V(3).Info("Validating Agent")
	agent, result, err := r.validateTaskAndAgent(ctx, &taskRun, statusUpdate)
	if err != nil || !result.IsZero() {
		return result, err
	}

	// Step 2: Initialize Phase if necessary
	logger.V(3).Info("Preparing for LLM")
	if result, err := r.prepareForLLM(ctx, &taskRun, statusUpdate, agent); err != nil || !result.IsZero() {
		return result, err
	}

	// Step 3: Handle tool calls phase
	logger.V(3).Info("Handling tool calls phase")
	if taskRun.Status.Phase == kubechainv1alpha1.TaskRunPhaseToolCallsPending {
		return r.processToolCalls(ctx, &taskRun)
	}

	// Step 4: Check for unexpected phase
	if taskRun.Status.Phase != kubechainv1alpha1.TaskRunPhaseReadyForLLM {
		logger.Info("TaskRun in unknown phase", "phase", taskRun.Status.Phase)
		return ctrl.Result{}, nil
	}

	// Step 5: Get API credentials (LLM is returned but not used)
	logger.V(3).Info("Getting API credentials")
	_, apiKey, err := r.getLLMAndCredentials(ctx, agent, &taskRun, statusUpdate)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Step 6: Create LLM client
	logger.V(3).Info("Creating LLM client")
	llmClient, err := r.newLLMClient(apiKey)
	if err != nil {
		logger.Error(err, "Failed to create OpenAI client")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFailed
		statusUpdate.Status.StatusDetail = "Failed to create OpenAI client: " + err.Error()
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(&taskRun, corev1.EventTypeWarning, "OpenAIClientCreationFailed", err.Error())

		// End span since we've failed with a terminal error
		r.endTaskRunSpan(ctx, &taskRun, codes.Error, "Failed to create OpenAI client: "+err.Error())

		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Step 7: Collect tools from all sources
	tools := r.collectTools(ctx, agent)

	r.recorder.Event(&taskRun, corev1.EventTypeNormal, "SendingContextWindowToLLM", "Sending context window to LLM")

	// Create child span for LLM call
	childCtx, childSpan := r.createLLMRequestSpan(ctx, &taskRun, len(taskRun.Status.ContextWindow), len(tools))
	if childSpan != nil {
		defer childSpan.End()
	}

	logger.V(3).Info("Sending LLM request")
	// Step 8: Send the prompt to the LLM
	output, err := llmClient.SendRequest(childCtx, taskRun.Status.ContextWindow, tools)
	if err != nil {
		logger.Error(err, "LLM request failed")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.StatusDetail = fmt.Sprintf("LLM request failed: %v", err)
		statusUpdate.Status.Error = err.Error()

		// Check for LLMRequestError with 4xx status code
		var llmErr *llmclient.LLMRequestError
		if errors.As(err, &llmErr) && llmErr.StatusCode >= 400 && llmErr.StatusCode < 500 {
			logger.Info("LLM request failed with 4xx status code, marking as failed",
				"statusCode", llmErr.StatusCode,
				"message", llmErr.Message)
			statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFailed
			r.recorder.Event(&taskRun, corev1.EventTypeWarning, "LLMRequestFailed4xx",
				fmt.Sprintf("LLM request failed with status %d: %s", llmErr.StatusCode, llmErr.Message))
		} else {
			r.recorder.Event(&taskRun, corev1.EventTypeWarning, "LLMRequestFailed", err.Error())
		}

		// Record error in span
		if childSpan != nil {
			childSpan.RecordError(err)
			childSpan.SetStatus(codes.Error, err.Error())
		}

		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status after LLM error")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Mark span as successful if we reach here
	if childSpan != nil {
		childSpan.SetStatus(codes.Ok, "LLM request succeeded")
	}

	logger.V(3).Info("Processing LLM response")
	// Step 9: Process LLM response
	var llmResult ctrl.Result
	llmResult, err = r.processLLMResponse(ctx, output, &taskRun, statusUpdate)
	if err != nil {
		logger.Error(err, "Failed to process LLM response")
		statusUpdate.Status.Status = StatusError
		statusUpdate.Status.Phase = kubechainv1alpha1.TaskRunPhaseFailed
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Failed to process LLM response: %v", err)
		statusUpdate.Status.Error = err.Error()
		r.recorder.Event(&taskRun, corev1.EventTypeWarning, "LLMResponseProcessingFailed", err.Error())

		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update TaskRun status after LLM response processing error")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, nil // Don't return the error to avoid requeuing
	}

	if !llmResult.IsZero() {
		return llmResult, nil
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
