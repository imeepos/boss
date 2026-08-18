//go:build ignore

// 压测账号种子(W11):幂等创建 loadtest 管理账号(sysadmin 角色)供 k6 登录。
// 用法: go run scripts/load/seed_account.go "$BOSS_DATABASE_DSN"
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dsn := os.Args[1]
	hash, err := bcrypt.GenerateFromPassword([]byte(os.Getenv("BOSS_LOAD_PASSWORD")), bcrypt.MinCost)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bcrypt:", err)
		os.Exit(1)
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect:", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)
	// 幂等:存在则刷新口令哈希,保持与 env 一致。
	_, err = conn.Exec(ctx, `
		INSERT INTO accounts(username, password_hash, real_name, role_id)
		SELECT 'loadtest', $1, '压测账号', id FROM roles WHERE code='sysadmin'
		ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash`, string(hash))
	if err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
	fmt.Println("loadtest account ready")
}
