# Loki Single Binary Deployment Plan

Below are the precise changes required to configure Loki for local development in pure single‐binary mode using filesystem storage, thus avoiding the scalable targets (backend, read, write, etc.) that require an object storage backend.

1. In the values file (grafana/loki-values.yaml):
   - Remove (or clear) the custom “config” block that defines schema_config, storage_config, or any scalable component configuration. This prevents the Helm validation template from detecting distributed targets.
   - Set the deployment mode to "single" by adding:
     •  mode: "single"
   - Enable single binary mode by setting:
     •  singleBinary: { enabled: true, replicas: 1 }
   - Explicitly disable scalable targets by adding the following keys under the “loki” block:
     •  backend: { enabled: false }
     •  read: { enabled: false }
     •  write: { enabled: false }
     •  querier: { enabled: false }
     •  queryFrontend: { enabled: false }
     •  ingester: { enabled: false }
     •  distributor: { enabled: false }
     •  (Optionally, also disable “ruler” and “compactor” if present)

2. Keep the basic filesystem storage configuration simple:
   •  storage: { type: filesystem }
   •  persistence: { enabled: true, size: 10Gi }
   •  service: { type: NodePort }

3. Example changes (showing key snippets):

Before (problematic section with scalable target configuration):
---------------------------------------------------------------
loki:
  auth_enabled: false
  commonConfig:
    replication_factor: 1
  storage:
    type: filesystem
    filesystem:
      chunks_directory: /var/loki/chunks
      rules_directory: /var/loki/rules
  config:
    ingester:
      wal:
        enabled: false
    schema_config:
      configs:
        - store: boltdb-shipper
          object_store: filesystem
          schema: v11
          index:
            prefix: index_
            period: 24h
    storage_config:
      boltdb_shipper:
        active_index_directory: /var/loki/index
        cache_location: /var/loki/cache
        shared_store: filesystem
      filesystem:
        directory: /var/loki/chunks
  singleBinary:
    enabled: true
    replicas: 1
  monitoring: …
---------------------------------------------------------------

After (minimal single-binary configuration):
---------------------------------------------------------------
loki:
  auth_enabled: false
  mode: "single"
  commonConfig:
    replication_factor: 1
  storage:
    type: filesystem
  singleBinary:
    enabled: true
    replicas: 1
  backend:
    enabled: false
  read:
    enabled: false
  write:
    enabled: false
  querier:
    enabled: false
  queryFrontend:
    enabled: false
  ingester:
    enabled: false
  distributor:
    enabled: false
  ruler:
    enabled: false
  compactor:
    enabled: false

persistence:
  enabled: true
  size: 10Gi

service:
  type: NodePort
---------------------------------------------------------------

4. Implementation Notes:
   - Remove (or comment out) any “config” block that adds distributed configuration. Rely on the default behavior of the chart when running in “single” mode.
   - Make sure that all scalable target keys (backend, read, write, etc.) are explicitly disabled, so the Helm chart’s validation does not require an object storage backend.

Once these changes have been applied in grafana/loki-values.yaml, run the deploy command (e.g. via “make loki-up”) to confirm that the Helm chart installs Loki without triggering the scalable targets validation error.

This plan ensures a working local single-binary Loki deployment using filesystem storage.
