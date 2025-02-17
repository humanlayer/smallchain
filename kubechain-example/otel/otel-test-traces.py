# /// script
# dependencies = [
#   "requests",
# ]
# ///

#!/usr/bin/env python3
import os
import time
import requests #  type: ignore

# Generate a 16-byte (128-bit) random trace ID, then hex-encode to 32 hex chars
trace_id_hex = os.urandom(16).hex()  # e.g. "1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d"
# Generate an 8-byte (64-bit) random span ID, then hex-encode to 16 hex chars
span_id_hex = os.urandom(8).hex()    # e.g. "1a2b3c4d5e6f7a8b"

# Current time in nanoseconds
start_time_nano = time.time_ns()
end_time_nano = start_time_nano + 1_000_000_000  # 1 second later

# Build minimal OTLP/HTTP JSON for one span
payload = {
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
                            "startTimeUnixNano": str(start_time_nano),
                            "endTimeUnixNano": str(end_time_nano)
                        }
                    ]
                }
            ]
        }
    ]
}

url = "http://localhost:4318/v1/traces"
headers = {"Content-Type": "application/json"}

response = requests.post(url, headers=headers, json=payload)
print("Status code:", response.status_code)
print("Response body:", response.text)
print("Generated traceId:", trace_id_hex)
print("Generated spanId: ", span_id_hex)
