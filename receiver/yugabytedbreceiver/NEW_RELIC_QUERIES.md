# YugabyteDB Receiver - New Relic Query Guide

## Issue: Metrics showing as null

When OpenTelemetry gauge metrics are sent to New Relic, they are not stored directly under the metric name. Instead, New Relic stores them with aggregation fields.

## Available Metrics

1. **yugabytedb.connection.count** - Number of connections broken down by state and user
   - Attributes: `connection.state`, `connection.user`
   - States: `active`, `idle`, `idle_in_transaction`, `waiting`, `unknown`
2. **yugabytedb.pg_stat_activity.running_queries** - Number of currently running queries (state = 'active')
3. **yugabytedb.pg_stat_activity.active_connections** - Total number of active connections

## Correct Queries to Use

### Connection Count by State

**All connections by state:**
```nrql
FROM Metric SELECT latest(yugabytedb.connection.count) 
FACET connection.state 
SINCE 5 minutes ago
```

**Active connections only:**
```nrql
FROM Metric SELECT latest(yugabytedb.connection.count) 
WHERE connection.state = 'active'
SINCE 5 minutes ago
```

**Idle connections:**
```nrql
FROM Metric SELECT latest(yugabytedb.connection.count) 
WHERE connection.state = 'idle'
SINCE 5 minutes ago
```

**Idle in transaction (potential issues):**
```nrql
FROM Metric SELECT latest(yugabytedb.connection.count) 
WHERE connection.state = 'idle_in_transaction'
TIMESERIES AUTO
SINCE 30 minutes ago
```

### Connection Count by User

**All connections by user:**
```nrql
FROM Metric SELECT latest(yugabytedb.connection.count) 
FACET connection.user 
SINCE 5 minutes ago
```

**Specific user's connections:**
```nrql
FROM Metric SELECT latest(yugabytedb.connection.count) 
WHERE connection.user = 'yugabyte'
FACET connection.state
SINCE 5 minutes ago
```

### Connection State Distribution

**Stacked area chart showing all states over time:**
```nrql
FROM Metric SELECT latest(yugabytedb.connection.count) 
FACET connection.state 
TIMESERIES AUTO 
SINCE 1 hour ago
```

**Breakdown by state and user:**
```nrql
FROM Metric SELECT latest(yugabytedb.connection.count) 
FACET connection.state, connection.user 
SINCE 5 minutes ago
```

### Running Queries

**Get latest value:**
```nrql
FROM Metric SELECT latest(yugabytedb.pg_stat_activity.running_queries) SINCE 5 minutes ago
```

**Time series:**
```nrql
FROM Metric SELECT latest(yugabytedb.pg_stat_activity.running_queries) TIMESERIES AUTO SINCE 30 minutes ago
```

### Active Connections

**Get latest value:**
```nrql
FROM Metric SELECT latest(yugabytedb.pg_stat_activity.active_connections) SINCE 5 minutes ago
```

**Time series:**
```nrql
FROM Metric SELECT latest(yugabytedb.pg_stat_activity.active_connections) TIMESERIES AUTO SINCE 30 minutes ago
```

### Combined Views

**All three metrics together:**
```nrql
FROM Metric SELECT 
  latest(yugabytedb.pg_stat_activity.running_queries) as 'Running Queries',
  latest(yugabytedb.pg_stat_activity.active_connections) as 'Active Connections',
  sum(latest(yugabytedb.connection.count)) as 'Total Connections'
TIMESERIES AUTO 
SINCE 30 minutes ago
```

**Connection health overview:**
```nrql
FROM Metric SELECT 
  sum(latest(yugabytedb.connection.count)) as 'Total',
  filter(sum(latest(yugabytedb.connection.count)), WHERE connection.state = 'active') as 'Active',
  filter(sum(latest(yugabytedb.connection.count)), WHERE connection.state = 'idle') as 'Idle',
  filter(sum(latest(yugabytedb.connection.count)), WHERE connection.state = 'idle_in_transaction') as 'Idle in Tx'
TIMESERIES AUTO
SINCE 1 hour ago
```

### Check What Metrics Are Available

**List all YugabyteDB metrics:**
```nrql
FROM Metric SELECT uniques(metricName) WHERE metricName LIKE 'yugabytedb%' SINCE 5 minutes ago
```

**See all attributes for connection.count:**
```nrql
FROM Metric SELECT keyset() WHERE metricName = 'yugabytedb.connection.count' SINCE 5 minutes ago LIMIT 1
```

## Dashboard Widgets

### Active Connections by State (Pie Chart)
```nrql
FROM Metric SELECT latest(yugabytedb.connection.count)
FACET connection.state
WHERE connection.state IN ('active', 'idle', 'idle_in_transaction', 'waiting')
```

### Top Users by Connection Count (Bar Chart)
```nrql
FROM Metric SELECT sum(latest(yugabytedb.connection.count))
FACET connection.user
SINCE 5 minutes ago
LIMIT 10
```

### Idle in Transaction Alert (potential issues)
```nrql
FROM Metric SELECT latest(yugabytedb.connection.count)
WHERE connection.state = 'idle_in_transaction'
FACET connection.user
```

### Connection State Timeline (Stacked Area)
```nrql
FROM Metric SELECT latest(yugabytedb.connection.count)
FACET connection.state
TIMESERIES AUTO
SINCE 1 hour ago
```

### Connection Utilization (if you have max_connections = 300)
```nrql
FROM Metric SELECT 
  sum(latest(yugabytedb.connection.count)) / 300 * 100 as 'Connection Utilization %'
TIMESERIES AUTO 
SINCE 1 hour ago
```

## Alert Conditions

### Too many idle in transaction connections
```nrql
FROM Metric SELECT latest(yugabytedb.connection.count)
WHERE connection.state = 'idle_in_transaction'
```
Alert when: `Query returns a value` > 10 for at least 5 minutes

### High connection count per user
```nrql
FROM Metric SELECT latest(yugabytedb.connection.count)
FACET connection.user
```
Alert when: `Query returns a value` > 50 for at least 2 minutes

## Notes

- Gauge metrics in New Relic require aggregation functions (latest, average, min, max)
- The raw metric name won't return values directly
- Use `latest()` for the most recent value
- Use `sum(latest())` when aggregating across multiple data points (different states/users)
- The `connection.state` attribute shows the PostgreSQL connection state
- The `connection.user` attribute shows the database user owning the connection
- States are normalized: "idle in transaction" becomes "idle_in_transaction"
