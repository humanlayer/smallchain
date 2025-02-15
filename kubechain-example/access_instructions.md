REVISION: 1
NOTES:

1. Get your 'admin' user password by running:

   kubectl get secret --namespace otel-demo grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo

2. The Grafana server can be accessed via port 80 on the following DNS name from within your cluster:

   grafana.otel-demo.svc.cluster.local

   Get the Grafana URL to visit by running these commands in the same shell:
   export NODE_PORT=$(kubectl get --namespace otel-demo -o jsonpath="{.spec.ports[0].nodePort}" services grafana)
     export NODE_IP=$(kubectl get nodes --namespace otel-demo -o jsonpath="{.items[0].status.addresses[0].address}")
   echo http://$NODE_IP:$NODE_PORT

   ###################################################################

### IMPORTANT: Ensure that storage is explicitly configured

### Default storage options are subject to change.

###

### IMPORTANT: The use of <component>.env: {...} is deprecated.

### Please use <component>.extraEnv: [] instead.

###################################################################

You can log into the Jaeger Query UI here:

export POD_NAME=$(kubectl get pods --namespace otel-demo -l "app.kubernetes.io/instance=jaeger,app.kubernetes.io/component=query" -o jsonpath="{.items[0].metadata.name}")
echo http://127.0.0.1:8080/
kubectl port-forward --namespace otel-demo $POD_NAME 8080:16686
Let's check if all the pods are running correctly now:

NOTES:
The Prometheus server can be accessed via port 80 on the following DNS name from within your cluster:
prometheus-server.otel-demo.svc.cluster.local

Get the Prometheus server URL by running these commands in the same shell:
export POD_NAME=$(kubectl get pods --namespace otel-demo -l "app.kubernetes.io/name=prometheus,app.kubernetes.io/instance=prometheus" -o jsonpath="{.items[0].metadata.name}")
kubectl --namespace otel-demo port-forward $POD_NAME 9090

The Prometheus alertmanager can be accessed via port 9093 on the following DNS name from within your cluster:
prometheus-alertmanager.otel-demo.svc.cluster.local

Get the Alertmanager URL by running these commands in the same shell:
export POD_NAME=$(kubectl get pods --namespace otel-demo -l "app.kubernetes.io/name=alertmanager,app.kubernetes.io/instance=prometheus" -o jsonpath="{.items[0].metadata.name}")
kubectl --namespace otel-demo port-forward $POD_NAME 9093
#################################################################################

###### WARNING: Pod Security Policy has been disabled by default since

###### it deprecated after k8s 1.25+. use

###### (index .Values "prometheus-node-exporter" "rbac"

###### . "pspEnabled") with (index .Values

###### "prometheus-node-exporter" "rbac" "pspAnnotations")

###### in case you still need it.

#################################################################################

The Prometheus PushGateway can be accessed via port 9091 on the following DNS name from within your cluster:
prometheus-prometheus-pushgateway.otel-demo.svc.cluster.local

Get the PushGateway URL by running these commands in the same shell:
export POD_NAME=$(kubectl get pods --namespace otel-demo -l "app=prometheus-pushgateway,component=pushgateway" -o jsonpath="{.items[0].metadata.name}")
kubectl --namespace otel-demo port-forward $POD_NAME 9091

For more information on running Prometheus, visit:
https://prometheus.io/
Deploying Grafana to cluster...
helm repo add grafana https://grafana.github.io/helm-charts
