# Example Application Resources

This document describes the sample resources provided in the `config/samples` directory and explains how they match our implemented CRDs for the example application.

Deploy them with 

```
kustomize build samples | kubectl apply -f -
```

---

## MCPServer Resource with Secret References

[./samples/kubechain_v1alpha1_mcpserver_with_secrets.yaml](./samples/kubechain_v1alpha1_mcpserver_with_secrets.yaml)

**Resource:** `MCPServer`  
**API Version:** `kubechain.humanlayer.dev/v1alpha1`  
**Kind:** `MCPServer`

**Sample File:** `config/samples/kubechain_v1alpha1_mcpserver_with_secrets.yaml`

**Key Fields:**

- **transport:** The connection type (e.g., `"stdio"`)
- **command:** The command to run for stdio MCP servers
- **args:** Arguments to pass to the command
- **env:** Environment variables to set for the server
  - Can include direct values:
    ```yaml
    - name: DIRECT_VALUE
      value: "some-direct-value"
    ```
  - Can reference secrets:
    ```yaml
    - name: SECRET_VALUE
      valueFrom:
        secretKeyRef:
          name: mcp-credentials
          key: api-key
    ```

**Required Secret:**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: mcp-credentials
  namespace: default
type: Opaque
data:
  api-key: c2VjcmV0LWFwaS1rZXktdmFsdWU=  # base64 encoded value of "secret-api-key-value"
```

---

## LLM

[./samples/kubechain_v1alpha1_llm.yaml](./samples/kubechain_v1alpha1_llm.yaml)

**Resource:** `LLM`  
**API Version:** `kubechain.humanlayer.dev/v1alpha1`  
**Kind:** `LLM`

**Sample File:** `config/samples/kubechain_v1alpha1_llm.yaml`

**Key Fields:**

- **provider:** e.g. `"openai"`
- **apiKeyFrom:**
  - References a secret (e.g. secret name: `openai`)
  - Key: e.g. `OPENAI_API_KEY`
- **maxTokens:** e.g. `1000`

_Note:_ Ensure that the referenced secret exists (for example, create a secret named `openai` with the appropriate API key).

---

## Agent

[./samples/kubechain_v1alpha1_agent.yaml](./samples/kubechain_v1alpha1_agent.yaml)

**Resource:** `Agent`  
**API Version:** `kubechain.humanlayer.dev/v1alpha1`  
**Kind:** `Agent`

**Sample File:** `config/samples/kubechain_v1alpha1_agent.yaml`

**Key Fields:**

- **llmRef:**
  - Must refer to the LLM resource (e.g. `gpt-4o`)
- **tools:**
  - A list of tool references (e.g. one tool with name `"add"`)
- **system:**
  - A system prompt (e.g. instructions for a calculator agent)

---

## Tool

[./samples/kubechain_v1alpha1_tool.yaml](./samples/kubechain_v1alpha1_tool.yaml)

**Resource:** `Tool`  
**API Version:** `kubechain.humanlayer.dev/v1alpha1`  
**Kind:** `Tool`

**Sample File:** `config/samples/kubechain_v1alpha1_tool.yaml`

**Key Fields:**

- **toolType:** e.g. `"function"`
- **name:** e.g. `"add"`
- **description:** A short description (e.g. “Add two numbers”)
- **arguments:**
  - A JSON schema defining the expected input arguments. For instance, properties “a” and “b” of type number.
- **execute:**
  - Configuration for how the tool is executed (e.g. use a builtin function called `"add"`)

---

## Task

[./samples/kubechain_v1alpha1_task.yaml](./samples/kubechain_v1alpha1_task.yaml)

**Resource:** `Task`  
**API Version:** `kubechain.humanlayer.dev/v1alpha1`  
**Kind:** `Task`

**Sample File:** `config/samples/kubechain_v1alpha1_task.yaml`

**Key Fields:**

- **agentRef:**
  - References an existing Agent (e.g. `"calculator-agent"`)
- **message:**
  - The task prompt or request (e.g. `"What is 2 + 2?"`)

_Note:_ The Task controller automatically launches a TaskRun resource when a Task is created. (The TaskRun's execution is managed by the TaskRun controller that builds the conversation context and handles tool calls.)

---

## Additional Notes

- **Secrets:** Make sure the secret referenced by the LLM (e.g. secret `openai` with key `OPENAI_API_KEY`) is created in your cluster.
- **CRDs & Controllers:** Before applying these sample files, ensure that the CRDs are installed (use `make manifests install`) and that the controllers are deployed (`make deploy`).
- **Auto-Launching TaskRuns:** When a Task is created (as shown in the Task sample), the Task controller automatically creates a corresponding TaskRun (with a name like `<task-name>-run-1`). This TaskRun then “executes” the task by invoking the associated agent and tool.

These sample files now match our example application design, where:

- An LLM (`gpt-4o`) is defined with the required API key reference.
- A Calculator Agent (`calculator-agent`) uses that LLM and has a system prompt suited for mathematical operations.
- A Tool (`add`) is implemented to perform addition.
- A Task (`calculate-sum`) uses the Agent to process an arithmetic question.

Ensure that your cluster includes the necessary prerequisites (such as the secret for the LLM) so that the status fields eventually show “ready” once the controllers have reconciled the objects.

Happy deploying!
