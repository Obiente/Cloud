# Traefik Request Tracking Middleware

This guide explains how to configure Traefik middleware to track HTTP requests and errors for deployment metrics.

## Overview

To track HTTP requests and errors per deployment, we can use Traefik's built-in features:
1. **Access Logs**: Track all HTTP requests with response codes
2. **Metrics (Prometheus)**: Expose request counts and response codes
3. **Custom Middleware**: Optionally add custom headers or tracking

## Configuration Options

### Option 1: Using Traefik Access Logs (Recommended)

Traefik access logs already contain request information. The API can parse these logs to extract:
- Request count per deployment
- HTTP status codes (errors)
- Response times
- Bandwidth (request/response sizes)

#### Enable Access Logs

In your Traefik configuration:

```yaml
# docker-compose.yml or traefik.yml
traefik:
  command:
    - --accesslog=true
    - --accesslog.format=json  # JSON format for easier parsing
    - --accesslog.filepath=/var/log/traefik/access.log
```

Or via Docker labels:

```yaml
labels:
  - "traefik.accesslog=true"
  - "traefik.accesslog.format=json"
  - "traefik.accesslog.filepath=/var/log/traefik/access.log"
```

#### Access Log Format

JSON format includes:
```json
{
  "ClientAddr": "192.168.1.1:12345",
  "ClientHost": "192.168.1.1",
  "ClientPort": "12345",
  "ClientUsername": "-",
  "DownstreamAddr": "10.0.0.1:8000",
  "DownstreamStatus": 200,
  "Duration": 12345678,
  "FrontendName": "deployment-abc123",
  "OriginContentSize": 1024,
  "OriginDuration": 10000000,
  "OriginStatus": 200,
  "RequestAddr": "/api/users",
  "RequestHost": "app.example.com",
  "RequestMethod": "GET",
  "RequestPath": "/api/users",
  "RequestPort": "-",
  "RequestProtocol": "HTTP/1.1",
  "RequestScheme": "https",
  "RetryAttempts": 0,
  "RouterName": "deployment-abc123",
  "StartLocal": "2024-01-01T12:00:00Z",
  "StartUTC": "2024-01-01T12:00:00Z"
}
```

### Option 2: Using Traefik Prometheus Metrics

Enable Prometheus metrics and scrape them:

```yaml
traefik:
  command:
    - --metrics.prometheus=true
    - --metrics.prometheus.entryPoint=websecure
```

Access metrics at `http://traefik:8080/metrics` (or configure Prometheus scraping).

Key metrics:
- `traefik_http_requests_total{code="200",method="GET",service="deployment-abc123"}`
- `traefik_http_request_duration_seconds_sum{service="deployment-abc123"}`

### Option 3: Custom Middleware for Request Tracking

Create a custom Traefik middleware plugin or use Traefik's built-in plugins to add custom tracking.

#### Example: Plugin Configuration (Go Plugin)

```go
// traefik-request-tracker/main.go
package main

import (
    "net/http"
    "context"
)

type Config struct {
    APIEndpoint string `json:"apiEndpoint,omitempty"`
}

func CreateConfig() *Config {
    return &Config{
        APIEndpoint: "http://api:3001/api/v1/metrics/request",
    }
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
    return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
        // Track request before forwarding
        trackRequest(req)
        
        // Call next handler
        next.ServeHTTP(rw, req)
        
        // Track response
        trackResponse(rw)
    }), nil
}
```

#### Using Traefik Plugin Provider

1. Build the plugin
2. Add to Traefik:

```yaml
traefik:
  command:
    - --experimental.plugins.requesttracker.modulename=github.com/obiente/traefik-request-tracker
    - --experimental.plugins.requesttracker.version=v1.0.0
```

3. Apply middleware via labels:

```yaml
labels:
  - "traefik.http.middlewares.request-tracker.plugin.requesttracker.apiendpoint=http://api:3001/api/v1/metrics/request"
  - "traefik.http.routers.deployment-abc123.middlewares=request-tracker"
```

### Option 4: Simple Header-Based Tracking

Use Traefik's built-in middleware to add custom headers that can be logged:

```yaml
labels:
  - "traefik.http.middlewares.add-deployment-id.headers.customrequestheaders.X-Deployment-ID=deployment-abc123"
  - "traefik.http.routers.deployment-abc123.middlewares=add-deployment-id"
```

Then parse access logs looking for `X-Deployment-ID` header.

## Implementation Recommendation

For the Obiente Cloud platform, we recommend:

1. **Enable Traefik access logs in JSON format** for all deployments
2. **Create a log parser service** that:
   - Reads Traefik access logs
   - Extracts deployment ID from router/service names
   - Aggregates request counts and errors
   - Updates `DeploymentMetrics` table in the database
3. **Use Prometheus metrics as a backup** for real-time monitoring

### Log Parser Service Integration

The orchestrator service can be extended to:
- Watch Traefik access log file
- Parse JSON entries
- Extract deployment metrics:
  - `request_count`: Count of requests (group by status code)
  - `error_count`: Count of 4xx/5xx responses
  - `bandwidth_rx_bytes`: Sum of request sizes
  - `bandwidth_tx_bytes`: Sum of response sizes

### Example Log Parser

```go
// apps/api/internal/orchestrator/log_parser.go
package orchestrator

import (
    "bufio"
    "encoding/json"
    "os"
    "strings"
)

type AccessLogEntry struct {
    RouterName      string `json:"RouterName"`
    DownstreamStatus int   `json:"DownstreamStatus"`
    OriginContentSize int64 `json:"OriginContentSize"`
    RequestContentSize int64 `json:"RequestContentSize"`
}

func (os *OrchestratorService) parseTraefikAccessLog(logPath string) {
    file, err := os.Open(logPath)
    if err != nil {
        return
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        var entry AccessLogEntry
        if err := json.Unmarshal([]byte(line), &entry); err != nil {
            continue
        }

        // Extract deployment ID from RouterName (e.g., "deployment-abc123")
        deploymentID := extractDeploymentID(entry.RouterName)
        if deploymentID == "" {
            continue
        }

        // Update metrics
        os.updateRequestMetrics(deploymentID, entry)
    }
}

func extractDeploymentID(routerName string) string {
    // Router names typically follow pattern: deployment-{id} or deployment-{id}-{service}
    if strings.HasPrefix(routerName, "deployment-") {
        parts := strings.Split(routerName, "-")
        if len(parts) >= 2 {
            return strings.Join(parts[1:], "-") // Handle IDs with dashes
        }
    }
    return ""
}
```

## Traefik Label Configuration for Deployments

When creating deployments, ensure Traefik labels include identifiable router names:

```go
labels["traefik.http.routers."+deploymentID+".rule"] = "Host(`app.example.com`)"
labels["traefik.http.routers."+deploymentID+".entrypoints"] = "websecure"
labels["traefik.http.routers."+deploymentID+".tls.certresolver"] = "letsencrypt"
```

This allows the log parser to identify which deployment each request belongs to.

## Next Steps

1. Enable access logs in Traefik configuration
2. Implement log parser in orchestrator service
3. Update `DeploymentMetrics` to include request/error counts from logs
4. Optionally integrate Prometheus scraping for additional metrics

