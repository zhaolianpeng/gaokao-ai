package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

var tableOrder = []string{
	"province_score_line",
	"score_rank",
	"college",
	"major_catalog",
	"source_import_batch",
	"crawl_job_log",
	"mini_auth_user",
	"college_program_group",
	"college_enrollment_plan",
	"college_major",
	"college_score_line",
	"college_admission_line",
	"major_ranking",
	"major_employment",
	"gaokao_application_snapshot",
	"gaokao_application_snapshot_stat",
	"college_major_admission_stat",
	"agent_recommend_task",
}

func main() {
	var (
		sourceDSN = flag.String("source", "host=127.0.0.1 port=5432 user=gaokao_app password=GaokaoApi_2026_Auth dbname=gaokao sslmode=disable", "PostgreSQL DSN")
		targetDSN = flag.String("target", "gaokao_app:GaokaoApi_2026_Auth@tcp(127.0.0.1:3306)/gaokao?charset=utf8mb4&parseTime=true&loc=Local", "MySQL DSN")
		tablesArg = flag.String("tables", "", "comma-separated tables to migrate; empty means all")
		truncate  = flag.Bool("truncate", true, "truncate MySQL target tables before import")
	)
	flag.Parse()

	tables := tableOrder
	if strings.TrimSpace(*tablesArg) != "" {
		tables = make([]string, 0)
		for _, item := range strings.Split(*tablesArg, ",") {
			table := strings.TrimSpace(item)
			if table != "" {
				tables = append(tables, table)
			}
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sourceDB, err := sql.Open("postgres", *sourceDSN)
	if err != nil {
		log.Fatalf("open postgres failed: %v", err)
	}
	defer sourceDB.Close()

	targetDB, err := sql.Open("mysql", *targetDSN)
	if err != nil {
		log.Fatalf("open mysql failed: %v", err)
	}
	defer targetDB.Close()

	if err := sourceDB.PingContext(ctx); err != nil {
		log.Fatalf("ping postgres failed: %v", err)
	}
	if err := targetDB.PingContext(ctx); err != nil {
		log.Fatalf("ping mysql failed: %v", err)
	}

	if _, err := targetDB.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		log.Fatalf("disable foreign key checks failed: %v", err)
	}
	defer func() {
		if _, err := targetDB.ExecContext(context.Background(), "SET FOREIGN_KEY_CHECKS = 1"); err != nil {
			log.Printf("reenable foreign key checks failed: %v", err)
		}
	}()

	for _, table := range tables {
		if err := migrateTable(ctx, sourceDB, targetDB, table, *truncate); err != nil {
			log.Fatalf("migrate %s failed: %v", table, err)
		}
	}

	log.Println("migration finished")
}

func migrateTable(ctx context.Context, sourceDB, targetDB *sql.DB, table string, truncate bool) error {
	columns, err := loadColumns(ctx, sourceDB, table)
	if err != nil {
		return err
	}

	if truncate {
		if _, err := targetDB.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s", table)); err != nil {
			return fmt.Errorf("truncate %s: %w", table, err)
		}
	}

	rows, err := sourceDB.QueryContext(ctx, fmt.Sprintf("SELECT %s FROM %s", strings.Join(columns, ", "), table))
	if err != nil {
		return fmt.Errorf("query %s: %w", table, err)
	}
	defer rows.Close()

	tx, err := targetDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx %s: %w", table, err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(quoteColumns(columns), ", "), strings.TrimRight(strings.Repeat("?,", len(columns)), ",")))
	if err != nil {
		return fmt.Errorf("prepare %s: %w", table, err)
	}
	defer stmt.Close()

	values := make([]interface{}, len(columns))
	scans := make([]interface{}, len(columns))
	for i := range values {
		scans[i] = &values[i]
	}

	count := 0
	startedAt := time.Now()
	for rows.Next() {
		if err := rows.Scan(scans...); err != nil {
			return fmt.Errorf("scan %s: %w", table, err)
		}
		args := make([]interface{}, len(values))
		for i, value := range values {
			if raw, ok := value.([]byte); ok {
				args[i] = string(raw)
				continue
			}
			args[i] = value
		}
		if _, err := stmt.ExecContext(ctx, args...); err != nil {
			return fmt.Errorf("insert %s row %d: %w", table, count+1, err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate %s: %w", table, err)
	}

	if _, err := tx.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s AUTO_INCREMENT = 1", table)); err != nil {
		return fmt.Errorf("reset auto increment %s: %w", table, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit %s: %w", table, err)
	}

	log.Printf("migrated %s: %d rows in %s", table, count, time.Since(startedAt).Round(time.Millisecond))
	return nil
}

func loadColumns(ctx context.Context, sourceDB *sql.DB, table string) ([]string, error) {
	rows, err := sourceDB.QueryContext(ctx, `
SELECT column_name
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = $1
ORDER BY ordinal_position`, table)
	if err != nil {
		return nil, fmt.Errorf("load columns for %s: %w", table, err)
	}
	defer rows.Close()

	columns := make([]string, 0)
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			return nil, fmt.Errorf("scan columns for %s: %w", table, err)
		}
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate columns for %s: %w", table, err)
	}
	return columns, nil
}

func quoteColumns(columns []string) []string {
	quoted := make([]string, 0, len(columns))
	for _, column := range columns {
		quoted = append(quoted, "`"+column+"`")
	}
	return quoted
}
