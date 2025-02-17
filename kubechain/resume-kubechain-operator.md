# Current State

## Completed
- Initialized Kubebuilder project with correct domain (humanlayer.dev)
- Implemented LLM CRD with:
  - Proper group name (kubechain.humanlayer.dev)
  - Kubernetes-native secret reference (apiKeyFrom.secretKeyRef)
  - Validation for required fields and value ranges
  - Status subresource
  - Custom columns (Provider, Ready)
  - Working controller with status updates
  - Proper logging

## Next Steps
1. Implement Tool CRD
   - Define schema for different tool types (builtin, container, delegate)
   - Add validation for tool arguments using JSON schema
   - Implement controller with proper status handling

2. Implement ToolSet CRD
   - Define schema for tool collections
   - Add validation for tool references
   - Implement controller for managing tool sets

3. Implement Agent CRD
   - Define schema with LLM and ToolSet references
   - Add system prompt configuration
   - Implement controller with dependency validation

4. Implement Task and TaskRun CRDs
   - Define schema for task definitions and runs
   - Add support for context window management
   - Implement async operation handling
   - Add support for tool call tracking

5. Add OpenTelemetry Integration
   - Add tracing to controllers
   - Implement metrics collection
   - Set up proper span relationships

6. Implement UI Components
   - Create task viewer
   - Add trace visualization
   - Implement task management interface
