// gis 服务入口(阶段8):消费 boss-order-events,环节12(更新 GIS 地图)触发地图状态同步。
// 口径:GIS 为派生聚合(无基表,见 alignment-audit),同步动作 = 事件驱动的状态回查与留痕;
// 地址坐标_geom 由 addresses.geom(PostGIS)承载,地图读侧经 cmd/server /gis/* 路由按需聚合。
package main

import (
	"context"
	"encoding/json"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/ymm-001/boss/internal/domain/gis"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/database"
)

// orderEvent boss-order-events 消息体(与 internal/pkg/events.Event 对齐)。
type orderEvent struct {
	Type      string    `json:"type"`
	OrderID   int64     `json:"orderId"`
	Stage     int8      `json:"stage"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// Syncer 地图同步器:环节12 事件到达即回查 GIS 聚合并留痕(变更联动地图验收)。
type Syncer struct {
	Gis gis.GISService
}

// Handle 处理一条事件;仅 stage=12(updateMap)触发同步。
func (s *Syncer) Handle(ctx context.Context, e orderEvent) {
	if e.Stage != 12 {
		return
	}
	counts, err := s.Gis.LevelCounts(ctx)
	if err != nil {
		log.Printf("gis: sync order=%d: %v", e.OrderID, err)
		return
	}
	total := int64(0)
	for _, n := range counts {
		total += n.Count
	}
	log.Printf("gis: order=%d updateMap synced, entities=%d (8 levels)", e.OrderID, total)
}

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(ctx, cfg.Database.DSN)
	if err != nil {
		log.Fatalf("gis: open database: %v", err)
	}
	defer pool.Close()

	syncer := &Syncer{Gis: gis.NewPGStore(pool)}
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.Kafka.Brokers, Topic: cfg.Events.Topic, GroupID: "boss-gis",
	})
	defer reader.Close()
	log.Printf("gis: consuming %s", cfg.Events.Topic)

	for {
		m, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("gis: stopped")
				return
			}
			log.Printf("gis: read: %v", err)
			continue
		}
		var e orderEvent
		if err := json.Unmarshal(m.Value, &e); err != nil {
			continue
		}
		syncer.Handle(ctx, e)
	}
}
