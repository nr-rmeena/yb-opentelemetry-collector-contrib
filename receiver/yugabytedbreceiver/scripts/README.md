# YugabyteDB Global View Setup Scripts

This directory contains scripts to set up YugabyteDB cluster monitoring with OpenTelemetry using Global Views.

## Files

- **`setup_yugabytedb_monitoring.sh`** - Main setup script (automated)
- **`setup_global_views.sql`** - Creates Global Views with Foreign Data Wrappers
- **`create_monitoring_user.sql`** - Creates read-only monitoring user
- **`README.md`** - This documentation

## Prerequisites

1. **PostgreSQL Client Tools** - The `psql` command must be installed:
   - macOS: `brew install postgresql`
   - Ubuntu/Debian: `apt-get install postgresql-client`
   - RHEL/CentOS: `yum install postgresql`

2. **YugabyteDB Cluster** - A running cluster (single node or multi-node)

3. **Admin Credentials** - Access to create databases, users, and extensions

## Quick Start

### One-Command Setup

Run the automated setup script:

```bash
cd receiver/yugabytedbreceiver/scripts
./setup_yugabytedb_monitoring.sh
```

The script will:
1. Prompt for YugabyteDB connection credentials
2. Test the database connection
3. Create a new database `gv$`
4. Set up Global Views with Foreign Data Wrappers
5. Create a monitoring user with read-only access
6. Display the monitoring credentials

**Example Session:**

```
YugabyteDB host [localhost]:
YugabyteDB port [5433]:
Admin username [yugabyte]:
Admin password [yugabyte]: ********
Database name [yugabyte]:

Testing connection to localhost:5433...
Setting up Global Views...
✓ Global Views setup completed successfully!

Creating monitoring user...

==========================================
Setup Completed Successfully!
==========================================

Created components:
  - Global Views (gv$pg_stat_activity, gv$pg_stat_statements, gv$pg_stat_database)
  - gv_history schema with global_pg_stat_statements table
  - Monitoring user for read-only access

==========================================
Monitoring User Credentials
==========================================
Username: nr_monitor
Password: nr_monitor_2024

Connection string:
postgresql://nr_monitor:nr_monitor_2024@localhost:5433/gv$

Test connection:
psql postgresql://nr_monitor:nr_monitor_2024@localhost:5433/gv$
==========================================
```

### Non-Interactive Mode (CI/CD)

For automated deployments, set environment variables:

```bash
export YB_HOST="localhost"
export YB_PORT="5433"
export YB_ADMIN_USER="yugabyte"
export YB_ADMIN_PASSWORD="yugabyte"
export YB_DATABASE="yugabyte"

./setup_yugabytedb_monitoring.sh
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `YB_HOST` | `localhost` | YugabyteDB host address |
| `YB_PORT` | `5433` | YugabyteDB port |
| `YB_ADMIN_USER` | `yugabyte` | Admin user for setup |
| `YB_ADMIN_PASSWORD` | `yugabyte` | Admin user password |
| `YB_DATABASE` | `yugabyte` | Database to connect to initially |

## What Gets Created

### 1. New Database: `gv$`

A dedicated database for Global Views and monitoring data.

### 2. Global View Infrastructure

- **postgres_fdw extension** - Enables foreign data wrapper functionality
- **Foreign servers** - One per YugabyteDB node (`gv$<hostname>`)
- **Foreign schemas** - One per node (`gv$<hostname>`)
- **Foreign tables** - System catalogs from each node:
  - `pg_stat_activity`
  - `pg_stat_statements`
  - `pg_stat_database`

### 3. Global Views

Unified views aggregating data from all nodes with metadata columns:

- **`gv$pg_stat_activity`** - Active connections and queries across all nodes
- **`gv$pg_stat_statements`** - Query statistics from all nodes
- **`gv$pg_stat_database`** - Database statistics from all nodes

Each view includes these metadata columns:
- `gv$host` - Node hostname
- `gv$zone` - Availability zone
- `gv$region` - Region
- `gv$cloud` - Cloud provider

Example view structure:
```sql
CREATE VIEW "gv$pg_stat_activity" AS
  SELECT 'node1' AS "gv$host", 'us-west1-a' AS "gv$zone",
         'us-west1' AS "gv$region", 'gcp' AS "gv$cloud", t.*
  FROM "gv$node1".pg_stat_activity AS t
  UNION ALL
  SELECT 'node2' AS "gv$host", 'us-west1-b' AS "gv$zone",
         'us-west1' AS "gv$region", 'gcp' AS "gv$cloud", t.*
  FROM "gv$node2".pg_stat_activity AS t;
```

### 4. Historical Storage Schema

- **`gv_history` schema** - For storing historical metrics
- **`gv_history.global_pg_stat_statements` table** - Query statistics history with timestamps

### 5. Monitoring User

- **Username**: `nr_monitor`
- **Password**: `nr_monitor_2024` (default)
- **Privileges**: Read-only access to all Global Views and tables
- **Access**: Can connect to `gv$` database and query all views

## Verification

### 1. Test Monitoring User Connection

```bash
psql postgresql://nr_monitor:nr_monitor_2024@localhost:5433/gv$
```

### 2. List Global Views

```sql
\c gv$
\dv gv$*
```

### 3. Query Sample Data

```sql
-- View all nodes in the cluster
SELECT "gv$host", "gv$zone", "gv$region", "gv$cloud"
FROM "gv$pg_stat_activity"
GROUP BY "gv$host", "gv$zone", "gv$region", "gv$cloud";

-- Get active connections per node
SELECT "gv$host", COUNT(*) as connection_count
FROM "gv$pg_stat_activity"
GROUP BY "gv$host"
ORDER BY connection_count DESC;

-- View active queries across all nodes
SELECT "gv$host", state, query
FROM "gv$pg_stat_activity"
WHERE state = 'active'
LIMIT 10;
```

### 4. Verify Monitoring User Privileges

```sql
-- This should work (read)
SELECT COUNT(*) FROM "gv$pg_stat_activity";

-- This should fail (write)
CREATE TABLE test_table (id INT);  -- ERROR: permission denied
```

## Integration with OpenTelemetry Collector

Update your `otel-collector-config.yaml`:

```yaml
receivers:
  yugabytedbreceiver:
    host: "localhost"
    port: 5433
    database: "gv$"                      # ← Connect to gv$ database
    user: "nr_monitor"                   # ← Use monitoring user
    password: "nr_monitor_2024"          # ← Default password
    use_global_view: true                # ← Enable Global View mode

exporters:
  otlphttp/newrelic:
    endpoint: "https://otlp.nr-data.net"
    headers:
      api-key: "${NEW_RELIC_LICENSE_KEY}"

service:
  pipelines:
    metrics:
      receivers: [yugabytedbreceiver]
      exporters: [otlphttp/newrelic]
```

## Manual Setup (Advanced)

If you prefer manual setup or need to customize:

### Step 1: Create Global Views

```bash
psql -h localhost -p 5433 -U yugabyte -d yugabyte -f setup_global_views.sql
```

### Step 2: Create Monitoring User

```bash
psql -h localhost -p 5433 -U yugabyte -d yugabyte -f create_monitoring_user.sql
```

## Troubleshooting

### Error: "psql: command not found"

Install PostgreSQL client tools:
```bash
# macOS
brew install postgresql

# Ubuntu/Debian
sudo apt-get install postgresql-client
```

### Error: "Failed to connect to YugabyteDB"

Test the connection manually:
```bash
psql -h localhost -p 5433 -U yugabyte -d yugabyte -c "SELECT version();"
```

### Error: "database gv$ already exists"

The database was created in a previous run. To recreate:
```sql
-- Connect to yugabyte database
psql -h localhost -p 5433 -U yugabyte -d yugabyte

-- Force drop the database
DROP DATABASE IF EXISTS "gv$" WITH (FORCE);
```

### Check User Privileges

```sql
-- Connect to gv$ database
\c gv$

-- Check user role
\du nr_monitor

-- List user's table privileges
SELECT table_schema, table_name, privilege_type
FROM information_schema.table_privileges
WHERE grantee = 'nr_monitor'
ORDER BY table_schema, table_name;
```

## Security Best Practices

### 1. Change Default Password

After setup, change the monitoring user password:

```sql
\c gv$
ALTER USER nr_monitor WITH PASSWORD 'your_secure_password_here';
```

Update your OpenTelemetry collector config accordingly.

### 2. Use Environment Variables

Don't hardcode passwords in config files:

```yaml
receivers:
  yugabytedbreceiver:
    password: "${MONITORING_PASSWORD}"
```

```bash
export MONITORING_PASSWORD="your_secure_password"
```

### 3. Limit Network Access

Use firewall rules to restrict access:
```bash
# Only allow from OpenTelemetry collector host
iptables -A INPUT -p tcp --dport 5433 -s <collector-ip> -j ACCEPT
```

### 4. Enable SSL/TLS (Production)

```yaml
receivers:
  yugabytedbreceiver:
    ssl_mode: "require"
```

### 5. Regular Password Rotation

```sql
-- Rotate password every 90 days
ALTER USER nr_monitor WITH PASSWORD 'new_password_here';
```

## Cleanup

To completely remove the Global View setup:

### Option 1: Quick Cleanup

```bash
psql -h localhost -p 5433 -U yugabyte -d yugabyte <<EOF
-- Force drop database and all objects
DROP DATABASE IF EXISTS "gv$" WITH (FORCE);
EOF
```

### Option 2: Manual Cleanup

```sql
-- Connect to gv$ database
\c gv$

-- Drop monitoring user
DROP OWNED BY nr_monitor CASCADE;
DROP USER IF EXISTS nr_monitor;

-- Drop Global Views
DROP VIEW IF EXISTS "gv$pg_stat_activity" CASCADE;
DROP VIEW IF EXISTS "gv$pg_stat_statements" CASCADE;
DROP VIEW IF EXISTS "gv$pg_stat_database" CASCADE;

-- Drop gv_history schema
DROP SCHEMA IF EXISTS gv_history CASCADE;

-- Drop node-specific schemas and foreign servers
DO $$
DECLARE
    schema_name TEXT;
BEGIN
    FOR schema_name IN
        SELECT nspname FROM pg_namespace WHERE nspname LIKE 'gv$%'
    LOOP
        EXECUTE format('DROP SCHEMA IF EXISTS %I CASCADE', schema_name);
        EXECUTE format('DROP SERVER IF EXISTS %I CASCADE', schema_name);
    END LOOP;
END $$;

-- Drop extension
DROP EXTENSION IF EXISTS postgres_fdw CASCADE;

-- Finally, drop the database
\c yugabyte
DROP DATABASE IF EXISTS "gv$";
```

## Architecture Diagram

```
┌─────────────────────────────────────────────────────┐
│           YugabyteDB Cluster (3 nodes)              │
├─────────────────────────────────────────────────────┤
│  Node 1          Node 2          Node 3             │
│  ├─pg_stat_*     ├─pg_stat_*     ├─pg_stat_*        │
└──┬───────────────┴───────────────┴──────────────────┘
   │
   │  Foreign Data Wrappers (postgres_fdw)
   │
┌──▼──────────────────────────────────────────────────┐
│            gv$ Database (Global Views)              │
├─────────────────────────────────────────────────────┤
│  gv$pg_stat_activity   (aggregated from all nodes)  │
│  gv$pg_stat_statements (aggregated from all nodes)  │
│  gv$pg_stat_database   (aggregated from all nodes)  │
│                                                      │
│  gv_history.global_pg_stat_statements (historical)  │
└──┬──────────────────────────────────────────────────┘
   │
   │  Read-Only Access (nr_monitor user)
   │
┌──▼──────────────────────────────────────────────────┐
│      OpenTelemetry Collector                        │
│      (yugabytedbreceiver)                           │
└──┬──────────────────────────────────────────────────┘
   │
   │  OTLP Export
   │
┌──▼──────────────────────────────────────────────────┐
│            New Relic / Observability Platform       │
└─────────────────────────────────────────────────────┘
```

## Default Credentials

**⚠️ IMPORTANT**: Change these in production!

- **Monitoring User**: `nr_monitor`
- **Monitoring Password**: `nr_monitor_2024`
- **Database**: `gv$`

## Support

For issues or questions:
- Check the [YugabyteDB documentation](https://docs.yugabyte.com/)
- Review the [OpenTelemetry Collector docs](https://opentelemetry.io/docs/collector/)
- File an issue in the repository
