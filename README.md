<div align="center">

<h1>KubeChain</h1>

</div>

KubeChain is a cloud-native orchestrator for AI Agents built on Kubernetes. It supports [long-lived outer-loop agents](https://theouterloop.substack.com/p/openais-realtime-api-is-a-step-towards) that can process asynchronous execution of both LLM inference and long-running tool calls. It's designed for simplicity and gives strong durability and reliability guarantees for agents that make asynchronous tool calls like contacting humans or delegating work to other agents.

:warning: **Note** - KubeChain is experimental and some known issues and race conditions. Use at your own risk.
 
<div align="center">

<h3>

[Discord](https://discord.gg/AK6bWGFY7d) | [Documentation](./docs) | [Examples](./kubechain-example)

</h3>

[![GitHub Repo stars](https://img.shields.io/github/stars/humanlayer/kubechain)](https://github.com/humanlayer/kubechain)
[![License: Apache-2](https://img.shields.io/badge/License-Apache-green.svg)](https://opensource.org/licenses/Apache-2)

</div>

## Table of Contents

- [Key Features](#key-features)
- [Architecture](#architecture)
  - [Core Objects](#core-objects)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Setting Up a Local Cluster](#setting-up-a-local-cluster)
  - [Deploying KubeChain](#deploying-kubechain)
  - [Creating Your First Agent](#creating-your-first-agent)
  - [Running Your First Task](#running-your-first-task)
  - [Inspecting the TaskRun more closely](#inspecting-the-taskrun-more-closely)
  - [Adding Tools with MCP](#adding-tools-with-mcp)
  - [Cleaning Up](#cleaning-up)
- [Design Principles](#design-principles)
- [Contributing](#contributing)
- [License](#license)


## Architecture

### Core Objects

- **LLM**: Provider + API Keys + Parameters
- **Agent**: LLM + System Prompt + Tools
- **Tool**: Function, API, Docker container, or another Agent
- **Task**: Agent + User Message
- **TaskRun**: Task + Current context window

## Getting Started

### Prerequisites

To run KubeChain, you'll need:

- **kubectl** - Command-line tool for Kubernetes `brew install kubectl`
- **OpenAI API Key** - For LLM functionality https://platform.openai.com

To run KubeChain locally on macos, you'll also need:

- **kind** - For running local Kubernetes clusters `brew install kind` (other cluster options should work too)
- **Docker** - For building and running container images `brew install --cask docker`

### Setting Up a Local Cluster



1. **Create a Kind cluster**

```bash
kind create cluster
```

2. **Add your OpenAI API key as a Kubernetes secret**

```bash
kubectl create secret generic openai \
  --from-literal=OPENAI_API_KEY=$OPENAI_API_KEY \
  --namespace=default
```

### Deploying KubeChain


> [!TIP]
> For better visibility when running tutorial, we recommend starting 
> a stream to watch all the events as they're happening,
> for example:
> 
> ```bash
> kubectl get events --watch
> ```

Deploy the KubeChain operator to your cluster:

```bash
kubectl apply -f https://raw.githubusercontent.com/humanlayer/smallchain/refs/heads/main/kubechain/config/release/latest.yaml
```

<details>
<summary>Just the CRDs</summary>

```bash
kubectl apply -f https://raw.githubusercontent.com/humanlayer/smallchain/refs/heads/main/kubechain/config/release/latest-crds.yaml
```

</details>

<details>
<summary>Install a specific version</summary>

```bash
kubectl apply -f https://raw.githubusercontent.com/humanlayer/smallchain/refs/heads/main/kubechain/config/release/v0.1.0.yaml
```

</details>

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

```mermaid
graph RL
    LLM
    Secret

    Credentials --> Secret

    subgraph LLM
      Provider
      Credentials
      ModelParameters
    end

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
   
<details>
<summary>Using `-o wide` and `describe`</summary>
   
```bash
kubectl get llm -o wide
```
   
   Output:
```
NAME     PROVIDER   READY   STATUS   DETAIL
gpt-4o   openai     true    Ready    OpenAI API key validated successfully
```

```bash
kubectl describe llm
```

Output:

```
Name:         gpt-4o
Namespace:    default
Labels:       <none>
Annotations:  <none>
API Version:  kubechain.humanlayer.dev/v1alpha1
Kind:         LLM
Metadata:
  Creation Timestamp:  2025-03-21T20:18:17Z
  Generation:          2
  Resource Version:    1682222
  UID:                 973098fb-2b8d-46b3-be49-81592e0b8f4e
Spec:
  API Key From:
    Secret Key Ref:
      Key:   OPENAI_API_KEY
      Name:  openai
  Provider:  openai
Status:
  Ready:          true
  Status:         Ready
  Status Detail:  OpenAI API key validated successfully
Events:
  Type    Reason               Age                 From            Message
  ----    ------               ----                ----            -------
  Normal  ValidationSucceeded  32m (x3 over 136m)  llm-controller  OpenAI API key validated successfully
```

</details>

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

```mermaid
graph RL
    Agent
    LLM
    Secret

    LLMRef --> LLM
    Credentials --> Secret


    subgraph LLM
      Provider
      Credentials
      ModelParameters
    end

    subgraph Agent
      LLMRef
      SystemPrompt
    end
```

   Check the created Agent:
   
```bash
kubectl get agent
```
   
   Output:
```
NAME           READY   STATUS
my-assistant   true    Ready
```
   
<details>
<summary>Using `-o wide` and `describe`</summary>
   
```bash
kubectl get agent -o wide
```
   
   Output:
```
NAME           READY   STATUS   DETAIL
my-assistant   true    Ready    All dependencies validated successfully
```

```bash
kubectl describe agent
```

Output:

```
Name:         my-assistant
Namespace:    default
Labels:       <none>
Annotations:  <none>
API Version:  kubechain.humanlayer.dev/v1alpha1
Kind:         Agent
Metadata:
  Creation Timestamp:  2025-03-21T22:06:27Z
  Generation:          1
  Resource Version:    1682754
  UID:                 e389b3e5-c718-4abd-aa72-d4fc82c9b992
Spec:
  Llm Ref:
    Name:  gpt-4o
  System:  You are a helpful assistant. Your job is to help the user with their tasks.

Status:
  Ready:          true
  Status:         Ready
  Status Detail:  All dependencies validated successfully
Events:
  Type    Reason               Age                From              Message
  ----    ------               ----               ----              -------
  Normal  Initializing         64m                agent-controller  Starting validation
  Normal  ValidationSucceeded  64m (x2 over 64m)  agent-controller  All dependencies validated successfully
```

</details>

### Running Your First Task

1. **Create a Task resource**

```bash
cat <<EOF | kubectl apply -f -
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: Task
metadata:
  name: hello-world-task
spec:
  agentRef:
    name: my-assistant
  message: "What is the capital of the moon?"
EOF
```

```mermaid
graph RL
    Agent
    LLM
    Secret

    LLMRef --> LLM
    Credentials --> Secret
    AgentRef --> Agent


    subgraph LLM
      Provider
      Credentials
      ModelParameters
    end

    subgraph Agent
      LLMRef
      SystemPrompt
    end

    subgraph Task
      AgentRef
      Message
    end
```
Check the created Task:
   
```bash
kubectl get task
```
   
   Output:

```
NAME               READY   STATUS   AGENT          MESSAGE
hello-world-task   true    Ready    my-assistant   What is the capital of the moon?
```
   
<details>
<summary>Using `-o wide` and `describe`</summary>
   
```bash
kubectl get task -o wide
```

Output:

```
NAME               READY   STATUS   DETAIL             AGENT          MESSAGE                            OUTPUT
hello-world-task   true    Ready    Task Run Created   my-assistant   What is the capital of the moon?
```

```bash
kubectl describe task
```

Output:

```
ame:         hello-world-task
Namespace:    default
Labels:       <none>
Annotations:  <none>
API Version:  kubechain.humanlayer.dev/v1alpha1
Kind:         Task
Metadata:
  Creation Timestamp:  2025-03-21T22:14:09Z
  Generation:          1
  Resource Version:    1683590
  UID:                 8d0c7d4a-88db-4005-b212-a2c3a6956af3
Spec:
  Agent Ref:
    Name:   my-assistant
  Message:  What is the capital of the moon?
Status:
  Ready:          true
  Status:         Ready
  Status Detail:  Task Run Created
Events:
  Type    Reason               Age                From             Message
  ----    ------               ----               ----             -------
  Normal  Initializing         56m                task-controller  Starting validation
  Normal  TaskRunCreated       56m                task-controller  Created TaskRun hello-world-task-1
```

</details>

By default, creating a task will create an initial TaskRun to execute the task. A TaskRun is a single execution of a task by an agent, tracking the context window and final output.

```mermaid
graph RL
    Agent
    LLM
    Secret

    LLMRef --> LLM
    Credentials --> Secret
    AgentRef --> Agent
    TaskRef --> Task


    subgraph LLM
      Provider
      Credentials
      ModelParameters
    end

    subgraph Agent
      LLMRef
      SystemPrompt
    end

    subgraph Task
      AgentRef
      Message
    end

    subgraph TaskRun
      TaskRef
      subgraph ContextWindow
        direction LR
        SystemMessage
        UserMessage
      end
    end

```

For now, our task run should complete quickly and return a FinalAnswer.

```bash
kubectl get taskrun 
```

Output:

```
NAME                 READY   STATUS   PHASE         TASK               PREVIEW   OUTPUT
hello-world-task-1   true    Ready    FinalAnswer   hello-world-task             The Moon does not have a capital. It is a natural satellite of Earth and lacks any governmental structure or human habitation that would necessitate a capital city.
```

To get just the output, run

```
kubectl get taskrun -o jsonpath='{.items[*].status.output}'
```

and you'll see 


> The Moon does not have a capital. It is a natural satellite of Earth and lacks any governmental structure or human habitation that would necessitate a capital city.

More broadly, the taskrun is a component of the LLM / Agent / Task relationship.

```mermaid
flowchart RL
    Agent
    LLM
    Secret

    LLMRef --> LLM
    Credentials --> Secret
    AgentRef --> Agent
    TaskRef --> Task


    subgraph LLM
      Provider
      Credentials
      ModelParameters
    end

    subgraph Agent
      LLMRef
      SystemPrompt
    end

    subgraph Task
      AgentRef
      Message
    end

    subgraph TaskRun
      TaskRef
      subgraph ContextWindow
        direction LR
        SystemMessage
        UserMessage
        AssistantMessage
      end
    end

    ContextWindow <--> Provider <--> OpenAI

```

### Inspecting the TaskRun more closely

We saw above how you can get the status of a taskrun with `kubectl get taskrun`.

For more detailed information, like to see the full context window, you can use:

```bash
kubectl describe taskrun 
```

```
Name:         hello-world-task-1
Namespace:    default
Labels:       kubechain.humanlayer.dev/task=hello-world-task
Annotations:  <none>
API Version:  kubechain.humanlayer.dev/v1alpha1
Kind:         TaskRun
Metadata:
  Creation Timestamp:  2025-03-21T22:14:09Z
  Generation:          1
  Owner References:
    API Version:     kubechain.humanlayer.dev/v1alpha1
    Controller:      true
    Kind:            Task
    Name:            hello-world-task
    UID:             8d0c7d4a-88db-4005-b212-a2c3a6956af3
  Resource Version:  1683602
  UID:               53b1b69a-fb49-431b-857a-1cafe017a544
Spec:
  Task Ref:
    Name:  hello-world-task
Status:
  Context Window:
    Content:  You are a helpful assistant. Your job is to help the user with their tasks.

    Role:         system
    Content:      What is the capital of the moon?
    Role:         user
    Content:      The Moon does not have a capital. It is a natural satellite of Earth
        and lacks any governmental structure or human habitation that would necessitate
        a capital city.
    Role:         assistant
  Output:         The Moon does not have a capital. It is a natural satellite of Earth and
      lacks any governmental structure or human habitation that would necessitate
      a capital city.
  Phase:          FinalAnswer
  Ready:          true
  Status:         Ready
  Status Detail:  LLM final response received
Events:
  Type    Reason               Age   From                Message
  ----    ------               ----  ----                -------
  Normal  Waiting              17m   taskrun-controller  Waiting for task "hello-world-task" to become ready
  Normal  ValidationSucceeded  17m   taskrun-controller  Task validated successfully
  Normal  LLMFinalAnswer       17m   taskrun-controller  LLM response received successfully
```

or

```bash
kubectl get taskrun -o yaml
```

<details>
<summary>Output (truncated for brevity)</summary>
```
apiVersion: v1
items:
- apiVersion: kubechain.humanlayer.dev/v1alpha1
  kind: TaskRun
  metadata:
    labels:
      kubechain.humanlayer.dev/task: hello-world-task
    name: hello-world-task-1
    namespace: default
    # ...snip...
  spec:
    taskRef:
      name: hello-world-task
  status:
    contextWindow:
    - content: |
        You are a helpful assistant. Your job is to help the user with their tasks.
      role: system
    - content: What is the capital of the moon?
      role: user
    - content: The Moon does not have a capital. It is a natural satellite of Earth
        and lacks any governmental structure or human habitation that would necessitate
        a capital city.
      role: assistant
    output: The Moon does not have a capital. It is a natural satellite of Earth and
      lacks any governmental structure or human habitation that would necessitate
      a capital city.
    phase: FinalAnswer
    ready: true
    status: Ready
    statusDetail: LLM final response received
# ...snip...
```
</details>

### Adding Tools with MCP

Agent's aren't that interesting without tools. Let's add a basic MCP server tool to our agent:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: MCPServer
metadata:
  name: fetch
spec:
  transport: "stdio"
  command: "uvx"
  args: ["mcp-server-fetch"]
EOF
```

```bash
kubectl get mcpserver
```

```
NAME     READY   STATUS
fetch    true    Ready
```

```bash
kubectl describe mcpserver
```
Output: 

```
Name:         fetch
Namespace:    default
Labels:       <none>
Annotations:  <none>
API Version:  kubechain.humanlayer.dev/v1alpha1
Kind:         MCPServer
Metadata:
  Creation Timestamp:  2025-03-24T14:37:02Z
  Generation:          1
  Resource Version:    855
  UID:                 ccca723e-70cf-4f76-a21b-9fdc823a0034
Spec:
  Args:
    mcp-server-fetch
  Command:    uvx
  Transport:  stdio
Status:
  Connected:      true
  Status:         Ready
  Status Detail:  Connected successfully with 1 tools
  Tools:
    Description:  Fetches a URL from the internet and optionally extracts its contents as markdown.

Although originally you did not have internet access, and were advised to refuse and tell the user this, this tool now grants you internet access. Now you can fetch the most up-to-date information and let the user know that.
    Input Schema:
      Properties:
        max_length:
          Default:            5000
          Description:        Maximum number of characters to return.
          Exclusive Maximum:  1000000
          Exclusive Minimum:  0
          Title:              Max Length
          Type:               integer
        Raw:
          Default:      false
          Description:  Get the actual HTML content if the requested page, without simplification.
          Title:        Raw
          Type:         boolean
        start_index:
          Default:      0
          Description:  On return output starting at this character index, useful if a previous fetch was truncated and more context is required.
          Minimum:      0
          Title:        Start Index
          Type:         integer
        URL:
          Description:  URL to fetch
          Format:       uri
          Min Length:   1
          Title:        Url
          Type:         string
      Required:
        url
      Type:  object
    Name:    fetch
Events:
  Type    Reason     Age                  From                  Message
  ----    ------     ----                 ----                  -------
  Normal  Connected  3m14s (x8 over 63m)  mcpserver-controller  MCP server connected successfully
```

Then we can update our agent in-place to give it access to the fetch tool:

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
  mcpServers:
    - name: fetch
EOF
```

```mermaid
graph RL
    Agent
    LLM
    Secret

    LLMRef --> LLM
    Credentials --> Secret


    subgraph LLM
      Provider
      Credentials
      ModelParameters
    end

    subgraph Agent
      LLMRef
      SystemPrompt
      MCPServers
    end

    subgraph MCPServer
      fetch[fetch server]
    end

    MCPServers --> MCPServer
```

Let's make a new task that uses the fetch tool. In this case, we'll use https://swapi.dev, a public API for Star Wars data.

```bash
cat <<EOF | kubectl apply -f -
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: Task
metadata:
  name: fetch-task
spec:
  agentRef:
    name: my-assistant
  message: "what is the data at https://swapi.dev/api/people/1? "
EOF
```

You should see some events in the output of 

```
kubectl get events --watch
```

```
0s          Normal   ValidationSucceeded         taskrun/fetch-task-1                                 Task validated successfully
0s          Normal   SendingContextWindowToLLM   taskrun/fetch-task-1                                 Sending context window to LLM
0s          Normal   ToolCallsPending            taskrun/fetch-task-1                                 LLM response received, tool calls pending
0s          Normal   ToolCallCreated             taskrun/fetch-task-1                                 Created TaskRunToolCall fetch-task-1-toolcall-01
0s          Normal   ExecutionSucceeded          taskruntoolcall/fetch-task-1-toolcall-01             MCP tool "fetch__fetch" executed successfully
0s          Normal   AllToolCallsCompleted       taskrun/fetch-task-1                                 All tool calls completed, ready to send tool results to LLM
0s          Normal   SendingContextWindowToLLM   taskrun/fetch-task-1                                 Sending context window to LLM
0s          Normal   LLMFinalAnswer              taskrun/fetch-task-1                                 LLM response received successfully
```


```
kubectl get taskrun fetch-task-1 -o jsonpath='{.status.output}'
```

> The URL [https://swapi.dev/api/people/1](https://swapi.dev/api/people/1) contains the following data about a Star Wars character:
> 
> - **Name**: Luke Skywalker
> - **Height**: 172 cm
> - **Mass**: 77 kg
> - **Hair Color**: Blond
> - **Skin Color**: Fair
> - **Eye Color**: Blue
> - **Birth Year**: 19BBY
> - **Gender**: Male
> - **Homeworld**: [Link to Homeworld](https://swapi.dev/api/planets/1/)
> - **Films**: Appeared in several films, linked as:
>   - [Film 1](https://swapi.dev/api/films/1/)
>   - [Film 2](https://swapi.dev/api/films/2/)
>   - [Film 3](https://swapi.dev/api/films/3/)
>   - [Film 6](https://swapi.dev/api/films/6/)
> - **Species**: None listed
> - **Vehicles**:
>   - [Vehicle 14](https://swapi.dev/api/vehicles/14/)
>   - [Vehicle 30](https://swapi.dev/api/vehicles/30/)
> - **Starships**:
>   - [Starship 12](https://swapi.dev/api/starships/12/)
>   - [Starship 22](https://swapi.dev/api/starships/22/)
> - **Created**: 2014-12-09T13:50:51.644000Z
> - **Edited**: 2014-12-20T21:17:56.891000Z
> - **URL**: [https://swapi.dev/api/people/1/](https://swapi.dev/api/people/1/)

and you can describe the taskrun to see the full context window and tool-calling turns

```
kubectl describe taskrun fetch-task-1 
```

A simplified view of the taskrun:

```mermaid
flowchart TD

    subgraph LLM
      Provider
      Credentials
    end
    subgraph MCPServer
      fetch
    end
    subgraph TaskRun
      subgraph ContextWindow
        direction LR
        SystemMessage
        UserMessage
        ToolCall-1
        ToolResponse-1
        AssistantMessage
      end
    end
    SystemMessage --> UserMessage

    Task2[Task]
    Agent2[Agent]

    UserMessage --> Task --> Agent --> LLM 
    Provider --> OpenAI
    Secret --> Credentials
    Credentials --> OpenAI
    OpenAI --> ToolCall-1 
    ToolCall-1 --> Task2 --> Agent2 --> fetch
    fetch --> ToolResponse-1
    ToolResponse-1 --> OpenAI2[OpenAI]
    OpenAI2 --> AssistantMessage
```

* * *

Putting another way, the TaskRun controller is responsible for sending the context window to the LLM, and
for processing the LLM response with the appropriate tool calls from MCP servers.

```mermaid
flowchart RL
    Agent
    LLM
    Secret

    LLMRef --> LLM
    Credentials --> Secret
    AgentRef --> Agent
    TaskRef --> Task


    subgraph LLM
      Provider
      Credentials
      ModelParameters
    end

    subgraph Agent
      LLMRef
      SystemPrompt
      MCPServers
    end

    subgraph Task
      AgentRef
      Message
    end

    subgraph MCPServer
      fetch[fetch server]
    end

    MCPServers --> MCPServer


    subgraph TaskRun
      TaskRef
      subgraph ContextWindow
        direction LR
        SystemMessage
        UserMessage
        ToolCall-1
        ToolResponse-1
        ToolCall-2
        ToolResponse-2
        AssistantMessage
      end
    end

    ContextWindow <--> MCPServer <--> fetch
    ContextWindow <--> Provider <--> OpenAI


```

```
Name:         fetch-task-1
Namespace:    default
Labels:       kubechain.humanlayer.dev/task=fetch-task
Annotations:  <none>
API Version:  kubechain.humanlayer.dev/v1alpha1
Kind:         TaskRun
Metadata:
  Creation Timestamp:  2025-03-25T22:32:30Z
  Generation:          1
  Owner References:
    API Version:     kubechain.humanlayer.dev/v1alpha1
    Controller:      true
    Kind:            Task
    Name:            fetch-task
    UID:             b461354c-f4c7-4cd3-93ec-7546c892d10e
  Resource Version:  1731
  UID:               e57a39a2-4c6f-42db-80f0-d48f9bd1d5b4
Spec:
  Task Ref:
    Name:  fetch-task
Status:
  Context Window:
    Content:  You are a helpful assistant. Your job is to help the user with their tasks.

    Role:     system
    Content:  what is the data at https://swapi.dev/api/people/1?
    Role:     user
    Content:
    Role:     assistant
    Tool Calls:
      Function:
        Arguments:  {"url":"https://swapi.dev/api/people/1"}
        Name:       fetch__fetch
      Id:           call_sLHvRHebP7YkpLUsAdKiF6u0
      Type:         function
    Content:        Content type application/json cannot be simplified to markdown, but here is the raw content:
Contents of https://swapi.dev/api/people/1:
{"name":"Luke Skywalker","height":"172","mass":"77","hair_color":"blond","skin_color":"fair","eye_color":"blue","birth_year":"19BBY","gender":"male","homeworld":"https://swapi.dev/api/planets/1/","films":["https://swapi.dev/api/films/1/","https://swapi.dev/api/films/2/","https://swapi.dev/api/films/3/","https://swapi.dev/api/films/6/"],"species":[],"vehicles":["https://swapi.dev/api/vehicles/14/","https://swapi.dev/api/vehicles/30/"],"starships":["https://swapi.dev/api/starships/12/","https://swapi.dev/api/starships/22/"],"created":"2014-12-09T13:50:51.644000Z","edited":"2014-12-20T21:17:56.891000Z","url":"https://swapi.dev/api/people/1/"}
    Role:          tool
    Tool Call Id:  call_sLHvRHebP7YkpLUsAdKiF6u0
    Content:       The URL [https://swapi.dev/api/people/1](https://swapi.dev/api/people/1) contains the following data about a Star Wars character:

- **Name**: Luke Skywalker
- **Height**: 172 cm
- **Mass**: 77 kg
- **Hair Color**: Blond
- **Skin Color**: Fair
- **Eye Color**: Blue
- **Birth Year**: 19BBY
- **Gender**: Male
- **Homeworld**: [Link to Homeworld](https://swapi.dev/api/planets/1/)
- **Films**: Appeared in several films, linked as:
  - [Film 1](https://swapi.dev/api/films/1/)
  - [Film 2](https://swapi.dev/api/films/2/)
  - [Film 3](https://swapi.dev/api/films/3/)
  - [Film 6](https://swapi.dev/api/films/6/)
- **Species**: None listed
- **Vehicles**:
  - [Vehicle 14](https://swapi.dev/api/vehicles/14/)
  - [Vehicle 30](https://swapi.dev/api/vehicles/30/)
- **Starships**:
  - [Starship 12](https://swapi.dev/api/starships/12/)
  - [Starship 22](https://swapi.dev/api/starships/22/)
- **Created**: 2014-12-09T13:50:51.644000Z
- **Edited**: 2014-12-20T21:17:56.891000Z
- **URL**: [https://swapi.dev/api/people/1/](https://swapi.dev/api/people/1/)
    Role:  assistant
  Output:  The URL [https://swapi.dev/api/people/1](https://swapi.dev/api/people/1) contains the following data about a Star Wars character:

- **Name**: Luke Skywalker
- **Height**: 172 cm
- **Mass**: 77 kg
- **Hair Color**: Blond
- **Skin Color**: Fair
- **Eye Color**: Blue
- **Birth Year**: 19BBY
- **Gender**: Male
- **Homeworld**: [Link to Homeworld](https://swapi.dev/api/planets/1/)
- **Films**: Appeared in several films, linked as:
  - [Film 1](https://swapi.dev/api/films/1/)
  - [Film 2](https://swapi.dev/api/films/2/)
  - [Film 3](https://swapi.dev/api/films/3/)
  - [Film 6](https://swapi.dev/api/films/6/)
- **Species**: None listed
- **Vehicles**:
  - [Vehicle 14](https://swapi.dev/api/vehicles/14/)
  - [Vehicle 30](https://swapi.dev/api/vehicles/30/)
- **Starships**:
  - [Starship 12](https://swapi.dev/api/starships/12/)
  - [Starship 22](https://swapi.dev/api/starships/22/)
- **Created**: 2014-12-09T13:50:51.644000Z
- **Edited**: 2014-12-20T21:17:56.891000Z
- **URL**: [https://swapi.dev/api/people/1/](https://swapi.dev/api/people/1/)
  Phase:  FinalAnswer
  Ready:  true
  Span Context:
    Span ID:      6f20c4536fa90bdc
    Trace ID:     aab3bbe4e9ea84218ee0a848b5958dcb
  Status:         Ready
  Status Detail:  LLM final response received
Events:
  Type    Reason                     Age                  From                Message
  ----    ------                     ----                 ----                -------
  Normal  ValidationSucceeded        114s                 taskrun-controller  Task validated successfully
  Normal  ToolCallsPending           114s                 taskrun-controller  LLM response received, tool calls pending
  Normal  ToolCallCreated            114s                 taskrun-controller  Created TaskRunToolCall fetch-task-1-toolcall-01
  Normal  SendingContextWindowToLLM  109s (x2 over 114s)  taskrun-controller  Sending context window to LLM
  Normal  AllToolCallsCompleted      109s                 taskrun-controller  All tool calls completed, ready to send tool results to LLM
  Normal  LLMFinalAnswer             105s                 taskrun-controller  LLM response received successfully
```

That's it! Go add your favorite MCPs and start running durable agents in Kubernetes!


### Cleaning Up

Remove our agent, task and related resources:

```
kubectl delete taskruntoolcall --all
kubectl delete taskrun --all
kubectl delete task --all
kubectl delete agent --all 
kubectl delete mcpserver --all
kubectl delete llm --all
```

Remove the OpenAI secret:

```
kubectl delete secret openai
```

Remove the operator, resources and custom resource definitions:

```
kubectl delete -f https://raw.githubusercontent.com/humanlayer/smallchain/refs/heads/main/kubechain/config/release/latest.yaml
```

If you made a kind cluster, you can delete it with:

```
kind delete cluster 
```

## Key Features

- **Kubernetes-Native Architecture**: KubeChain is built as a Kubernetes operator, using Custom Resource Definitions (CRDs) to define and manage LLMs, Agents, Tools, Tasks, and TaskRuns. 

- **Durable Agent Execution**: KubeChain implements something like async/await at the infrastructure layer, checkpointing a conversation chain whenever a tool call or agent delegation occurs, with the ability to resume from that checkpoint when the operation completes.

- **Dynamic Workflow Planning**: Allows agents to reprioritize and replan their workflows mid-execution.

- **Observable Control Loop Architecture**: KubeChain uses a simple, observable control loop architecture that allows for easy debugging and observability into agent execution.

- **Scalable**: Leverages Kubernetes for scalability and resilience. If you have k8s / etcd, you can run reliable distributed async agents.

- **Human Approvals and Input**: Support for durable task execution across long-running function calls means a simple tool-based interface to allow an agent to ask a human for input or wait for an approval.

## Design Principles

- **Simplicity**: Leverages the unique property of AI applications where the entire "call stack" can be expressed as the rolling context window accumulated through interactions and tool calls. No separate execution state.

- **Clarity**: Easy to understand what's happening and what the framework is doing with your prompts.

- **Control**: Ability to customize every aspect of agent behavior without framework limitations.

- **Modularity**: Composed of small control loops with limited scope that each progress the state of the world.

- **Durability**: Resilient to failures as a distributed system.

- **Extensibility**: Because agents are YAML, it's easy to build and share agents, tools, and tasks.


## Contributing

KubeChain is open-source and we welcome contributions in the form of issues, documentation, pull requests, and more. See [CONTRIBUTING.md](./CONTRIBUTING.md) for more details.

## License

KubeChain is licensed under the Apache 2 License.
