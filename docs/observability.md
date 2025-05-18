# Observability in Wallet Service

This document provides detailed information about the observability features implemented in the wallet-service.

## Overview

The wallet-service implements a comprehensive observability stack consisting of:

1. **Structured Logging**: Using zap for context-rich, efficient, and leveled logging
2. **Metrics Collection**: Using Prometheus for capturing system and business metrics
3. **Distributed Tracing**: Using OpenTelemetry for end-to-end request tracking

## Structured Logging

### Implementation

- Uses the `zap` library for high-performance, structured logging
- Logs are output in JSON format for easy parsing by log aggregation tools
- Contextual information (request IDs, user IDs, parameters) is included in log entries
- Log levels allow for proper filtering and priority

### Log Levels

- `debug`: Detailed information useful during development and troubleshooting
- `info`: General operational information about the system's behavior
- `warn`: Warning events that might require attention
- `error`: Error events that affect functionality but don't crash the system

### Contextual Information

Logs include:
- Component name (e.g., `wallet_service`, `wallet_handler`)
- Request ID for correlating log entries for a single request
- User ID where applicable
- Operation parameters
- Error details
- Duration information for performance tracking

## Metrics Collection

### Implementation

- Uses Prometheus client for metrics collection and exposure
- Metrics are exposed on the `/metrics` endpoint
- Pre-configured dashboards are available (see Monitoring section)

### Key Metrics

1. **HTTP Metrics**
   - Request counts by endpoint and status code
   - Request duration distributions

2. **Database Metrics**
   - Query duration distributions by operation type
   - Connection pool statistics

3. **Business Metrics**
   - Wallet operation counts (view, exchange, spend) by outcome (success, error)
   - Wallet balances by user

## Distributed Tracing

### Implementation

- Uses OpenTelemetry for standardized tracing
- Traces are exported to a configurable collector (default: Jaeger)
- Automatic instrumentation for HTTP requests via Fiber middleware
- Manual instrumentation for database queries and business logic

### Trace Information

- Service operations with timing information
- Detailed context for each span (user ID, parameters)
- Error information when operations fail
- Connection to logs and metrics through correlation IDs

## Configuration

### Environment Variables

```
# Logging
LOG_LEVEL=debug|info|warn|error (default: info)

# Tracing
TRACING_ENABLED=true|false (default: false)
TRACING_ENDPOINT=host:port (default: localhost:4317)
TRACING_SAMPLING_RATIO=0.0-1.0 (default: 0.1)

# Metrics
METRICS_ENABLED=true|false (default: true)
```

## Monitoring Setup

### Recommended Tools

1. **Logging**: 
   - Elasticsearch + Kibana
   - Loki + Grafana

2. **Metrics**:
   - Prometheus + Grafana
   
3. **Tracing**:
   - Jaeger UI
   - Tempo + Grafana

### Docker Compose Setup

A sample Docker Compose configuration for a full observability stack is available in the `observability` directory.

## Best Practices

1. **Log Levels**:
   - Use `debug` for detailed development information
   - Use `info` for regular operations
   - Use `warn` for potential issues
   - Use `error` for actual errors

2. **Metrics**:
   - Focus on the Four Golden Signals: latency, traffic, errors, and saturation
   - Add business-specific metrics that matter to your domain

3. **Tracing**:
   - Ensure sampling rate is appropriate for your traffic volume
   - Don't overload spans with too much data
   - Connect traces to logs via request IDs

## Future Improvements

1. Integration with a centralized logging solution
2. Enhanced metrics with SLO/SLI tracking
3. Expanded business metrics for better operational insights
4. Health check endpoint with detailed service status
