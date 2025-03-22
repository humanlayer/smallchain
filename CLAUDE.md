# SmallChain Development Guide

## Root-Level Makefile Commands

The root-level Makefile provides convenient commands for managing the entire project without having to change directories. You can run all commands from the project root.

### Pattern-Matching Commands
- `make kubechain-<command>`: Run any target from the kubechain Makefile (e.g., `make kubechain-fmt`)
- `make example-<command>`: Run any target from the kubechain-example Makefile (e.g., `make example-kind-up`)
- `make ts-<command>`: Run any npm script from the ts directory (e.g., `make ts-build`)

### Composite Commands
- `make build`: Build both kubechain and ts components
- `make test`: Run tests for both kubechain and ts components

### Cluster Management
- `make cluster-up`: Create the Kind cluster
- `make cluster-down`: Delete the Kind cluster

### Operator Management
- `make build-operator`: Build the Kubechain operator binary
- `make deploy-operator`: Deploy the Kubechain operator to the local Kind cluster
- `make undeploy-operator`: Undeploy the operator and remove CRDs

### Resource Management
- `make deploy-samples`: Deploy sample resources to the cluster
- `make undeploy-samples`: Remove sample resources
- `make show-samples`: Show status of sample resources
- `make watch-samples`: Watch status of sample resources with continuous updates

### UI and Observability
- `make deploy-ui`: Deploy the Kubechain UI
- `make deploy-otel`: Deploy the observability stack
- `make undeploy-otel`: Remove the observability stack
- `make otel-access`: Display access instructions for monitoring stack

### Testing
- `make test-operator`: Run unit tests for the operator
- `make test-e2e`: Run end-to-end tests (requires a running cluster)

### All-in-One Commands
- `make setup-all`: Set up the entire environment (cluster, operator, samples, UI, observability)
- `make clean-all`: Clean up everything (samples, operator, observability, cluster)

### Help
- `make help`: Display all available commands with descriptions

## Go (Kubechain) Commands

You can run these commands directly in the kubechain directory or use the pattern-matching syntax from the root:

- Build: `cd kubechain && make build` or `make kubechain-build`
- Format: `cd kubechain && make fmt` or `make kubechain-fmt`
- Lint: `cd kubechain && make lint` or `make kubechain-lint`
- Run tests: `cd kubechain && make test` or `make kubechain-test`
- Run single test: `cd kubechain && go test -v ./internal/controller/llm -run TestLLMController`
- Run e2e tests: `cd kubechain && make test-e2e` or `make kubechain-test-e2e`

## TypeScript Commands

You can run these commands directly in the ts directory or use the pattern-matching syntax from the root:

- Build: `cd ts && npm run build` or `make ts-build`
- Dev mode: `cd ts && npm run dev` or `make ts-dev`
- Run tests: `cd ts && npm test` or `make ts-test`
- Run single test: `cd ts && npm test -- -t "ChainService constructor"`

## Makefiles Overview

### Main Kubechain Makefile (/kubechain/Makefile)

#### Development Commands
- `make fmt`: Format Go code
- `make vet`: Run Go vet
- `make lint`: Run golangci-lint on code
- `make lint-fix`: Run golangci-lint and fix issues
- `make test`: Run unit tests
- `make test-e2e`: Run end-to-end tests (requires a running Kind cluster)
- `make manifests`: Generate Kubernetes manifests (CRDs, RBAC) - **Important:** Run this after modifying CRD types or controller RBAC annotations
- `make generate`: Generate Go code (DeepCopy methods) - **Important:** Run this after adding new struct fields

#### Build Commands
- `make build`: Build the manager binary
- `make run`: Run the controller locally
- `make docker-build`: Build the controller Docker image
- `make docker-push`: Push the controller Docker image
- `make docker-load-kind`: Load Docker image into Kind cluster
- `make build-installer`: Generate install.yaml in the dist directory

#### Deployment Commands
- `make install`: Install CRDs into cluster
- `make uninstall`: Uninstall CRDs from cluster
- `make deploy`: Deploy controller to cluster (builds and pushes image)
- `make deploy-local-kind`: Deploy controller to local Kind cluster
- `make deploy-samples`: Deploy sample resources
- `make undeploy-samples`: Remove sample resources
- `make undeploy`: Undeploy controller
- `make show-samples`: Show status of sample resources
- `make watch-samples`: Watch status of sample resources with continuous updates

### Kubechain Example Makefile (/kubechain-example/Makefile)

#### Cluster Management
- `make kind-up`: Create Kind cluster
- `make kind-down`: Delete Kind cluster

#### Application Deployment
- `make operator-build`: Build Kubechain operator Docker image
- `make operator-deploy`: Deploy Kubechain operator to cluster
- `make ui-deploy`: Deploy Kubechain UI

#### Observability Stack
- `make otel-stack`: Deploy full observability stack (Prometheus, OpenTelemetry, Grafana, Tempo, Loki)
- `make otel-stack-down`: Remove observability stack
- `make otel-test`: Run test to generate OpenTelemetry data
- `make otel-access`: Display access instructions for monitoring stack

Individual components can be managed separately:
- `make prometheus-up/down`
- `make grafana-up/down`
- `make otel-up/down`
- `make tempo-up/down`
- `make loki-up/down`

## Documentation

The project includes detailed documentation in the `/kubechain/docs/` directory:

- [MCP Server Guide](/kubechain/docs/mcp-server.md) - Working with Model Control Protocol servers
- [CRD Reference](/kubechain/docs/crd-reference.md) - Complete reference for all Custom Resource Definitions
- [Kubebuilder Guide](/kubechain/docs/kubebuilder-guide.md) - How to develop with Kubebuilder in this project

## Typical Workflow

### Local Development with Kind Cluster

#### Using Root Makefile (Recommended)
1. Create local Kubernetes cluster: `make cluster-up`
2. Deploy the operator (includes installing CRDs): `make deploy-operator`
3. Deploy observability stack: `make deploy-otel`
4. Deploy sample resources: `make deploy-samples`
5. Watch resources: `make watch-samples`

Alternatively, use: `make setup-all` to perform steps 1-4 in a single command.

#### Using Directory Makefiles (Alternative)
1. Create local Kubernetes cluster: `cd kubechain-example && make kind-up`
2. Install CRDs: `cd kubechain && make install`
3. Build and deploy the controller: `cd kubechain && make deploy-local-kind`
4. Deploy observability stack: `cd kubechain-example && make otel-stack`
5. Deploy sample resources: `cd kubechain && make deploy-samples`
6. Watch resources: `cd kubechain && make watch-samples`

### Clean Up

#### Using Root Makefile (Recommended)
1. Clean up everything: `make clean-all`

Alternatively, clean up components individually:
1. Remove sample resources: `make undeploy-samples`
2. Undeploy the operator: `make undeploy-operator`
3. Remove observability stack: `make undeploy-otel`
4. Delete cluster: `make cluster-down`

#### Using Directory Makefiles (Alternative)
1. Remove sample resources: `cd kubechain && make undeploy-samples`
2. Undeploy the controller: `cd kubechain && make undeploy`
3. Remove observability stack: `cd kubechain-example && make otel-stack-down`
4. Delete cluster: `cd kubechain-example && make kind-down`

## Code Style Guidelines
### Go
- Follow standard Go code style with `gofmt`
- Use meaningful error handling with context
- Use dependency injection for controllers
- Test with Ginkgo/Gomega framework
- Document public functions with godoc

### Kubebuilder and CRD Development
- All resources should be in the `kubechain.humanlayer.dev` API group
- Use proper kubebuilder annotations for validation and RBAC
- Add RBAC annotations to all controllers to generate proper permissions
- Run `make manifests` after modifying CRD types or controller annotations
- Run `make generate` after adding new struct fields to generate DeepCopy methods
- When creating new resources, use `kubebuilder create api --group kubechain --version v1alpha1 --kind YourResource --namespaced true --resource true --controller true`
- Ensure the PROJECT file contains entries for all resources before running `make manifests`
- Follow the detailed guidance in the [Kubebuilder Guide](/kubechain/docs/kubebuilder-guide.md)

### TypeScript
- Use 2-space indentation
- No semicolons (per prettier config)
- Double quotes for strings
- Strong typing with TypeScript interfaces
- Use ES6+ features (arrow functions, destructuring)
- Jest for testing

### General
- Descriptive variable/function names (camelCase in TS, CamelCase for exported Go)
- Use consistent error handling patterns within each language
- Add tests for new functionality
- Keep functions small and focused


### Markdown
- When writing markdown code blocks, do not indent the block, just use backticks to offset the code