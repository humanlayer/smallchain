# Makefile for kubechain-example

# Cluster name and configuration
CLUSTER_NAME = kubechain-example-cluster
KIND_CONFIG = kind/kind-config.yaml
OTEL_CONFIG = otel/otel-config.yaml

# Environment variable to customize NodePort range (if supported by your setup)
export KIND_NODE_PORT_RANGE = 30000-35000

.PHONY: kind-up kind-down operator-build operator-deploy ui-deploy otel-deploy all

kind-up:
	@echo "Creating kind cluster '$(CLUSTER_NAME)'..."
	kind create cluster --name $(CLUSTER_NAME) --config $(KIND_CONFIG)

kind-down:
	@echo "Deleting kind cluster '$(CLUSTER_NAME)'..."
	kind delete cluster --name $(CLUSTER_NAME)

operator-build:
	@echo "Building kubechain operator Docker image..."
	# Adjust the Docker build context if necessary.
	docker build -t humanlayer/kubechain:latest ../kubechain

operator-deploy:
	@echo "Deploying kubechain operator to cluster..."
	# Apply the CRDs and operator deployment manifests.
	kubectl apply -f ../kubechain/config/crd/bases
	kubectl apply -f ../kubechain/config/manager/manager.yaml

ui-deploy:
	@echo "Deploying kubechain-ui to cluster..."
	kubectl apply -f deploy/kubechain-ui.yaml

otel-deploy:
	@echo "Deploying OpenTelemetry collector to cluster..."
	kubectl apply -f $(OTEL_CONFIG)

all: kind-up operator-build operator-deploy ui-deploy otel-deploy
