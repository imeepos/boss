package app

import (
	"context"
	"fmt"
	"time"

	"github.com/ymm-001/boss/internal/domain/aaa"
	aaability "github.com/ymm-001/boss/internal/domain/aaa/billing"
	"github.com/ymm-001/boss/internal/domain/analytics"
	"github.com/ymm-001/boss/internal/pkg/audit"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/events"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ensureAuditPartitions E14:预建当月起 2 个月的审计分区,避免写入全部落入 default。
func ensureAuditPartitions(ctx context.Context, pool *pgxpool.Pool) error {
	if err := audit.NewPGWriter(pool).EnsurePartitions(ctx, time.Now(), 2); err != nil {
		return fmt.Errorf("wiring: audit partitions: %w", err)
	}
	return nil
}

// emitters 外发通道装配结果;关闭函数由 New 统一挂到 app.close。
type emitters struct {
	cdr      aaability.Emitter // 话单投递(PG 必选,Kafka 双写)
	pub      events.Publisher  // W8 状态变更链路
	closeCdr func()
	closePub func()
}

// wireEmitters 阶段9债务偿还:gRPC aaa/v1 话单 + W8 事件链。
// Kafka brokers 可用即接双写/发布,否则 PG 单写/Noop 降级。
func wireEmitters(aaastore *aaa.PGStore, cfg *config.Config) emitters {
	var e emitters
	cdrStore := aaability.NewPGEmitter(aaastore)
	e.cdr = cdrStore
	if len(cfg.Kafka.Brokers) > 0 {
		ke := aaability.NewKafkaEmitter(cfg.Kafka.Brokers, cfg.AAA.CDRTopic)
		e.cdr = &aaability.FanoutEmitter{Realtime: ke, Store: cdrStore}
		e.closeCdr = func() { _ = ke.Close() }
	}

	e.pub = events.Noop{}
	if len(cfg.Kafka.Brokers) > 0 {
		kp := events.NewKafkaPublisher(cfg.Kafka.Brokers, cfg.Events.Topic)
		e.pub = kp
		e.closePub = func() { _ = kp.Close() }
	}
	return e
}

// selectAnalytics 阶段9经营分析后端选择(pg 派生聚合 | starrocks OLAP 宽表)。
func selectAnalytics(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) (analytics.AnalyticsService, func(), error) {
	store := analytics.NewPGStore(pool, cfg.Analytics.MaintUnitCost, cfg.Analytics.PortUnitCost)
	if cfg.Analytics.Backend != "starrocks" || cfg.OLAP.StarRocksDSN == "" {
		return store, nil, nil
	}
	sr, err := analytics.NewStarRocksStore(cfg.OLAP.StarRocksDSN, cfg.Analytics.MaintUnitCost, cfg.Analytics.PortUnitCost)
	if err != nil {
		return nil, nil, fmt.Errorf("wiring: starrocks: %w", err)
	}
	if err := sr.Ping(ctx); err != nil {
		_ = sr.Close()
		return nil, nil, fmt.Errorf("wiring: starrocks ping: %w", err)
	}
	return sr, func() { _ = sr.Close() }, nil
}
