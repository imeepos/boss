package analytics

// 分析域单测:五大指标口径(比率分母 0、健康度空、成本公式)、热力图、维护表排序。

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestFiveIndicators_UnitOfWork(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	health := 80.5
	// 总指标行。
	mock.ExpectQuery(`SELECT`).
		WillReturnRows(mock.NewRows([]string{
			"port_used", "port_total", "ord_done", "ord_eff", "faults", "active_cust", "health"}).
			AddRow(int64(8), int64(10), int64(6), int64(9), int64(100), int64(50), health))
	// 区域 ROI 行。
	mock.ExpectQuery(`WITH rev AS`).WithArgs(800.0).
		WillReturnRows(mock.NewRows([]string{"id", "name", "rev", "inv"}).
			AddRow(int64(11), "大区A", 10000.0, 5000.0))
	s := NewPGStore(mock, 50, 800)

	ind, rois, err := s.FiveIndicators(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	byKey := map[string]Indicator{}
	for _, i := range ind {
		byKey[i.Key] = i
	}
	if byKey["portUtilization"].Value != 0.8 || byKey["installConversion"].Value != 6.0/9.0 {
		t.Fatalf("ratios=%+v", ind)
	}
	// 维护成本 = 100×50/50 = 100。
	if byKey["maintenanceCostPerUser"].Value != 100 {
		t.Fatalf("maint=%+v", byKey["maintenanceCostPerUser"])
	}
	if byKey["assetHealth"].Value != 80.5 {
		t.Fatalf("health=%+v", byKey["assetHealth"])
	}
	// ROI = 10000/5000 = 2。
	if byKey["regionROI"].Value != 2 || len(rois) != 1 || rois[0].ROI != 2 {
		t.Fatalf("roi ind=%+v rois=%+v", byKey["regionROI"], rois)
	}
}

func TestFiveIndicators_EmptyGuard(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`SELECT`).
		WillReturnRows(mock.NewRows([]string{
			"port_used", "port_total", "ord_done", "ord_eff", "faults", "active_cust", "health"}).
			AddRow(int64(0), int64(0), int64(0), int64(0), int64(0), int64(0), nil))
	mock.ExpectQuery(`WITH rev AS`).WithArgs(800.0).
		WillReturnRows(mock.NewRows([]string{"id", "name", "rev", "inv"}))
	s := NewPGStore(mock, 0, 0) // 0 → 缺省 50/800
	ind, rois, err := s.FiveIndicators(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range ind {
		if i.Value != 0 {
			t.Fatalf("empty db should yield 0, got %+v", i)
		}
	}
	if len(rois) != 0 {
		t.Fatalf("rois=%+v", rois)
	}
}

func TestHeatmap(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`FROM addresses a`).
		WillReturnRows(mock.NewRows([]string{"id", "name", "level", "total", "used"}).
			AddRow(int64(1), "小区A", int16(4), int64(10), int64(9)).
			AddRow(int64(2), "楼栋B", int16(5), int64(4), int64(1)))
	s := NewPGStore(mock, 50, 800)
	cells, err := s.Heatmap(context.Background())
	if err != nil || len(cells) != 2 {
		t.Fatalf("cells=%+v err=%v", cells, err)
	}
	if cells[0].Utilization != 0.9 || cells[1].Utilization != 0.25 {
		t.Fatalf("cells=%+v", cells)
	}
}

func TestMaintenanceList(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`FROM device_maintenances`).
		WillReturnRows(mock.NewRows([]string{
			"device_no", "device_type", "health_score", "fault_count", "age_years", "reason", "priority"}).
			AddRow("OLT-01", "PON 9口", int16(30), int32(5), 6.0, "老化", "MUST_REPLACE").
			AddRow("OLT-02", "PON 4口", int16(70), int32(1), 2.0, "丢包偏高", "WATCH"))
	s := NewPGStore(mock, 50, 800)
	items, err := s.MaintenanceList(context.Background())
	if err != nil || len(items) != 2 || items[0].Priority != "MUST_REPLACE" {
		t.Fatalf("items=%+v err=%v", items, err)
	}
}
