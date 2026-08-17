package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/database"
)

// Application 持有各域服务的装配结果,是模块化单体依赖绑定的唯一入口。
//
// 依赖倒置(见 docs/ADR-001):域之间只经接口依赖;本层把「接口 → 实现」绑定。
// 增量装配(见 docs/architecture-review.md 发现 3.4):每个阶段只构造已实现的域服务,未实现域字段保持零值。
type Application struct {
	User user.Service
	// 后续域随阶段实现并注入(阶段 2-9)。
}

// ErrNotImplemented 域尚未接入装配时返回,便于调用方降级/提示。
var ErrNotImplemented = errors.New("app: domain service not wired yet")

// New 装配依赖:打开 PG → 应用迁移 → 构造 user 域 PGStore。
// migrationsDir 为 migrations/*.up.sql 所在目录(通常相对工作目录为 "migrations")。
func New(ctx context.Context, cfg *config.Config, migrationsDir string) (*Application, error) {
	pool, err := database.Open(ctx, cfg.Database.DSN)
	if err != nil {
		return nil, fmt.Errorf("app: open database: %w", err)
	}
	if err := database.Migrate(ctx, pool, migrationsDir); err != nil {
		pool.Close()
		return nil, fmt.Errorf("app: migrate: %w", err)
	}
	return &Application{
		User: user.NewPGStoreWithSecret(pool, []byte(cfg.JWT.Secret)),
	}, nil
}
