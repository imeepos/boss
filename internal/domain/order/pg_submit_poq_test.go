package order

// 下单资格预检回归(TMF Product Offering Qualification,adopted note
// 2026-09-01-provision-correctness-followup 裁定二):tpl 已接线时套餐不可解析
// → 发单号前拒单;可解析 → 正常建单。tpl 未接线(内存桩)自动跳过。

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/ymm-001/boss/internal/domain/provision"
)

func TestPGStore_SubmitQualificationPrecheck(t *testing.T) {
	t.Run("套餐不可解析:发单号前拒单", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		expectSubmitRefs(mock, 10, 5)
		expectDirectRiskClean(mock, 1, 100)
		mock.ExpectQuery(`SELECT cov.legal_entity_id`).
			WithArgs(int64(100)).
			WillReturnRows(mock.NewRows([]string{"legal_entity_id", "path"}).AddRow(int64(1), "root.luzon"))
		// tpl 桩报 ErrTemplateUnresolved → 拒单,不发单号不插单(无后续 SQL)。
		tpl := &stubProvCreator{err: provision.ErrTemplateUnresolved}

		s := NewPGStore(mock, stubExists{ok: true}, tpl)
		_, err = s.Submit(context.Background(), SubmitReq{
			CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5,
		})
		if err == nil || !errors.Is(err, provision.ErrTemplateUnresolved) {
			t.Fatalf("want ErrTemplateUnresolved, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("套餐可解析:正常建单", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		expectSubmitRefs(mock, 10, 5)
		expectDirectRiskClean(mock, 1, 100)
		mock.ExpectQuery(`SELECT cov.legal_entity_id`).
			WithArgs(int64(100)).
			WillReturnRows(mock.NewRows([]string{"legal_entity_id", "path"}).AddRow(int64(1), "root.luzon"))
		mock.ExpectQuery(`SELECT 'ORD-'`).
			WithArgs(pgxmock.AnyArg()).
			WillReturnRows(mock.NewRows([]string{"order_no"}).AddRow("ORD-20250817-000001"))
		mock.ExpectQuery(`INSERT INTO orders`).
			WithArgs(pgxmock.AnyArg(), int64(1), int64(10), int64(100), int8(1), "PENDING", int64(5), int64(1), int64(0), "root.luzon", "POSTPAID", 0, pgxmock.AnyArg()).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(7)))
		mock.ExpectExec(`INSERT INTO order_stages`).
			WithArgs(int64(7), int8(1), "DONE").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		tpl := &stubProvCreator{tplID: 16}
		s := NewPGStore(mock, stubExists{ok: true}, tpl)
		o, err := s.Submit(context.Background(), SubmitReq{
			CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5,
		})
		if err != nil {
			t.Fatalf("Submit: %v", err)
		}
		if o.ID != 7 {
			t.Fatalf("o=%+v", o)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}
