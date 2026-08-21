// backup 域装配辅助:目录从 env BOSS_BACKUP_DIR 读取,默认 data/backups(相对工作目录)。
package app

import (
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ymm-001/boss/internal/domain/backup"
)

func newBackupService(pool *pgxpool.Pool) *backup.Service {
	svc, err := backup.NewService(pool, os.Getenv("BOSS_BACKUP_DIR"))
	if err != nil {
		// 目录建不出来属部署级故障:留 nil,路由层判空返回服务不可用。
		return nil
	}
	return svc
}
