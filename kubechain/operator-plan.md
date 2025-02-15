# Kubernetes Operator Implementation Plan

This document details the concrete implementation steps for developing the operator components outlined in kubechain-plan.md. It covers the concrete implementation of CRDs, controllers with reconciliation logic, asynchronous operations, status updates, observability, testing, and deployment.

---

## 1. CRD Components Development

### LLM CRD
- **Schema:** Fields for `provider`, `apiKeySecretRef`, and additional config (e.g., temperature, maxTokens).
- **Controller Responsibilities:**
  - Fetch the LLM instance.
  - Validate secret references (e.g., check API key existence/validity).
  - Update `LLM.Status.Ready` and log events.
- **Pseudocode Example:**
  ```
  func Reconcile(ctx, req) {
      llm := fetchLLM(req)
      if !validateSecret(llm.Spec.apiKeySecretRef) {
          llm.Status.Ready = false
          recordEvent(llm, "Invalid or missing API key")
      } else {
          llm.Status.Ready = true
      }
      updateStatus(llm)
  }
  ```

### Tool & ToolSet CRDs
- **Schema:** 
  - For Tool: Define `name`, `description`, `arguments` (JSON schema), and execution details (builtin, container, delegate).
  - For ToolSet: Array of tool definitions.
- **Controller Responsibilities:**
  - Validate each tool’s configuration.
  - If a container is specified, trigger pod creation.
  - Set status (e.g., `Toolset.Status.Ready`) once all tools are validated.
- **Pseudocode Example:**
  ```
  func ReconcileToolset(ctx, req) {
      toolset := fetchToolset(req)
      for _, tool := range toolset.Spec.Tools {
          if !validateTool(tool) {
              recordEvent(toolset, "Tool validation failed for " + tool.name)
          }
      }
      toolset.Status.Ready = allToolsValid(toolset)
      updateStatus(toolset)
  }
  ```

### Agent CRD
- **Schema:** Contains `llmRef`, `systemPrompt`, and `toolsetRefs`.
- **Controller Responsibilities:**
  - Ensure referenced LLM and ToolSets exist and are in a ready state.
  - Update `Agent.Status.Ready` only if all dependencies are ready.
- **Pseudocode Example:**
  ```
  func ReconcileAgent(ctx, req) {
      agent := fetchAgent(req)
      if !isLLMReady(agent.Spec.LLMRef) || !areToolsetsReady(agent.Spec.ToolsetRefs) {
         agent.Status.Ready = false
         recordEvent(agent, "One or more dependencies not ready")
      } else {
         agent.Status.Ready = true
      }
      updateStatus(agent)
  }
  ```

### Task CRD
- **Schema:** Contains `agentRef`, `userInput`, and optionally a reference for external context storage.
- **Controller Responsibilities:**
  - On task creation, check for fields like `launchImmediately`.
  - Spawn an associated TaskRun resource with appropriate owner references.
- **Pseudocode:**
  ```
  if task.Spec.launchImmediately {
      taskRun := newTaskRun(task)
      create(taskRun)
      recordEvent(task, "TaskRun created")
  }
  ```

### TaskRun CRD
- **Schema:** Includes `taskRef`, inline or external `contextWindow`, `toolCallHistory`, and a `Phase` field.
- **Phases:** Recommended phases include `Pending`, `Running`, `PausedForUserInput`, and `Completed`.
- **Controller Responsibilities:**
  - On `Pending`: Initialize the conversation (e.g., start context with Agent.systemPrompt and Task.userInput).
  - On `Running`:  
    - Determine if an LLM call or tool call is needed.
    - Update the context window and record any tool call requests.
    - Transition to `Completed` if done or to `PausedForUserInput` if awaiting human feedback.
- **Pseudocode Example:**
  ```
  func ReconcileTaskRun(ctx, req) {
      taskRun := fetchTaskRun(req)
      switch taskRun.Status.Phase {
          case Pending:
              taskRun.Status.ContextWindow = [
                  "system: " + agent.systemPrompt,
                  "user: " + task.userInput
              ]
              taskRun.Status.Phase = Running
          case Running:
              if needsLLMCall(taskRun) {
                  response = callLLM(taskRun.Status.ContextWindow)
                  taskRun.Status.ContextWindow.append("assistant: " + response)
              }
              if toolCallRequested(response) {
                  recordToolCall(taskRun, toolDetails)
              }
              if conversationComplete(response) {
                  taskRun.Status.Phase = Completed
              }
          case PausedForUserInput:
              // Await external update to resume processing
      }
      updateStatus(taskRun)
  }
  ```

---

## 2. Asynchronous Operations & Long-Running Tasks

- **Requeue Mechanisms:**  
  Use the controller requeue pattern to keep trying if external calls (LLM, tool execution) have not yet completed.
- **Work Queue / Timer:**  
  Use timers within the reconcile loop to schedule rechecks.
- **Delegation:**  
  If a tool call indicates delegation, create a sub-resource (or spawn a sub-chain) and link it via owner references.
- **Pseudocode:**
  ```
  if externalCallInProgress {
      requeueAfter(delay)
  } else if responseReceived {
      processResponse(taskRun)
  }
  ```

---

## 3. Status Conditions and Event Handling

- **Status Fields:**  
  Each CRD will include:
  - `Phase` with detailed values.
  - A history log (timestamped phase transitions).
  - Additional conditions/reasons where applicable.
- **Event Recording:**  
  Use Kubernetes event recorder:
  ```
  recordEvent(resource, "Transitioned to phase: " + newPhase)
  ```
- **Example:**  
  In the TaskRun controller, record each phase transition for traceability.

---

## 4. Observability & Metrics Integration

- **Tracing:**  
  Instrument every Reconcile loop with OpenTelemetry:
  - Start a new span at the beginning.
  - Attach events (e.g., external call initiation, completion).
- **Metrics:**  
  Include Prometheus metrics such as:
  - Total reconciliations per CRD.
  - Duration of reconciliations.
  - Error counts.
- **Pseudocode:**
  ```
  ctx, span = tracer.Start(ctx, "ReconcileTaskRun")
  defer span.End()
  prometheusCounter.WithLabelValues("TaskRun").Inc()
  ```

---

## 5. Upgrade and Backwards Compatibility

- **Conversion Webhooks:**  
  Set up webhooks to support CRD version upgrades.
- **Defaulting:**  
  In validation and defaulting webhooks, ensure missing fields are populated with defaults.
- **Schema Evolution:**  
  Maintain backward compatibility by only adding new fields and preserving existing phase history.
- **Pseudocode:**
  ```
  if resource.Spec.newField is not set {
      resource.Spec.newField = defaultValue
  }
  ```

---

## 6. Testing Strategy

- **Unit Tests:**  
  - Use fake clients (e.g., controller-runtime’s fake client) to test Reconcile logic for each CRD.
- **Integration Tests:**  
  - Leverage the `envtest` environment to simulate API server interactions.
- **End-to-End Tests:**  
  - Deploy the operator in a KIND cluster via the kubechain-example make tasks.
- **Test Cases:**  
  Write tests for:
  - Successful reconciliation and phase transitions.
  - Failure modes (e.g., missing secret, dependency not ready).
  - Observability instrumentation (spans, events).
- **Example (Pseudocode Unit Test):**
  ```
  func TestReconcileTaskRun(t *testing.T) {
      fakeClient := NewFakeClient(initialObjects...)
      reconciler := NewTaskRunReconciler(fakeClient, ...)
      result, err := reconciler.Reconcile(ctx, request)
      assert.NoError(t, err)
      assert.Equal(t, expectedPhase, taskRun.Status.Phase)
  }
  ```

---

## 7. Deployment

- **Manifests:**  
  Create YAML manifests for CRDs and the operator deployment.
- **Make Tasks:**  
  Define make tasks to:
  - Launch a KIND cluster with custom settings.
  - Build and deploy the operator image.
  - Deploy kubechain-ui and an OpenTelemetry collector.
- **Example Make Task:**
  ```
  build:
      docker build -t myorg/kubechain-operator:latest .
  deploy:
      kubectl apply -f config/crd/bases
      kubectl apply -f config/manager/manager.yaml
  ```

---

## 8. Code Structure and Interfaces

- **Directory Layout:**
  - `/api/v1alpha1`: CRD API definitions.
  - `/controllers`: Controller reconciler logic for each CRD.
  - `/pkg/observability`: OTel and Prometheus instrumentation.
  - `/pkg/client`: Abstractions for external calls (LLM, tool executor).
- **Interface Definitions:**  
  Define clear interfaces to abstract external interactions. For example:
  ```
  type LLMClient interface {
      CallLLM(ctx context.Context, messages []Message) (string, error)
  }

  type ToolExecutor interface {
      Execute(toolName string, args json.RawMessage) (json.RawMessage, error)
  }
  ```
- **Code Modularity:**  
  Keep business logic separate from Kubernetes API interactions to facilitate unit testing and future enhancement.

---

This plan provides unambiguous, detailed instructions and pseudocode snippets for implementing each operator component. Editor engineers should use these guidelines to scaffold, iteratively develop, and test each module according to Kubernetes operator best practices.
