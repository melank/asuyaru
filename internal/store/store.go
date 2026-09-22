// Package store は SQLite ファイル一つの読み書きとマイグレーションを持つ。
package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Open は path の SQLite を開く。接続はプロセス内で一つに絞る。
func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("保存先のディレクトリを作る: %w", err)
	}
	db, err := sql.Open("sqlite3", dsn(path))
	if err != nil {
		return nil, fmt.Errorf("SQLite を開く: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("SQLite に接続する: %w", err)
	}
	return db, nil
}

func dsn(path string) string {
	u := url.URL{
		Scheme:   "file",
		Path:     path,
		RawQuery: "_foreign_keys=on&_busy_timeout=5000",
	}
	return u.String()
}

// Migrate は番号順に SQL を適用する。適用済みは同じファイルに残す。
func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL
		)`); err != nil {
		return fmt.Errorf("schema_migrations を作る: %w", err)
	}
	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return err
	}
	names, err := fs.Glob(migrationFS, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("マイグレーションを読む: %w", err)
	}
	sort.Strings(names)
	for _, name := range names {
		version := filepath.Base(name)
		if applied[version] {
			continue
		}
		body, err := migrationFS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("%s を読む: %w", version, err)
		}
		if err := apply(ctx, db, version, string(body)); err != nil {
			return err
		}
	}
	return nil
}

func appliedVersions(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("適用済みを読む: %w", err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("適用済みを読む: %w", err)
		}
		out[version] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("適用済みを読む: %w", err)
	}
	return out, nil
}

func apply(ctx context.Context, db *sql.DB, version, body string) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("%s の接続を取る: %w", version, err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("%s を開始する: %w", version, err)
	}
	if _, err := conn.ExecContext(ctx, body); err != nil {
		_, _ = conn.ExecContext(ctx, "ROLLBACK")
		return fmt.Errorf("%s を適用する: %w", version, err)
	}
	appliedAt := time.Now().Format("2006-01-02T15:04:05")
	if _, err := conn.ExecContext(ctx, `INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`, version, appliedAt); err != nil {
		_, _ = conn.ExecContext(ctx, "ROLLBACK")
		return fmt.Errorf("%s の適用を記録する: %w", version, err)
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("%s を確定する: %w", version, err)
	}
	return nil
}
