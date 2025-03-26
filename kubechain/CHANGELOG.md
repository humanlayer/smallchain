### v0.1.12 (March 24, 2025)

Features:
- Added OpenTelemetry tracing support
  - Spans for LLM requests with context window and tool metrics
  - Parent spans for TaskRun lifecycle tracking
  - Completion spans for terminal states
  - Status and error propagation to spans

Changes:
- Refactored TaskRun phase transitions and improved phase transition logging
- Enhanced testing infrastructure
  - Improved TaskRun and TaskRunToolCall test suites
  - Added test utilities for common setup patterns

### v0.1.11 (March 24, 2025)

Features:
- Added support for contact channels with Slack and email integration
  - New ContactChannel CRD with validation fields, printer columns, and status tracking
  - Support for API key authentication
  - Email message customization options
  - Channel configuration validation

Fixes:
- Updated MCPServer CRD to support approval channels for tool execution

### v0.1.10 (March 24, 2025)

Features:
- Added MCP (Model Control Protocol) server support
  - New MCPServer CRD for tool execution
  - Support for stdio and http transport protocols
  - Tool discovery and validation
  - Resource configuration options
- Enhanced task run statuses and tracking
- Improved agent validation for MCP server access
- Added status details fields across CRDs for better observability

Infrastructure:
- Increased resource limits for controller
  - CPU: 1000m (up from 500m)
  - Memory: 512Mi (up from 128Mi)
- Updated base resource requests
  - CPU: 100m (up from 10m)
  - Memory: 256Mi (up from 64Mi)
