package provision

// FindTemplateForOffer 回归:带宽匹配命中 / 法人默认回退 / 无可用模板报错。
// 背景:环节7 曾把 offer_id 当 template_id 传入,102 实测 100M 套餐静默套错 tnet 模板。

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestFindTemplateForOffer(t *testing.T) {
	ctx := context.Background()

	t.Run("带宽命中:content.bandwidth=套餐带宽", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectQuery(`SELECT t.id FROM provision_templates t`).
			WithArgs(int64(101), int64(1)).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(16)))

		s := NewPGStore(mock)
		id, err := s.FindTemplateForOffer(ctx, 101, 1)
		if err != nil || id != 16 {
			t.Fatalf("id=%d err=%v, want 16/nil", id, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("带宽无匹配:回退法人默认模板", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectQuery(`SELECT t.id FROM provision_templates t`).
			WithArgs(int64(178), int64(1)).
			WillReturnError(pgx.ErrNoRows)
		mock.ExpectQuery(`SELECT id FROM provision_templates`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(16)))

		s := NewPGStore(mock)
		id, err := s.FindTemplateForOffer(ctx, 178, 1)
		if err != nil || id != 16 {
			t.Fatalf("id=%d err=%v, want fallback 16/nil", id, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("法人无可用模板:显性报错", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectQuery(`SELECT t.id FROM provision_templates t`).
			WithArgs(int64(101), int64(9)).
			WillReturnError(pgx.ErrNoRows)
		mock.ExpectQuery(`SELECT id FROM provision_templates`).
			WithArgs(int64(9)).
			WillReturnError(pgx.ErrNoRows)

		s := NewPGStore(mock)
		if _, err := s.FindTemplateForOffer(ctx, 101, 9); err == nil {
			t.Fatal("want error when no enabled template")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("DB 故障:错误透传不误入回退", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectQuery(`SELECT t.id FROM provision_templates t`).
			WithArgs(int64(101), int64(1)).
			WillReturnError(errors.New("connection reset"))

		s := NewPGStore(mock)
		if _, err := s.FindTemplateForOffer(ctx, 101, 1); err == nil {
			t.Fatal("want db error passthrough")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}
