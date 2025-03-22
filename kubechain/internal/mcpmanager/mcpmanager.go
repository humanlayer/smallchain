package mcpmanager

import (
	"context"
	"fmt"
	"os/exec"
	"sync"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

// MCPServerManager manages MCP server connections and tools
type MCPServerManager struct {
	connections map[string]*MCPConnection
	mu          sync.RWMutex
}

// MCPConnection represents a connection to an MCP server
type MCPConnection struct {
	// ServerName is the name of the MCPServer resource
	ServerName string
	// ServerType is "stdio" or "http"
	ServerType string
	// Command is the stdio process (if ServerType is "stdio")
	Command *exec.Cmd
	// Client is the MCP client
	Client mcp.MCPClient
	// Tools is the list of tools provided by this server
	Tools []kubechainv1alpha1.MCPTool
}

// NewMCPServerManager creates a new MCPServerManager
func NewMCPServerManager() *MCPServerManager {
	return &MCPServerManager{
		connections: make(map[string]*MCPConnection),
		mu:          sync.RWMutex{},
	}
}

// GetConnection returns the MCPConnection for the given server name
func (m *MCPServerManager) GetConnection(serverName string) (*MCPConnection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, exists := m.connections[serverName]
	return conn, exists
}

// convertEnvVars converts kubechain EnvVar to string slice of env vars
func convertEnvVars(envVars []kubechainv1alpha1.EnvVar) []string {
	env := make([]string, 0, len(envVars))
	for _, e := range envVars {
		env = append(env, fmt.Sprintf("%s=%s", e.Name, e.Value))
	}
	return env
}

// ConnectServer establishes a connection to an MCP server
func (m *MCPServerManager) ConnectServer(ctx context.Context, mcpServer *kubechainv1alpha1.MCPServer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we already have a connection for this server
	if conn, exists := m.connections[mcpServer.Name]; exists {
		// If the server exists and the specs are the same, reuse the connection
		// TODO: Add logic to detect if specs changed and reconnect if needed
		if conn.ServerType == mcpServer.Spec.Type {
			return nil
		}

		// Clean up existing connection
		m.disconnectServerLocked(mcpServer.Name)
	}

	var mcpClient mcp.MCPClient
	var err error

	if mcpServer.Spec.Type == "stdio" {
		// Create a stdio-based MCP client
		mcpClient, err = client.NewStdioMCPClient(mcpServer.Spec.Command, convertEnvVars(mcpServer.Spec.Env), mcpServer.Spec.Args...)
		if err != nil {
			return fmt.Errorf("failed to create stdio MCP client: %w", err)
		}
	} else if mcpServer.Spec.Type == "http" {
		// Create an HTTP-based MCP client
		mcpClient, err = client.NewHTTPMCPClient(mcpServer.Spec.URL)
		if err != nil {
			return fmt.Errorf("failed to create HTTP MCP client: %w", err)
		}
	} else {
		return fmt.Errorf("unsupported MCP server type: %s", mcpServer.Spec.Type)
	}

	// Initialize the client
	_, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{})
	if err != nil {
		mcpClient.Close() // Clean up on error
		return fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	// Get the list of tools
	toolsResp, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		mcpClient.Close() // Clean up on error
		return fmt.Errorf("failed to list tools: %w", err)
	}

	// Convert tools to kubechain format
	tools := make([]kubechainv1alpha1.MCPTool, 0, len(toolsResp.Tools))
	for _, tool := range toolsResp.Tools {
		tools = append(tools, kubechainv1alpha1.MCPTool{
			Name:        tool.Name,
			Description: tool.Description,
			// We'll need to handle the InputSchema conversion here
			// This is a simplified version
			// TODO: Properly convert InputSchema
		})
	}

	// Store the connection
	m.connections[mcpServer.Name] = &MCPConnection{
		ServerName: mcpServer.Name,
		ServerType: mcpServer.Spec.Type,
		Client:     mcpClient,
		Tools:      tools,
	}

	return nil
}

// DisconnectServer closes the connection to an MCP server
func (m *MCPServerManager) DisconnectServer(serverName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disconnectServerLocked(serverName)
}

// disconnectServerLocked is the internal implementation of DisconnectServer
// that assumes the lock is already held
func (m *MCPServerManager) disconnectServerLocked(serverName string) {
	conn, exists := m.connections[serverName]
	if !exists {
		return
	}

	// Close the connection
	if conn.Client != nil {
		conn.Client.Close()
	}

	// Remove the connection from the map
	delete(m.connections, serverName)
}

// GetTools returns the tools for the given server
func (m *MCPServerManager) GetTools(serverName string) ([]kubechainv1alpha1.MCPTool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, exists := m.connections[serverName]
	if !exists {
		return nil, false
	}
	return conn.Tools, true
}

// GetToolsForAgent returns all tools from the MCP servers referenced by the agent
func (m *MCPServerManager) GetToolsForAgent(agent *kubechainv1alpha1.Agent) []kubechainv1alpha1.MCPTool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var allTools []kubechainv1alpha1.MCPTool
	for _, serverRef := range agent.Spec.MCPServers {
		conn, exists := m.connections[serverRef.Name]
		if !exists {
			continue
		}
		allTools = append(allTools, conn.Tools...)
	}
	return allTools
}

// CallTool calls a tool on an MCP server
func (m *MCPServerManager) CallTool(ctx context.Context, serverName, toolName string, arguments map[string]interface{}) (string, error) {
	m.mu.RLock()
	conn, exists := m.connections[serverName]
	m.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("MCP server not found: %s", serverName)
	}

	result, err := conn.Client.CallTool(ctx, mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name:      toolName,
			Arguments: arguments,
		},
	})

	if err != nil {
		return "", fmt.Errorf("error calling tool %s on server %s: %w", toolName, serverName, err)
	}

	// Process the result
	var output string
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			output += textContent.Text
		} else {
			// Handle other content types as needed
			output += "[Non-text content]"
		}
	}

	if result.IsError {
		return output, fmt.Errorf("tool execution error: %s", output)
	}

	return output, nil
}

// FindServerForTool finds which MCP server provides a given tool
// Format of the tool name is expected to be "serverName__toolName"
func (m *MCPServerManager) FindServerForTool(fullToolName string) (serverName string, toolName string, found bool) {
	// In our implementation, we'll use serverName__toolName as the format
	// This is consistent with the MCP integration example
	parts := client.ParseMCPToolName(fullToolName)
	if len(parts) != 2 {
		return "", "", false
	}

	serverName = parts[0]
	toolName = parts[1]

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if the server exists
	conn, exists := m.connections[serverName]
	if !exists {
		return "", "", false
	}

	// Check if the tool exists on this server
	for _, tool := range conn.Tools {
		if tool.Name == toolName {
			return serverName, toolName, true
		}
	}

	return "", "", false
}

// Close closes all connections
func (m *MCPServerManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for serverName := range m.connections {
		m.disconnectServerLocked(serverName)
	}
}
