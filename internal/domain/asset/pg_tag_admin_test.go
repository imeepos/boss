package asset

// 建标签服务层测试(P2-W2-T1):法人存在性校验 + 编号/EPC 唯一冲突 40900 分类。

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_CreateTag_LegalEntityMissing(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(9)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))

	s := NewPGStore(mock)
	_, err = s.CreateTag(context.Background(), Tag{LegalEntityID: 9, TagNo: "T-1", EpcCode: "E1", Band: "UHF"})
	if !errors.Is(err, ErrForeignKeyViolation) {
		t.Fatalf("err=%v, want ErrForeignKeyViolation", err)
	}
}

func TestPGStore_CreateTag_NoDuplicateConflict(t *testing.T) {
	cases := map[string]*pgconn.PgError{
		"tagNo": {Code: "23505", ConstraintName: "tags_tag_no_key"},
		"epc":   {Code: "23505", ConstraintName: "tags_epc_code_key"},
	}
	for name, pgErr := range cases {
		t.Run(name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatal(err)
			}
			defer mock.Close()

			mock.ExpectQuery(`SELECT EXISTS`).
				WithArgs(int64(1)).
				WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO tags`).
				WithArgs(int64(1), "T-1", "E1", "UHF", nil, "", "").
				WillReturnError(pgErr)

			s := NewPGStore(mock)
			_, err = s.CreateTag(context.Background(), Tag{LegalEntityID: 1, TagNo: "T-1", EpcCode: "E1", Band: "UHF"})
			if !errors.Is(err, ErrCodeDuplicate) {
				t.Fatalf("err=%v, want ErrCodeDuplicate", err)
			}
		})
	}
}

// DisableTag:UNBOUND 停用成功 / BOUND 拒绝先解绑 / 已停用幂等 / 不存在 404。
func TestPGStore_DisableTag(t *testing.T) {
	ctx := context.Background()
	t.Run("UNBOUND 停用成功", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec("UPDATE tags SET status = 'DISABLED'").
			WithArgs(int64(5)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		s := NewPGStore(mock)
		if err := s.DisableTag(ctx, 5, "损耗"); err != nil {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("BOUND 拒绝须先解绑", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec("UPDATE tags SET status = 'DISABLED'").
			WithArgs(int64(5)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		mock.ExpectQuery("SELECT status, COALESCE").
			WithArgs(int64(5)).
			WillReturnRows(mock.NewRows([]string{"status", "bound"}).AddRow("BOUND", int64(7)))
		s := NewPGStore(mock)
		err := s.DisableTag(ctx, 5, "损耗")
		if !errors.Is(err, ErrBindingConflict) {
			t.Fatalf("err=%v, want ErrBindingConflict", err)
		}
	})
	t.Run("已停用幂等成功", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec("UPDATE tags SET status = 'DISABLED'").
			WithArgs(int64(5)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		mock.ExpectQuery("SELECT status, COALESCE").
			WithArgs(int64(5)).
			WillReturnRows(mock.NewRows([]string{"status", "bound"}).AddRow("DISABLED", nil))
		s := NewPGStore(mock)
		if err := s.DisableTag(ctx, 5, ""); err != nil {
			t.Fatalf("idempotent disable must succeed: %v", err)
		}
	})
	t.Run("不存在404", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec("UPDATE tags SET status = 'DISABLED'").
			WithArgs(int64(5)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		mock.ExpectQuery("SELECT status, COALESCE").
			WithArgs(int64(5)).
			WillReturnError(pgx.ErrNoRows)
		s := NewPGStore(mock)
		if err := s.DisableTag(ctx, 5, ""); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
	})
}

// EnableTag:DISABLED 启用成功 / 非 DISABLED 幂等 / 不存在 404。
func TestPGStore_EnableTag(t *testing.T) {
	ctx := context.Background()
	t.Run("DISABLED 启用成功", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec("UPDATE tags SET status = 'UNBOUND'").
			WithArgs(int64(5)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		s := NewPGStore(mock)
		if err := s.EnableTag(ctx, 5); err != nil {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("非DISABLED幂等成功", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec("UPDATE tags SET status = 'UNBOUND'").
			WithArgs(int64(5)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		mock.ExpectQuery("SELECT status FROM tags").
			WithArgs(int64(5)).
			WillReturnRows(mock.NewRows([]string{"status"}).AddRow("BOUND"))
		s := NewPGStore(mock)
		if err := s.EnableTag(ctx, 5); err != nil {
			t.Fatalf("idempotent enable must succeed: %v", err)
		}
	})
	t.Run("不存在404", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec("UPDATE tags SET status = 'UNBOUND'").
			WithArgs(int64(5)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		mock.ExpectQuery("SELECT status FROM tags").
			WithArgs(int64(5)).
			WillReturnError(pgx.ErrNoRows)
		s := NewPGStore(mock)
		if err := s.EnableTag(ctx, 5); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
	})
}

// ensureTagBindable 绑定前置:DISABLED 拒绝(ErrTagDisabled)/不存在 FK/正常放行。
func TestPGStore_EnsureTagBindable(t *testing.T) {
	ctx := context.Background()
	t.Run("DISABLED 拒绝绑定", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery("SELECT status FROM tags").
			WithArgs(int64(9)).
			WillReturnRows(mock.NewRows([]string{"status"}).AddRow("DISABLED"))
		s := NewPGStore(mock)
		err := s.ensureTagBindable(ctx, mock, 9)
		if !errors.Is(err, ErrTagDisabled) {
			t.Fatalf("err=%v, want ErrTagDisabled", err)
		}
	})
	t.Run("不存在FK拒绝", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery("SELECT status FROM tags").
			WithArgs(int64(9)).
			WillReturnError(pgx.ErrNoRows)
		s := NewPGStore(mock)
		if err := s.ensureTagBindable(ctx, mock, 9); !errors.Is(err, ErrForeignKeyViolation) {
			t.Fatalf("err=%v, want ErrForeignKeyViolation", err)
		}
	})
	t.Run("UNBOUND 放行", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery("SELECT status FROM tags").
			WithArgs(int64(9)).
			WillReturnRows(mock.NewRows([]string{"status"}).AddRow("UNBOUND"))
		s := NewPGStore(mock)
		if err := s.ensureTagBindable(ctx, mock, 9); err != nil {
			t.Fatalf("err=%v", err)
		}
	})
}

// ListTagEvents 事件流只读回放:按时间倒序,JSONB changed 映射。
func TestPGStore_ListTagEvents(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	base := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery("event_id::text").
		WithArgs(int64(9)).
		WillReturnRows(mock.NewRows([]string{"id", "event_id", "tag_id", "asset_id", "action", "actor_account_id", "detail", "changed", "created_at"}).
			AddRow(int64(2), "uuid-2", int64(9), int64(3), "RECYCLE", int64(1), "报废回收", map[string]any{"bound_asset_id": []int64{3, 0}}, base.Add(time.Second)).
			AddRow(int64(1), "uuid-1", int64(9), int64(3), "BIND", int64(0), "", map[string]any{"bound_asset_id": 3}, base))

	s := NewPGStore(mock)
	events, err := s.ListTagEvents(context.Background(), 9)
	if err != nil {
		t.Fatalf("ListTagEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("events=%d, want 2", len(events))
	}
	// 倒序:最新 RECYCLE 在前。
	if events[0].Action != "RECYCLE" || events[1].Action != "BIND" {
		t.Fatalf("order=[%s %s], want [RECYCLE BIND]", events[0].Action, events[1].Action)
	}
	if events[0].EventID != "uuid-2" || events[0].AssetID != 3 || events[0].ActorAccountID != 1 {
		t.Fatalf("event0=%+v", events[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// CreateBatch 法人不存在 → ErrForeignKeyViolation(P2-W2-T1 F 领用侧前置)。
func TestPGStore_CreateBatch_LegalEntityMissing(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(9)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))

	s := NewPGStore(mock)
	_, err = s.CreateBatch(context.Background(), AssetBatch{LegalEntityID: 9, Code: "RK-20260905-00001", Name: "批次"})
	if !errors.Is(err, ErrForeignKeyViolation) {
		t.Fatalf("err=%v, want ErrForeignKeyViolation", err)
	}
}

// CreateAssignment 领用(P2-W2-T1 G):仅 IN_STOCK 可领用;领用不改资产状态
// (无 UPDATE assets 期望=状态口径断言);师傅/资产不存在 FK 拒绝。
func TestPGStore_CreateAssignment(t *testing.T) {
	ctx := context.Background()
	assetCols := []string{"id", "asset_code", "batch_id", "legal_entity_id", "legal_entity_name", "tag_id", "address_id", "region_id", "region_name", "type", "status", "model_id"}
	in := AssetAssignment{AssetID: 5, WorkerID: 7, Reason: "装机备件领用"}
	t.Run("IN_STOCK 领用成功且不改资产状态", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery("FROM assets WHERE id").
			WithArgs(int64(5)).
			WillReturnRows(mock.NewRows(assetCols).AddRow(int64(5), "A-1", int64(1), int64(1), "企业", nil, nil, nil, "", "ONU", "IN_STOCK", nil))
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("INSERT INTO asset_assignments").
			WithArgs(int64(5), int64(7), "", nil, "", "装机备件领用", nil, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(12)))
		s := NewPGStore(mock)
		id, err := s.CreateAssignment(ctx, in)
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if id != 12 {
			t.Fatalf("id=%d", id)
		}
		// 无 UPDATE assets 期望:领用不改资产状态(装机才置 DEPLOYED)。
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("非库存态拒绝", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery("FROM assets WHERE id").
			WithArgs(int64(5)).
			WillReturnRows(mock.NewRows(assetCols).AddRow(int64(5), "A-1", int64(1), int64(1), "企业", nil, nil, nil, "", "ONU", "DEPLOYED", nil))
		s := NewPGStore(mock)
		_, err := s.CreateAssignment(ctx, in)
		if !errors.Is(err, ErrAssetNotInStock) {
			t.Fatalf("err=%v, want ErrAssetNotInStock", err)
		}
	})
	t.Run("资产不存在FK拒绝", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery("FROM assets WHERE id").
			WithArgs(int64(5)).
			WillReturnError(pgx.ErrNoRows)
		s := NewPGStore(mock)
		_, err := s.CreateAssignment(ctx, in)
		if !errors.Is(err, ErrForeignKeyViolation) {
			t.Fatalf("err=%v, want ErrForeignKeyViolation", err)
		}
	})
}

// ReturnAssignment 归还(P2-W2-T1 H):闭合开段 / 重复归还 40900 / 不存在 404。
func TestPGStore_ReturnAssignment(t *testing.T) {
	ctx := context.Background()
	t.Run("闭合开段成功", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		now := time.Now().UTC()
		mock.ExpectQuery("UPDATE asset_assignments SET effective_to").
			WithArgs(int64(12)).
			WillReturnRows(mock.NewRows([]string{"effective_to"}).AddRow(now))
		s := NewPGStore(mock)
		closedAt, err := s.ReturnAssignment(ctx, 12)
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if closedAt == nil {
			t.Fatalf("closedAt nil")
		}
	})
	t.Run("重复归还40900", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		closed := time.Now().UTC().Add(-time.Hour)
		mock.ExpectQuery("UPDATE asset_assignments SET effective_to").
			WithArgs(int64(12)).
			WillReturnError(pgx.ErrNoRows)
		mock.ExpectQuery("SELECT effective_to FROM asset_assignments").
			WithArgs(int64(12)).
			WillReturnRows(mock.NewRows([]string{"effective_to"}).AddRow(closed))
		s := NewPGStore(mock)
		_, err := s.ReturnAssignment(ctx, 12)
		if !errors.Is(err, ErrAssignmentClosed) {
			t.Fatalf("err=%v, want ErrAssignmentClosed", err)
		}
	})
	t.Run("不存在404", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery("UPDATE asset_assignments SET effective_to").
			WithArgs(int64(12)).
			WillReturnError(pgx.ErrNoRows)
		mock.ExpectQuery("SELECT effective_to FROM asset_assignments").
			WithArgs(int64(12)).
			WillReturnError(pgx.ErrNoRows)
		s := NewPGStore(mock)
		if _, err := s.ReturnAssignment(ctx, 12); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
	})
}
