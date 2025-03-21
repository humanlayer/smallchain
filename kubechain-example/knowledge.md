# Best Practices

- Never use `cat <<EOF` for creating resources. Instead, create proper YAML files and use `kubectl apply -f`.
- Use proper version control for configuration files.
- Use `uv run` to execute Python scripts - it automatically handles dependencies without requiring manual package installation.
- Tempo TraceQL queries should use `resource.service.name` for service name filtering, e.g. `{resource.service.name="my-service"}`
- Traces need parent-child relationships between spans to visualize properly in Tempo's trace view. Each child span should include a `parentSpanId` that matches a parent span's `spanId` within the same `traceId`.
