//go:build tools

// 一次性排查脚本:查 102 库迁移记录与 api_keys 表状态。
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	conn, err := pgx.Connect(context.Background(), os.Getenv("BOSS_PG_DSN"))
	if err != nil {
		fmt.Println("connect:", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())
	ctx := context.Background()
	var n int
	_ = conn.QueryRow(ctx, `SELECT count(*) FROM schema_migrations WHERE version LIKE '000042%' OR version LIKE '000043%'`).Scan(&n)
	fmt.Println("000042/43 rows:", n)
	rows, _ := conn.Query(ctx, `SELECT version FROM schema_migrations ORDER BY version`)
	for rows.Next() {
		var v string
		_ = rows.Scan(&v)
		fmt.Println("applied:", v)
	}
	rows.Close()
	var tbl int
	_ = conn.QueryRow(ctx, `SELECT count(*) FROM pg_tables WHERE tablename = 'api_keys'`).Scan(&tbl)
	fmt.Println("api_keys table exists:", tbl == 1)
	var pk int
	_ = conn.QueryRow(ctx, `SELECT count(*) FROM permissions WHERE code = 'menu:apikey'`).Scan(&pk)
	fmt.Println("menu:apikey permission rows:", pk)
}
