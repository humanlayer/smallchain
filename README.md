<div align="center">

<h1>KubeChain</h1>

</div>

KubeChain is a cloud-native orchestrator for Autonomous AI Agents built on Kubernetes. It supports long-lived outer-loop agents that can process asynchronous execution of both LLM inference and long-running tool calls.

:warning: **Note** - KubeChain is highly experimental and has several known issues and race conditions. Use at your own risk.
 
<div align="center">

<h3>

[Discord](https://discord.gg/AK6bWGFY7d) | [Documentation](./docs) | [Examples](./kubechain-example)

</h3>

[![GitHub Repo stars](https://img.shields.io/github/stars/humanlayer/kubechain)](https://github.com/humanlayer/kubechain)
[![License: Apache-2](https://img.shields.io/badge/License-Apache-green.svg)](https://opensource.org/licenses/Apache-2)

</div>

## Table of Contents

- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Setting Up a Local Cluster](#setting-up-a-local-cluster)
  - [Deploying KubeChain](#deploying-kubechain)
  - [Creating Your First Agent](#creating-your-first-agent)
  - [Running Your First Task](#running-your-first-task)
  - [Monitoring Resources](#monitoring-resources)
- [Key Features](#key-features)
- [Design Principles](#design-principles)
- [Architecture](#architecture)
  - [Core Objects](#core-objects)
- [Contributing](#contributing)
- [License](#license)

## Getting Started

### Prerequisites

To run KubeChain, you'll need:

- **kubectl** - Command-line tool for Kubernetes
- **kind** - For running local Kubernetes clusters
- **OpenAI API Key** - For LLM functionality
- **Docker** - For building and running container images

### Setting Up a Local Cluster

1. **Create a Kind cluster**

   ```bash
   kind create cluster --config kubechain-example/kind/kind-config.yaml
   ```

2. **Add your OpenAI API key as a Kubernetes secret**

   ```bash
   kubectl create secret generic openai \
     --from-literal=OPENAI_API_KEY=$OPENAI_API_KEY \
     --namespace=default
   ```

### Deploying KubeChain

Deploy the KubeChain operator to your cluster:

```bash
make deploy-operator
```

This command will build the operator, create necessary CRDs, and deploy the KubeChain components to your cluster.

### Creating Your First Agent

1. **Define an LLM resource**

```bash
cat <<EOF | kubectl apply -f -
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
EOF
```

   Check the created LLM:
   
   ```bash
   kubectl get llm
   ```
   
   Output:
   ```
   NAME     PROVIDER   READY   STATUS
   gpt-4o   openai     true    Ready
   ```
   
   For more detailed information:
   
   ```bash
   kubectl get llm -o wide
   ```
   
   Output:
   ```
   NAME     PROVIDER   READY   STATUS   DETAIL
   gpt-4o   openai     true    Ready    OpenAI API key validated successfully
   ```

2. **Create an Agent resource**

```bash
cat <<EOF | kubectl apply -f -
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: Agent
metadata:
  name: my-assistant
spec:
  llmRef:
    name: gpt-4o
  system: |
    You are a helpful assistant. Your job is to help the user with their tasks.
EOF
```

   Check the created Agent:
   
```bash
kubectl get agent
```
   
   Output:
```
NAME           LLM      READY   STATUS
my-assistant   gpt-4o   true    Ready
```
   
   For more detailed information:
   
```bash
kubectl get agent -o wide
```
   
   Output:
```
NAME          READY   STATUS   DETAIL
my-assistant  true    Ready    All dependencies validated successfully
```

### Running Your First Task

1. **Create a Task resource**

   ```bash
   cat <<EOF | kubectl apply -f -
   apiVersion: kubechain.humanlayer.dev/v1alpha1
   kind: Task
   metadata:
     name: hello-world-task
   spec:
     agent: my-assistant
     userMessage: "Say hello to the world using the echo tool"
   EOF
   ```

   Check the created Task:
   
   ```bash
   kubectl get task
   ```
   
   Output:
   ```
   NAME               AGENT          READY   STATUS
   hello-world-task   my-assistant   true    Ready
   ```
   
   For more detailed information:
   
   ```bash
   kubectl get task -o wide
   ```

2. **Create a TaskRun to execute the task**

   ```bash
   cat <<EOF | kubectl apply -f -
   apiVersion: kubechain.humanlayer.dev/v1alpha1
   kind: TaskRun
   metadata:
     name: hello-world-run
   spec:
     task: hello-world-task
   EOF
   ```

### Monitoring Resources

Monitor the progress of your TaskRun:

```bash
kubectl get taskrun
```

Output:
```
NAME              TASK               PHASE      AGE
hello-world-run   hello-world-task   Running    30s
```

After completion:

```
NAME              TASK               PHASE         AGE
hello-world-run   hello-world-task   FinalAnswer   2m
```

For more detailed information:

```bash
kubectl get taskrun -o wide
```

View the detailed output:

```bash
kubectl get taskrun hello-world-run -o yaml
```

Sample output:
```yaml
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: TaskRun
metadata:
  name: hello-world-run
  namespace: default
spec:
  task: hello-world-task
status:
  completionTime: "2023-10-20T14:30:45Z"
  messages:
    - role: system
      content: You are a helpful assistant. Your job is to help the user with their tasks.
        You have access to a tool called simple-echo-tool that can echo messages back.
    - role: user
      content: Say hello to the world using the echo tool
    - role: assistant
      content: I'll help you say hello to the world using the echo tool.
      toolCalls:
        - id: call_01
          type: function
          function:
            name: simple-echo-tool
            arguments: '{"message":"Hello, World!"}'
    - role: tool
      toolCallId: call_01
      content: 'Echo: Hello, World!'
    - role: assistant
      content: I've used the echo tool to say hello to the world. The response was "Echo: Hello, World!"
  phase: FinalAnswer
  startTime: "2023-10-20T14:30:30Z"
```

Describe the TaskRun to see events and detailed status:

```bash
kubectl describe taskrun hello-world-run
```

### Adding Tools with MCP


### Cleaning Up

## Key Features

- **Kubernetes-Native Architecture**: KubeChain is built as a Kubernetes operator, using Custom Resource Definitions (CRDs) to define and manage LLMs, Agents, Tools, Tasks, and TaskRuns.

- **Durable Agent Execution**: KubeChain implements something like async/await at the infrastructure layer, checkpointing a conversation chain whenever a tool call or agent delegation occurs, with the ability to resume from that checkpoint when the operation completes.

- **Dynamic Workflow Planning**: Allows agents to reprioritize and replan their workflows mid-execution.

- **Observable Control Loop Architecture**: KubeChain uses a simple, observable control loop architecture that allows for easy debugging and observability into agent execution.

- **Scalable**: Leverages Kubernetes for scalability and resilience.

- **Human Approvals and Input**: Support for durable task execution across long-running function calls means a simple tool-based interface to allow an agent to ask a human for input or wait for an approval.

## Design Principles

- **Clarity**: Easy to understand what's happening and what the framework is doing with your prompts.

- **Control**: Ability to customize every aspect of agent behavior without framework limitations.

- **Modularity**: Composed of small control loops with limited scope that each progress the state of the world.

- **Durability**: Resilient to failures as a distributed system.

- **Simplicity**: Leverages the unique property of AI applications where the entire "call stack" can be expressed as the rolling context window accumulated through interactions and tool calls.

- **Extensibility**: Easy to build and share agents, tools, and tasks.

## Architecture

### Core Objects

- **LLM**: Provider + API Keys + Parameters
- **Agent**: LLM + System Prompt + Tools
- **Tool**: Function, API, Docker container, or another Agent
- **Task**: Agent + User Message
- **TaskRun**: Task + Current context window

## Contributing

KubeChain is open-source and we welcome contributions in the form of issues, documentation, pull requests, and more. See [CONTRIBUTING.md](./CONTRIBUTING.md) for more details.

## License

KubeChain is licensed under the Apache 2 License.