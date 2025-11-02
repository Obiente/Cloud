# Monitoring

Obiente Cloud provides built-in monitoring capabilities and supports integration with external monitoring tools like Prometheus and Grafana.

## Built-in Observability

### Health Check Endpoint

The API provides a health check endpoint that includes metrics system status:

```bash
curl http://localhost:3001/health
```

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "metrics_healthy": true
}
```

If metrics collection is unhealthy, the endpoint returns `503 Service Unavailable` with details about consecutive failures.

### Metrics Observability Endpoint

Real-time statistics about the metrics collection system:

```bash
curl http://localhost:3001/metrics/observability
```

**Response includes:**
- Collection rates and error counts
- Container processing statistics
- Database write success/failure rates
- Retry queue status
- Subscriber counts and backpressure metrics
- Circuit breaker state
- Health status and consecutive failures
- Cache sizes
- Last collection/storage/health check times

**Example:**
```json
{
  "collection_count": 15234,
  "collection_errors": 12,
  "collections_per_second": 0.2,
  "containers_processed": 45678,
  "containers_failed": 45,
  "storage_batches_written": 254,
  "storage_batches_failed": 2,
  "storage_metrics_written": 25400,
  "active_subscribers": 3,
  "circuit_breaker_state": 0,
  "healthy": true,
  "consecutive_failures": 0
}
```

**Circuit Breaker States:**
- `0` = Closed (normal operation)
- `1` = Open (too many failures, blocking requests)
- `2` = Half-Open (testing if service recovered)

## Metrics System Architecture

### Collection Flow

1. **Parallel Collection**: Metrics streamer collects container stats in parallel (configurable workers, default: 50)
2. **Live Cache**: Recent metrics stored in memory (5-minute retention, configurable)
3. **Streaming**: Real-time metrics delivered to UI subscribers via channels
4. **Batch Storage**: Aggregated metrics written to TimescaleDB every 60 seconds

### Resilience Features

- **Circuit Breaker**: Protects Docker API from cascade failures
- **Retry Mechanism**: Exponential backoff for failed Docker API calls
- **Retry Queue**: Failed database writes automatically retried
- **Graceful Degradation**: Collection rate reduced under load
- **Health Monitoring**: Automatic detection of collection lagging or failures
- **Backpressure Handling**: Detects and cleans up slow/dead subscribers

## External Monitoring (Optional)

### Prometheus Integration

Prometheus can scrape metrics from the API and other services.

**Configuration Example:**
```yaml
scrape_configs:
  - job_name: 'obiente-api'
    static_configs:
      - targets: ['api:3001']
    metrics_path: '/metrics/prometheus'
    
  - job_name: 'obiente-containers'
    static_configs:
      - targets: ['node1:9323', 'node2:9323']
```

### Grafana Dashboards

Create dashboards to visualize:

**Cluster Metrics:**
- Node CPU and memory usage
- Deployment count per node
- Network throughput
- Disk I/O

**Deployment Metrics:**
- Total deployments
- Resource usage per deployment
- Request latency
- Error rates

**Platform Metrics:**
- API response times
- Database query performance
- Cache hit rates
- Metrics collection health

### Key Metrics to Monitor

#### System Health
- `/health` endpoint status (should always return 200)
- Metrics collection consecutive failures (should be 0)
- Circuit breaker state (should be Closed/0)
- Database connection pool utilization

#### Performance
- Collection rate (collections per second)
- Database write success rate
- Average container processing time
- Retry queue size (should stay low)

#### Capacity
- Live metrics cache size
- Previous stats cache size
- Active subscribers count
- Storage batch sizes

## Alerting

Set up alerts for:

1. **Metrics Collection Failures**
   - Alert if `consecutive_failures` > 3
   - Alert if `collection_errors` rate > 10%

2. **Circuit Breaker Open**
   - Alert if `circuit_breaker_state` == 1 (Open)
   - Indicates Docker API issues

3. **Database Write Failures**
   - Alert if `storage_batches_failed` rate > 5%
   - Check retry queue size

4. **High Backpressure**
   - Alert if `subscriber_overflows` increasing
   - May indicate slow subscribers

## Troubleshooting

### Metrics Collection Not Working

```bash
# Check metrics streamer health
curl http://localhost:3001/metrics/observability | jq '.healthy'

# Check circuit breaker state
curl http://localhost:3001/metrics/observability | jq '.circuit_breaker_state'

# Check error counts
curl http://localhost:3001/metrics/observability | jq '.collection_errors'
```

### High Error Rates

1. Check Docker API connectivity
2. Verify container access permissions
3. Review circuit breaker state
4. Check network latency between API and Docker daemon

### Slow Collection

1. Increase `METRICS_MAX_WORKERS` for more parallel workers
2. Check Docker API response times
3. Monitor container count (may need to scale workers)
4. Review circuit breaker failure threshold

### Database Write Issues

1. Check retry queue size: `retry_queue_size`
2. Verify TimescaleDB connectivity
3. Check database disk space
4. Review batch size configuration

## Configuration

All metrics monitoring can be configured via environment variables. See:
[Environment Variables Reference](../reference/environment-variables.md#metrics-collection-configuration)

**Key Settings:**
- `METRICS_COLLECTION_INTERVAL`: How often to collect (default: 5s)
- `METRICS_MAX_WORKERS`: Parallel workers (default: 50)
- `METRICS_HEALTH_CHECK_INTERVAL`: Health check frequency (default: 30s)
- `METRICS_CIRCUIT_BREAKER_FAILURE_THRESHOLD`: Failures before opening circuit (default: 5)

---

[← Back to Guides](index.md)
