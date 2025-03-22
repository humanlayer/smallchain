# SmallChain Root Makefile
# Orchestrates commands from kubechain, kubechain-example, and ts directories

# Define directories
KUBECHAIN_DIR = kubechain
EXAMPLE_DIR = kubechain-example
TS_DIR = ts

.PHONY: help build test cluster-up cluster-down build-operator deploy-operator undeploy-operator \
        deploy-samples undeploy-samples deploy-ui deploy-otel undeploy-otel \
        test-operator test-e2e setup-all clean-all \
        kubechain-% example-% ts-%

##@ General Commands

help: ## Display this help information
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Pattern Matching (Run child directory commands directly)

kubechain-%: ## Run any kubechain Makefile target: make kubechain-<target>
	$(MAKE) -C $(KUBECHAIN_DIR) $*

example-%: ## Run any kubechain-example Makefile target: make example-<target>
	$(MAKE) -C $(EXAMPLE_DIR) $*

ts-%: ## Run any ts npm script: make ts-<script>
	npm --prefix $(TS_DIR) run $*

##@ Composite Commands

build: kubechain-build ts-build ## Build both kubechain and ts components

test: kubechain-test ts-test ## Run tests for both kubechain and ts components

##@ Cluster Management

cluster-up: ## Create the Kind cluster
	$(MAKE) -C $(EXAMPLE_DIR) kind-up

cluster-down: ## Delete the Kind cluster
	$(MAKE) -C $(EXAMPLE_DIR) kind-down

##@ Operator Management

build-operator: ## Build the Kubechain operator binary
	$(MAKE) -C $(KUBECHAIN_DIR) build

deploy-operator: ## Deploy the Kubechain operator to the local Kind cluster
	$(MAKE) -C $(KUBECHAIN_DIR) deploy-local-kind

undeploy-operator: ## Undeploy the operator and remove CRDs
	$(MAKE) -C $(KUBECHAIN_DIR) undeploy
	$(MAKE) -C $(KUBECHAIN_DIR) uninstall

##@ Resource Management

deploy-samples: ## Deploy sample resources to the cluster
	$(MAKE) -C $(KUBECHAIN_DIR) deploy-samples

undeploy-samples: ## Remove sample resources
	$(MAKE) -C $(KUBECHAIN_DIR) undeploy-samples

show-samples: ## Show status of sample resources
	$(MAKE) -C $(KUBECHAIN_DIR) show-samples

watch-samples: ## Watch status of sample resources with continuous updates
	$(MAKE) -C $(KUBECHAIN_DIR) watch-samples

##@ UI and Observability

deploy-ui: ## Deploy the Kubechain UI
	$(MAKE) -C $(EXAMPLE_DIR) ui-deploy

deploy-otel: ## Deploy the observability stack (Prometheus, OpenTelemetry, Grafana, Tempo, Loki)
	$(MAKE) -C $(EXAMPLE_DIR) otel-stack

undeploy-otel: ## Remove the observability stack
	$(MAKE) -C $(EXAMPLE_DIR) otel-stack-down

otel-access: ## Display access instructions for monitoring stack
	$(MAKE) -C $(EXAMPLE_DIR) otel-access

##@ Testing

test-operator: ## Run unit tests for the operator
	$(MAKE) -C $(KUBECHAIN_DIR) test

test-e2e: ## Run end-to-end tests (requires a running cluster)
	$(MAKE) -C $(KUBECHAIN_DIR) test-e2e

##@ All-in-One Commands

setup-all: cluster-up deploy-operator deploy-samples deploy-ui deploy-otel ## Set up the entire environment
	@echo "Complete environment setup finished successfully"

clean-all: undeploy-samples undeploy-operator undeploy-otel cluster-down ## Clean up everything
	@echo "Complete environment cleanup finished successfully"

.PHONY: githooks
githooks:
	ln -s hack/git_pre_push.sh .git/hooks/pre-push

