package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/google/uuid"
	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	externalapi "github.com/humanlayer/smallchain/kubechain/internal/externalAPI"
	"github.com/humanlayer/smallchain/kubechain/internal/humanlayer"
)

// TaskRunToolCallReconciler reconciles a TaskRunToolCall object.
type TaskRunToolCallReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
	server   *http.Server
}

func (r *TaskRunToolCallReconciler) webhookHandler(w http.ResponseWriter, req *http.Request) {
	logger := log.FromContext(context.Background())
	var webhook humanlayer.FunctionCall
	if err := json.NewDecoder(req.Body).Decode(&webhook); err != nil {
		logger.Error(err, "Failed to decode webhook payload")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	logger.Info("Received webhook", "webhook", webhook)

	if webhook.Status != nil && webhook.Status.Approved != nil {
		if *webhook.Status.Approved {
			logger.Info("Email approved", "comment", webhook.Status.Comment)
		} else {
			logger.Info("Email request denied")
		}

		// Update TaskRunToolCall status
		if err := r.updateTaskRunToolCall(context.Background(), webhook); err != nil {
			logger.Error(err, "Failed to update TaskRunToolCall status")
			http.Error(w, "Failed to update status", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok"}`))
}

func (r *TaskRunToolCallReconciler) updateTaskRunToolCall(ctx context.Context, webhook humanlayer.FunctionCall) error {
	logger := log.FromContext(ctx)
	var trtc kubechainv1alpha1.TaskRunToolCall

	if err := r.Get(ctx, client.ObjectKey{Namespace: "default", Name: webhook.RunID}, &trtc); err != nil {
		return fmt.Errorf("failed to get TaskRunToolCall: %w", err)
	}

	if webhook.Status != nil && webhook.Status.Approved != nil {
		// Update the TaskRunToolCall status with the webhook data
		if *webhook.Status.Approved {
			trtc.Status.Result = fmt.Sprintf("Approved: %s", *webhook.Status.Comment)
			trtc.Status.Phase = kubechainv1alpha1.TaskRunToolCallPhaseSucceeded
			trtc.Status.Status = "Ready"
			trtc.Status.StatusDetail = "Tool executed successfully"
		} else {
			trtc.Status.Result = "Rejected"
			trtc.Status.Phase = kubechainv1alpha1.TaskRunToolCallPhaseFailed
			trtc.Status.Status = "Error"
			trtc.Status.StatusDetail = "Tool execution rejected"
		}

		// if webhook.Status.RespondedAt != nil {
		// 		trtc.Status.RespondedAt = &metav1.Time{Time: *webhook.Status.RespondedAt}
		// }

		// if webhook.Status.Approved != nil {
		// 		trtc.Status.Approved = webhook.Status.Approved
		// }

		if err := r.Status().Update(ctx, &trtc); err != nil {
			return fmt.Errorf("failed to update TaskRunToolCall status: %w", err)
		}
		logger.Info("TaskRunToolCall status updated", "name", trtc.Name, "phase", trtc.Status.Phase)
	}

	return nil
}

// Helper function to convert various value types to float64
func convertToFloat(val interface{}) (float64, error) {
	switch v := val.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", val)
	}
}

// Reconcile processes TaskRunToolCall objects.
func (r *TaskRunToolCallReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var trtc kubechainv1alpha1.TaskRunToolCall
	if err := r.Get(ctx, req.NamespacedName, &trtc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.Info("Reconciling TaskRunToolCall", "name", trtc.Name)

	// Initialize status if not set.
	if trtc.Status.Phase == "" {
		trtc.Status.Phase = kubechainv1alpha1.TaskRunToolCallPhasePending
		trtc.Status.Status = "Pending"
		trtc.Status.StatusDetail = "Initializing"
		trtc.Status.StartTime = &metav1.Time{Time: time.Now()}
		if err := r.Status().Update(ctx, &trtc); err != nil {
			logger.Error(err, "Failed to update initial status on TaskRunToolCall")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if trtc.Status.Phase == kubechainv1alpha1.TaskRunToolCallPhaseSucceeded || trtc.Status.Phase == kubechainv1alpha1.TaskRunToolCallPhaseFailed {
		logger.Info("TaskRunToolCall already completed, nothing to do", "phase", trtc.Status.Phase)
		return ctrl.Result{}, nil
	}

	// Check if a child TaskRun already exists for this tool call.
	var taskRunList kubechainv1alpha1.TaskRunList
	if err := r.List(ctx, &taskRunList, client.InNamespace(trtc.Namespace), client.MatchingLabels{"kubechain.humanlayer.dev/taskruntoolcall": trtc.Name}); err != nil {
		logger.Error(err, "Failed to list child TaskRuns")
		return ctrl.Result{}, err
	}
	if len(taskRunList.Items) > 0 {
		logger.Info("Child TaskRun already exists", "childTaskRun", taskRunList.Items[0].Name)
		// Optionally, sync status from child to parent.
		return ctrl.Result{}, nil
	}

	// Fetch the referenced Tool.
	var tool kubechainv1alpha1.Tool
	if err := r.Get(ctx, client.ObjectKey{Namespace: trtc.Namespace, Name: trtc.Spec.ToolRef.Name}, &tool); err != nil {
		logger.Error(err, "Failed to get Tool", "tool", trtc.Spec.ToolRef.Name)
		trtc.Status.Status = "Error"
		trtc.Status.StatusDetail = fmt.Sprintf("Failed to get Tool: %v", err)
		trtc.Status.Error = err.Error()
		r.recorder.Event(&trtc, corev1.EventTypeWarning, "ValidationFailed", err.Error())
		if err := r.Status().Update(ctx, &trtc); err != nil {
			logger.Error(err, "Failed to update status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	// Determine tool type from the Tool resource
	var toolType string
	if tool.Spec.Execute.Builtin != nil {
		toolType = "function"
	} else if tool.Spec.AgentRef != nil {
		toolType = "delegateToAgent"
	} else if tool.Spec.Execute.ExternalAPI != nil {
		toolType = "externalAPI"
	} else if tool.Spec.ToolType != "" {
		toolType = tool.Spec.ToolType
	} else {
		err := fmt.Errorf("unknown tool type: tool doesn't have valid execution configuration")
		logger.Error(err, "Invalid tool configuration")
		trtc.Status.Status = "Error"
		trtc.Status.StatusDetail = err.Error()
		trtc.Status.Error = err.Error()
		r.recorder.Event(&trtc, corev1.EventTypeWarning, "ValidationFailed", err.Error())
		if err := r.Status().Update(ctx, &trtc); err != nil {
			logger.Error(err, "Failed to update status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	// Handle different tool types based on determined type
	if toolType == "delegateToAgent" {
		err := fmt.Errorf("delegation is not implemented yet; only direct execution is supported")
		logger.Error(err, "Delegation not implemented")
		trtc.Status.Status = "Error"
		trtc.Status.StatusDetail = err.Error()
		trtc.Status.Error = err.Error()
		r.recorder.Event(&trtc, corev1.EventTypeWarning, "ValidationFailed", err.Error())
		if err := r.Status().Update(ctx, &trtc); err != nil {
			logger.Error(err, "Failed to update status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	} else if toolType == "function" {
		// Parse the arguments string as JSON
		var args map[string]float64
		if err := json.Unmarshal([]byte(trtc.Spec.Arguments), &args); err != nil {
			logger.Error(err, "Failed to parse arguments as numeric values")
			trtc.Status.Status = "Error"
			trtc.Status.StatusDetail = "Invalid arguments: expected numeric values"
			trtc.Status.Error = err.Error()
			r.recorder.Event(&trtc, corev1.EventTypeWarning, "ExecutionFailed", err.Error())
			if err := r.Status().Update(ctx, &trtc); err != nil {
				logger.Error(err, "Failed to update status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}

		logger.Info("Tool call arguments", "toolName", tool.Name, "arguments", args)

		// Try to convert arguments - be more flexible with parameter names and types
		var a, b float64
		var err error

		// Check for different possible parameter names and formats
		if aVal, ok := args["a"]; ok {
			a, err = convertToFloat(aVal)
		} else if aVal, ok := args["first"]; ok {
			a, err = convertToFloat(aVal)
		} else if aVal, ok := args["num1"]; ok {
			a, err = convertToFloat(aVal)
		} else if aVal, ok := args["x"]; ok {
			a, err = convertToFloat(aVal)
		} else {
			err = fmt.Errorf("missing first number parameter")
		}

		if err != nil {
			logger.Error(err, "Failed to parse first argument")
			// Error handling...
			return ctrl.Result{}, err
		}

		if bVal, ok := args["b"]; ok {
			b, err = convertToFloat(bVal)
		} else if bVal, ok := args["second"]; ok {
			b, err = convertToFloat(bVal)
		} else if bVal, ok := args["num2"]; ok {
			b, err = convertToFloat(bVal)
		} else if bVal, ok := args["y"]; ok {
			b, err = convertToFloat(bVal)
		} else {
			err = fmt.Errorf("missing second number parameter")
		}

		if err != nil {
			logger.Error(err, "Failed to parse second argument")
			// Error handling...
			return ctrl.Result{}, err
		}

		// Execute the appropriate function
		functionName := tool.Spec.Execute.Builtin.Name
		var res float64

		switch functionName {
		case "add":
			res = a + b
		case "subtract":
			res = a - b
		case "multiply":
			res = a * b
		case "divide":
			if b == 0 {
				err := fmt.Errorf("division by zero")
				logger.Error(err, "Division by zero")
				trtc.Status.Status = "Error"
				trtc.Status.StatusDetail = "Division by zero"
				trtc.Status.Error = err.Error()
				r.recorder.Event(&trtc, corev1.EventTypeWarning, "ExecutionFailed", err.Error())
				if err := r.Status().Update(ctx, &trtc); err != nil {
					logger.Error(err, "Failed to update status")
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, err
			}
			res = a / b
		default:
			err := fmt.Errorf("unsupported builtin function %q", functionName)
			logger.Error(err, "Unsupported builtin")
			trtc.Status.Status = "Error"
			trtc.Status.StatusDetail = err.Error()
			trtc.Status.Error = err.Error()
			r.recorder.Event(&trtc, corev1.EventTypeWarning, "ExecutionFailed", err.Error())
			if err := r.Status().Update(ctx, &trtc); err != nil {
				logger.Error(err, "Failed to update status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}

		// Update TaskRunToolCall status with the function result
		trtc.Status.Result = fmt.Sprintf("%v", res)
		trtc.Status.Phase = kubechainv1alpha1.TaskRunToolCallPhaseSucceeded
		trtc.Status.Status = "Ready"
		trtc.Status.StatusDetail = "Tool executed successfully"
		if err := r.Status().Update(ctx, &trtc); err != nil {
			logger.Error(err, "Failed to update TaskRunToolCall status after execution")
			return ctrl.Result{}, err
		}
		logger.Info("Direct execution completed", "result", res)
		r.recorder.Event(&trtc, corev1.EventTypeNormal, "ExecutionSucceeded", fmt.Sprintf("Tool %q executed successfully", tool.Name))
		return ctrl.Result{}, nil
	} else if toolType == "externalAPI" {
		if tool.Spec.Execute.ExternalAPI == nil {
			err := fmt.Errorf("externalAPI tool missing execution details")
			logger.Error(err, "Missing execution details")
			trtc.Status.Status = "Error"
			trtc.Status.StatusDetail = err.Error()
			trtc.Status.Error = err.Error()
			r.recorder.Event(&trtc, corev1.EventTypeWarning, "ValidationFailed", err.Error())
			if err := r.Status().Update(ctx, &trtc); err != nil {
				logger.Error(err, "Failed to update status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}

		// Get API key from secret
		var apiKey string
		if tool.Spec.Execute.ExternalAPI.CredentialsFrom != nil {
			var secret corev1.Secret
			err := r.Get(ctx, client.ObjectKey{
				Namespace: trtc.Namespace,
				Name:      tool.Spec.Execute.ExternalAPI.CredentialsFrom.Name,
			}, &secret)
			if err != nil {
				logger.Error(err, "Failed to get API credentials")
				trtc.Status.Status = "Error"
				trtc.Status.StatusDetail = fmt.Sprintf("Failed to get API credentials: %v", err)
				trtc.Status.Error = err.Error()
				r.recorder.Event(&trtc, corev1.EventTypeWarning, "ValidationFailed", err.Error())
				if err := r.Status().Update(ctx, &trtc); err != nil {
					logger.Error(err, "Failed to update status")
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, err
			}

			apiKey = string(secret.Data[tool.Spec.Execute.ExternalAPI.CredentialsFrom.Key])
			logger.Info("Retrieved API key", "key", apiKey)
			if apiKey == "" {
				err := fmt.Errorf("empty API key in secret")
				logger.Error(err, "Empty API key")
				trtc.Status.Status = "Error"
				trtc.Status.StatusDetail = err.Error()
				trtc.Status.Error = err.Error()
				r.recorder.Event(&trtc, corev1.EventTypeWarning, "ValidationFailed", err.Error())
				if err := r.Status().Update(ctx, &trtc); err != nil {
					logger.Error(err, "Failed to update status")
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, err
			}
		}

		var argsMap map[string]interface{}
		if err := json.Unmarshal([]byte(trtc.Spec.Arguments), &argsMap); err != nil {
			logger.Error(err, "Failed to parse arguments")
			trtc.Status.Status = "Error"
			trtc.Status.StatusDetail = "Invalid arguments JSON"
			trtc.Status.Error = err.Error()
			trtc.Status.Phase = kubechainv1alpha1.TaskRunToolCallPhaseFailed
			r.recorder.Event(&trtc, corev1.EventTypeWarning, "ExecutionFailed", err.Error())
			if err := r.Status().Update(ctx, &trtc); err != nil {
				logger.Error(err, "Failed to update status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}
		// And modify it to:
		if len(argsMap) == 0 && tool.Name == "humanlayer-function-call" {
			// RegisterClient adds the HumanLayer client to the external API registry
			humanlayer.RegisterClient()

			// Create kwargs map first to ensure it's properly initialized
			kwargs := map[string]interface{}{
				"tool_name": trtc.Spec.ToolRef.Name,
				"task_run":  trtc.Spec.TaskRunRef.Name,
				"namespace": trtc.Namespace,
			}

			// Default function call for HumanLayer with verified kwargs
			argsMap = map[string]interface{}{
				"fn":     "approve_tool_call",
				"kwargs": kwargs,
			}

			// Log to verify
			logger.Info("Created humanlayer function call args",
				"argsMap", argsMap,
				"kwargs", kwargs)
		}

		// Get the external client
		externalClient, err := externalapi.DefaultRegistry.GetClient(
			tool.Name,
			r.Client,
			trtc.Namespace,
			tool.Spec.Execute.ExternalAPI.CredentialsFrom,
		)
		if err != nil {
			logger.Error(err, "Failed to get external client")
			trtc.Status.Status = "Error"
			trtc.Status.StatusDetail = fmt.Sprintf("Failed to get external client: %v", err)
			trtc.Status.Error = err.Error()
			trtc.Status.Phase = kubechainv1alpha1.TaskRunToolCallPhaseFailed
			r.recorder.Event(&trtc, corev1.EventTypeWarning, "ExecutionFailed", err.Error())
			if err := r.Status().Update(ctx, &trtc); err != nil {
				logger.Error(err, "Failed to update status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}

		var fn string
		var kwargs map[string]interface{}

		// Extract function name
		if fnVal, fnExists := argsMap["fn"]; fnExists && fnVal != nil {
			fn, _ = fnVal.(string)
		}

		// Extract kwargs
		if kwargsVal, kwargsExists := argsMap["kwargs"]; kwargsExists && kwargsVal != nil {
			kwargs, _ = kwargsVal.(map[string]interface{})
		}

		// Generate call ID
		callID := "call-" + uuid.New().String()

		// Prepare function call spec
		functionSpec := map[string]interface{}{
			"fn":     fn,
			"kwargs": kwargs,
		}

		// Make the API call
		result, err := externalClient.Call(ctx, trtc.Name, callID, functionSpec)
		if err != nil {
			logger.Error(err, "External API call failed")
			// Error handling...
			return ctrl.Result{}, err
		}

		// Update TaskRunToolCall with the result
		trtc.Status.Result = string(result)
		trtc.Status.Phase = kubechainv1alpha1.TaskRunToolCallPhaseSucceeded
		trtc.Status.Status = "Ready"
		trtc.Status.StatusDetail = "Tool executed successfully"
		if err := r.Status().Update(ctx, &trtc); err != nil {
			logger.Error(err, "Failed to update TaskRunToolCall status")
			return ctrl.Result{}, err
		}
		logger.Info("TaskRunToolCall completed", "phase", trtc.Status.Phase)
		return ctrl.Result{}, nil
	}

	// Fallback: if tool type is not recognized.
	err := fmt.Errorf("unsupported tool configuration")
	logger.Error(err, "Unsupported tool configuration")
	trtc.Status.Status = "Error"
	trtc.Status.StatusDetail = err.Error()
	trtc.Status.Error = err.Error()
	r.recorder.Event(&trtc, corev1.EventTypeWarning, "ExecutionFailed", err.Error())
	if err := r.Status().Update(ctx, &trtc); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, err
}

func (r *TaskRunToolCallReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("taskruntoolcall-controller")
	r.server = &http.Server{Addr: ":8080"} // Choose a port
	http.HandleFunc("/webhook/inbound", r.webhookHandler)

	go func() {
		if err := r.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Log.Error(err, "Failed to start HTTP server")
		}
	}()

	return ctrl.NewControllerManagedBy(mgr).
		For(&kubechainv1alpha1.TaskRunToolCall{}).
		Complete(r)
}

func (r *TaskRunToolCallReconciler) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := r.server.Shutdown(ctx); err != nil {
		log.Log.Error(err, "Failed to shut down HTTP server")
	}
}
