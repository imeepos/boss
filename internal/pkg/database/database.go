package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Open 建立 PostgreSQL 连接池。
// 默认走简单协议(SimpleProtocol),以支持迁移文件里的多语句(BEGIN/COMMIT + 多 CREATE/INSERT)。
func Open(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("database: parse dsn: %w", err)
	}
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	configurePool(cfg)
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("database: new pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database: ping: %w", err)
	}
	return pool, nil
}

func configurePool(cfg *pgxpool.Config) {
	cfg.MaxConns = int32(envInt("BOSS_DB_MAX_CONNS", 50))
	cfg.MinConns = int32(envInt("BOSS_DB_MIN_CONNS", 10))
	cfg.MaxConnIdleTime = 5 * time.Minute
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

// Migrate 按文件名序执行 dir 下所有 *.up.sql,幂等(已应用版本记录在 schema_migrations)。
func Migrate(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	const create = `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`
	if _, err := pool.Exec(ctx, create); err != nil {
		return fmt.Errorf("database: create schema_migrations: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("database: glob migrations: %w", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		return fmt.Errorf("database: no migrations found in %s (deploy image must ship migrations/)", dir)
	}
	for _, f := range files {
		version := strings.TrimSuffix(filepath.Base(f), ".up.sql")
		var n int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM schema_migrations WHERE version = $1`, version).Scan(&n); err != nil {
			return fmt.Errorf("database: check %s: %w", version, err)
		}
		if n > 0 {
			continue
		}
		body, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("database: read %s: %w", version, err)
		}
		// 迁移文件自带 BEGIN/COMMIT,简单协议下整段原子执行。
		if _, err := pool.Exec(ctx, string(body)); err != nil {
			return fmt.Errorf("database: apply %s: %w", version, err)
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO schema_migrations(version) VALUES($1)`, version); err != nil {
			return fmt.Errorf("database: record %s: %w", version, err)
		}
	}
	return nil
}
