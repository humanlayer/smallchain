#!/bin/bash
TIMESTAMP=$(date -u +%s%N)  # Get current UTC time in nanoseconds

# Generate logs
cat > otel-test-logs.json << EOF
{
  "resourceLogs": [
    {
      "resource": {
        "attributes": [
          {
            "key": "service.name",
            "value": {
              "stringValue": "curl-test-service"
            }
          }
        ]
      },
      "scopeLogs": [
        {
          "scope": {
            "name": "curl-test-scope"
          },
          "logRecords": [
            {
              "timeUnixNano": "$TIMESTAMP",
              "severityNumber": 9,
              "severityText": "INFO",
              "body": {
                "stringValue": "Hello from curl!"
              },
              "attributes": [
                {
                  "key": "service.name",
                  "value": {
                    "stringValue": "curl-test-service"
                  }
                }
              ]
            }
          ]
        }
      ]
    }
  ]
}
EOF

# Generate metrics
cat > otel-test-metrics.json << EOF
{
  "resourceMetrics": [
    {
      "resource": {
        "attributes": [
          {
            "key": "service.name",
            "value": { "stringValue": "curl-test-service" }
          }
        ]
      },
      "scopeMetrics": [
        {
          "scope": {
            "name": "curl-test-scope"
          },
          "metrics": [
            {
              "name": "curl.test.metric",
              "description": "Demo metric from curl",
              "unit": "1",
              "gauge": {
                "dataPoints": [
                  {
                    "timeUnixNano": "$TIMESTAMP",
                    "asInt": 42
                  }
                ]
              }
            }
          ]
        }
      ]
    }
  ]
}
EOF
