package analytics

// 阶段9 真实 PG 集成测试:五大指标口径与明细数据自洽、热力图聚合、维护表排序。
// 运行: BOSS_PG_TEST_DSN="host=192.168.0.102 port=25432 user=boss password=boss dbname=boss sslmode=disable" go test ./internal/domain/analytics/ -run TestAnalytics -v -count=1

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/pkg/database"
)

func TestAnalytics_Integration(t *testing.T) {
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
	s := NewPGStore(pool, 50, 800)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1e12)

	// 种子:地址(楼栋,五级路径满足 nlevel=level)+2 端口(1 USED)+订单(DONE)+客户 ACTIVE+维护档案+区域收入/投资。
	var addrID int64
	path := fmt.Sprintf("an1%s.an2%s.an3%s.an4%s.an5%s", suffix, suffix, suffix, suffix, suffix)
	if err := pool.QueryRow(ctx, `
		INSERT INTO addresses(path, level, name, geom)
		VALUES(CAST($1 AS ltree), 5, '分析楼栋', ST_SetSRID(ST_MakePoint(121.0,31.0),4326))
		RETURNING id`, path).Scan(&addrID); err != nil {
		t.Fatalf("seed address: %v", err)
	}
	// 自清理(defer 后注册先执行,早于 pool.Close):防共享库残留。
	defer func() {
		for _, q := range []string{
			`DELETE FROM ports WHERE address_id = ` + fmt.Sprint(addrID),
			`DELETE FROM orders WHERE order_no = 'ORD-AN-` + suffix + `'`,
			`DELETE FROM device_maintenances WHERE device_no LIKE 'OLT-AN%` + suffix + `'`,
			`DELETE FROM addresses WHERE path::text LIKE 'an1` + suffix + `%' AND level = 5`,
			`DELETE FROM addresses WHERE path::text LIKE 'an1` + suffix + `%' AND level = 4`,
			`DELETE FROM addresses WHERE path::text LIKE 'an1` + suffix + `%' AND level = 3`,
			`DELETE FROM addresses WHERE path::text LIKE 'an1` + suffix + `%' AND level = 2`,
			`DELETE FROM addresses WHERE path::text LIKE 'an1` + suffix + `%' AND level = 1`,
		} {
			if _, err := pool.Exec(ctx, q); err != nil {
				t.Logf("analytics cleanup(尽力而为): %v", err)
			}
		}
	}()
	if _, err := pool.Exec(ctx, `
		INSERT INTO ports(port_code, quad_code, resource_id, legal_entity_id, legal_entity_name, address_id, region_id, region_name, status)
		SELECT 'PA-'||$1||'-'||v.s, 'QA-'||$1||'-'||v.s, r.id, 1, '主品牌', $2, 1, '马尼拉', v.s
		FROM resources r, (VALUES ('USED'),('IDLE')) AS v(s)
		WHERE r.code = (SELECT code FROM resources ORDER BY id LIMIT 1)`, "X"+suffix, addrID); err != nil {
		t.Fatal(err)
	}
	var ordID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO orders(order_no, customer_id, offer_id, address_id, channel_id, legal_entity_id, status)
		VALUES('ORD-AN-'||$1, 1, 1, $2, 1, 1, 'DONE') RETURNING id`, suffix, addrID).Scan(&ordID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO device_maintenances(device_no, device_type, health_score, fault_count, age_years, reason, priority)
		VALUES('OLT-AN-'||$1, 'PON', 30, 4, 6, '老化', 'MUST_REPLACE'), ('OLT-AN2-'||$1, 'PON', 90, 0, 1, '', 'WATCH')`, suffix); err != nil {
		t.Fatal(err)
	}

	t.Run("五大指标与明细自洽", func(t *testing.T) {
		ind, rois, err := s.FiveIndicators(ctx)
		if err != nil || len(ind) != 5 {
			t.Fatalf("ind=%+v err=%v", ind, err)
		}
		for _, i := range ind {
			if i.Detail == "" {
				t.Fatalf("indicator %s 无口径明细", i.Key)
			}
		}
		if len(rois) == 0 {
			t.Fatal("region ROI 为空")
		}
	})

	t.Run("热力图含种子楼栋", func(t *testing.T) {
		cells, err := s.Heatmap(ctx)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, c := range cells {
			if c.AddressID == addrID {
				found = true
				if c.PortsTotal != 2 || c.PortsUsed != 1 || c.Utilization != 0.5 {
					t.Fatalf("cell=%+v, want 2/1/0.5", c)
				}
			}
		}
		if !found {
			t.Fatalf("楼栋 %d 未入热力图", addrID)
		}
	})

	t.Run("维护表 MUST_REPLACE 在前", func(t *testing.T) {
		items, err := s.MaintenanceList(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var mustIdx, watchIdx = -1, -1
		for i, m := range items {
			if m.DeviceNo == "OLT-AN-"+suffix {
				mustIdx = i
			}
			if m.DeviceNo == "OLT-AN2-"+suffix {
				watchIdx = i
			}
		}
		if mustIdx < 0 || watchIdx < 0 || mustIdx > watchIdx {
			t.Fatalf("mustIdx=%d watchIdx=%d items=%d", mustIdx, watchIdx, len(items))
		}
	})
}
