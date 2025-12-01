# YugabyteDB Receiver - Complete NRQL Query Reference

## 🎯 Quick Reference - All Metrics

The YugabyteDB receiver collects 3 main metrics:

1. **yugabytedb.connection.count** - Connections by state and user (with attributes)
2. **yugabytedb.pg_stat_activity.running_queries** - Count of running queries
3. **yugabytedb.pg_stat_activity.active_connections** - Total connections

---

## 📊 Essential Queries

### 1. Connection State Overview (Most Important!)

**See all connection states as a pie chart:**
```nrql
FROM Metric 
SELECT latest(yugabytedb.connection.count) 
FACET connection.state 
SINCE 5 minutes ago
```

**Timeline of connection states (Stacked Area Chart):**
```nrql
FROM Metric 
SELECT latest(yugabytedb.connection.count) 
FACET connection.state 
TIMESERIES AUTO 
SINCE 1 hour ago
```

### 2. Connections by User

**Which users have the most connections:**
```nrql
FROM Metric 
SELECT sum(latest(yugabytedb.connection.count)) 
FACET connection.user 
SINCE 5 minutes ago
```

**Specific user's connection breakdown by state:**
```nrql
FROM Metric 
SELECT latest(yugabytedb.connection.count) 
WHERE connection.user = 'yugabyte'
FACET connection.state
SINCE 5 minutes ago
```

### 3. Running Queries

**Current running queries count:**
```nrql
FROM Metric 
SELECT latest(yugabytedb.pg_stat_activity.running_queries) 
SINCE 5 minutes ago
```

**Running queries timeline:**
```nrql
FROM Metric 
SELECT latest(yugabytedb.pg_stat_activity.running_queries) 
TIMESERIES AUTO 
SINCE 1 hour ago
```

### 4. Total Active Connections

**Current total connections:**
```nrql
FROM Metric 
SELECT latest(yugabytedb.pg_stat_activity.active_connections) 
SINCE 5 minutes ago
```

**Total connections over time:**
```nrql
FROM Metric 
SELECT latest(yugabytedb.pg_stat_activity.active_connections) 
TIMESERIES AUTO 
SINCE 1 hour ago
```

---

## 🚨 Alert Queries

### Alert 1: Too Many Idle in Transaction Connections
```nrql
FROM Metric 
SELECT latest(yugabytedb.connection.count) 
WHERE connection.state = 'idle_in_transaction'
```
**Alert when:** Value > 10 for at least 5 minutes

### Alert 2: High Connection Count per User
```nrql
FROM Metric 
SELECT sum(latest(yugabytedb.connection.count)) 
FACET connection.user
```
**Alert when:** Any facet > 50 for at least 2 minutes

### Alert 3: Connection Utilization (if max_connections = 300)
```nrql
FROM Metric 
SELECT (latest(yugabytedb.pg_stat_activity.active_connections) / 300) * 100 as 'utilization'
```
**Alert when:** utilization > 80% for at least 5 minutes

### Alert 4: Spike in Running Queries
```nrql
FROM Metric 
SELECT latest(yugabytedb.pg_stat_activity.running_queries)
```
**Alert when:** Value > 100 for at least 3 minutes

---

## 📈 Dashboard Queries

### Widget 1: Connection State Distribution (Pie Chart)
```nrql
FROM Metric 
SELECT latest(yugabytedb.connection.count) 
FACET connection.state
```

### Widget 2: Top 10 Users by Connection Count (Bar Chart)
```nrql
FROM Metric 
SELECT sum(latest(yugabytedb.connection.count)) 
FACET connection.user 
LIMIT 10
```

### Widget 3: Idle in Transaction Timeline (Line Chart)
```nrql
FROM Metric 
SELECT latest(yugabytedb.connection.count) 
WHERE connection.state = 'idle_in_transaction'
TIMESERIES AUTO 
SINCE 1 hour ago
```

### Widget 4: Complete Health Overview (Multiple Series)
```nrql
FROM Metric 
SELECT 
  latest(yugabytedb.pg_stat_activity.running_queries) as 'Running Queries',
  latest(yugabytedb.pg_stat_activity.active_connections) as 'Total Connections',
  filter(sum(latest(yugabytedb.connection.count)), WHERE connection.state = 'active') as 'Active',
  filter(sum(latest(yugabytedb.connection.count)), WHERE connection.state = 'idle') as 'Idle',
  filter(sum(latest(yugabytedb.connection.count)), WHERE connection.state = 'idle_in_transaction') as 'Idle in Tx'
TIMESERIES AUTO 
SINCE 1 hour ago
```

### Widget 5: Connection Utilization Gauge
```nrql
FROM Metric 
SELECT (latest(yugabytedb.pg_stat_activity.active_connections) / 300) * 100 as 'Connection Utilization %'
```

### Widget 6: Connections by User and State (Table)
```nrql
FROM Metric 
SELECT latest(yugabytedb.connection.count) 
FACET connection.user, connection.state
SINCE 5 minutes ago
```

---

## 🔍 Investigation Queries

### Find Idle in Transaction Connections by User
```nrql
FROM Metric 
SELECT latest(yugabytedb.connection.count) 
WHERE connection.state = 'idle_in_transaction'
FACET connection.user
TIMESERIES AUTO 
SINCE 1 hour ago
```

### Compare Active vs Idle Connections
```nrql
FROM Metric 
SELECT 
  filter(sum(latest(yugabytedb.connection.count)), WHERE connection.state = 'active') as 'Active',
  filter(sum(latest(yugabytedb.connection.count)), WHERE connection.state = 'idle') as 'Idle'
TIMESERIES AUTO 
SINCE 1 hour ago
```

### Connection Efficiency (Running Queries / Total Connections)
```nrql
FROM Metric 
SELECT 
  latest(yugabytedb.pg_stat_activity.running_queries) as 'Running Queries',
  latest(yugabytedb.pg_stat_activity.active_connections) as 'Total Connections',
  (latest(yugabytedb.pg_stat_activity.running_queries) / latest(yugabytedb.pg_stat_activity.active_connections)) * 100 as 'Efficiency %'
TIMESERIES AUTO 
SINCE 1 hour ago
```

### All Connection States Summary
```nrql
FROM Metric 
SELECT 
  sum(latest(yugabytedb.connection.count)) as 'Total',
  filter(sum(latest(yugabytedb.connection.count)), WHERE connection.state = 'active') as 'Active',
  filter(sum(latest(yugabytedb.connection.count)), WHERE connection.state = 'idle') as 'Idle',
  filter(sum(latest(yugabytedb.connection.count)), WHERE connection.state = 'idle_in_transaction') as 'Idle in Tx',
  filter(sum(latest(yugabytedb.connection.count)), WHERE connection.state = 'waiting') as 'Waiting'
SINCE 5 minutes ago
```

---

## 🎨 Advanced Queries

### Peak Connection Time Analysis
```nrql
FROM Metric 
SELECT max(latest(yugabytedb.pg_stat_activity.active_connections)) 
FACET hourOf(timestamp) 
SINCE 7 days ago
```

### User Connection Trends
```nrql
FROM Metric 
SELECT average(latest(yugabytedb.connection.count)) 
FACET connection.user 
TIMESERIES 1 hour 
SINCE 24 hours ago
```

### Connection State Percentage Distribution
```nrql
FROM Metric 
SELECT percentage(
  filter(sum(latest(yugabytedb.connection.count)), WHERE connection.state = 'active'),
  sum(latest(yugabytedb.connection.count))
) as 'Active %',
percentage(
  filter(sum(latest(yugabytedb.connection.count)), WHERE connection.state = 'idle'),
  sum(latest(yugabytedb.connection.count))
) as 'Idle %',
percentage(
  filter(sum(latest(yugabytedb.connection.count)), WHERE connection.state = 'idle_in_transaction'),
  sum(latest(yugabytedb.connection.count))
) as 'Idle in Tx %'
SINCE 5 minutes ago
```

---

## 💡 Important Notes

1. **Use `latest()` for gauge metrics** - All YugabyteDB metrics are gauges
2. **Use `sum(latest())` when aggregating** - When combining multiple state/user combinations
3. **Use `filter()` for conditional aggregation** - To get specific states
4. **Remember the attribute names:**
   - `connection.state` - active, idle, idle_in_transaction, waiting, unknown
   - `connection.user` - Database username

## 🎯 Quick Copy-Paste for Dashboard

```nrql
-- Widget 1: Connection States (Pie)
FROM Metric SELECT latest(yugabytedb.connection.count) FACET connection.state

-- Widget 2: Connections by User (Bar)
FROM Metric SELECT sum(latest(yugabytedb.connection.count)) FACET connection.user LIMIT 10

-- Widget 3: Running Queries (Line)
FROM Metric SELECT latest(yugabytedb.pg_stat_activity.running_queries) TIMESERIES AUTO SINCE 1 hour ago

-- Widget 4: Total Connections (Line)
FROM Metric SELECT latest(yugabytedb.pg_stat_activity.active_connections) TIMESERIES AUTO SINCE 1 hour ago

-- Widget 5: Idle in Transaction (Line)
FROM Metric SELECT latest(yugabytedb.connection.count) WHERE connection.state = 'idle_in_transaction' TIMESERIES AUTO SINCE 1 hour ago

-- Widget 6: Utilization (Billboard)
FROM Metric SELECT (latest(yugabytedb.pg_stat_activity.active_connections) / 300) * 100 as 'Utilization %'
```

