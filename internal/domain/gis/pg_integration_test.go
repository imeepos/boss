package gis

// GIS 真实 PG 集成测试(阶段8 验收):八级下钻层级分发、数量统计、资产详情合并。
// 运行: BOSS_PG_TEST_DSN="host=192.168.0.102 port=25432 user=boss password=boss dbname=boss sslmode=disable" go test ./internal/domain/gis/ -run TestGIS -v -count=1

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/pkg/database"
)

func TestGIS_DrillAndDetail_Integration(t *testing.T) {
	dsn := os.Getenv("BOSS_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("BOSS_PG_TEST_DSN 未设置,跳过集成测试")
	}
	ctx := context.Background()
	pool, err := database.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool, "../../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	s := NewPGStore(pool)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1e12)
	// 地址五级链(nlevel(path)=level,逐级累积标签)。
	ids := make([]int64, 0, 5)
	labels := make([]string, 0, 5)
	var parent int64
	for i := 1; i <= 5; i++ {
		labels = append(labels, fmt.Sprintf("g%d%s", i, suffix))
		path := labels[0]
		for _, l := range labels[1:] {
			path += "." + l
		}
		var id int64
		err := pool.QueryRow(ctx, `
			INSERT INTO addresses(path, level, name, parent_id, geom)
			VALUES(CAST($1 AS ltree), $2, $3, $4, ST_SetSRID(ST_MakePoint(121.0, 31.0), 4326))
			RETURNING id`, path, i, fmt.Sprintf("GIS层%d", i), nilIf0(parent)).Scan(&id)
		if err != nil {
			t.Fatalf("addr %d: %v", i, err)
		}
		ids = append(ids, id)
		parent = id
	}
	buildingID := ids[4]

	// OLT(弱电井)→SPLITTER(分光器)→端口。
	var oltID, splID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO resources(legal_entity_id, code, name, type, address_id, status)
		VALUES(1, $1, 'GIS-OLT', 'OLT', $2, 'ONLINE') RETURNING id`, "OLT-GIS-"+suffix, buildingID).Scan(&oltID)
	if err != nil {
		t.Fatal(err)
	}
	err = pool.QueryRow(ctx, `
		INSERT INTO resources(legal_entity_id, code, name, type, parent_id, address_id, status)
		VALUES(1, $1, 'GIS-SPL', 'SPLITTER', $2, $3, 'ONLINE') RETURNING id`, "SPL-GIS-"+suffix, oltID, buildingID).Scan(&splID)
	if err != nil {
		t.Fatal(err)
	}
	var portID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO ports(port_code, quad_code, resource_id, legal_entity_id, legal_entity_name, address_id, region_id, region_name, status)
		VALUES($1, $2, $3, 1, '主品牌', $4, 1, '马尼拉', 'USED') RETURNING id`,
		"P-GIS-"+suffix, "Q-GIS-"+suffix, splID, buildingID).Scan(&portID)
	if err != nil {
		t.Fatal(err)
	}
	// 指标 + 客户 + 四码。
	if _, err := pool.Exec(ctx, `
		INSERT INTO device_metrics(resource_id, optical_power, packet_loss, status, collected_at)
		VALUES($1, -24.5, 12, 'ONLINE', now())`, oltID); err != nil {
		t.Fatal(err)
	}
	var custID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO customers(name, phone, id_type, address_id, region_id, region_name, legal_entity_id)
		VALUES('GIS客户', '0917', '身份证', $1, 1, '马尼拉', 1) RETURNING id`, buildingID).Scan(&custID); err != nil {
		t.Fatal(err)
	}
	// asset_id 软引用:用时间派生唯一值(quad_links.asset_id 唯一,防与其它测试撞号)。
	uniq := time.Now().UnixNano()
	if _, err := pool.Exec(ctx, `
		INSERT INTO quad_links(asset_id, customer_id, port_id, address_id, legal_entity_id, legal_entity_name, status)
		VALUES($1, $2, $3, $4, 1, '主品牌', 'LINKED')`, uniq, custID, portID, buildingID); err != nil {
		t.Fatalf("quad link: %v", err)
	}

	t.Run("八级下钻层级分发", func(t *testing.T) {
		l1, err := s.Drill(ctx, 1, 0)
		if err != nil || len(l1) == 0 || l1[0].Level != 1 {
			t.Fatalf("l1=%+v err=%v", l1, err)
		}
		l6, err := s.Drill(ctx, 6, buildingID)
		if err != nil || len(l6) == 0 || l6[0].ID != oltID || l6[0].Count < 1 {
			t.Fatalf("l6=%+v err=%v", l6, err)
		}
		l7, err := s.Drill(ctx, 7, oltID)
		if err != nil || len(l7) == 0 || l7[0].ID != splID {
			t.Fatalf("l7=%+v err=%v", l7, err)
		}
		l8, err := s.Drill(ctx, 8, splID)
		if err != nil || len(l8) == 0 || l8[0].ID != portID {
			t.Fatalf("l8=%+v err=%v", l8, err)
		}
	})

	t.Run("数量统计", func(t *testing.T) {
		counts, err := s.LevelCounts(ctx)
		if err != nil || len(counts) != 8 {
			t.Fatalf("counts=%d err=%v", len(counts), err)
		}
	})

	t.Run("资产详情合并", func(t *testing.T) {
		d, err := s.ResourceDetail(ctx, oltID)
		if err != nil {
			t.Fatal(err)
		}
		if d.Status != "ONLINE" || d.OpticalPower == nil || d.PortsUsed < 1 || d.CustomerName != "GIS客户" {
			t.Fatalf("d=%+v", d)
		}
	})
}

// nilIf0 0 归 NULL(地址 parent_id 可空)。
func nilIf0(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}
