# Makefile for kubechain-example

# Cluster name and configuration
CLUSTER_NAME = kubechain-example-cluster
KIND_CONFIG = kind/kind-config.yaml
OTEL_CONFIG = otel/otel-config.yaml

# Environment variable to customize NodePort range (if supported by your setup)
export KIND_NODE_PORT_RANGE = 30000-35000

.PHONY: kind-up kind-down operator-build operator-deploy ui-deploy otel-deploy prometheus-deploy grafana-deploy tempo-deploy all

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

otel-deploy-base:
	@echo "Deploying OpenTelemetry collector to cluster..."
	helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
	helm repo update
	helm upgrade --install otel-collector open-telemetry/opentelemetry-collector \
		--values otel/values.yaml 

otel-patch-service:
	@echo "Patching OTEL collector service to use NodePorts..."
	kubectl patch svc otel-collector-opentelemetry-collector -p '{"spec": {"type": "NodePort", "ports": [ \
		{"name": "jaeger-compact", "port": 6831, "protocol": "UDP", "targetPort": 6831, "nodePort": 6831}, \
		{"name": "jaeger-grpc", "port": 14250, "protocol": "TCP", "targetPort": 14250, "nodePort": 14250}, \
		{"name": "jaeger-thrift", "port": 14268, "protocol": "TCP", "targetPort": 14268, "nodePort": 14268}, \
		{"name": "otlp", "port": 4317, "protocol": "TCP", "targetPort": 4317, "nodePort": 4317}, \
		{"name": "otlp-http", "port": 4318, "protocol": "TCP", "targetPort": 4318, "nodePort": 4318}, \
		{"name": "zipkin", "port": 9411, "protocol": "TCP", "targetPort": 9411, "nodePort": 9411} \
	]}}'

otel-up: otel-deploy-base otel-patch-service 

otel-test:
	curl -v -X POST \
		-H "Content-Type: application/json" \
		-d @otel/otel-test-logs.json \
		http://localhost:4318/v1/logs
	curl -v -X POST \
		-H "Content-Type: application/json" \
		-d @otel/otel-test-metrics.json \
		http://localhost:4318/v1/metrics
	uv run otel/otel-test-traces.py

	
otel-down:
	@echo "Deleting OpenTelemetry collector from cluster..."
	helm delete otel-collector

prometheus-up:
	@echo "Deploying Prometheus to cluster..."
	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
	helm repo update
	helm upgrade --install prometheus prometheus-community/prometheus \
		--set server.service.type=NodePort \
		--set server.service.nodePort=9090

prometheus-down:
	@echo "Deleting Prometheus from cluster..."
	helm delete prometheus

grafana-up:
	@echo "Deploying Grafana to cluster..."
	kubectl apply -f grafana/config.yaml
	kubectl apply -f grafana/datasources.yaml
	helm repo add grafana https://grafana.github.io/helm-charts
	helm repo update
	helm upgrade --install grafana grafana/grafana \
		--values grafana/values.yaml

grafana-down:
	@echo "Deleting Grafana from cluster..."
	helm delete grafana

jaeger-up:
	@echo "Deploying Jaeger to cluster..."
	helm repo add jaegertracing https://jaegertracing.github.io/helm-charts
	helm repo update
	helm upgrade --install jaeger jaegertracing/jaeger \
		--history-max 3 \
		--values jaeger/jaeger.yaml

jaeger-down:
	@echo "Deleting Jaeger from cluster..."
	helm delete jaeger

tempo-up:
	@echo "Deploying Tempo to cluster..."
	helm repo add grafana https://grafana.github.io/helm-charts
	helm repo update
	helm upgrade --install tempo grafana/tempo \
		--set tempo.receivers.otlp.protocols.grpc.endpoint=0.0.0.0:4317 \
		--set tempo.receivers.otlp.protocols.http.endpoint=0.0.0.0:4318 \
		--set service.type=NodePort \
		--set service.ports.otlp-grpc.nodePort=4317 \
		--set service.ports.otlp-http.nodePort=4318

tempo-down:
	@echo "Deleting Tempo from cluster..."
	helm delete tempo

otel-stack: otel-up prometheus-up grafana-up tempo-up
otel-stack-down: otel-down prometheus-down grafana-down tempo-down

otel-access:
	@echo "Access instructions for monitoring stack:"
	@echo "\nGrafana:"
	@echo "Password: admin"
	@echo "Access: http://localhost:13000"
	@echo "\nJaeger:"
	@echo "Access: http://localhost:16686"
	@echo "\nPrometheus:"
	@echo "Access: http://localhost:9090"

all: kind-up operator-build operator-deploy ui-deploy otel-deploy prometheus-deploy grafana-deploy tempo-deploy
