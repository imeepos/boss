// devseed 开发种子:向 PG 写入/更新 admin 账号(sysadmin),供 admin 前端冒烟登录。
// 仅用于开发环境;密码经 bcrypt 哈希入库。用法: go run ./scripts/devseed
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dsn := "host=192.168.0.102 port=25432 user=boss password=boss dbname=boss sslmode=disable"
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()

	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO accounts (username, password_hash, real_name, role_id, status)
		 SELECT 'admin', $1, '开发管理员', id, 1 FROM roles WHERE code = 'sysadmin'
		 ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash, status = 1`,
		string(hash)); err != nil {
		log.Fatalf("devseed: upsert admin: %v", err)
	}
	var id int64
	if err := db.QueryRowContext(ctx,
		`SELECT a.id FROM accounts a JOIN roles r ON r.id = a.role_id
		 WHERE a.username = 'admin' AND r.code = 'sysadmin'`).Scan(&id); err != nil {
		log.Fatalf("devseed: verify: %v", err)
	}
	fmt.Printf("devseed: admin ready (account_id=%d, password=admin123)\n", id)
}
