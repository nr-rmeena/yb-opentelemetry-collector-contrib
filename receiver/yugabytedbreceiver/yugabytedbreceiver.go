// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package yugabytedbreceiver

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/yugabytedbreceiver/internal/metadata"
)

// yugabytedbReceiver collects metrics from YugabyteDB
type yugabytedbReceiver struct {
	config         *Config
	consumer       consumer.Metrics
	cancel         context.CancelFunc
	metricsBuilder *metadata.MetricsBuilder
	logger         *zap.Logger
}

// connectionMetric represents connection count grouped by state and user
type connectionMetric struct {
	state string
	user  string
	count int64
}

// createMetricsReceiver creates a new YugabyteDB metrics receiver
func createMetricsReceiver(_ context.Context, settings receiver.Settings, cfg component.Config, consumer consumer.Metrics) (receiver.Metrics, error) {
	c := cfg.(*Config)
	mbConfig := metadata.DefaultMetricsBuilderConfig()
	mb := metadata.NewMetricsBuilder(mbConfig, settings)
	return &yugabytedbReceiver{
		config:         c,
		consumer:       consumer,
		metricsBuilder: mb,
		logger:         settings.Logger,
	}, nil
}

// Start begins the metric collection process
func (r *yugabytedbReceiver) Start(ctx context.Context, _ component.Host) error {
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	go r.scrapeLoop(ctx)
	return nil
}

// Shutdown stops the metric collection process
func (r *yugabytedbReceiver) Shutdown(_ context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}
	return nil
}

// scrapeLoop continuously collects metrics at regular intervals
func (r *yugabytedbReceiver) scrapeLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.collectMetrics(ctx)
		}
	}
}

// collectMetrics gathers all metrics from YugabyteDB and emits them
func (r *yugabytedbReceiver) collectMetrics(ctx context.Context) {
	db, err := r.connectToDatabase()
	if err != nil {
		r.logger.Error("failed to connect to YugabyteDB", zap.Error(err))
		return
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			r.logger.Error("failed to close database connection", zap.Error(closeErr))
		}
	}()
	now := pcommon.NewTimestampFromTime(time.Now())

	// Collect metrics from Global Views
	r.collectRunningQueries(ctx, db, now)
	r.collectActiveConnections(ctx, db, now)
	r.collectConnectionsByStateAndUser(ctx, db, now)
	r.collectActiveUserCount(ctx, db, now)

	// Emit all metrics to the consumer
	r.emitMetrics(ctx)
}

// connectToDatabase establishes a connection to YugabyteDB
func (r *yugabytedbReceiver) connectToDatabase() (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=require",
		r.config.Host, r.config.Port, r.config.User, r.config.Password, r.config.Database)
	return sql.Open("postgres", dsn)
}

// normalizeConnectionState normalizes PostgreSQL connection states to our metric format
func (r *yugabytedbReceiver) normalizeConnectionState(state string) string {
	switch state {
	case "idle in transaction", "idle in transaction (aborted)":
		return "idle_in_transaction"
	case "":
		return "unknown"
	default:
		return state
	}
}

// collectRunningQueries collects running queries count from Global Views
func (r *yugabytedbReceiver) collectRunningQueries(ctx context.Context, db *sql.DB, now pcommon.Timestamp) {
	rows, err := db.QueryContext(ctx, globalViewRunningQueriesQuery)
	if err != nil {
		r.logger.Error("failed to query running queries", zap.Error(err))
		return
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			r.logger.Error("failed to close rows", zap.Error(closeErr))
		}
	}()

	totalCount := int64(0)
	for rows.Next() {
		var host, zone, region, cloud string
		var count int64
		err := rows.Scan(&host, &zone, &region, &cloud, &count)
		if err != nil {
			r.logger.Error("failed to scan row", zap.Error(err))
			continue
		}
		totalCount += count
		r.logger.Debug("running queries", zap.String("host", host), zap.Int64("count", count))
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("error iterating rows", zap.Error(err))
		return
	}

	r.metricsBuilder.RecordYugabytedbPgStatActivityRunningQueriesDataPoint(now, totalCount)
	r.logger.Debug("collected running queries", zap.Int64("total", totalCount))
}

// collectActiveConnections collects active connections count from Global Views
func (r *yugabytedbReceiver) collectActiveConnections(ctx context.Context, db *sql.DB, now pcommon.Timestamp) {
	rows, err := db.QueryContext(ctx, globalViewActiveConnectionsQuery)
	if err != nil {
		r.logger.Error("failed to query active connections", zap.Error(err))
		return
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			r.logger.Error("failed to close rows", zap.Error(closeErr))
		}
	}()

	totalCount := int64(0)
	for rows.Next() {
		var host, zone, region, cloud string
		var count int64
		err := rows.Scan(&host, &zone, &region, &cloud, &count)
		if err != nil {
			r.logger.Error("failed to scan row", zap.Error(err))
			continue
		}
		totalCount += count
		r.logger.Debug("active connections", zap.String("host", host), zap.Int64("count", count))
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("error iterating rows", zap.Error(err))
		return
	}

	r.metricsBuilder.RecordYugabytedbPgStatActivityActiveConnectionsDataPoint(now, totalCount)
	r.logger.Debug("collected active connections", zap.Int64("total", totalCount))
}

// collectConnectionsByStateAndUser collects connection counts by state and user from Global Views
func (r *yugabytedbReceiver) collectConnectionsByStateAndUser(ctx context.Context, db *sql.DB, now pcommon.Timestamp) {
	rows, err := db.QueryContext(ctx, globalViewConnectionsByStateAndUserQuery)
	if err != nil {
		r.logger.Error("failed to query connections by state and user", zap.Error(err))
		return
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			r.logger.Error("failed to close rows", zap.Error(closeErr))
		}
	}()

	// Aggregate by state and user across all nodes
	aggregates := make(map[string]map[string]int64) // state -> user -> count

	for rows.Next() {
		var host, zone, region, cloud, state, user string
		var count int64
		err := rows.Scan(&host, &zone, &region, &cloud, &state, &user, &count)
		if err != nil {
			r.logger.Error("failed to scan row", zap.Error(err))
			continue
		}

		normalizedState := r.normalizeConnectionState(state)
		if aggregates[normalizedState] == nil {
			aggregates[normalizedState] = make(map[string]int64)
		}
		aggregates[normalizedState][user] += count

		r.logger.Debug("connection",
			zap.String("host", host),
			zap.String("state", normalizedState),
			zap.String("user", user),
			zap.Int64("count", count))
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("error iterating rows", zap.Error(err))
		return
	}

	// Record aggregated metrics
	for state, users := range aggregates {
		for user, count := range users {
			r.metricsBuilder.RecordYugabytedbConnectionCountDataPoint(now, count, state, user)
		}
	}

	r.logger.Debug("collected connections by state and user", zap.Int("state_count", len(aggregates)))
}

// collectActiveUserCount collects unique active user count from Global Views
func (r *yugabytedbReceiver) collectActiveUserCount(ctx context.Context, db *sql.DB, now pcommon.Timestamp) {
	rows, err := db.QueryContext(ctx, globalViewActiveUserCountQuery)
	if err != nil {
		r.logger.Error("failed to query active user count", zap.Error(err))
		return
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			r.logger.Error("failed to close rows", zap.Error(closeErr))
		}
	}()

	// Aggregate session counts per user across all nodes
	userSessionCounts := make(map[string]int64) // username -> total session count

	for rows.Next() {
		var host, zone, region, cloud, user string
		var userSessionCount int64
		err := rows.Scan(&host, &zone, &region, &cloud, &user, &userSessionCount)
		if err != nil {
			r.logger.Error("failed to scan row", zap.Error(err))
			continue
		}

		userSessionCounts[user] += userSessionCount

		r.logger.Debug("active user session",
			zap.String("host", host),
			zap.String("user", user),
			zap.Int64("session_count", userSessionCount))
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("error iterating rows", zap.Error(err))
		return
	}

	// Record metric per user with their aggregated session count
	for user, count := range userSessionCounts {
		r.metricsBuilder.RecordYugabytedbActiveUsersCountDataPoint(now, count, user)
	}

	r.logger.Debug("collected active user count", zap.Int("unique_users", len(userSessionCounts)))
}

// emitMetrics sends all collected metrics to the consumer
func (r *yugabytedbReceiver) emitMetrics(ctx context.Context) {
	metrics := r.metricsBuilder.Emit()
	if err := r.consumer.ConsumeMetrics(ctx, metrics); err != nil {
		r.logger.Error("failed to consume metrics", zap.Error(err))
	}
}
