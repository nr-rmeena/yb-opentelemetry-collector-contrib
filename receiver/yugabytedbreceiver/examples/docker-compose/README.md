# YugabyteDB Receiver - Docker Compose Example

This example demonstrates how to run the YugabyteDB receiver locally with a YugabyteDB cluster and send metrics to New Relic.

## Prerequisites

- Docker and Docker Compose installed
- New Relic account with an API key
- OpenTelemetry Collector built with yugabytedbreceiver

## Metrics Collected

The YugabyteDB receiver collects the following metrics from `pg_stat_activity`:

1. **yugabytedb.connection.count** - Number of connections broken down by state and user
   - **Attributes:**
     - `connection.state`: active, idle, idle_in_transaction, waiting, unknown
     - `connection.user`: Database user (e.g., yugabyte, app_user)
   - **Use cases:** Identify connection leaks, monitor idle transactions, track per-user usage

2. **yugabytedb.pg_stat_activity.running_queries** - Number of currently running queries (state = 'active')
   - **Use cases:** Monitor query load, identify query spikes

3. **yugabytedb.pg_stat_activity.active_connections** - Total number of active connections to YugabyteDB
   - **Use cases:** Track overall connection count, connection utilization

## Quick Start

### 1. Set Environment Variables

```bash
export NEW_RELIC_API_KEY="your-api-key-here"
```

### 2. Start the Stack

```bash
docker-compose up -d
```

This will start:
- YugabyteDB cluster (single node)
- OpenTelemetry Collector with yugabytedbreceiver

### 3. Verify Metrics

The collector will:
- Connect to YugabyteDB on port 5433
- Query `pg_stat_activity` every 10 seconds
- Send metrics to New Relic OTLP endpoint
- Display debug output in the logs

View collector logs:
```bash
docker-compose logs -f otel-collector
```

### 4. Query Metrics in New Relic

#### Connection State Distribution
```nrql
FROM Metric SELECT latest(yugabytedb.connection.count) 
FACET connection.state 
TIMESERIES AUTO 
SINCE 30 minutes ago
```

#### Connections by User
```nrql
FROM Metric SELECT latest(yugabytedb.connection.count) 
FACET connection.user 
SINCE 5 minutes ago
```

#### Idle in Transaction (Potential Issues)
```nrql
FROM Metric SELECT latest(yugabytedb.connection.count) 
WHERE connection.state = 'idle_in_transaction'
TIMESERIES AUTO
SINCE 30 minutes ago
```

#### Running Queries
```nrql
FROM Metric SELECT latest(yugabytedb.pg_stat_activity.running_queries) 
TIMESERIES AUTO 
SINCE 30 minutes ago
```

#### Active Connections
```nrql
FROM Metric SELECT latest(yugabytedb.pg_stat_activity.active_connections) 
TIMESERIES AUTO 
SINCE 30 minutes ago
```

#### Complete Overview
```nrql
FROM Metric SELECT 
  sum(latest(yugabytedb.connection.count)) as 'Total Connections',
  filter(sum(latest(yugabytedb.connection.count)), WHERE connection.state = 'active') as 'Active',
  filter(sum(latest(yugabytedb.connection.count)), WHERE connection.state = 'idle') as 'Idle',
  filter(sum(latest(yugabytedb.connection.count)), WHERE connection.state = 'idle_in_transaction') as 'Idle in Tx',
  latest(yugabytedb.pg_stat_activity.running_queries) as 'Running Queries'
TIMESERIES AUTO
SINCE 1 hour ago
```

## Configuration

The collector configuration is in `otel-collector-config.yaml`:

- **Receiver:** yugabytedb receiver configured to connect to localhost:5433
- **Exporters:** 
  - otlphttp/newrelic - sends metrics to New Relic staging environment
  - debug - prints metrics to console for verification

## Understanding Connection States

The `yugabytedb.connection.count` metric tracks connections in different states:

- **active**: Currently executing a query
- **idle**: Connection open but not executing anything
- **idle_in_transaction**: Connection has an open transaction but isn't executing a query (⚠️ potential issue)
- **waiting**: Connection is waiting for a lock
- **unknown**: State could not be determined

### Why Monitor Idle in Transaction?

Connections in the "idle in transaction" state can:
- Hold locks and block other queries
- Prevent VACUUM from cleaning up old rows
- Indicate application bugs (e.g., missing commits)

**Alert Threshold:** > 5 idle_in_transaction connections for > 5 minutes

## Stopping the Stack

```bash
docker-compose down
```

## Troubleshooting

### Check YugabyteDB is running
```bash
docker-compose ps
```

### Connect to YugabyteDB manually
```bash
docker exec -it yugabytedb psql -U yugabyte -d yugabyte
```

### Check pg_stat_activity
```sql
-- See all connections with their states
SELECT usename, state, count(*) 
FROM pg_stat_activity 
GROUP BY usename, state 
ORDER BY count(*) DESC;

-- Total count
SELECT count(*) FROM pg_stat_activity;

-- Active queries only
SELECT count(*) FROM pg_stat_activity WHERE state = 'active';

-- Idle in transaction (potential issues)
SELECT usename, query_start, state_change, query
FROM pg_stat_activity 
WHERE state = 'idle in transaction'
ORDER BY state_change;
```

### View collector logs
```bash
docker-compose logs -f otel-collector
```

Look for:
- ✅ "Everything is ready. Begin running and processing data"
- ✅ Debug output showing metrics being collected with different connection states and users
- ❌ Any error messages about database connection or query failures

## Dashboard Recommendations

Create a New Relic dashboard with these widgets:

1. **Connection State Distribution (Pie Chart)**
   - Shows breakdown by state (active, idle, idle_in_transaction, etc.)

2. **Connections by User (Bar Chart)**
   - Identifies which users/applications are using the most connections

3. **Idle in Transaction Timeline (Line Chart)**
   - Tracks potentially problematic idle transactions over time

4. **Running Queries vs Total Connections (Line Chart)**
   - Compare active query load with total connection count

5. **Connection Utilization (Billboard)**
   - Shows percentage: `(total connections / max_connections) * 100`

## Notes

- The collector scrapes metrics every 10 seconds
- YugabyteDB runs on port 5433 (default YugabyteDB YSQL port)
- Default credentials: user=yugabyte, password=yugabyte, database=yugabyte
- Metrics are sent to New Relic staging environment (staging-otlp.nr-data.net)
- Connection states are automatically normalized (e.g., "idle in transaction" → "idle_in_transaction")
