# Makefile for kubechain-example

# Cluster name and configuration
CLUSTER_NAME = kubechain-example-cluster
KIND_CONFIG = kind/kind-config.yaml
OTEL_CONFIG = otel/otel-config.yaml

# Environment variable to customize NodePort range (if supported by your setup)
export KIND_NODE_PORT_RANGE = 30000-35000

.PHONY: kind-up kind-down operator-build operator-deploy ui-deploy otel-deploy prometheus-deploy grafana-deploy jaeger-deploy all

kind-up:
	@echo "Creating kind cluster '$(CLUSTER_NAME)'..."
	kind create cluster --name $(CLUSTER_NAME) --config $(KIND_CONFIG)

kind-down:
	@echo "Deleting kind cluster '$(CLUSTER_NAME)'..."
	kind delete cluster --name $(CLUSTER_NAME)

operator-build:
	@echo "Building kubechain operator Docker image..."
	docker build -t humanlayer/kubechain:latest ../kubechain

operator-deploy:
	@echo "Deploying kubechain operator to cluster..."
	kubectl apply -f ../kubechain/config/crd/bases
	kubectl apply -f ../kubechain/config/manager/manager.yaml

ui-deploy:
	@echo "Deploying kubechain-ui to cluster..."
	kubectl apply -f deploy/kubechain-ui.yaml

otel-deploy:
	@echo "Deploying OpenTelemetry collector to cluster..."
	helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
	helm repo update
	helm upgrade --install otel-collector open-telemetry/opentelemetry-collector \
		--values otel/values.yaml

prometheus-deploy:
	@echo "Deploying Prometheus to cluster..."
	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
	helm repo update
	helm install prometheus prometheus-community/prometheus \
		--set server.service.type=NodePort \
		--set server.service.nodePort=9090

grafana-deploy:
	@echo "Deploying Grafana to cluster..."
	helm repo add grafana https://grafana.github.io/helm-charts
	helm repo update
	helm install grafana grafana/grafana \
		--set adminPassword=admin \
		--set service.type=NodePort \
		--set service.nodePort=13000

jaeger-deploy:
	@echo "Deploying Jaeger to cluster..."
	helm repo add jaegertracing https://jaegertracing.github.io/helm-charts
	helm repo update
	helm install jaeger jaegertracing/jaeger \
		--set provisionDataStore.cassandra=false \
		--set provisionDataStore.elastic=false \
		--set query.service.type=NodePort \
		--set query.service.nodePort=16686

otel-stack: otel-deploy prometheus-deploy grafana-deploy jaeger-deploy

otel-access:
	@echo "Access instructions for monitoring stack:"
	@echo "\nGrafana:"
	@echo "Password: admin"
	@echo "Access: http://localhost:13000"
	@echo "\nJaeger:"
	@echo "Access: http://localhost:16686"
	@echo "\nPrometheus:"
	@echo "Access: http://localhost:9090"

all: kind-up operator-build operator-deploy ui-deploy otel-deploy prometheus-deploy grafana-deploy jaeger-deploy
