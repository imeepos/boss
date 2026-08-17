package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

var ts = time.Date(2025, 8, 17, 10, 0, 0, 0, time.UTC)

func TestPGStore_ListGroups(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, legal_entity_id, code, name, COALESCE\(leader_id, 0\), COALESCE\(leader_name, ''\) FROM worker_groups`).
		WillReturnRows(mock.NewRows([]string{"id", "legal_entity_id", "code", "name", "leader_id", "leader_name"}).
			AddRow(int64(1), int64(1), "装机一组", "装机一组", int64(1024), "张师傅"))

	s := NewPGStore(mock)
	got, err := s.ListGroups(context.Background())
	if err != nil {
		t.Fatalf("ListGroups: %v", err)
	}
	if len(got) != 1 || got[0].LeaderID != 1024 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateGroup(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO worker_groups`).
		WithArgs(int64(1), "抢修组", "抢修组", nil, "").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.CreateGroup(context.Background(), Group{LegalEntityID: 1, Code: "抢修组", Name: "抢修组"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListWorkers(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "staff_no", "name", "group_id", "region_id", "phone", "status", "joined_at", "left_at"}
	mock.ExpectQuery(`SELECT id, staff_no, name, group_id, region_id, phone, status, joined_at, left_at FROM workers`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), "WK-1024", "张师傅", int64(1), int64(11), "138****8899", int16(1), ts, nil).
			AddRow(int64(2), "WK-1025", "李师傅", int64(1), int64(11), "136****1177", int16(0), ts, ts))

	s := NewPGStore(mock)
	got, err := s.ListWorkers(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListWorkers: %v", err)
	}
	if len(got) != 2 || got[0].StaffNo != "WK-1024" || got[0].LeftAt != nil || got[1].LeftAt == nil {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateWorker(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	var nilTime *time.Time
	mock.ExpectQuery(`INSERT INTO workers`).
		WithArgs("WK-1026", "王师傅", int64(1), int64(11), "137****3366", int16(1), ts, nilTime).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(3)))

	s := NewPGStore(mock)
	id, err := s.CreateWorker(context.Background(), Worker{
		StaffNo: "WK-1026", Name: "王师傅", GroupID: 1, RegionID: 11, Phone: "137****3366", Status: 1, JoinedAt: ts,
	})
	if err != nil {
		t.Fatalf("CreateWorker: %v", err)
	}
	if id != 3 {
		t.Fatalf("id=%d, want 3", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_GetWorker(t *testing.T) {
	t.Run("命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		cols := []string{"id", "staff_no", "name", "group_id", "region_id", "phone", "status", "joined_at", "left_at"}
		mock.ExpectQuery(`SELECT id, staff_no, name, group_id, region_id, phone, status, joined_at, left_at FROM workers WHERE id`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows(cols).
				AddRow(int64(1), "WK-1024", "张师傅", int64(1), int64(11), "138****8899", int16(1), ts, nil))

		s := NewPGStore(mock)
		w, err := s.GetWorker(context.Background(), 1)
		if err != nil {
			t.Fatalf("GetWorker: %v", err)
		}
		if w.Name != "张师傅" || w.LeftAt != nil {
			t.Fatalf("w=%+v", w)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("未命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT id, staff_no`).
			WithArgs(int64(99)).
			WillReturnError(pgx.ErrNoRows)

		s := NewPGStore(mock)
		_, err = s.GetWorker(context.Background(), 99)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}
