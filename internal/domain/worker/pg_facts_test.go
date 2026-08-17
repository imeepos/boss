package worker

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_ListPerformances(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "worker_id", "group_id", "group_name", "legal_entity_id", "legal_entity_name", "region_id", "region_name", "period", "finished", "on_time_rate", "score"}
	mock.ExpectQuery(`SELECT id, worker_id, group_id, group_name, legal_entity_id, legal_entity_name, region_id, region_name, period, finished, on_time_rate, score FROM worker_performances`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(1), int64(1), "装机一组", int64(1), "主品牌·企业", int64(11), "马尼拉市", "2026-08", int32(50), int16(98), 4.9))

	s := NewPGStore(mock)
	got, err := s.ListPerformances(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListPerformances: %v", err)
	}
	if len(got) != 1 || got[0].Finished != 50 || got[0].Score != 4.9 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_UpsertPerformance(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO worker_performances`).
		WithArgs(int64(1), int64(1), "装机一组", int64(1), "主品牌·企业", int64(11), "马尼拉市", "2026-08", int32(50), int16(98), 4.9).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))

	s := NewPGStore(mock)
	id, err := s.UpsertPerformance(context.Background(), Performance{
		WorkerID: 1, GroupID: 1, GroupName: "装机一组", LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		RegionID: 11, RegionName: "马尼拉市", Period: "2026-08", Finished: 50, OnTimeRate: 98, Score: 4.9,
	})
	if err != nil {
		t.Fatalf("UpsertPerformance: %v", err)
	}
	if id != 1 {
		t.Fatalf("id=%d, want 1", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListCommissions(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "worker_id", "group_id", "group_name", "legal_entity_id", "legal_entity_name", "region_id", "region_name", "period", "formula", "amount"}
	mock.ExpectQuery(`SELECT id, worker_id, group_id, group_name, legal_entity_id, legal_entity_name, region_id, region_name, period, formula, amount FROM worker_commissions`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(1), int64(1), "装机一组", int64(1), "主品牌·企业", int64(11), "马尼拉市", "2026-08", "新装×40", 2000.0))

	s := NewPGStore(mock)
	got, err := s.ListCommissions(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListCommissions: %v", err)
	}
	if len(got) != 1 || got[0].Amount != 2000.0 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_UpsertCommission(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO worker_commissions`).
		WithArgs(int64(1), int64(1), "装机一组", int64(1), "主品牌·企业", int64(11), "马尼拉市", "2026-08", "新装×40", 2000.0).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))

	s := NewPGStore(mock)
	id, err := s.UpsertCommission(context.Background(), Commission{
		WorkerID: 1, GroupID: 1, GroupName: "装机一组", LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		RegionID: 11, RegionName: "马尼拉市", Period: "2026-08", Formula: "新装×40", Amount: 2000.0,
	})
	if err != nil {
		t.Fatalf("UpsertCommission: %v", err)
	}
	if id != 1 {
		t.Fatalf("id=%d, want 1", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListSchedules(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "worker_id", "group_id", "group_name", "legal_entity_id", "legal_entity_name", "region_id", "region_name", "month", "busy_days"}
	mock.ExpectQuery(`SELECT id, worker_id, group_id, group_name, legal_entity_id, legal_entity_name, region_id, region_name, month, busy_days FROM worker_schedules`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(1), int64(1), "装机一组", int64(1), "主品牌·企业", int64(11), "马尼拉市", "2026-08", int16(22)))

	s := NewPGStore(mock)
	got, err := s.ListSchedules(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListSchedules: %v", err)
	}
	if len(got) != 1 || got[0].BusyDays != 22 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_UpsertSchedule(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO worker_schedules`).
		WithArgs(int64(1), int64(1), "装机一组", int64(1), "主品牌·企业", int64(11), "马尼拉市", "2026-08", int16(22)).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))

	s := NewPGStore(mock)
	id, err := s.UpsertSchedule(context.Background(), Schedule{
		WorkerID: 1, GroupID: 1, GroupName: "装机一组", LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		RegionID: 11, RegionName: "马尼拉市", Month: "2026-08", BusyDays: 22,
	})
	if err != nil {
		t.Fatalf("UpsertSchedule: %v", err)
	}
	if id != 1 {
		t.Fatalf("id=%d, want 1", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
