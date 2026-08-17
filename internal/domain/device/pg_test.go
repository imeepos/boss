package device

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

var ts = time.Date(2025, 8, 17, 10, 0, 0, 0, time.UTC)

func TestPGStore_ListMetrics(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "resource_id", "optical_power", "packet_loss", "status", "collected_at"}
	mock.ExpectQuery(`SELECT id, resource_id, optical_power, packet_loss, status, collected_at`).
		WithArgs(int64(10)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(10), nil, nil, "OFFLINE", ts).
			AddRow(int64(2), int64(10), float64(-18.2), float64(0.01), "ONLINE", ts))

	s := NewPGStore(mock)
	got, err := s.ListMetrics(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListMetrics: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	if got[0].OpticalPower != nil {
		t.Fatal("got[0].OpticalPower non-nil, want nil(离线)")
	}
	if got[1].OpticalPower == nil || *got[1].OpticalPower != -18.2 {
		t.Fatalf("got[1].OpticalPower=%v", got[1].OpticalPower)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_AppendMetric(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	opt := -18.6
	pkt := 0.03
	mock.ExpectQuery(`INSERT INTO device_metrics`).
		WithArgs(int64(11), &opt, &pkt, "ONLINE").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(3)))

	s := NewPGStore(mock)
	id, err := s.AppendMetric(context.Background(), DeviceMetric{
		ResourceID: 11, OpticalPower: &opt, PacketLoss: &pkt, Status: "ONLINE",
	})
	if err != nil {
		t.Fatalf("AppendMetric: %v", err)
	}
	if id != 3 {
		t.Fatalf("id=%d, want 3", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListMaintenances(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "device_no", "device_type", "health_score", "fault_count", "age_years", "reason", "priority"}
	mock.ExpectQuery(`SELECT id, device_no, COALESCE\(device_type, ''\), health_score, fault_count, age_years, COALESCE\(reason, ''\), priority`).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), "OLT-01", "PON 9口", int16(58), int32(0), float64(3.5), "丢包率偏高", "WATCH"))

	s := NewPGStore(mock)
	got, err := s.ListMaintenances(context.Background())
	if err != nil {
		t.Fatalf("ListMaintenances: %v", err)
	}
	if len(got) != 1 || got[0].Priority != "WATCH" || got[0].AgeYears == nil || *got[0].AgeYears != 3.5 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateMaintenance(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	age := 2.0
	mock.ExpectQuery(`INSERT INTO device_maintenances`).
		WithArgs("OLT-02", "PON 16口", int16(70), int32(1), &age, "光功率偏低", "SUGGEST").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.CreateMaintenance(context.Background(), DeviceMaintenance{
		DeviceNo: "OLT-02", DeviceType: "PON 16口", HealthScore: 70, FaultCount: 1, AgeYears: &age, Reason: "光功率偏低", Priority: "SUGGEST",
	})
	if err != nil {
		t.Fatalf("CreateMaintenance: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
