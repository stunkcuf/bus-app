package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

// QueryOptimizer provides database query optimization features
type QueryOptimizer struct {
	db                 *sql.DB
	slowQueryThreshold time.Duration
}

// NewQueryOptimizer creates a new query optimizer
func NewQueryOptimizer(db *sql.DB) *QueryOptimizer {
	return &QueryOptimizer{
		db:                 db,
		slowQueryThreshold: 1 * time.Second,
	}
}

// SlowQuery represents a slow query for analysis
type SlowQuery struct {
	Query         string
	Duration      time.Duration
	RowsAffected  int64
	ExecutionTime time.Time
}

// QueryStats represents statistics for a query pattern
type QueryStats struct {
	Query         string
	TotalCalls    int
	TotalDuration time.Duration
	AvgDuration   time.Duration
	MaxDuration   time.Duration
	MinDuration   time.Duration
}

// DatabaseStats represents overall database statistics
type DatabaseStats struct {
	TableSizes      map[string]TableSize
	IndexUsage      []IndexUsage
	ConnectionStats ConnectionStats
	CacheHitRatio   float64
}

// TableSize represents the size of a database table
type TableSize struct {
	TableName string
	RowCount  int64
	TotalSize string
	IndexSize string
	ToastSize string
}

// IndexUsage represents index usage statistics
type IndexUsage struct {
	SchemaName string
	TableName  string
	IndexName  string
	IndexScans int64
	IndexSize  string
	Unused     bool
}

// ConnectionStats represents database connection statistics
type ConnectionStats struct {
	ActiveConnections  int
	IdleConnections    int
	WaitingConnections int
	MaxConnections     int
	ConnectionsInUse   float64
}

// AnalyzeSlowQueries analyzes and returns slow queries
func (qo *QueryOptimizer) AnalyzeSlowQueries(ctx context.Context) ([]SlowQuery, error) {
	// In a real implementation, you would track queries and their execution times
	// For now, we'll analyze the pg_stat_statements if available

	query := `
		SELECT 
			query,
			mean_time::numeric / 1000 as avg_duration_seconds,
			calls,
			total_time::numeric / 1000 as total_duration_seconds
		FROM pg_stat_statements
		WHERE mean_time > $1
		ORDER BY mean_time DESC
		LIMIT 20
	`

	rows, err := qo.db.QueryContext(ctx, query, qo.slowQueryThreshold.Seconds()*1000)
	if err != nil {
		// pg_stat_statements might not be enabled
		log.Printf("Could not analyze slow queries (pg_stat_statements may not be enabled): %v", err)
		return nil, nil
	}
	defer rows.Close()

	var slowQueries []SlowQuery
	for rows.Next() {
		var queryText string
		var avgDuration, totalDuration float64
		var calls int64

		err := rows.Scan(&queryText, &avgDuration, &calls, &totalDuration)
		if err != nil {
			continue
		}

		slowQueries = append(slowQueries, SlowQuery{
			Query:         queryText,
			Duration:      time.Duration(avgDuration * float64(time.Second)),
			RowsAffected:  calls,
			ExecutionTime: time.Now(),
		})
	}

	return slowQueries, nil
}

// GetDatabaseStats retrieves comprehensive database statistics
func (qo *QueryOptimizer) GetDatabaseStats(ctx context.Context) (*DatabaseStats, error) {
	stats := &DatabaseStats{
		TableSizes: make(map[string]TableSize),
	}

	// Get table sizes
	tableSizeQuery := `
		SELECT 
			schemaname,
			tablename,
			pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS total_size,
			pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) AS table_size,
			pg_size_pretty(pg_indexes_size(schemaname||'.'||tablename)) AS indexes_size,
			n_live_tup as row_count
		FROM pg_stat_user_tables
		WHERE schemaname = 'public'
		ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
	`

	rows, err := qo.db.QueryContext(ctx, tableSizeQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get table sizes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, tableName, totalSize, tableSize, indexSize string
		var rowCount int64

		err := rows.Scan(&schemaName, &tableName, &totalSize, &tableSize, &indexSize, &rowCount)
		if err != nil {
			continue
		}

		stats.TableSizes[tableName] = TableSize{
			TableName: tableName,
			RowCount:  rowCount,
			TotalSize: totalSize,
			IndexSize: indexSize,
		}
	}

	// Get index usage
	indexUsageQuery := `
		SELECT 
			schemaname,
			tablename,
			indexname,
			idx_scan,
			pg_size_pretty(pg_relation_size(indexrelid)) as index_size,
			idx_scan = 0 as unused
		FROM pg_stat_user_indexes
		WHERE schemaname = 'public'
		ORDER BY idx_scan
	`

	rows, err = qo.db.QueryContext(ctx, indexUsageQuery)
	if err != nil {
		log.Printf("Failed to get index usage: %v", err)
	} else {
		defer rows.Close()

		for rows.Next() {
			var usage IndexUsage
			err := rows.Scan(&usage.SchemaName, &usage.TableName, &usage.IndexName,
				&usage.IndexScans, &usage.IndexSize, &usage.Unused)
			if err != nil {
				continue
			}
			stats.IndexUsage = append(stats.IndexUsage, usage)
		}
	}

	// Get connection stats
	connStatsQuery := `
		SELECT 
			count(*) FILTER (WHERE state = 'active') as active,
			count(*) FILTER (WHERE state = 'idle') as idle,
			count(*) FILTER (WHERE wait_event_type = 'Lock') as waiting,
			setting::int as max_connections
		FROM pg_stat_activity, pg_settings
		WHERE pg_settings.name = 'max_connections'
		GROUP BY setting
	`

	var active, idle, waiting, maxConn int
	err = qo.db.QueryRowContext(ctx, connStatsQuery).Scan(&active, &idle, &waiting, &maxConn)
	if err != nil {
		log.Printf("Failed to get connection stats: %v", err)
	} else {
		stats.ConnectionStats = ConnectionStats{
			ActiveConnections:  active,
			IdleConnections:    idle,
			WaitingConnections: waiting,
			MaxConnections:     maxConn,
			ConnectionsInUse:   float64(active+idle) / float64(maxConn) * 100,
		}
	}

	// Get cache hit ratio
	cacheQuery := `
		SELECT 
			sum(heap_blks_hit)::float / (sum(heap_blks_hit) + sum(heap_blks_read)) * 100 as cache_hit_ratio
		FROM pg_statio_user_tables
	`

	err = qo.db.QueryRowContext(ctx, cacheQuery).Scan(&stats.CacheHitRatio)
	if err != nil {
		log.Printf("Failed to get cache hit ratio: %v", err)
		stats.CacheHitRatio = 0
	}

	return stats, nil
}

// OptimizeQuery provides query optimization suggestions
func (qo *QueryOptimizer) OptimizeQuery(ctx context.Context, query string) ([]string, error) {
	suggestions := []string{}

	// Use EXPLAIN ANALYZE to get query plan
	explainQuery := fmt.Sprintf("EXPLAIN (ANALYZE, BUFFERS) %s", query)

	rows, err := qo.db.QueryContext(ctx, explainQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze query: %w", err)
	}
	defer rows.Close()

	var planLines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			continue
		}
		planLines = append(planLines, line)
	}

	// Analyze the plan for common issues
	planText := strings.Join(planLines, "\n")

	// Check for sequential scans on large tables
	if strings.Contains(planText, "Seq Scan") {
		suggestions = append(suggestions, "Consider adding an index to avoid sequential scan")
	}

	// Check for nested loops with high cost
	if strings.Contains(planText, "Nested Loop") && strings.Contains(planText, "cost=") {
		suggestions = append(suggestions, "High cost nested loop detected - consider query restructuring")
	}

	// Check for missing indexes
	if strings.Contains(planText, "Filter:") {
		suggestions = append(suggestions, "Filter condition detected - consider adding an index on filter columns")
	}

	// Check for sorting operations
	if strings.Contains(planText, "Sort") {
		suggestions = append(suggestions, "Sort operation detected - consider adding an index for ORDER BY columns")
	}

	return suggestions, nil
}

// CreateMissingIndexes creates indexes that would improve performance
func (qo *QueryOptimizer) CreateMissingIndexes(ctx context.Context) error {
	// Additional indexes that could improve performance based on common query patterns
	additionalIndexes := []string{
		// Composite indexes for common JOIN operations
		`CREATE INDEX IF NOT EXISTS idx_students_route_driver ON students(route_id, driver) WHERE active = true`,
		`CREATE INDEX IF NOT EXISTS idx_route_assignments_composite ON route_assignments(driver, bus_id, route_id)`,
		`CREATE INDEX IF NOT EXISTS idx_driver_logs_composite ON driver_logs(driver, date, period)`,

		// Partial indexes for common WHERE conditions
		`CREATE INDEX IF NOT EXISTS idx_buses_active ON buses(bus_id) WHERE status = 'active'`,
		`CREATE INDEX IF NOT EXISTS idx_vehicles_active ON vehicles(vehicle_id) WHERE status = 'active'`,
		`CREATE INDEX IF NOT EXISTS idx_students_active ON students(student_id) WHERE active = true`,

		// Indexes for maintenance queries
		`CREATE INDEX IF NOT EXISTS idx_buses_maintenance_status ON buses(bus_id, oil_status, tire_status)`,
		`CREATE INDEX IF NOT EXISTS idx_vehicles_maintenance_status ON vehicles(vehicle_id, oil_status, tire_status)`,

		// Indexes for reporting
		`CREATE INDEX IF NOT EXISTS idx_trip_logs_reporting ON trip_logs(date, driver, bus_id)`,
		`CREATE INDEX IF NOT EXISTS idx_mileage_composite ON mileage_reports(year, month, vehicle_id)`,

		// Text search indexes for searching
		`CREATE INDEX IF NOT EXISTS idx_students_name_search ON students USING gin(to_tsvector('english', name))`,
		`CREATE INDEX IF NOT EXISTS idx_ecse_students_name_search ON ecse_students USING gin(to_tsvector('english', first_name || ' ' || last_name))`,
	}

	for _, indexSQL := range additionalIndexes {
		log.Printf("Creating index: %s", indexSQL)
		if _, err := qo.db.ExecContext(ctx, indexSQL); err != nil {
			log.Printf("Failed to create index: %v", err)
			// Continue with other indexes even if one fails
		}
	}

	// Update table statistics for query planner
	tables := []string{"buses", "vehicles", "students", "routes", "route_assignments",
		"driver_logs", "trip_logs", "mileage_reports", "users"}

	for _, table := range tables {
		if _, err := qo.db.ExecContext(ctx, fmt.Sprintf("ANALYZE %s", table)); err != nil {
			log.Printf("Failed to analyze table %s: %v", table, err)
		}
	}

	return nil
}

// VacuumDatabase performs database maintenance
func (qo *QueryOptimizer) VacuumDatabase(ctx context.Context, full bool) error {
	vacuumCmd := "VACUUM ANALYZE"
	if full {
		vacuumCmd = "VACUUM FULL ANALYZE"
	}

	log.Printf("Running %s on database...", vacuumCmd)

	// For VACUUM, we need to use a separate connection without a transaction
	conn, err := qo.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer conn.Close()

	_, err = conn.ExecContext(ctx, vacuumCmd)
	if err != nil {
		return fmt.Errorf("vacuum failed: %w", err)
	}

	log.Printf("Database maintenance completed")
	return nil
}

// GetQueryRecommendations provides specific recommendations based on current database state
func (qo *QueryOptimizer) GetQueryRecommendations(ctx context.Context) ([]string, error) {
	recommendations := []string{}

	stats, err := qo.GetDatabaseStats(ctx)
	if err != nil {
		return nil, err
	}

	// Check cache hit ratio
	if stats.CacheHitRatio < 90 {
		recommendations = append(recommendations,
			fmt.Sprintf("Cache hit ratio is %.2f%% - consider increasing shared_buffers", stats.CacheHitRatio))
	}

	// Check for unused indexes
	unusedCount := 0
	for _, idx := range stats.IndexUsage {
		if idx.Unused {
			unusedCount++
		}
	}
	if unusedCount > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Found %d unused indexes - consider removing them to improve write performance", unusedCount))
	}

	// Check connection usage
	if stats.ConnectionStats.ConnectionsInUse > 80 {
		recommendations = append(recommendations,
			fmt.Sprintf("Connection usage at %.2f%% - consider increasing max_connections or using connection pooling",
				stats.ConnectionStats.ConnectionsInUse))
	}

	// Check table sizes
	for tableName, size := range stats.TableSizes {
		if size.RowCount > 100000 {
			recommendations = append(recommendations,
				fmt.Sprintf("Table '%s' has %d rows - ensure proper indexing and consider partitioning",
					tableName, size.RowCount))
		}
	}

	return recommendations, nil
}

// InitializeQueryOptimizer sets up the query optimizer
func InitializeQueryOptimizer() *QueryOptimizer {
	if db == nil {
		log.Printf("Database not initialized for query optimizer")
		return nil
	}

	optimizer := NewQueryOptimizer(db.DB)

	// Create missing indexes in the background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := optimizer.CreateMissingIndexes(ctx); err != nil {
			log.Printf("Failed to create missing indexes: %v", err)
		}
	}()

	return optimizer
}
