# KubeChain Development Guide

## Build & Test Commands
- Build: `make build`
- Run controller locally: `make run`
- Lint code: `make lint`
- Fix linting issues: `make lint-fix`
- Run all tests: `make test`
- Run specific test: `go test -v ./internal/controller -run TestName`
- End-to-end tests: `make test-e2e`
- Deploy to local kind cluster: `make deploy-local-kind`
- Deploy samples: `make deploy-samples`
- End-to-end testing: `kubectl delete task,taskrun,agent,llm --all` then `kustomize build samples | kubectl apply -f -`

## Code Style Guidelines
- Follow Go standard formatting (gofmt)
- Use interfaces for external services to enable mocking
- Error handling: Check errors and update status with detailed messages
- Status pattern: Ready (bool), Status (Ready/Error/Pending), StatusDetail (string)
- Controllers are stateless
- Resources start in Pending state until dependencies are ready
- Import order: standard library, external deps, internal packages
- Use the Ginkgo/Gomega testing framework for controller tests
- Test phases with BeforeEach/AfterEach for setup/cleanup

## Architecture Overview
KubeChain is a Kubernetes Operator that orchestrates AI agents in a distributed manner with these core components:

- **LLM**: Model/provider config (OpenAI, Anthropic) with API keys
- **Tool**: Reusable functions for agents (in-process Go, external containers, other agents)
- **ToolSet**: Groups of tools (MCP server, delegation tools, built-ins)
- **Agent**: Combines LLM reference, system prompts, and tool references
- **Task**: User requests targeting an agent with input and metadata
- **TaskRun**: Execution instance with conversation context, tool calls, and results

## TaskRun Phase Progression
Pending → ReadyForLLM → ToolCallsPending/FinalAnswer

## Current Development Status
- TaskRun controller can send messages to LLM
- Working on: Context window management and resource locking
- Next up: TaskRun with tool usage implementation