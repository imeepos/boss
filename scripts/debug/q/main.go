package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	conn, _ := pgx.Connect(context.Background(), os.Getenv("BOSS_PG_DSN"))
	var id int64
	var name string
	var st int16
	err := conn.QueryRow(context.Background(), `SELECT id, name, status FROM api_keys`).Scan(&id, &name, &st)
	fmt.Println(err, id, name, st)
}
