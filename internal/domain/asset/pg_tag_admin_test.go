package asset

// 建标签服务层测试(P2-W2-T1):法人存在性校验 + 编号/EPC 唯一冲突 40900 分类。

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

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
