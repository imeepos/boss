package analytics

// StarRocks 实链路集成测试:DDL 建表 → 写宽表样例 → 三口查询断言口径。
// 运行: BOSS_STARROCKS_TEST_DSN="root@tcp(192.168.0.102:29030)/" go test ./internal/domain/analytics/ -run TestStarRocks -v -count=1

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestStarRocksStoreEndToEnd(t *testing.T) {
	dsn := os.Getenv("BOSS_STARROCKS_TEST_DSN")
	if dsn == "" {
		t.Skip("BOSS_STARROCKS_TEST_DSN 未设置,跳过 StarRocks 实链路测试")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	s, err := NewStarRocksStore(dsn, 50, 800)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()
	if err := s.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}

	for _, q := range []string{
		`CREATE TABLE IF NOT EXISTS boss_olap.wide_region_roi (region_id BIGINT NOT NULL, region_name VARCHAR(64), revenue DOUBLE, investment DOUBLE) PRIMARY KEY(region_id) DISTRIBUTED BY HASH(region_id)`,
		`TRUNCATE TABLE boss_olap.wide_region_roi`,
		`INSERT INTO boss_olap.wide_region_roi VALUES (1,'城东',100000,50000),(2,'城西',0,10000)`,
		`CREATE TABLE IF NOT EXISTS boss_olap.wide_port_heat (address_id BIGINT NOT NULL, name VARCHAR(64), level TINYINT, ports_total BIGINT, ports_used BIGINT) PRIMARY KEY(address_id) DISTRIBUTED BY HASH(address_id)`,
		`TRUNCATE TABLE boss_olap.wide_port_heat`,
		`INSERT INTO boss_olap.wide_port_heat VALUES (10,'小区A',4,10,9),(11,'楼栋B',5,8,2)`,
		`CREATE TABLE IF NOT EXISTS boss_olap.wide_device_health (device_no VARCHAR(64) NOT NULL, device_type VARCHAR(32), health_score SMALLINT, fault_count INT, age_years DOUBLE, reason VARCHAR(255), priority VARCHAR(16)) PRIMARY KEY(device_no) DISTRIBUTED BY HASH(device_no)`,
		`TRUNCATE TABLE boss_olap.wide_device_health`,
		`INSERT INTO boss_olap.wide_device_health VALUES ('OLT-01','OLT',40,6,8.0,'超年限','MUST_REPLACE'),('OLT-02','OLT',70,2,3.0,'观察','WATCH')`,
		`CREATE TABLE IF NOT EXISTS boss_olap.wide_order_flow (id TINYINT NOT NULL, orders_done BIGINT, orders_effective BIGINT, active_customers BIGINT) PRIMARY KEY(id) DISTRIBUTED BY HASH(id)`,
		`TRUNCATE TABLE boss_olap.wide_order_flow`,
		`INSERT INTO boss_olap.wide_order_flow VALUES (1,60,100,50)`,
	} {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			t.Fatalf("exec %q: %v", q, err)
		}
	}

	ind, rois, err := s.FiveIndicators(ctx)
	if err != nil {
		t.Fatal(err)
	}
	byKey := map[string]Indicator{}
	for _, i := range ind {
		byKey[i.Key] = i
	}
	// 端口利用率 = (9+2)/(10+8) = 11/18。
	if byKey["portUtilization"].Value < 0.611 || byKey["portUtilization"].Value > 0.612 {
		t.Fatalf("portUtilization=%v", byKey["portUtilization"])
	}
	// 转化率 60/100;维护成本 (6+2)×50/50=8;ROI 100000/60000。
	if byKey["installConversion"].Value != 0.6 {
		t.Fatalf("installConversion=%v", byKey["installConversion"])
	}
	if byKey["maintenanceCostPerUser"].Value != 8 {
		t.Fatalf("maintCost=%v", byKey["maintenanceCostPerUser"])
	}
	wantROI := 100000.0 / 60000.0
	if byKey["regionROI"].Value < wantROI-1e-9 || byKey["regionROI"].Value > wantROI+1e-9 {
		t.Fatalf("regionROI=%v want %v", byKey["regionROI"], wantROI)
	}
	if len(rois) != 2 || rois[0].ROI != 2 {
		t.Fatalf("rois=%+v", rois)
	}

	heat, err := s.Heatmap(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(heat) != 2 || heat[0].AddressID != 10 || heat[0].Utilization < 0.89 {
		t.Fatalf("heat=%+v", heat)
	}

	maint, err := s.MaintenanceList(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(maint) != 2 || maint[0].DeviceNo != "OLT-01" {
		t.Fatalf("maint=%+v", maint)
	}
}
