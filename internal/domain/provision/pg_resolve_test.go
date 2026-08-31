package provision

// FindTemplateForOffer 回归(方案B):显式绑定优先 / 带宽兜底 / 绑定模板禁用报错 / 无解报错。
// 背景:原实现按带宽匹配后回退法人最旧模板,102 实测无带宽套餐(IPTV/云存储)全落到测试模板、
// 有套餐无模板法人卡死环节7。裁定见 adopted note 2026-09-01-offer-provision-binding。

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestFindTemplateForOffer(t *testing.T) {
	ctx := context.Background()

	t.Run("显式绑定优先:走了绑定就不再按带宽猜", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectQuery(`SELECT t\.id, t\.status FROM offer_provision_bindings b`).
			WithArgs(int64(101)).
			WillReturnRows(mock.NewRows([]string{"id", "status"}).AddRow(int64(42), "ENABLED"))

		s := NewPGStore(mock)
		id, err := s.FindTemplateForOffer(ctx, 101, 1)
		if err != nil || id != 42 {
			t.Fatalf("id=%d err=%v, want 42/nil", id, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("绑定模板被禁用:显性报错不落到带宽兜底", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectQuery(`SELECT t\.id, t\.status FROM offer_provision_bindings b`).
			WithArgs(int64(101)).
			WillReturnRows(mock.NewRows([]string{"id", "status"}).AddRow(int64(42), "DISABLED"))

		s := NewPGStore(mock)
		_, err = s.FindTemplateForOffer(ctx, 101, 1)
		if err == nil || !errors.Is(err, ErrBoundTemplateDisabled) || !strings.Contains(err.Error(), "BOUND BUT DISABLED") {
			t.Fatalf("want ErrBoundTemplateDisabled, got %v", err)
		}
	})

	t.Run("无绑定走带宽兜底:content.bandwidth=套餐带宽", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectQuery(`SELECT t\.id, t\.status FROM offer_provision_bindings b`).
			WithArgs(int64(101)).
			WillReturnError(pgx.ErrNoRows)
		mock.ExpectQuery(`SELECT t\.id FROM provision_templates t`).
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

	t.Run("无绑定且带宽无匹配:显性报错,禁止回退法人任意模板", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectQuery(`SELECT t\.id, t\.status FROM offer_provision_bindings b`).
			WithArgs(int64(178)).
			WillReturnError(pgx.ErrNoRows)
		mock.ExpectQuery(`SELECT t\.id FROM provision_templates t`).
			WithArgs(int64(178), int64(1)).
			WillReturnError(pgx.ErrNoRows)

		s := NewPGStore(mock)
		_, err = s.FindTemplateForOffer(ctx, 178, 1)
		if err == nil || !errors.Is(err, ErrTemplateUnresolved) || !strings.Contains(err.Error(), "TEMPLATE UNRESOLVED") {
			t.Fatalf("want ErrTemplateUnresolved, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("DB 故障:绑定查询错误透传不误入带宽", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectQuery(`SELECT t\.id, t\.status FROM offer_provision_bindings b`).
			WithArgs(int64(101)).
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
