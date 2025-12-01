# YugabyteDB Receiver - Summary of Changes

## Date: November 25, 2025

## Changes Made

### 1. Added New Metric: `yugabytedb.pg_stat_activity.active_connections`

**Description:** Tracks the total number of active connections to YugabyteDB.

**SQL Query:** `SELECT count(*) FROM pg_stat_activity`

**Metric Type:** Gauge (int64)

**Unit:** {connections}

### 2. Updated Files

#### metadata.yaml
- Added `yugabytedb.pg_stat_activity.active_connections` metric definition
- Metrics are alphabetically sorted (required by mdatagen):
  1. active_connections
  2. running_queries

#### yugabytedbreceiver.go
- Refactored `scrapeLoop` to use separate `collectMetrics` function
- Now collects TWO metrics in each scrape cycle:
  - **running_queries**: `SELECT count(*) FROM pg_stat_activity WHERE state = 'active'`
  - **active_connections**: `SELECT count(*) FROM pg_stat_activity`
- Improved error handling - continues collecting other metrics even if one fails
- Single database connection per scrape (better performance)

#### Generated Code (by mdatagen)
- `internal/metadata/generated_metrics.go` now includes:
  - `RecordYugabytedbPgStatActivityActiveConnectionsDataPoint()`
  - `RecordYugabytedbPgStatActivityRunningQueriesDataPoint()`
  - Both metrics are automatically emitted together

### 3. Reorganized Folder Structure

Created new folder structure for better organization:

```
receiver/yugabytedbreceiver/
├── config.go
├── doc.go
├── factory.go
├── go.mod
├── go.sum
├── metadata.yaml
├── receiver_test.go
├── yugabytedbreceiver.go
├── NEW_RELIC_QUERIES.md          # Query guide for New Relic
├── internal/
│   └── metadata/                  # Generated code by mdatagen
│       ├── generated_*.go
│       └── testdata/
└── examples/                      # NEW: Examples folder
    └── docker-compose/            # NEW: Docker Compose setup
        ├── README.md              # Comprehensive setup guide
        ├── docker-compose.yaml    # YugabyteDB + Collector setup
        └── otel-collector-config.yaml  # Collector configuration
```

**Moved files:**
- `docker-compose.yaml` → `examples/docker-compose/`
- `otel-collector-config.yaml` → `examples/docker-compose/`

### 4. Updated Documentation

#### NEW_RELIC_QUERIES.md
- Added queries for both metrics
- Combined query examples
- Dashboard widget examples
- Connection utilization calculation example

#### examples/docker-compose/README.md
- Complete setup guide
- Prerequisites and quick start
- Troubleshooting section
- NRQL query examples for both metrics

## Metrics Summary

| Metric Name | Description | Query | Unit |
|-------------|-------------|-------|------|
| `yugabytedb.pg_stat_activity.running_queries` | Currently executing queries | `WHERE state = 'active'` | {queries} |
| `yugabytedb.pg_stat_activity.active_connections` | Total active connections | All rows | {connections} |

## New Relic Queries

### Individual Metrics

```nrql
FROM Metric SELECT latest(yugabytedb.pg_stat_activity.running_queries) SINCE 5 minutes ago
```

```nrql
FROM Metric SELECT latest(yugabytedb.pg_stat_activity.active_connections) SINCE 5 minutes ago
```

### Combined View

```nrql
FROM Metric SELECT 
  latest(yugabytedb.pg_stat_activity.running_queries) as 'Running Queries',
  latest(yugabytedb.pg_stat_activity.active_connections) as 'Active Connections'
TIMESERIES AUTO 
SINCE 30 minutes ago
```

## Testing

### Build
```bash
cd /Users/rmeena/Documents/Projects/yb-opentelemetry-collector-contrib
make otelcontribcol
```

### Run Locally
```bash
bin/otelcontribcol_darwin_arm64 --config receiver/yugabytedbreceiver/examples/docker-compose/otel-collector-config.yaml
```

### Expected Output
Every 10 seconds you should see debug output showing:
1. `yugabytedb.pg_stat_activity.active_connections` with value (e.g., 5)
2. `yugabytedb.pg_stat_activity.running_queries` with value (e.g., 1)

## Next Steps

1. **Test with Docker Compose:**
   ```bash
   cd receiver/yugabytedbreceiver/examples/docker-compose
   docker-compose up
   ```

2. **Verify in New Relic:**
   - Use the queries from NEW_RELIC_QUERIES.md
   - Create dashboards with both metrics
   - Set up alerts for connection spikes

3. **Add More Metrics (Future):**
   - Database size
   - Transaction rate
   - Cache hit ratio
   - Replication lag
   - Table-level statistics

## Benefits

✅ **Two metrics for better monitoring** - Track both connections and active queries
✅ **Better code organization** - Examples in separate folder
✅ **Comprehensive documentation** - Setup guides and query examples
✅ **Type-safe metric recording** - Using mdatagen-generated builders
✅ **Proper error handling** - Continues collecting even if one query fails
✅ **Production-ready** - Follows OpenTelemetry best practices

