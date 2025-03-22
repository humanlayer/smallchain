package mcpserver

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	"github.com/humanlayer/smallchain/kubechain/internal/mcpmanager"
)

// MCPServerReconciler reconciles a MCPServer object
type MCPServerReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	recorder   record.EventRecorder
	MCPManager *mcpmanager.MCPServerManager
}

// Reconcile processes the MCPServer resource and establishes a connection to the MCP server
func (r *MCPServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the MCPServer instance
	var mcpServer kubechainv1alpha1.MCPServer
	if err := r.Get(ctx, req.NamespacedName, &mcpServer); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Starting reconciliation", "name", mcpServer.Name)

	// Create a status update copy
	statusUpdate := mcpServer.DeepCopy()

	// Basic validation
	if err := r.validateMCPServer(&mcpServer); err != nil {
		statusUpdate.Status.Connected = false
		statusUpdate.Status.Status = "Error"
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Validation failed: %v", err)
		r.recorder.Event(&mcpServer, corev1.EventTypeWarning, "ValidationFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update MCPServer status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Try to connect to the MCP server
	err := r.MCPManager.ConnectServer(ctx, &mcpServer)
	if err != nil {
		statusUpdate.Status.Connected = false
		statusUpdate.Status.Status = "Error"
		statusUpdate.Status.StatusDetail = fmt.Sprintf("Connection failed: %v", err)
		r.recorder.Event(&mcpServer, corev1.EventTypeWarning, "ConnectionFailed", err.Error())
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update MCPServer status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil // Retry after 30 seconds
	}

	// Get tools from the manager
	tools, exists := r.MCPManager.GetTools(mcpServer.Name)
	if !exists {
		statusUpdate.Status.Connected = false
		statusUpdate.Status.Status = "Error"
		statusUpdate.Status.StatusDetail = "Failed to get tools from manager"
		r.recorder.Event(&mcpServer, corev1.EventTypeWarning, "GetToolsFailed", "Failed to get tools from manager")
		if updateErr := r.Status().Update(ctx, statusUpdate); updateErr != nil {
			logger.Error(updateErr, "Failed to update MCPServer status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil // Retry after 30 seconds
	}

	// Update status with tools
	statusUpdate.Status.Connected = true
	statusUpdate.Status.Status = "Ready"
	statusUpdate.Status.StatusDetail = fmt.Sprintf("Connected successfully with %d tools", len(tools))
	statusUpdate.Status.Tools = tools
	r.recorder.Event(&mcpServer, corev1.EventTypeNormal, "Connected", "MCP server connected successfully")

	// Update status
	if err := r.Status().Update(ctx, statusUpdate); err != nil {
		logger.Error(err, "Failed to update MCPServer status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled MCPServer",
		"name", mcpServer.Name,
		"connected", statusUpdate.Status.Connected,
		"toolCount", len(statusUpdate.Status.Tools))

	// Schedule periodic reconciliation to refresh tool list
	return ctrl.Result{RequeueAfter: time.Minute * 10}, nil
}

// validateMCPServer performs basic validation on the MCPServer spec
func (r *MCPServerReconciler) validateMCPServer(mcpServer *kubechainv1alpha1.MCPServer) error {
	// Check server type
	if mcpServer.Spec.Type != "stdio" && mcpServer.Spec.Type != "http" {
		return fmt.Errorf("invalid server type: %s", mcpServer.Spec.Type)
	}

	// Validate stdio type
	if mcpServer.Spec.Type == "stdio" {
		if mcpServer.Spec.Command == "" {
			return fmt.Errorf("command is required for stdio servers")
		}
		// Other validations as needed
	}

	// Validate http type
	if mcpServer.Spec.Type == "http" {
		if mcpServer.Spec.URL == "" {
			return fmt.Errorf("url is required for http servers")
		}
		// Other validations as needed
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MCPServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("mcpserver-controller")

	// Initialize the MCP manager if not already set
	if r.MCPManager == nil {
		r.MCPManager = mcpmanager.NewMCPServerManager()
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&kubechainv1alpha1.MCPServer{}).
		Complete(r)
}
