# Database Connection Pool Tuning Guide

## Overview

The database connection pool tuning feature provides automatic optimization of database connections based on system resources and workload patterns. This ensures optimal performance while preventing connection exhaustion.

## Features

- **Automatic Pool Sizing**: Calculates optimal connection numbers based on CPU cores
- **Dynamic Adjustment**: Adapts pool size based on real-time usage patterns
- **Health Monitoring**: Continuous health checks with automatic recovery
- **Performance Metrics**: Detailed metrics collection and reporting
- **Visual Dashboard**: Real-time monitoring interface for managers

## Configuration

### Environment Variables

Configure the connection pool using these environment variables:

```bash
# Maximum number of open connections (default: calculated based on CPU)
DB_MAX_OPEN_CONNS=25

# Maximum number of idle connections (default: max_open / 2)
DB_MAX_IDLE_CONNS=12

# Maximum lifetime of a connection (default: 1h)
DB_CONN_MAX_LIFETIME=1h

# Maximum idle time before connection is closed (default: 10m)
DB_CONN_MAX_IDLE_TIME=10m

# Health check interval (default: 30s)
DB_HEALTH_CHECK_INTERVAL=30s
```

### Automatic Configuration

If no environment variables are set, the system automatically configures:

```
Max Connections = (CPU_CORES * 2) + 1
Max Idle = Max Connections / 2
```

Example:
- 4 CPU cores = 9 max connections, 4 idle
- 8 CPU cores = 17 max connections, 8 idle
- 16 CPU cores = 33 max connections (capped at 25)

## API Endpoints

### Pool Metrics
```
GET /api/db/pool/metrics
```

Returns current pool statistics:
```json
{
  "timestamp": "2025-01-25T10:30:00Z",
  "pool": {
    "open_connections": 15,
    "in_use": 8,
    "idle": 7,
    "wait_count": 0,
    "wait_duration_ms": 0
  },
  "performance": {
    "total_queries": 12543,
    "total_errors": 3,
    "error_rate": 0.024,
    "health_status": true
  },
  "configuration": {
    "max_open_conns": 25,
    "max_idle_conns": 12,
    "conn_max_lifetime": "1h0m0s",
    "conn_max_idle_time": "10m0s"
  }
}
```

### Pool Health
```
GET /api/db/pool/health
```

Returns health status:
```json
{
  "status": "healthy",
  "open_connections": 15,
  "utilization_rate": 60.0,
  "database_healthy": true
}
```

Status codes:
- `200 OK`: Healthy
- `503 Service Unavailable`: Unhealthy

### Pool Optimization
```
POST /api/db/pool/optimize
```

Triggers dynamic pool optimization based on current load.

## Monitoring Dashboard

Access the visual monitoring dashboard at:
```
/db-pool-monitor
```

Features:
- Real-time connection metrics
- Performance graphs
- Health status indicators
- Optimization recommendations
- One-click optimization

## Best Practices

### 1. Initial Setup

Let the system auto-configure based on your hardware:
```go
// In database.go
poolConfig := LoadPoolConfigFromEnv()
ConfigureDBPool(db, poolConfig)
```

### 2. Monitor Metrics

Watch for these warning signs:
- Utilization > 80%: Consider increasing max connections
- High wait count: Connections are exhausted
- High error rate: Database issues or query problems

### 3. Optimization Strategy

- **Low Traffic**: Reduce connections to save resources
- **High Traffic**: Increase connections for better concurrency
- **Burst Traffic**: Keep more idle connections ready

### 4. Health Checks

The system performs automatic health checks every 30 seconds:
- Pings database connection
- Recovers failed connections
- Logs health status

## Troubleshooting

### High Connection Wait Count

**Symptom**: `wait_count` continuously increasing

**Solutions**:
1. Increase `DB_MAX_OPEN_CONNS`
2. Optimize slow queries
3. Add read replicas for scaling

### Connection Pool Exhaustion

**Symptom**: "too many connections" errors

**Solutions**:
1. Check for connection leaks
2. Ensure proper connection closing
3. Review transaction durations

### Idle Connection Timeout

**Symptom**: "connection reset" errors

**Solutions**:
1. Decrease `DB_CONN_MAX_IDLE_TIME`
2. Enable connection keepalive
3. Check firewall/proxy timeouts

## Performance Optimization

### Query Optimization

Wrap queries with metrics collection:
```go
err := WrapQueryWithMetrics(func() error {
    return db.Query("SELECT * FROM users")
})
```

### Connection Reuse

Ensure connections are properly released:
```go
rows, err := db.Query("SELECT ...")
if err != nil {
    return err
}
defer rows.Close() // Always close!
```

### Transaction Management

Keep transactions short:
```go
tx, err := db.Begin()
if err != nil {
    return err
}
defer tx.Rollback() // Safety net

// Quick operations only
err = tx.Commit()
```

## Monitoring Alerts

The system shows alerts for:

- **High Utilization** (>90%): Connection pool near capacity
- **Elevated Utilization** (>80%): Monitor closely
- **High Wait Count** (>100): Connections being queued
- **Health Check Failed**: Database connectivity issues

## Integration with Load Balancers

For production deployments:

1. **Health Check Endpoint**: `/api/db/health`
   - No authentication required
   - Returns 200 for healthy, 503 for unhealthy

2. **Metrics Export**: Use `/api/db/pool/metrics` for:
   - Prometheus integration
   - Grafana dashboards
   - Custom monitoring tools

## Advanced Tuning

### PostgreSQL Configuration

Ensure PostgreSQL is configured to handle the connection load:

```sql
-- Check current max connections
SHOW max_connections;

-- Recommended: Set to 2-3x your app's max connections
ALTER SYSTEM SET max_connections = 100;

-- Reload configuration
SELECT pg_reload_conf();
```

### Connection Pooling Layers

Consider additional pooling:
1. **PgBouncer**: For very high connection counts
2. **Read Replicas**: For read-heavy workloads
3. **Connection Proxy**: For multi-region deployments

## Metrics Collection

The system tracks:
- Total queries executed
- Query error count
- Connection wait times
- Pool utilization rates
- Health check results

Access metrics programmatically:
```go
metrics := GetPoolMetrics()
fmt.Printf("Queries: %d, Errors: %d\n", 
    metrics.QueryCount, metrics.ErrorCount)
```

## Future Enhancements

1. **Predictive Scaling**: ML-based pool size prediction
2. **Multi-Database Support**: Pool per database
3. **Connection Priorities**: Priority queues for critical queries
4. **Detailed Query Analytics**: Per-query performance tracking