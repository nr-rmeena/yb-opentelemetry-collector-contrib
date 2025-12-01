// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package yugabytedbreceiver

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/yugabytedbreceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
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
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
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
	// Collect all metrics
	r.collectRunningQueries(ctx, db, now)
	r.collectActiveConnections(ctx, db, now)
	r.collectConnectionsByStateAndUser(ctx, db, now)
	// Emit all metrics to the consumer
	r.emitMetrics(ctx)
}

// connectToDatabase establishes a connection to YugabyteDB
func (r *yugabytedbReceiver) connectToDatabase() (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		r.config.Host, r.config.Port, r.config.User, r.config.Password, r.config.Database)
	return sql.Open("postgres", dsn)
}

// collectRunningQueries collects the count of currently running queries (active state)
func (r *yugabytedbReceiver) collectRunningQueries(ctx context.Context, db *sql.DB, now pcommon.Timestamp) {
var count int64
row := db.QueryRowContext(ctx, runningQueriesQuery)
	if err := row.Scan(&count); err != nil {
		r.logger.Error("failed to scan running queries count", zap.Error(err))
		return
	}
	r.metricsBuilder.RecordYugabytedbPgStatActivityRunningQueriesDataPoint(now, count)
	r.logger.Debug("collected running queries metric", zap.Int64("count", count))
}

// collectActiveConnections collects the total count of active connections
func (r *yugabytedbReceiver) collectActiveConnections(ctx context.Context, db *sql.DB, now pcommon.Timestamp) {
	var count int64
	row := db.QueryRowContext(ctx, activeConnectionsQuery)
	if err := row.Scan(&count); err != nil {
		r.logger.Error("failed to scan active connections count", zap.Error(err))
		return
	}
	r.metricsBuilder.RecordYugabytedbPgStatActivityActiveConnectionsDataPoint(now, count)
	r.logger.Debug("collected active connections metric", zap.Int64("count", count))
}

// collectConnectionsByStateAndUser collects connection counts grouped by state and user
func (r *yugabytedbReceiver) collectConnectionsByStateAndUser(ctx context.Context, db *sql.DB, now pcommon.Timestamp) {
	rows, err := db.QueryContext(ctx, connectionsByStateAndUserQuery)
	if err != nil {
		r.logger.Error("failed to query connection metrics by state and user", zap.Error(err))
		return
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			r.logger.Error("failed to close rows", zap.Error(closeErr))
		}
	}()
	metricsCollected := 0
	for rows.Next() {
		metric, err := r.scanConnectionMetric(rows)
		if err != nil {
			r.logger.Error("failed to scan connection metric row", zap.Error(err))
			continue
		}
		normalizedState := r.normalizeConnectionState(metric.state)
		r.metricsBuilder.RecordYugabytedbConnectionCountDataPoint(
			now,
			metric.count,
			normalizedState,
			metric.user,
		)
		metricsCollected++
	}
	if err := rows.Err(); err != nil {
		r.logger.Error("error iterating connection metric rows", zap.Error(err))
		return
	}
	r.logger.Debug("collected connection state metrics",
		zap.Int("metrics_count", metricsCollected))
}

// scanConnectionMetric scans a single row into a connectionMetric struct
func (r *yugabytedbReceiver) scanConnectionMetric(rows *sql.Rows) (connectionMetric, error) {
	var metric connectionMetric
	err := rows.Scan(&metric.state, &metric.user, &metric.count)
	return metric, err
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

// emitMetrics sends all collected metrics to the consumer
func (r *yugabytedbReceiver) emitMetrics(ctx context.Context) {
	metrics := r.metricsBuilder.Emit()
	if err := r.consumer.ConsumeMetrics(ctx, metrics); err != nil {
		r.logger.Error("failed to consume metrics", zap.Error(err))
	}
}
