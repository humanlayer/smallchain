AI Agent Orchestration on Kubernetes – Design & Development Plan
Note: This document is tailored as a comprehensive plan for building a Kubernetes Operator (using Kubebuilder) that orchestrates agent-based AI workloads in a distributed manner. The design draws inspiration from:

HumanLayer’s docs
Got-Agents “Linear Assistant” code
SmallChain code from “humanlayer/smallchain”
Existing frameworks like LangChain, OpenAI Tools & Function Calling approach, Anthropic, etc.
Below you will find:

Executive Summary
Detailed Architecture & Requirements
Mermaid Diagrams for:
High-Level System Architecture
CRD Relationships
Workflow Process (Task orchestration, tool calling, observability)
Step-by-Step Development Guide (with Kubebuilder)
Detailed Code Samples in Go
Comparison and Discussion of Agent Frameworks
Additional Notes on Observability (OpenTelemetry), pausing/resuming tasks, best practices, trade-offs, etc.

1. Executive Summary
   We propose a Kubernetes Operator that manages AI “agents” capable of handling user tasks, delegating work, maintaining context, and retrieving responses from large language models (LLMs) and specialized tools. The approach leverages:

Kubernetes CRDs to represent:

LLM: Which model/provider (OpenAI, Anthropic) along with keys/config like temperature, etc. It should allow for keys from a secretRef or from an env var in the controller container
Tool: Reusable “tools” that can be invoked by an agent (e.g., in-process Go functions, external containers, remote calls, or other agents in the cluster). For delegating to other agents, the tool call input is a structured object {"message": str, "goal": str, "everything_that_happened_so_far": str} and the description should instruct the calling agent to provide lots and lots of detail! delegation toolsets can have an agentRef: {name: str} or agentRef: {selector: {matchLabels: {...}}. We will have support for a builtin ToolSet called "contact human" - this will include a HumanLayer ContactChannel object (see below for humanlayer models info)
ToolSet: a way to specify a set of tools that can be used by an agent, e.g. an MCP server is a single object in the system, but may expose multiple tools and schemas. Similarly a delegation tool may use a labelSelector to specifiy mutiple agents in a single object
Agent: Combines an LLM with system prompts and references to one or more Tools and/or ToolSets.
Task: A user request (prompt) targeted at a particular Agent.
TaskRun: The actual “instance” of a run, containing the conversation context, tool calls, and final results. When calling LLM during a taskRun, assemble all relevant toolsets into a flat set of tools to send to the llm.
TaskRunToolCall: any tool calls that are a descendant of a task run
TaskRunEvent: either: launching the taskRun with an input object, sending context to an llm, llm response received, TaskRunToolCalls call(s) created, TaskRunToolCalls result received

The TaskRun controller should poll for TaskRun objects where either 1) the taskrun was just launched but not sent to an llm, or 2) all the TaskRunToolCalls are resolved (ready to send back to llm) and assemble a context window to send to the LLM

A taskRunToolCall might have an agentRef, in which case it will also have a child object to represent the TaskRun for that delegation

A TaskRun might have a parentTaskRunToolCall ref

OpenTelemetry (OTEL) for tracing every step, including:

Timestamps for tool calls (requested, execution start, finish, result returned).
Automatic correlation between tasks, sub-tasks, and user involvement.

Workflow Interruption / Human-in-the-Loop:

Agents can pause a workflow and ask for human input. The system must checkpoint conversation state and allow the user to resume. This is done via CRD-based status and operators.

There are two cases in which we pause for human input:

- agent calls a human_contact tool - this allows the agent to contact one or more humans for input
- agent calls a tool that requires human approval

Example Application

```yaml
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: LLM
metadata:
  name: gpt-4o
spec:
  provider: openai
  apiKeyFrom:
    secretKeyRef:
      name: openai
      key: OPENAI_API_KEY
  config:
    model: gpt-4o
---
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: Agent
metadata:
  name: project-manager
spec:
  llmRef:
    name: gpt-4o
  systemPrompt: |
    You are a project manager.
    You are responsible for overseeing the project and ensuring it is completed on time and within budget.
    You are also responsible for delegating tasks to the team and tracking their progress.
  toolsetRefs:
    - name: delegate-to-calculator-operator
---
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: Tool
metadata:
  name: delegate-to-calculator-operator
spec:
  type: delegateToAgent
  agentRef:
    name: calculator-operator
---
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: Agent
metadata:
  name: calculator-operator
spec:
  llmRef:
    name: gpt-4o
  systemPrompt: |
    You are a calculator operator.
    You are responsible for calculating the result of a mathematical expression.
  toolsetRefs:
    - name: calculator-toolset

---
# a single inline tool
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: Tool
metadata:
  name: add
spec:
  name: add
  description: Add two numbers
  arguments:
    type: object
    properties:
      a:
        type: number
      b:
        type: number
    required:
      - a
      - b
  execute:
    builtin:
      name: add
---
# a set of tools
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: ToolSet
metadata:
  name: calculator-toolset
spec:
  tools:
    - name: subtract
      description: Subtract two numbers
      arguments:
        type: object
        properties:
          a:
            type: number
          b:
            type: number
        required:
          - a
          - b
    - name: multiply
      description: Multiply two numbers
      arguments:
        type: object
        properties:
          a:
            type: number
          b:
            type: number
        required:
          - a
          - b
    - name: divide
      description: Divide two numbers
      arguments:
        type: object
        properties:
          a:
            type: number
          b:
            type: number
        required:
          - a
          - b
---
# a toolset seeded by an MCP server
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: ToolSet
metadata:
  name: mcp-tools
spec:
  # this api is tbd
---
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: ContactChannel
metadata:
  name: contact-manager-in-slack
spec:
  slack:
    channel_or_user_id: U0711111111
    context_about_channel_or_user: "A dm with your manager"
    slackBotTokenFrom:
      secretKeyRef:
        name: slack-bot-token
        key: SLACK_BOT_TOKEN
---
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: Tool
metadata:
  name: contact-manager-in-slack
spec:
  type: humanContact
  contactChannelRef:
    name: contact-manager-in-slack
---
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: Task
metadata:
  name: add-task
spec:
  launchImmediately: true # causes a TaskRun to be created immediately
  agentRef:
    name: project-manager
  input:
    # message is required
    message: What is the result of 2 + 2?
    # these are optional, shown here for completeness, used in delegation
    goal: null
    everything_that_happened_so_far: null
```

this would cause the following objects to be created:

```yaml
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: TaskRun
metadata:
  name: add-task-run-01
  ownerReferences:
    apiVersion: kubechain.humanlayer.dev/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: Task
    name: add-task
    uid: f51b2779-de31-4cef-a165-bca5d8599615
spec:
  taskRef:
    name: add-task
status:
  parentTaskRunToolCallRef: null
  phase: Pending
  phaseHistory:
    - phase: Pending
      transitionTime: 2024-01-01T00:00:00Z
  contextWindow: # this is the default inline context window. To support larger context windows, we will want to store it in a DB in the cluster. redis or rqlite or etc etc
    - role: system
      content: |
        You are a project manager.
        You are responsible for overseeing the project and ensuring it is completed on time and within budget.
        You are also responsible for delegating tasks to the team and tracking their progress.
    - role: user
      content: |
        What is the result of 2 + 2?
```

after the controller picks up the TaskRun as needing action, it updates the phase to:

```yaml
phase: SendContextWindowToLLM
phaseHistory:
  - phase: Pending
    transitionTime: 2024-01-01T00:00:00Z
  - phase: SendContextWindowToLLM
    transitionTime: 2024-01-01T00:01:00Z
```

This locks the resource and prevents other controllers from working on it.
The timestamp can be used to expire the lock according to some policy (mechanism for this is TBD)

Then the controller sends the context window to the LLM. Upon receiving the response, the controller updates the phase to:

```yaml
phase: LLMResponseReceived
phaseHistory:
  - phase: Pending
    transitionTime: 2024-01-01T00:00:00Z
  - phase: SendContextWindowToLLM
    transitionTime: 2024-01-01T00:01:00Z
  - phase: ToolCallsPending
    transitionTime: 2024-01-01T00:02:00Z
```

and the context window to:

```yaml
contextWindow:
  - role: system
    content: |
      You are a project manager.
      You are responsible for overseeing the project and ensuring it is completed on time and within budget.
      You are also responsible for delegating tasks to the team and tracking their progress.
  - role: user
    content: |
      What is the result of 2 + 2?
  - role: assistant
    content: null
    tool_calls:
      - name: delegate-to-calculator-operator
        arguments: # llm generated
          message: What is the result of 2 + 2?
          goal: Calculate the result of 2 + 2
          everything_that_happened_so_far: "The user requested the result of 2 + 2"
```

The controller then creates a TaskRunToolCall object to represent the tool call:

```yaml
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: TaskRunToolCall
metadata:
  name: add-task-run-tool-call-01
  ownerReferences:
    apiVersion: kubechain.humanlayer.dev/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: TaskRun
    name: add-task-run-01
    uid: f51b2779-de31-4cef-a165-bca5d8599615
spec:
  taskRunRef:
    name: add-task-run-01
  toolRef:
    name: delegate-to-calculator-operator
  arguments:
    message: What is the result of 2 + 2?
    goal: Calculate the result of 2 + 2
    everything_that_happened_so_far: "The user requested the result of 2 + 2"
status:
  phase: Pending
  phaseHistory:
    - phase: Pending
      transitionTime: 2024-01-01T00:00:00Z
```

The TaskRunToolCall controller will then create a TaskRun for the delegation tool call:

```yaml
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: TaskRun
metadata:
  name: add-task-run-01
  ownerReferences:
    apiVersion: kubechain.humanlayer.dev/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: TaskRunToolCall
    name: add-task-run-tool-call-01
    uid: f51b2779-de31-4cef-a165-bca5d8599615
spec:
  taskRunToolCallRef:
    name: add-task-run-tool-call-01
  agentRef:
    name: calculator-operator
status:
  phase: Pending
  phaseHistory:
    - phase: Pending
      transitionTime: 2024-01-01T00:00:00Z
```

### checkpointing context remotely

All these examples use the Custom Resource to store the context window. To support larger context windows, we will want to store it in a DB in the cluster. redis or rqlite or etc etc

```yaml
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: ContextStore
metadata:
  name: redis-context-store
spec:
  default: false
  redis:
    host: redis.example.com
    port: 6379
```

and then in the task and taskRun

```yaml
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: Task
metadata:
  name: add-task
spec:
  contextStoreRef:
    name: redis-context-store
  agentRef:
    name: project-manager
  input:
    message: What is the result of 2 + 2?
```

```yaml
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: TaskRun
metadata:
  name: add-task-run-01
spec:
  taskRef:
    name: add-task
  contextRef:
    store: redis-context-store
    key: add-task-run-01
status:
  # ...
  contextWindow: null # because there is a contextRef, the contextWindow is not stored inline in the TaskRun status
```

If there is no default context store and none is specified, the context window is stored inline in the TaskRun.

Implementation:

Kubebuilder scaffolding for CRDs and controllers.
Asynchronous orchestration (event-driven reconciliations) for large-scale reliability.
Embodiment of logic (like that found in the TypeScript references from the “Linear Assistant” and “smallchain” examples) into a Go-based operator that manages the lifecycle of tasks, function calls, sub-agent creation, etc.
Big Picture
Goal: Provide a platform where you can create an Agent CRD describing capabilities (e.g. “project_manager” or “calculator_operator” from the Got-Agents code sample), then submit a Task that triggers a multi-step, LLM-driven conversation. Each step can call tools, spawn sub-agents, or ask for user input. The entire chain-of-thought is orchestrated and stored in the cluster’s etcd via CRDs, with robust checkpointing, observability, and concurrency management.

2. Detailed Architecture & Requirements
   2.1 Architectural Overview
   Core Concepts

LLM: Models the Large Language Model “profile” and provider configs (OpenAI, Anthropic, local model, etc.).
Toolset: Defines a set of “Tools” an agent can use, each tool containing:
In-Process Function (exposed in Go)
Container Reference (a Pod/Job to run ephemeral tasks)
Remote Agent (sub-agent delegation, transforms the input to that sub-agent’s user prompt)
In-Cluster or Remote Service (like a microservice call)
Agent: Ties together:
System Prompt (the agent’s personality or role)
LLM reference
Zero or More Toolsets
Task: A single high-level request from a user, specifying:
The Agent to use
The user prompt
TaskRun: The ephemeral or ongoing “execution” state for a Task:
Accumulated conversation context (token window)
Tool calls (and results)
Sub-agent invocations
Timestamps, logs, etc.
Observability:
OTEL instrumented at each operator reconciliation
Timestamps for tool calls (requested, started, finished, result returned)
Human-in-the-Loop: Pausing, waiting for user input, checkpointing “where we left off.”
Detailed Requirements

CRDs:
LLM
spec.provider: (openai, anthropic, local, etc.)
spec.apiKeySecretRef: reference to a Secret with the API key
spec.config: optional config like model name, temperature, max tokens, etc.
Toolset
spec.tools: array describing tools, each with:
toolType: e.g. “function”, “container”, “delegateToAgent”, “remoteService”
functionSpec: if toolType=function, has the function’s schema
containerSpec: if toolType=container, the container image + possible PodSpec overrides
…
Agent
spec.llmRef: reference to an LLM
spec.systemPrompt: system instructions for the LLM
spec.toolsetRefs: references to one or more Toolsets
Task
spec.agentRef: which agent to invoke
spec.userInput: the user’s message / request
TaskRun
spec.taskRef: the Task being run
status: dynamic fields:
contextWindow: the conversation so far
toolCallHistory: each tool invocation with timestamps
phase: e.g. “Pending”, “Running”, “PausedForUserInput”, “Completed”
Asynchronous Reconcilers
The operator watches for creation or changes to Tasks, spawns or updates a TaskRun.
TaskRun controllers handle orchestration steps:
Checking if the LLM needs to be called again
Checking if a tool call is pending
Sub-agent creation if needed
Pausing the pipeline if user input is requested
Marking the run as complete
OpenTelemetry
Each step (LLM call, Tool call, sub-agent creation) is a sub-span.
Timestamps for tool calls: “requested,” “start,” “finish,” “result.”
Pausing & Resuming
TaskRun.status.phase can be set to PausedForUserInput.
A separate human approval or user input CRD update can resume the TaskRun by providing the new user input.
2.2 Trade-Offs
Kubernetes Operators vs. Direct Scripts
Pros: Native Kubernetes constructs, better portability, automatic concurrency management, high availability (HA) via the K8s control plane, strong state reconciliation.
Cons: More initial setup complexity, requires cluster-level knowledge.
LangChain vs. Native Implementation
LangChain: Great variety of pre-built abstractions for memory, chains, retrievers. However, it’s Python-based (or JS/TS) and not always easy to port entirely into a Go operator.
OpenAI Function Calling: Very streamlined approach for describing function schemas. Works seamlessly with GPT-4. But lacks the broader library features (vector stores, chain logic, etc.).
BAML (like in “Linear Assistant”) or “SmallChain\*\*:
BAML: focuses on bridging large LLM calls with business logic (issue tracking, clarifications, etc.).
SmallChain: more minimal chain-of-thought pattern with DB integration.
Both are TS-based references but the design patterns can be mirrored in Go.
Key: The best approach is to borrow from each, adopting an operator-based approach with function-calling style tool interfaces. The “chain-of-thought” can be stored in CRDs or external DBs.

3.  Mermaid Diagrams
    3.1 High-Level System Architecture
    mermaid
    Copy
    flowchart LR
    A[User] -->|Creates Task via kubectl or API| B((K8s API Server))
    B -->|Stores CRD in etcd| C[Task CRD]
    C -->|Event| D[Task Controller]
    D -->|Creates/Updates| E[TaskRun CRD]
    E -->|Reconciliation Logic| D
    D -->|Reads config from| F[Agent CRD]
    F --> G[LLM CRD]
    F --> H[Toolset CRD]
    D -->|Calls LLM or Tools| I[(External Services)]
    D -->|Pauses if needed| J[(User Input)]
    D -->|Updates status| E
    3.2 CRD Relationships
    mermaid
    Copy
    erDiagram
    LLM }|--|| Agent : "used by"
    Toolset }|--|| Agent : "referenced by"
    Agent ||--|{ Task : "requested by"
    Task ||--|| TaskRun : "executed by"

        LLM {
          string provider
          string apiKeySecretRef
          string config
        }
        Toolset {
          string name
          string toolType
          json functionSpec
          json containerSpec
          json remoteSpec
        }
        Agent {
          string name
          ref llmRef
          list toolsetRefs
          string systemPrompt
        }
        Task {
          string name
          ref agentRef
          string userInput
        }
        TaskRun {
          string name
          ref taskRef
          json contextWindow
          json toolCallHistory
          string phase
        }

    3.3 Workflow Process (Tool Calls & Context Updates)
    mermaid
    Copy
    sequenceDiagram
    participant TaskController
    participant TaskRun
    participant LLM
    participant Tool
    participant OTel

        rect rgb(230, 230, 230)
        Note over TaskController,TaskRun: Reconcile loop triggered by new or updated Task/TaskRun
        end

        TaskController->>OTel: Start "TaskRun" Span
        TaskController->>TaskRun: Check status (context, needed calls, etc.)

        alt LLM call needed
            TaskRun->>OTel: Start sub-span for LLM call
            TaskRun->>LLM: Provide prompt + context
            LLM->>TaskRun: Return assistant content + optional function calls
            TaskRun->>OTel: End sub-span for LLM call
        end

        alt Tool call needed
            TaskRun->>OTel: Start sub-span for Tool call
            TaskRun->>Tool: Send function arguments
            Tool->>TaskRun: Return result
            TaskRun->>OTel: End sub-span for Tool call
        end

        alt Sub-agent or Delegation
            TaskRun->>Agent: Spawn sub-agent CRD if needed
            Agent->>TaskRun: Sub-agent results
        end

        alt Pause for user input
            TaskRun->>TaskRun: Update status=PausedForUserInput
            Note over TaskRun: Wait for external signal to resume
        end

        TaskRun->>TaskController: Update status with conversation
        TaskController->>OTel: End "TaskRun" Span

4.  Step-by-Step Development Guide
    Below is a step-by-step outline of how to implement the operator using Kubebuilder.

4.1 Repository Initialization
Install Kubebuilder (if not already):

bash
Copy

# MacOS (Homebrew)

brew install kubebuilder

# Alternatively, from release tarball

# see https://book.kubebuilder.io/quick-start.html for up-to-date instructions

Create a new Go module for your operator:

bash
Copy
mkdir ai-operator
cd ai-operator
go mod init github.com/your-org/ai-operator
Initialize Kubebuilder:

bash
Copy
kubebuilder init --domain=example.com --repo=github.com/your-org/ai-operator
4.2 Creating CRDs
We’ll create the following API groups/versions:

LLM CRD
Toolset CRD
Agent CRD
Task CRD
TaskRun CRD
Note: Each CRD can be placed under a single API group (e.g., ai.example.com) or multiple subgroups if you prefer.

4.2.1 LLM CRD & Controller
bash
Copy
kubebuilder create api --group ai --version v1alpha1 --kind LLM --resource --controller
api/v1alpha1/llm_types.go (simplified example):

go
Copy
package v1alpha1

import (
metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE! YOUR CUSTOM CRD DEFINITIONS GO HERE!

// LLMSpec defines the desired state of LLM
type LLMSpec struct {
Provider string `json:"provider,omitempty"`
APIKeySecretRef string `json:"apiKeySecretRef,omitempty"`
// Additional config parameters
Temperature *float64 `json:"temperature,omitempty"`
MaxTokens *int `json:"maxTokens,omitempty"`
// ... other fields if needed
}

// LLMStatus defines the observed state of LLM
type LLMStatus struct {
// e.g. store last ping or some connectivity checks if desired
Ready bool `json:"ready,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// LLM is the Schema for the llms API
type LLM struct {
metav1.TypeMeta `json:",inline"`
metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   LLMSpec   `json:"spec,omitempty"`
    Status LLMStatus `json:"status,omitempty"`

}

//+kubebuilder:object:root=true

// LLMList contains a list of LLM
type LLMList struct {
metav1.TypeMeta `json:",inline"`
metav1.ListMeta `json:"metadata,omitempty"`
Items []LLM `json:"items"`
}
The controller (controllers/llm_controller.go) could simply set status.ready=true if we can confirm e.g. that the secret is valid. Usually, this is minimal. For example:

go
Copy
package controllers

import (
"context"
"fmt"

    "github.com/go-logr/logr"
    apierrors "k8s.io/apimachinery/pkg/api/errors"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"

    aiv1alpha1 "github.com/your-org/ai-operator/api/v1alpha1"

)

type LLMReconciler struct {
client.Client
Log logr.Logger
}

//+kubebuilder:rbac:groups=ai.example.com,resources=llms,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ai.example.com,resources=llms/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ai.example.com,resources=llms/finalizers,verbs=update

func (r \*LLMReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
log := r.Log.WithValues("llm", req.NamespacedName)

    var llm aiv1alpha1.LLM
    if err := r.Client.Get(ctx, req.NamespacedName, &llm); err != nil {
        if apierrors.IsNotFound(err) {
            return ctrl.Result{}, nil
        }
        return ctrl.Result{}, err
    }

    // Example logic: mark LLM ready if we can find the secret with the API key
    // (In reality we'd check we can read the secret, maybe do a quick test call.)
    llm.Status.Ready = false

    // Pseudocode: check secret existence
    // ...
    // If found and valid:
    llm.Status.Ready = true

    if err := r.Client.Status().Update(ctx, &llm); err != nil {
        log.Error(err, "unable to update LLM status")
        return ctrl.Result{}, err
    }

    log.Info(fmt.Sprintf("LLM provider = %s, ready = %t", llm.Spec.Provider, llm.Status.Ready))

    // Return and don't requeue
    return ctrl.Result{}, nil

}

func (r \*LLMReconciler) SetupWithManager(mgr ctrl.Manager) error {
return ctrl.NewControllerManagedBy(mgr).
For(&aiv1alpha1.LLM{}).
Complete(r)
}
4.2.2 Toolset CRD & Controller
bash
Copy
kubebuilder create api --group ai --version v1alpha1 --kind Toolset --resource --controller
api/v1alpha1/toolset_types.go example:

go
Copy
type ToolSpec struct {
ToolType string `json:"toolType,omitempty"`
FunctionSpec *FunctionSpec `json:"functionSpec,omitempty"`
ContainerSpec *ContainerSpec `json:"containerSpec,omitempty"`
DelegateAgent *DelegateSpec `json:"delegateAgent,omitempty"`
RemoteService *RemoteSpec `json:"remoteService,omitempty"`
}

type FunctionSpec struct {
// define your function schema
Name string `json:"name,omitempty"`
Description string `json:"description,omitempty"`
// Possibly JSON schema for arguments
// ...
}

type ContainerSpec struct {
Image string `json:"image,omitempty"`
// Could embed a PodSpec or partial fields
}

type DelegateSpec struct {
AgentRef string `json:"agentRef,omitempty"`
}

type RemoteSpec struct {
Endpoint string `json:"endpoint,omitempty"`
}

type ToolsetSpec struct {
Tools []ToolSpec `json:"tools,omitempty"`
}

type ToolsetStatus struct {
Ready bool `json:"ready,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Toolset struct {
metav1.TypeMeta `json:",inline"`
metav1.ObjectMeta `json:"metadata,omitempty"`
Spec ToolsetSpec `json:"spec,omitempty"`
Status ToolsetStatus `json:"status,omitempty"`
}
// +kubebuilder:object:root=true
type ToolsetList struct {
metav1.TypeMeta `json:",inline"`
metav1.ListMeta `json:"metadata,omitempty"`
Items []Toolset `json:"items"`
}
Controller logic for a Toolset might just ensure references are valid or spin up any needed container backends.

4.2.3 Agent CRD & Controller
bash
Copy
kubebuilder create api --group ai --version v1alpha1 --kind Agent --resource --controller
api/v1alpha1/agent_types.go snippet:

go
Copy
type AgentSpec struct {
LLMRef string `json:"llmRef,omitempty"`
SystemPrompt string `json:"systemPrompt,omitempty"`
ToolsetRefs []string `json:"toolsetRefs,omitempty"`
// Possibly store delegation logic, default arguments, etc.
}

type AgentStatus struct {
Ready bool `json:"ready,omitempty"`
}

type Agent struct {
metav1.TypeMeta `json:",inline"`
metav1.ObjectMeta `json:"metadata,omitempty"`
Spec AgentSpec `json:"spec,omitempty"`
Status AgentStatus `json:"status,omitempty"`
}
Controller might ensure the LLMRef and ToolsetRefs exist and are “Ready.”

4.2.4 Task CRD & Controller
bash
Copy
kubebuilder create api --group ai --version v1alpha1 --kind Task --resource --controller
api/v1alpha1/task_types.go:

go
Copy
type TaskSpec struct {
AgentRef string `json:"agentRef,omitempty"`
UserInput string `json:"userInput,omitempty"`
}

type TaskStatus struct {
// We might store references to the active TaskRun
TaskRunName string `json:"taskRunName,omitempty"`
}

type Task struct {
metav1.TypeMeta `json:",inline"`
metav1.ObjectMeta `json:"metadata,omitempty"`
Spec TaskSpec `json:"spec,omitempty"`
Status TaskStatus `json:"status,omitempty"`
}
The Task controller orchestrates creation or updates to a TaskRun.

4.2.5 TaskRun CRD & Controller
bash
Copy
kubebuilder create api --group ai --version v1alpha1 --kind TaskRun --resource --controller
api/v1alpha1/taskrun_types.go:

go
Copy
type ToolCallRecord struct {
Name string `json:"name,omitempty"`
RequestedAt metav1.Time `json:"requestedAt,omitempty"`
StartedAt metav1.Time `json:"startedAt,omitempty"`
FinishedAt metav1.Time `json:"finishedAt,omitempty"`
ResultReceived metav1.Time `json:"resultReceived,omitempty"`
Arguments string `json:"arguments,omitempty"`
Result string `json:"result,omitempty"`
}

type TaskRunSpec struct {
TaskRef string `json:"taskRef,omitempty"`
// We could store ephemeral instructions for the run
}

type TaskRunPhase string

const (
TaskRunPhasePending TaskRunPhase = "Pending"
TaskRunPhaseRunning TaskRunPhase = "Running"
TaskRunPhasePaused TaskRunPhase = "PausedForUserInput"
TaskRunPhaseCompleted TaskRunPhase = "Completed"
)

type TaskRunStatus struct {
Phase TaskRunPhase `json:"phase,omitempty"`
ContextWindow []string `json:"contextWindow,omitempty"`
ToolCalls []ToolCallRecord `json:"toolCalls,omitempty"`
// More fields as needed...
}

type TaskRun struct {
metav1.TypeMeta `json:",inline"`
metav1.ObjectMeta `json:"metadata,omitempty"`
Spec TaskRunSpec `json:"spec,omitempty"`
Status TaskRunStatus `json:"status,omitempty"`
}
Controller implements the core orchestration logic:

Load the associated Task.
Load the Agent (and LLM, Toolsets) used by that Task.
If Phase=Pending, begin the conversation, set Phase=Running.
If the LLM is invoked, store partial results in status.contextWindow.
If a tool is called, create a record in status.toolCalls.
If the user input is needed, set Phase=PausedForUserInput.
When done, set Phase=Completed. 5. Example Operator Reconciliation Logic in Go
Below is a conceptual snippet for the TaskRun reconciliation. It sketches how you might handle LLM calls, tool calls, and sub-agent delegation. In practice, you would also add OpenTelemetry instrumentation in each step.

go
Copy
package controllers

import (
"context"
"time"

    "github.com/go-logr/logr"
    aiv1alpha1 "github.com/your-org/ai-operator/api/v1alpha1"
    corev1 "k8s.io/api/core/v1"
    apierrors "k8s.io/apimachinery/pkg/api/errors"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

)

type TaskRunReconciler struct {
client.Client
Log logr.Logger
}

// Reconcile is the main reconciliation loop for a TaskRun
func (r \*TaskRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
log := r.Log.WithValues("taskrun", req.NamespacedName)

    // 1) Fetch the TaskRun
    var taskRun aiv1alpha1.TaskRun
    if err := r.Client.Get(ctx, req.NamespacedName, &taskRun); err != nil {
        if apierrors.IsNotFound(err) {
            return ctrl.Result{}, nil
        }
        return ctrl.Result{}, err
    }

    // 2) If Completed or Paused, no further action
    if taskRun.Status.Phase == aiv1alpha1.TaskRunPhaseCompleted ||
       taskRun.Status.Phase == aiv1alpha1.TaskRunPhasePaused {
        return ctrl.Result{}, nil
    }

    // 3) Fetch the Task
    var task aiv1alpha1.Task
    if err := r.Client.Get(ctx, client.ObjectKey{Name: taskRun.Spec.TaskRef, Namespace: taskRun.Namespace}, &task); err != nil {
        log.Error(err, "Cannot find Task reference", "taskRef", taskRun.Spec.TaskRef)
        return ctrl.Result{}, nil
    }

    // 4) Fetch the Agent
    var agent aiv1alpha1.Agent
    if err := r.Client.Get(ctx, client.ObjectKey{Name: task.Spec.AgentRef, Namespace: task.Namespace}, &agent); err != nil {
        log.Error(err, "Cannot find Agent reference", "agentRef", task.Spec.AgentRef)
        return ctrl.Result{}, nil
    }

    // 5) If Phase=Pending, initialize conversation
    if taskRun.Status.Phase == "" || taskRun.Status.Phase == aiv1alpha1.TaskRunPhasePending {
        taskRun.Status.Phase = aiv1alpha1.TaskRunPhaseRunning
        // Start context window with system prompt, user input
        taskRun.Status.ContextWindow = append(taskRun.Status.ContextWindow,
            "system: "+agent.Spec.SystemPrompt,
            "user: "+task.Spec.UserInput,
        )
        if err := r.Status().Update(ctx, &taskRun); err != nil {
            return ctrl.Result{}, err
        }
        return ctrl.Result{Requeue: true}, nil
    }

    // 6) If running, we do the next step of orchestration
    //    (Pseudo-code: check if LLM call is needed, or if a tool call is in-flight, etc.)

    // Example: we call the LLM with the last context
    // (In reality, we'd have a client to call OpenAI or Anthropic, etc.)
    lastMessage := "assistant: This is a mock LLM response."
    // Append the LLM response to the context window
    taskRun.Status.ContextWindow = append(taskRun.Status.ContextWindow, lastMessage)

    // 7) Possibly parse the response to see if a tool call was requested
    //    For now, we just assume no tool calls. If we find a tool call, we record it:
    // toolCall := aiv1alpha1.ToolCallRecord{
    //     Name: "add",
    //     RequestedAt: metav1.Now(),
    //     Arguments: `{"x":2,"y":3}`,
    // }
    // taskRun.Status.ToolCalls = append(taskRun.Status.ToolCalls, toolCall)
    // we might set the phase or requeue the run

    // 8) If the conversation is done, set Completed
    taskRun.Status.Phase = aiv1alpha1.TaskRunPhaseCompleted

    if err := r.Status().Update(ctx, &taskRun); err != nil {
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil

}

func (r \*TaskRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
return ctrl.NewControllerManagedBy(mgr).
For(&aiv1alpha1.TaskRun{}).
Complete(r)
}
OTEL Instrumentation: For each step, you would open a span via e.g. opentelemetry-go, typically inside Reconcile(), or in sub-functions.

6. Comparison of Frameworks
   6.1 LangChain
   Pros:
   Rich ecosystem of pre-built connectors (vector DBs, retrievers, memory).
   Rapid prototyping in Python or TypeScript.
   Cons:
   Not Go-native. Integration with a Go-based operator might require bridging Python code or rewriting logic.
   Potentially heavy overhead if you only need a simpler chain-of-thought approach.
   6.2 OpenAI Function Calling
   Pros:
   Straightforward JSON schema-based approach for tools.
   Native to GPT-3.5/4; reduces “hallucination” risk because the model must strictly return JSON.
   Cons:
   Ties the system somewhat to OpenAI’s model capabilities. Other LLM providers might not have an exact equivalent.
   Less robust chain-of-thought memory utilities.
   6.3 BAML (from Notorious R.A.G & got-agents)
   Pros:
   Focuses on bridging business logic (like linear issues or email flows).
   Clear patterns for next-step logic and fallback to human contact.
   Cons:
   TS-based, so you would have to replicate the logic in Go or keep a sidecar approach.
   6.4 “SmallChain”
   Pros:
   Minimal, flexible approach for storing chain-of-thought in a DB.
   Good reference for an event-driven function-calling loop.
   Cons:
   Very minimal; you have to handle concurrency, re-entrancy, etc. yourself.
   Recommendation:
   Leverage the function-calling approach from OpenAI for structured tool definitions, while borrowing the multi-step orchestration patterns from “BAML” and “SmallChain.” For memory or advanced workflows, you could incorporate patterns from LangChain or simply replicate them in Go.

7. Additional Topics
   7.1 Observability Setup (OpenTelemetry)
   Installation: Deploy the OTEL Collector in your cluster.
   Instrument the Operator:
   In main.go, initialize OTel with a tracer provider (e.g. Jaeger or Zipkin exporter).
   In each Reconcile() method, create a new span:
   go
   Copy
   func (r \*TaskRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
   ctx, span := tracer.Start(ctx, "TaskRunReconcile")
   defer span.End()
   // ...
   }
   For tool calls, record the timestamps in a sub-span or attach them as events.
   7.2 Pausing & Resuming (Human-in-the-Loop)
   If the LLM indicates additional input from a human is required (like a “request_more_information” in BAML), the operator sets status.phase=PausedForUserInput.
   A separate UI or process updates the TaskRun to Running with spec.newUserInput or similar once the user has provided feedback.
   The operator picks up the new input on the next reconciliation cycle, continuing from the saved context.
   7.3 Distributed Systems Reliability
   Backpressure: Ensure your operator doesn’t overwhelm LLM providers if many TaskRuns are active.
   Retries & Timeouts: Use exponential backoff for external calls.
   Checkpoints: Each step is stored in TaskRun.status, so if the operator restarts, it can resume from the last known state.
   7.4 Example Flows
   User Flow:

A user runs:
bash
Copy
kubectl apply -f my-agent.yaml
kubectl apply -f my-task.yaml
The operator sees the new Task.
The operator creates a TaskRun to handle it.
The TaskRun transitions from Pending -> Running.
The LLM is called with the system prompt + user message.
The LLM might request a tool call or sub-agent.
On each step, the operator updates TaskRun.status, eventually finishing or pausing for user input.
If paused, a user might update the TaskRun with additional info.
The operator resumes, eventually marking TaskRun as Completed. 8. Conclusion
This design merges robust, Kubernetes-native concurrency, reliability, and state management with an agent-based approach to orchestrating AI tasks. By using CRDs for each major concept (LLM, Toolset, Agent, Task, TaskRun) and hooking them together with an operator, we gain:

Clear Separation of concerns
Scalability across large clusters
Extensibility for new tool types and LLM providers
Observability via OpenTelemetry traces & metrics
Final Recommendation
Begin with the CRDs outlined above.
Implement incremental reconcilers for Task and TaskRun.
Integrate an LLM library to call out to your chosen providers.
Provide an event-driven approach for sub-agent creation or function calls (similar to the TypeScript references).
Add pause/resume with CRD status changes for human involvement.
Instrument with OpenTelemetry for robust tracing.
Appendices
Appendix A: Tools & Functions Snippet (from got-agents / smallchain)
The provided TypeScript code from SmallChain and Linear Assistant demonstrates:

Tool definitions:
ts
Copy
const add_tools = (): ChatCompletionTool[] => [
{
type: "function",
function: {
name: "add",
description: "add two numbers",
parameters: {
// JSON schema
},
},
},
]
Tool invocation:
Storing each function call in a DB with timestamps, chaining sub-agents, etc.
We replicate these patterns in Go by storing tool invocation requests in the TaskRun.status.toolCalls[], including the needed timestamps and arguments.

Appendix B: References
Kubebuilder Book -
OpenTelemetry Go docs
LangChain GitHub
OpenAI Function Calls docs
Anthropic docs
got-agents/agents
humanlayer/smallchain

### HumanLayer models

<humanlayer_models>

type FunctionCallStatus = {
requested_at: Date
responded_at?: Date
approved?: boolean
comment?: string
reject_option_name?: string
}

type SlackContactChannel = {
// the slack channel or user id to contact
channel_or_user_id: string
// the context about the channel or user to contact
context_about_channel_or_user?: string
// the bot token to use to contact the channel or user
bot_token?: string
experimental_slack_blocks?: boolean
}

type SMSContactChannel = {
phone_number: string
context_about_user?: string
}

type WhatsAppContactChannel = {
phone_number: string
context_about_user?: string
}

type EmailContactChannel = {
address: string
context_about_user?: string

experimental_subject_line?: string
experimental_in_reply_to_message_id?: string
experimental_references_message_id?: string

// If provided, this Jinja2 template will be used to render the email body
template?: string
}

type ContactChannel = {
slack?: SlackContactChannel
sms?: SMSContactChannel
whatsapp?: WhatsAppContactChannel
email?: EmailContactChannel
}

type ResponseOption = {
name: string
title?: string
description?: string
prompt_fill?: string
interactive?: boolean
}

type FunctionCallSpec = {
// the function name to call
fn: string
// the function arguments
kwargs: Record<string, any>
// the contact channel to use to contact the human
channel?: ContactChannel
reject_options?: ResponseOption[]
// Optional state to be preserved across the request lifecycle
state?: Record<string, any>
}

type FunctionCall = {
// the run id
run_id: string
call_id: string
spec: FunctionCallSpec
status?: FunctionCallStatus
}

type HumanContactSpec = {
// the message to send to the human
msg: string
// the contact channel to use to contact the human
channel?: ContactChannel
response_options?: ResponseOption[]
// Optional state to be preserved across the request lifecycle
state?: Record<string, any>
}

type HumanContactStatus = {
requested_at?: Date
responded_at?: Date
// the response from the human
response?: string
// the name of the selected response option
response_option_name?: string
}

type HumanContact = {
// the run id
run_id: string
// the call id
call_id: string
// the spec for the human contact
spec: HumanContactSpec
status?: HumanContactStatus
}

export {
SlackContactChannel,
SMSContactChannel,
WhatsAppContactChannel,
EmailContactChannel,
ContactChannel,
ResponseOption,
FunctionCallSpec,
FunctionCallStatus,
FunctionCall,
HumanContactSpec,
HumanContactStatus,
HumanContact,
}

import { FunctionCall, HumanContact } from './models'

type EmailMessage = {
from_address: string
to_address: string[]
cc_address: string[]
bcc_address: string[]
subject: string
content: string
datetime: string
}

type EmailPayload = {
from_address: string
to_address: string
subject: string
body: string
message_id: string
previous_thread?: EmailMessage[]
raw_email: string
is_test?: boolean
}

type SlackMessage = {
from_user_id: string
channel_id: string
content: string
message_id: string
}

type SlackThread = {
thread_ts: string
channel_id: string
events: SlackMessage[]
}

type V1Beta2EmailEventReceived = {
is_test?: boolean
type: 'agent_email.received'
event: EmailPayload
}

type V1Beta2SlackEventReceived = {
is_test?: boolean
type: 'agent_slack.received'
event: SlackThread
}

type V1Beta2FunctionCallCompleted = {
is_test?: boolean
type: 'function_call.completed'
event: FunctionCall
}

type V1Beta2HumanContactCompleted = {
is_test?: boolean
type: 'human_contact.completed'
event: HumanContact
}

</humanlayer_models>

### kubechain-ui

create a small nextjs UI that can be run as a docker container, that enables a user to inspect the state
of all kubernetes CRs from kubechain

This should include api routes on the nextjs a kubernetes API client to fetch resources for the frontend

it does not need authorization or oauth for now, but it should include stubs for implementing those

If possible, include also a taskrun viewer that uses the otel data

### kubechain-example

create make tasks that can:

- launch a kubernetes in docker (kind) cluster with a custom config to open specific ports on the host (customize the nodePortRange too)
- build and deploy the current local kubechain implementation to the cluster, including CRDs
- build and deploy the current kubechain-ui implementation to the cluster
- deploy an off the shelf opentelemetry collector and visualizer to the cluster
