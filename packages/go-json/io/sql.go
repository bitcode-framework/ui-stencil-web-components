package io

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"
)

// SQLModule provides SQL database functions for go-json programs.
type SQLModule struct {
	security *SecurityConfig
	config   map[string]any

	hostedDB *sql.DB

	pools   map[string]*sql.DB
	poolsMu sync.Mutex

	mu sync.Mutex
	tx *sql.Tx
	sp int
}

// NewSQLModule creates a new SQL I/O module in standalone mode.
func NewSQLModule(security *SecurityConfig) *SQLModule {
	if security == nil {
		security = DefaultSecurityConfig()
	}
	return &SQLModule{
		security: security,
		pools:    make(map[string]*sql.DB),
	}
}

// NewSQLModuleHosted creates a new SQL I/O module in hosted mode with a pre-configured connection.
func NewSQLModuleHosted(security *SecurityConfig, db *sql.DB) *SQLModule {
	m := NewSQLModule(security)
	m.hostedDB = db
	return m
}

func (m *SQLModule) Name() string { return "sql" }

func (m *SQLModule) SetConfig(cfg map[string]any) { m.config = cfg }

// Close closes all pooled connections and rolls back any active transaction.
func (m *SQLModule) Close() error {
	m.mu.Lock()
	if m.tx != nil {
		m.tx.Rollback()
		m.tx = nil
		m.sp = 0
	}
	m.mu.Unlock()

	m.poolsMu.Lock()
	defer m.poolsMu.Unlock()

	var firstErr error
	for dsn, db := range m.pools {
		if err := db.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		delete(m.pools, dsn)
	}
	return firstErr
}

// Cleanup rolls back any uncommitted transaction without closing pools.
func (m *SQLModule) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.tx != nil {
		m.tx.Rollback()
		m.tx = nil
		m.sp = 0
	}
}

func (m *SQLModule) Functions() map[string]any {
	return map[string]any{
		"query":    m.sqlQuery,
		"execute":  m.sqlExecute,
		"begin":    m.sqlBegin,
		"commit":   m.sqlCommit,
		"rollback": m.sqlRollback,
	}
}

var defaultBlockedKeywords = []string{"DROP", "TRUNCATE", "ALTER", "GRANT", "REVOKE", "CREATE INDEX", "CREATE TABLE"}

func (m *SQLModule) validateQuery(query string) error {
	maxLen := m.security.SQL.MaxQueryLength
	if maxLen <= 0 {
		maxLen = 10000
	}
	if len(query) > maxLen {
		return fmt.Errorf("sql: query exceeds max length (%d chars, max %d)", len(query), maxLen)
	}

	blocked := m.security.SQL.BlockedKeywords
	if blocked == nil {
		blocked = defaultBlockedKeywords
	}

	upper := strings.ToUpper(strings.TrimSpace(query))
	for _, kw := range blocked {
		if strings.Contains(upper, strings.ToUpper(kw)) {
			return fmt.Errorf("sql: query contains blocked keyword '%s'", kw)
		}
	}

	return nil
}

func (m *SQLModule) sqlQuery(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("sql.query: query is required")
	}

	query, args, dsn, err := m.parseQueryParams(params)
	if err != nil {
		return nil, err
	}

	if err := m.validateQuery(query); err != nil {
		return nil, err
	}

	db, err := m.getDB(dsn)
	if err != nil {
		return nil, err
	}

	effectiveDSN := dsn
	if effectiveDSN == "" {
		effectiveDSN = m.security.SQL.DefaultDSN
	}
	driver := detectDriverFromDSN(effectiveDSN)
	query, translatedArgs, translateErr := TranslateQuery(query, driver, args)
	if translateErr != nil {
		return nil, fmt.Errorf("sql.query: %w", translateErr)
	}

	timeout := time.Duration(m.security.SQL.MaxQueryTime) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var rows *sql.Rows
	m.mu.Lock()
	if m.tx != nil {
		rows, err = m.tx.QueryContext(ctx, query, translatedArgs...)
	} else {
		rows, err = db.QueryContext(ctx, query, translatedArgs...)
	}
	m.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("sql.query: %s", err.Error())
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("sql.query: %s", err.Error())
	}

	var result []any
	maxRows := m.security.SQL.MaxRows
	if maxRows <= 0 {
		maxRows = 10000
	}
	rowCount := 0

	for rows.Next() {
		if rowCount >= maxRows {
			break
		}

		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("sql.query: %s", err.Error())
		}

		row := make(map[string]any)
		for i, col := range columns {
			row[col] = convertSQLValue(values[i])
		}
		result = append(result, row)
		rowCount++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sql.query: %s", err.Error())
	}

	if result == nil {
		result = []any{}
	}

	colsAny := make([]any, len(columns))
	for i, c := range columns {
		colsAny[i] = c
	}

	return map[string]any{
		"rows":    result,
		"columns": colsAny,
		"count":   rowCount,
	}, nil
}

func (m *SQLModule) sqlExecute(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("sql.execute: query is required")
	}

	query, args, dsn, err := m.parseQueryParams(params)
	if err != nil {
		return nil, err
	}

	if err := m.validateQuery(query); err != nil {
		return nil, err
	}

	db, err := m.getDB(dsn)
	if err != nil {
		return nil, err
	}

	effectiveDSN := dsn
	if effectiveDSN == "" {
		effectiveDSN = m.security.SQL.DefaultDSN
	}
	driver := detectDriverFromDSN(effectiveDSN)
	query, translatedArgs, translateErr := TranslateQuery(query, driver, args)
	if translateErr != nil {
		return nil, fmt.Errorf("sql.execute: %w", translateErr)
	}

	timeout := time.Duration(m.security.SQL.MaxQueryTime) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var res sql.Result
	m.mu.Lock()
	if m.tx != nil {
		res, err = m.tx.ExecContext(ctx, query, translatedArgs...)
	} else {
		res, err = db.ExecContext(ctx, query, translatedArgs...)
	}
	m.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("sql.execute: %s", err.Error())
	}

	rowsAffected, _ := res.RowsAffected()
	lastInsertID, _ := res.LastInsertId()

	return map[string]any{
		"rows_affected":  rowsAffected,
		"last_insert_id": lastInsertID,
	}, nil
}

func (m *SQLModule) sqlBegin(params ...any) (any, error) {
	db, err := m.getDB("")
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tx != nil {
		// Nested transaction — use savepoint.
		m.sp++
		spName := fmt.Sprintf("sp_%d", m.sp)
		_, err := m.tx.Exec(fmt.Sprintf("SAVEPOINT %s", spName))
		if err != nil {
			return nil, fmt.Errorf("sql.begin: %s", err.Error())
		}
		return nil, nil
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("sql.begin: %s", err.Error())
	}
	m.tx = tx
	m.sp = 0
	return nil, nil
}

func (m *SQLModule) sqlCommit(params ...any) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tx == nil {
		return nil, fmt.Errorf("sql.commit: no active transaction")
	}

	if m.sp > 0 {
		spName := fmt.Sprintf("sp_%d", m.sp)
		_, err := m.tx.Exec(fmt.Sprintf("RELEASE SAVEPOINT %s", spName))
		m.sp--
		if err != nil {
			return nil, fmt.Errorf("sql.commit: %s", err.Error())
		}
		return nil, nil
	}

	err := m.tx.Commit()
	m.tx = nil
	m.sp = 0
	if err != nil {
		return nil, fmt.Errorf("sql.commit: %s", err.Error())
	}
	return nil, nil
}

func (m *SQLModule) sqlRollback(params ...any) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tx == nil {
		return nil, fmt.Errorf("sql.rollback: no active transaction")
	}

	if m.sp > 0 {
		spName := fmt.Sprintf("sp_%d", m.sp)
		_, err := m.tx.Exec(fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", spName))
		m.sp--
		if err != nil {
			return nil, fmt.Errorf("sql.rollback: %s", err.Error())
		}
		return nil, nil
	}

	err := m.tx.Rollback()
	m.tx = nil
	m.sp = 0
	if err != nil {
		return nil, fmt.Errorf("sql.rollback: %s", err.Error())
	}
	return nil, nil
}

func (m *SQLModule) parseQueryParams(params []any) (string, any, string, error) {
	query, ok := params[0].(string)
	if !ok {
		return "", nil, "", fmt.Errorf("sql: query must be a string")
	}

	var args any
	var dsn string

	for i := 1; i < len(params); i++ {
		switch v := params[i].(type) {
		case []any:
			args = v
		case map[string]any:
			if d, ok := v["dsn"].(string); ok {
				dsn = d
			}
			if a, ok := v["args"]; ok {
				args = a
			} else {
				args = v
			}
		case string:
			dsn = v
		}
	}

	return query, args, dsn, nil
}

func (m *SQLModule) getDB(dsn string) (*sql.DB, error) {
	if m.hostedDB != nil {
		return m.hostedDB, nil
	}

	if dsn == "" {
		dsn = m.security.SQL.DefaultDSN
	}
	if dsn == "" {
		return nil, fmt.Errorf("sql: DSN is required — set DefaultDSN in config or pass dsn parameter")
	}

	m.poolsMu.Lock()
	defer m.poolsMu.Unlock()

	if db, ok := m.pools[dsn]; ok {
		return db, nil
	}

	maxPools := m.security.SQL.MaxPools
	if maxPools <= 0 {
		maxPools = 5
	}
	if len(m.pools) >= maxPools {
		return nil, fmt.Errorf("sql: max pool limit reached (%d)", maxPools)
	}

	driver := detectDriverFromDSN(dsn)
	if err := m.security.ValidateSQLDriver(driver); err != nil {
		return nil, err
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("sql: cannot open database: %s", err.Error())
	}

	maxPoolSize := m.security.SQL.MaxPoolSize
	if maxPoolSize <= 0 {
		maxPoolSize = 10
	}
	db.SetMaxOpenConns(maxPoolSize)
	db.SetMaxIdleConns(maxPoolSize / 2)
	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetConnMaxLifetime(30 * time.Minute)

	m.pools[dsn] = db
	return db, nil
}

func detectDriverFromDSN(dsn string) string {
	switch {
	case strings.HasPrefix(dsn, "postgres://"), strings.HasPrefix(dsn, "postgresql://"):
		return "postgres"
	case strings.HasPrefix(dsn, "mysql://"):
		return "mysql"
	case strings.HasPrefix(dsn, "sqlserver://"):
		return "sqlserver"
	case strings.HasPrefix(dsn, "oracle://"):
		return "oracle"
	case strings.HasPrefix(dsn, "sqlite3://"), strings.HasPrefix(dsn, "sqlite://"), strings.HasPrefix(dsn, "file:"):
		return "sqlite"
	default:
		return "sqlite"
	}
}

func convertSQLValue(v any) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []byte:
		return string(val)
	case int64:
		return val
	case float64:
		return val
	case bool:
		return val
	case string:
		return val
	case time.Time:
		return val.Format(time.RFC3339)
	default:
		return fmt.Sprintf("%v", val)
	}
}
