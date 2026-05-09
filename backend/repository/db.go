package repository

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func NewDB(driver, dsn string) (*sql.DB, error) {
	normalizedDriver := strings.ToLower(strings.TrimSpace(driver))
	if normalizedDriver == "" {
		normalizedDriver = "mysql"
	}

	openDSN := dsn
	switch normalizedDriver {
	case "postgres", "postgresql":
		normalizedDriver = "postgres"
		openDSN = normalizeDSN(dsn)
	case "mysql":
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}

	db, err := sql.Open(normalizedDriver, openDSN)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func normalizeDSN(dsn string) string {
	if !strings.Contains(dsn, "=") || !strings.Contains(dsn, " ") {
		return dsn
	}

	parts := strings.Fields(dsn)
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			normalized = append(normalized, part)
			continue
		}
		if key == "password" && value == "" {
			continue
		}
		normalized = append(normalized, key+"="+value)
	}
	return strings.Join(normalized, " ")
}
