// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package yugabytedbreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/yugabytedbreceiver"

// SQL queries for collecting YugabyteDB metrics from pg_stat_activity

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
)
