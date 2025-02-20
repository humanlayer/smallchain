_this is a knowledge file for codebuff and other coding agents, with instructions and guidelines for working on this project if you are a human reading this, some of this may not apply to you_

## Status Pattern

Resources follow a consistent status pattern:
- Ready: Boolean indicating if resource is ready
- Status: Enum with values "Ready" or "Error" or "Pending"
- StatusDetail: Detailed message about the current status
- Events: Emit events for validation success/failure and significant state changes

Example:
```yaml
status:
  ready: true
  status: Ready
  statusDetail: "OpenAI API key validated successfully"
```

Events:
- ValidationSucceeded: When resource validation passes
- ValidationFailed: When resource validation fails
- ResourceCreated: When child resources are created (e.g. TaskRunCreated)

New resources start in Pending state while validating dependencies.
Use Pending (not Error) when upstream dependencies exist but aren't ready.

## using the controller

The controller is running in the local kind cluster in the default namespace. The cluster is called `kubechain-example-cluster`.

You can use `make deploy-local-kind` to rebuild the controller and push it to the local kind cluster.

## progress tracking

BEFORE every change, update resume-kubechain-operator.md with your recent progress and whats next

## tests

after every change, validate with

```
make test
```

## end to end tests

to test things end to end, you can delete all existing examples

```
k delete task,taskrun,agent,llm  --all
```

then apply the new resources

```
kustomize build samples | kubectl apply -f -
```

then

```
kubectl get llm,tool,agent,task,taskrun
```

## things not to do

- IF YOU ARE RUNNING THE `kind` CLI you are doing something wrong
- DO NOT TRY to port-foward to grafana or anything else in the OTEL stack - i have that handled via node ports
- DO NOT TRY TO CHECK THINGS IN GRAFANA OR PROMETHEUS AT ALL - I will go look at them when you are ready for me to, just ask
- DO NOT USE `cat <<EOF` to generate files, just edit the files directly

## development principles

- controllers are stateless
- Resources should be in a Pending state until ready
- If a resource is waiting on a parent resource to become ready, it should be in the Pending state
- TaskRun phases progress through:
  1. Pending (initial state)
  2. SendContextWindowToLLM (locks the resource)
  3. ToolCallsPending
  4. FinalAnswer

## Application archicecture

### Example Application

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

## Testing Patterns

### Mocking External Services
- Use interfaces to abstract external service clients
- Inject client factories into controllers for easy mocking
- Mock clients should be simple and return predictable responses
- Example:
```go
type Client interface {
  DoSomething() error
}

type Controller struct {
  newClient func() Client  // factory function for easy mocking
}
```

## Build Optimizations

Docker builds use BuildKit caching:
- Enable with DOCKER_BUILDKIT=1
- Cache Go module downloads with `--mount=type=cache,target=/go/pkg/mod`
- Cache Go build cache with `--mount=type=cache,target=/root/.cache/go-build`
