// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package yugabytedbreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/yugabytedbreceiver"

// SQL queries for collecting YugabyteDB metrics

// ============================================================================
// Local Queries (single-node mode, use_global_view: false)
// Query pg_stat_activity table on the connected node only
// ============================================================================

const (
	// runningQueriesQuery counts the number of currently active queries
	runningQueriesQuery = `SELECT count(*) FROM pg_stat_activity WHERE state = 'active'`

	// activeConnectionsQuery counts the total number of active connections
	activeConnectionsQuery = `SELECT count(*) FROM pg_stat_activity`

	// connectionsByStateAndUserQuery retrieves connection counts grouped by connection state and user
	connectionsByStateAndUserQuery = `
		SELECT
			COALESCE(state, 'unknown') as state,
			COALESCE(usename, 'unknown') as usename,
			count(*) as count
		FROM pg_stat_activity
		GROUP BY state, usename`

	// activeUserCountQuery counts unique active users with client backend connections
	activeUserCountQuery = `
		SELECT
			COALESCE(usename, 'unknown') as usename
		FROM pg_stat_activity
		WHERE state = 'active'
		AND backend_type = 'client backend'
		GROUP BY usename`
)

// ============================================================================
// Global View Queries (cluster-wide mode, use_global_view: true)
// Query gv$pg_stat_activity view that aggregates data from all nodes
// Includes node metadata: gv$host, gv$zone, gv$region, gv$cloud
// ============================================================================

const (
	// globalViewRunningQueriesQuery counts active queries per node
	globalViewRunningQueriesQuery = `
		SELECT
			"gv$host",
			"gv$zone",
			"gv$region",
			"gv$cloud",
			count(*) as count
		FROM gv_history."gv$pg_stat_activity"
		WHERE state = 'active'
		GROUP BY "gv$host", "gv$zone", "gv$region", "gv$cloud"`

	// globalViewActiveConnectionsQuery counts total connections per node
	globalViewActiveConnectionsQuery = `
		SELECT
			"gv$host",
			"gv$zone",
			"gv$region",
			"gv$cloud",
			count(*) as count
		FROM gv_history."gv$pg_stat_activity"
		GROUP BY "gv$host", "gv$zone", "gv$region", "gv$cloud"`

	// globalViewConnectionsByStateAndUserQuery retrieves connection counts by state, user, and node
	globalViewConnectionsByStateAndUserQuery = `
		SELECT
			"gv$host",
			"gv$zone",
			"gv$region",
			"gv$cloud",
			COALESCE(state, 'unknown') as state,
			COALESCE(usename, 'unknown') as usename,
			count(*) as count
		FROM gv_history."gv$pg_stat_activity"
		GROUP BY "gv$host", "gv$zone", "gv$region", "gv$cloud", state, usename`

	// globalViewActiveUserCountQuery counts unique active users per node with client backend connections
	globalViewActiveUserCountQuery = `
		SELECT
			"gv$host",
			"gv$zone",
			"gv$region",
			"gv$cloud",
			COALESCE(usename, 'unknown') as usename,
			COUNT(*) as user_session_count
		FROM gv_history."gv$pg_stat_activity"
		WHERE state = 'active'
		AND backend_type = 'client backend'
		GROUP BY "gv$host", "gv$zone", "gv$region", "gv$cloud", usename`
)
