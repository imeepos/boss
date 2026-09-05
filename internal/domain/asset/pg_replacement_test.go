// 换新单读写与状态机单测(守卫 UPDATE + 快照回填 + 非法流转拒止)。
package asset

import (
	"context"
	"errors"
	"testing"
	"time"

	pgxmock "github.com/pashagolub/pgxmock/v4"
)

// replCols 与 replacementCols 查询列严格对序。
var replCols = []string{"id", "replacement_no", "asset_id", "legal_entity_id", "legal_entity_name",
	"reason", "priority", "status", "worker_id", "worker_name", "finished_at"}

func replRow(id int64, no string, status string, workerID int64) []any {
	return []any{int64(id), no, int64(5), int64(1), "主品牌·企业", "光猫故障", "HIGH", status, workerID, "张三", nil}
}

func TestPGStore_ListReplacements(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, replacement_no.*FROM replacements ORDER BY id`).
		WillReturnRows(mock.NewRows(replCols).AddRow(replRow(1, "RPL-20260817-001", "DOING", 7)...))

	s := NewPGStore(mock)
	got, err := s.ListReplacements(context.Background())
	if err != nil {
		t.Fatalf("ListReplacements: %v", err)
	}
	if len(got) != 1 || got[0].ReplacementNo != "RPL-20260817-001" ||
		got[0].WorkerID != 7 || got[0].WorkerName != "张三" || got[0].FinishedAt != nil {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateReplacement_BackfillsLegalEntity(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	assetCols := []string{"id", "asset_code", "batch_id", "legal_entity_id", "legal_entity_name",
		"tag_id", "address_id", "region_id", "region_name", "type", "status", "model_id"}
	mock.ExpectQuery(`FROM assets WHERE id`).
		WithArgs(int64(6)).
		WillReturnRows(mock.NewRows(assetCols).
			AddRow(int64(6), "A-20260006", int64(1), int64(2), "副品牌·家宽", nil, nil, nil, "", "光猫", "DEPLOYED", int64(0)))
	mock.ExpectQuery(`INSERT INTO replacements`).
		WithArgs("RPL-20260817-002", int64(6), int64(2), "副品牌·家宽", "光猫故障", "MEDIUM", "PENDING").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.CreateReplacement(context.Background(), Replacement{
		ReplacementNo: "RPL-20260817-002", AssetID: 6,
		Reason: "光猫故障", Priority: "MEDIUM", Status: "PENDING",
	})
	if err != nil {
		t.Fatalf("CreateReplacement: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateReplacement_AssetMissing(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`FROM assets WHERE id`).WithArgs(int64(99)).
		WillReturnRows(mock.NewRows([]string{"id"}))

	s := NewPGStore(mock)
	if _, err := s.CreateReplacement(context.Background(),
		Replacement{ReplacementNo: "RPL-X", AssetID: 99}); !errors.Is(err, ErrForeignKeyViolation) {
		t.Fatalf("err=%v, want ErrForeignKeyViolation", err)
	}
}

func TestPGStore_AssignReplacement(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(7)).WillReturnRows(
		mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`UPDATE replacements SET worker_id`).
		WithArgs(int64(1), int64(7), "张三").
		WillReturnRows(mock.NewRows(replCols).AddRow(replRow(1, "RPL-1", "DOING", 7)...))

	s := NewPGStore(mock)
	got, err := s.AssignReplacement(context.Background(), 1, 7, "张三")
	if err != nil {
		t.Fatalf("AssignReplacement: %v", err)
	}
	if got.Status != "DOING" || got.WorkerID != 7 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_AssignReplacement_WrongStatus(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(7)).WillReturnRows(
		mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`UPDATE replacements SET worker_id`).
		WithArgs(int64(1), int64(7), "张三").
		WillReturnRows(mock.NewRows(replCols)) // 0 行 = 状态非 PENDING
	mock.ExpectQuery(`FROM replacements WHERE id`).WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(replCols).AddRow(replRow(1, "RPL-1", "DONE", 7)...))

	s := NewPGStore(mock)
	if _, err := s.AssignReplacement(context.Background(), 1, 7, "张三"); !errors.Is(err, ErrIllegalTransition) {
		t.Fatalf("err=%v, want ErrIllegalTransition", err)
	}
}

func TestPGStore_CompleteReplacement(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	now := time.Now()
	rows := replRow(1, "RPL-1", "DONE", 7)
	rows[len(rows)-1] = now // finished_at 列
	mock.ExpectQuery(`UPDATE replacements SET status = \$2, finished_at`).
		WithArgs(int64(1), "DONE").
		WillReturnRows(mock.NewRows(replCols).AddRow(rows...))

	s := NewPGStore(mock)
	got, err := s.CompleteReplacement(context.Background(), 1, "DONE")
	if err != nil {
		t.Fatalf("CompleteReplacement: %v", err)
	}
	if got.Status != "DONE" || got.FinishedAt == nil {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CompleteReplacement_BadResult(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	s := NewPGStore(mock)
	if _, err := s.CompleteReplacement(context.Background(), 1, "DOING"); !errors.Is(err, ErrIllegalTransition) {
		t.Fatalf("err=%v, want ErrIllegalTransition", err)
	}
}

func TestPGStore_ListReplacementsByWorker_OnlyDoing(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`FROM replacements WHERE worker_id = \$1 AND status = 'DOING' ORDER BY id`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows(replCols).AddRow(replRow(1, "RPL-1", "DOING", 7)...))

	s := NewPGStore(mock)
	got, err := s.ListReplacementsByWorker(context.Background(), 7)
	if err != nil {
		t.Fatalf("ListReplacementsByWorker: %v", err)
	}
	if len(got) != 1 || got[0].Status != "DOING" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
