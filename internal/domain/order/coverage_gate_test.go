package order

// 下单覆盖门控回归(P2,路线图 T12;odnGate 注入后地址未 SERVED → 发单号前拒单;
// 未注入(灰度关)自动跳过。仿 pg_submit_poq_test.go 桩骨架)。

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

type stubCoverageGate struct{ err error }

func (s *stubCoverageGate) CheckOrderCoverage(ctx context.Context, addressID int64) error {
	return s.err
}

var errGateReject = errors.New("gate: not servable")

func TestPGStore_SubmitCoverageGate(t *testing.T) {
	t.Run("门控拒单:发单号前拒绝(无后续 SQL)", func(t *testing.T) {
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
		s := NewPGStore(mock, stubExists{ok: true}, &stubProvCreator{tplID: 16}, &stubCoverageGate{err: errGateReject})
		_, err = s.Submit(context.Background(), SubmitReq{
			CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5,
		})
		if err == nil || !errors.Is(err, errGateReject) {
			t.Fatalf("want gate reject, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("门控放行:正常建单", func(t *testing.T) {
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
			WillReturnRows(mock.NewRows([]string{"order_no"}).AddRow("ORD-20250817-000002"))
		mock.ExpectQuery(`INSERT INTO orders`).
			WithArgs(pgxmock.AnyArg(), int64(1), int64(10), int64(100), int8(1), "PENDING", int64(5), int64(1), int64(0), "root.luzon", "POSTPAID", 0, pgxmock.AnyArg()).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(8)))
		mock.ExpectExec(`INSERT INTO order_stages`).
			WithArgs(int64(8), int8(1), "DONE").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		s := NewPGStore(mock, stubExists{ok: true}, &stubProvCreator{tplID: 16}, &stubCoverageGate{})
		o, err := s.Submit(context.Background(), SubmitReq{
			CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5,
		})
		if err != nil {
			t.Fatalf("Submit: %v", err)
		}
		if o.ID != 8 {
			t.Fatalf("o=%+v", o)
		}
	})

	t.Run("门控未注入:灰度关自动跳过", func(t *testing.T) {
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
			WillReturnRows(mock.NewRows([]string{"order_no"}).AddRow("ORD-20250817-000003"))
		mock.ExpectQuery(`INSERT INTO orders`).
			WithArgs(pgxmock.AnyArg(), int64(1), int64(10), int64(100), int8(1), "PENDING", int64(5), int64(1), int64(0), "root.luzon", "POSTPAID", 0, pgxmock.AnyArg()).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(9)))
		mock.ExpectExec(`INSERT INTO order_stages`).
			WithArgs(int64(9), int8(1), "DONE").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		s := NewPGStore(mock, stubExists{ok: true}, &stubProvCreator{tplID: 16})
		if _, err := s.Submit(context.Background(), SubmitReq{
			CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5,
		}); err != nil {
			t.Fatalf("Submit: %v", err)
		}
	})
}
