# /// script
# dependencies = [
#   "requests",
# ]
# ///

#!/usr/bin/env python3
import os
import time
import json
import requests #  type: ignore

# Current time in nanoseconds
current_time_ns = time.time_ns()

# Generate logs
logs_payload = {
    "resourceLogs": [
        {
            "resource": {
                "attributes": [
                    {
                        "key": "service.name",
                        "value": {"stringValue": "curl-test-service"}
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
                            "timeUnixNano": str(current_time_ns),
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

# Generate metrics
metrics_payload = {
    "resourceMetrics": [
        {
            "resource": {
                "attributes": [
                    {
                        "key": "service.name",
                        "value": {"stringValue": "curl-test-service"}
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
                                        "timeUnixNano": str(current_time_ns),
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

# Generate trace
trace_id_hex = os.urandom(16).hex()
span_id_hex = os.urandom(8).hex()

trace_payload = {
    "resourceSpans": [
        {
            "resource": {
                "attributes": [
                    {
                        "key": "service.name",
                        "value": {"stringValue": "python-random-service"}
                    }
                ]
            },
            "scopeSpans": [
                {
                    "scope": {
                        "name": "python-random-scope"
                    },
                    "spans": [
                        {
                            "traceId": trace_id_hex,
                            "spanId": span_id_hex,
                            "name": "python-random-span",
                            "kind": "SPAN_KIND_SERVER",
                            "startTimeUnixNano": str(current_time_ns),
                            "endTimeUnixNano": str(current_time_ns + 30_000_000_000)  # 30 seconds later
                        }
                    ]
                }
            ]
        }
    ]
}

url_base = "http://localhost:4318/v1"
headers = {"Content-Type": "application/json"}

# Send logs
logs_response = requests.post(f"{url_base}/logs", headers=headers, json=logs_payload)
print("Logs Status code:", logs_response.status_code)
print("Logs Response body:", logs_response.text)

# Send metrics
metrics_response = requests.post(f"{url_base}/metrics", headers=headers, json=metrics_payload)
print("Metrics Status code:", metrics_response.status_code)
print("Metrics Response body:", metrics_response.text)

# Send traces
traces_response = requests.post(f"{url_base}/traces", headers=headers, json=trace_payload)
print("Traces Status code:", traces_response.status_code)
print("Traces Response body:", traces_response.text)
print("Generated traceId:", trace_id_hex)
print("Generated spanId: ", span_id_hex)
