package worker

// 装维队管理(TeamService,000141)单测:pgxmock 验证 SQL 形状与错误分支。

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_UpdateGroup(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()

	// 队长须为本队在职成员:命中取姓名快照。
	mock.ExpectQuery(`SELECT name FROM workers WHERE id = \$1 AND group_id = \$2 AND status = 1`).
		WithArgs(int64(9), int64(2)).
		WillReturnRows(mock.NewRows([]string{"name"}).AddRow("李队长"))
	mock.ExpectQuery(`UPDATE worker_groups SET name`).
		WithArgs(int64(2), "装维一队", int64(9), "李队长").
		WillReturnRows(mock.NewRows([]string{"id", "legal_entity_id", "code", "name", "leader_id", "leader_name"}).
			AddRow(int64(2), int64(1), "TEAM-1", "装维一队", int64(9), "李队长"))

	s := NewPGStore(mock)
	g, err := s.UpdateGroup(context.Background(), 2, "装维一队", 9)
	if err != nil {
		t.Fatalf("UpdateGroup: %v", err)
	}
	if g.LeaderID != 9 || g.LeaderName != "李队长" || g.Name != "装维一队" {
		t.Fatalf("got=%+v", g)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_UpdateGroup_LeaderNotMember(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()

	mock.ExpectQuery(`SELECT name FROM workers WHERE id = \$1 AND group_id = \$2 AND status = 1`).
		WithArgs(int64(9), int64(2)).
		WillReturnRows(mock.NewRows([]string{"name"})) // 无行=非本队成员

	s := NewPGStore(mock)
	if _, err := s.UpdateGroup(context.Background(), 2, "装维一队", 9); !errors.Is(err, ErrLeaderNotMember) {
		t.Fatalf("err=%v want ErrLeaderNotMember", err)
	}
}

func TestPGStore_SoftDeleteGroup_NotEmpty(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM workers WHERE group_id`).
		WithArgs(int64(2)).
		WillReturnRows(mock.NewRows([]string{"count"}).AddRow(3))

	s := NewPGStore(mock)
	if err := s.SoftDeleteGroup(context.Background(), 2); !errors.Is(err, ErrGroupNotEmpty) {
		t.Fatalf("err=%v want ErrGroupNotEmpty", err)
	}
}

func TestPGStore_SoftDeleteGroup_OK(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM workers WHERE group_id`).
		WithArgs(int64(2)).
		WillReturnRows(mock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec(`UPDATE worker_groups SET deleted_at`).
		WithArgs(int64(2)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(mock)
	if err := s.SoftDeleteGroup(context.Background(), 2); err != nil {
		t.Fatalf("SoftDeleteGroup: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_TransferWorker_SameGroup(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()

	// CTE 无行写入(已在目标队) → 分辨分支:目标队在职存在 + 师傅在职 → ErrSameGroup。
	mock.ExpectExec(`WITH target AS`).
		WithArgs(int64(9), int64(2), "调整", nil).
		WillReturnResult(pgxmock.NewResult("INSERT", 0))
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM worker_groups WHERE id`).
		WithArgs(int64(2)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT ` + workerCols + ` FROM workers WHERE id = \$1`).
		WithArgs(int64(9)).
		WillReturnRows(mock.NewRows(cols("id", "staff_no", "name", "group_id", "region_id", "phone", "status", "joined_at", "left_at")).
			AddRow(int64(9), "WK-9", "张师傅", int64(2), 1, "13800000000", 1, ts, nil))

	s := NewPGStore(mock)
	err := s.TransferWorker(context.Background(), Transfer{WorkerID: 9, TargetGroupID: 2, Reason: "调整"})
	if !errors.Is(err, ErrSameGroup) {
		t.Fatalf("err=%v want ErrSameGroup", err)
	}
}

func cols(names ...string) []string { return names }

func TestPGStore_ListTeamPerformances(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()

	mock.ExpectQuery(`FROM workers w\s+JOIN worker_groups g`).
		WithArgs(int64(2), "2026-08").
		WillReturnRows(mock.NewRows([]string{"id", "staff_no", "name", "phone", "is_leader", "finished", "on_time_rate", "score"}).
			AddRow(int64(9), "WK-9", "李队长", "13800000000", true, 21, 96, 4.8).
			AddRow(int64(10), "WK-10", "张师傅", "13900000000", false, 15, 90, 4.5))

	s := NewPGStore(mock)
	items, err := s.ListTeamPerformances(context.Background(), 2, "2026-08")
	if err != nil {
		t.Fatalf("ListTeamPerformances: %v", err)
	}
	if len(items) != 2 || !items[0].IsLeader || items[1].Finished != 15 {
		t.Fatalf("items=%+v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_GroupOfLeader_NotLeader(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()

	mock.ExpectQuery(`WHERE g\.leader_id = \$1 AND g\.deleted_at IS NULL`).
		WithArgs(int64(9)).
		WillReturnRows(mock.NewRows([]string{"id", "legal_entity_id", "code", "name", "leader_id", "leader_name", "cnt"}))

	s := NewPGStore(mock)
	if _, err := s.GroupOfLeader(context.Background(), 9); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v want ErrNotFound", err)
	}
}
