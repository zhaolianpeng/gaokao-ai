package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"gaokao-ai/backend/logging"
)

type observedDB struct {
	raw *sql.DB
}

type observedRows struct {
	rows      *sql.Rows
	query     string
	args      []any
	startedAt time.Time
	rowCount  int
	previews  [][]any
	finalized bool
	mu        sync.Mutex
}

type observedRow struct {
	row       *sql.Row
	query     string
	args      []any
	startedAt time.Time
}

func observeDB(db *sql.DB) *observedDB {
	if db == nil {
		return nil
	}
	return &observedDB{raw: db}
}

func (db *observedDB) Exec(query string, args ...any) (sql.Result, error) {
	return db.exec(context.Background(), query, args...)
}

func (db *observedDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.exec(ctx, query, args...)
}

func (db *observedDB) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	startedAt := time.Now()
	result, err := db.raw.ExecContext(ctx, query, args...)
	fields := map[string]any{
		"query":     compactSQL(query),
		"args":      previewArgs(args),
		"latencyMs": time.Since(startedAt).Milliseconds(),
	}
	if err != nil {
		fields["error"] = err.Error()
		logging.LogEvent("db_exec", fields)
		return nil, err
	}
	if rowsAffected, rowsErr := result.RowsAffected(); rowsErr == nil {
		fields["rowsAffected"] = rowsAffected
	}
	if lastInsertID, idErr := result.LastInsertId(); idErr == nil {
		fields["lastInsertId"] = lastInsertID
	}
	logging.LogEvent("db_exec", fields)
	return result, nil
}

func (db *observedDB) QueryContext(ctx context.Context, query string, args ...any) (*observedRows, error) {
	startedAt := time.Now()
	rows, err := db.raw.QueryContext(ctx, query, args...)
	if err != nil {
		logging.LogEvent("db_query", map[string]any{
			"query":     compactSQL(query),
			"args":      previewArgs(args),
			"latencyMs": time.Since(startedAt).Milliseconds(),
			"error":     err.Error(),
		})
		return nil, err
	}
	return &observedRows{rows: rows, query: query, args: cloneArgs(args), startedAt: startedAt, previews: make([][]any, 0, 3)}, nil
}

func (db *observedDB) QueryRowContext(ctx context.Context, query string, args ...any) *observedRow {
	return &observedRow{row: db.raw.QueryRowContext(ctx, query, args...), query: query, args: cloneArgs(args), startedAt: time.Now()}
}

func (r *observedRows) Next() bool {
	return r.rows.Next()
}

func (r *observedRows) Scan(dest ...any) error {
	err := r.rows.Scan(dest...)
	if err != nil {
		logging.LogEvent("db_query_scan", map[string]any{
			"query":     compactSQL(r.query),
			"args":      previewArgs(r.args),
			"rowCount":  r.rowCount,
			"latencyMs": time.Since(r.startedAt).Milliseconds(),
			"error":     err.Error(),
		})
		return err
	}
	r.mu.Lock()
	r.rowCount++
	if len(r.previews) < 3 {
		r.previews = append(r.previews, previewScanDest(dest))
	}
	r.mu.Unlock()
	return nil
}

func (r *observedRows) Close() error {
	err := r.rows.Close()
	r.finalize(err)
	return err
}

func (r *observedRows) Err() error {
	err := r.rows.Err()
	r.finalize(err)
	return err
}

func (r *observedRows) finalize(err error) {
	r.mu.Lock()
	if r.finalized {
		r.mu.Unlock()
		return
	}
	r.finalized = true
	rowCount := r.rowCount
	previews := append(make([][]any, 0, len(r.previews)), r.previews...)
	r.mu.Unlock()
	fields := map[string]any{
		"query":       compactSQL(r.query),
		"args":        previewArgs(r.args),
		"latencyMs":   time.Since(r.startedAt).Milliseconds(),
		"rowCount":    rowCount,
		"rowPreviews": previews,
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	logging.LogEvent("db_query", fields)
}

func (r *observedRow) Scan(dest ...any) error {
	err := r.row.Scan(dest...)
	fields := map[string]any{
		"query":     compactSQL(r.query),
		"args":      previewArgs(r.args),
		"latencyMs": time.Since(r.startedAt).Milliseconds(),
	}
	if err != nil {
		fields["error"] = err.Error()
		logging.LogEvent("db_query_row", fields)
		return err
	}
	fields["result"] = previewScanDest(dest)
	logging.LogEvent("db_query_row", fields)
	return nil
}

func compactSQL(query string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(query)), " ")
}

func cloneArgs(args []any) []any {
	if len(args) == 0 {
		return nil
	}
	cloned := make([]any, len(args))
	copy(cloned, args)
	return cloned
}

func previewArgs(args []any) []any {
	if len(args) == 0 {
		return []any{}
	}
	preview := make([]any, 0, len(args))
	for _, arg := range args {
		preview = append(preview, previewValue(arg))
	}
	return preview
}

func previewScanDest(dest []any) []any {
	preview := make([]any, 0, len(dest))
	for _, item := range dest {
		preview = append(preview, previewValue(item))
	}
	return preview
}

func previewValue(value any) any {
	if value == nil {
		return nil
	}
	if valuer, ok := value.(driver.Valuer); ok {
		if driverValue, err := valuer.Value(); err == nil {
			return previewValue(driverValue)
		}
	}
	switch typed := value.(type) {
	case []byte:
		return logging.PreviewString(string(typed), 512)
	case string:
		return logging.PreviewString(typed, 512)
	case time.Time:
		return typed.Format(time.RFC3339)
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		return previewValue(rv.Elem().Interface())
	}
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		length := rv.Len()
		limit := length
		if limit > 5 {
			limit = 5
		}
		items := make([]any, 0, limit)
		for index := 0; index < limit; index++ {
			items = append(items, previewValue(rv.Index(index).Interface()))
		}
		if length > limit {
			return fmt.Sprintf("%v...[truncated %d items]", items, length-limit)
		}
		return items
	}
	return value
}
