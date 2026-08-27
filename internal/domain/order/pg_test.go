package order

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// stubExists 桩 CustomerLookup。
type stubExists struct{ ok bool }

func (s stubExists) Exists(context.Context, int64) (bool, error) { return s.ok, nil }

// stubProfileCreator 桩 UserProfileCreator。
type stubProfileCreator struct {
	UserProfileCreator
	err error
}

func (s *stubProfileCreator) GetLoAccountByCustomer(context.Context, int64) (*LoidAccount, error) {
	return &LoidAccount{ID: 88, Loid: "LOID-TEST"}, s.err
}
func (s *stubProfileCreator) CreateLoAccount(context.Context, LoidReq) (int64, error) { return 1, nil }

// stubProvCreator 桩 ProvisionTaskCreator。
type stubProvCreator struct {
	ProvisionTaskCreator
	err error
}

func (s *stubProvCreator) CreateTask(context.Context, ProvisionTask) (int64, error) { return 1, s.err }

var ts = time.Date(2025, 8, 17, 10, 0, 0, 0, time.UTC)

// expectRefExists 桩 Submit 关联存在性校验(addresses/product_offers/channels,均返回存在)。
func expectRefExists(mock pgxmock.PgxPoolIface, table string, id int64) {
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM ` + table).
		WithArgs(id).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
}

// expectSubmitRefs 桩 Submit 三项关联存在性校验(address/offer/channel 默认全通过)。
func expectSubmitRefs(mock pgxmock.PgxPoolIface, offerID, channelID int64) {
	expectRefExists(mock, "addresses", 100)
	expectRefExists(mock, "product_offers", offerID)
	expectRefExists(mock, "channels", channelID)
}

func TestPGStore_Submit(t *testing.T) {
	t.Run("成功", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		expectSubmitRefs(mock, 10, 5)
		mock.ExpectQuery(`SELECT cov.legal_entity_id`).
			WithArgs(int64(100)).
			WillReturnRows(mock.NewRows([]string{"legal_entity_id", "path"}).AddRow(int64(1), "root.luzon"))
		mock.ExpectQuery(`SELECT 'ORD-'`).
			WithArgs(pgxmock.AnyArg()).
			WillReturnRows(mock.NewRows([]string{"order_no"}).AddRow("ORD-20250817-000001"))
		mock.ExpectQuery(`INSERT INTO orders`).
			WithArgs(pgxmock.AnyArg(), int64(1), int64(10), int64(100), int8(1), "PENDING", int64(5), int64(1), "root.luzon", "POSTPAID", 0, pgxmock.AnyArg()).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(7)))
		mock.ExpectExec(`INSERT INTO order_stages`).
			WithArgs(int64(7), int8(1), "DOING").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		s := NewPGStore(mock, stubExists{ok: true})
		o, err := s.Submit(context.Background(), SubmitReq{
			CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5,
		})
		if err != nil {
			t.Fatalf("Submit: %v", err)
		}
		if o.LegalEntityID != 1 || o.RegionPath != "root.luzon" {
			t.Fatalf("ownership not derived: %+v", o)
		}
		if o.ID != 7 || o.Status != "PENDING" || o.Stage != 1 || o.OrderNo == "" {
			t.Fatalf("o=%+v", o)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("未覆盖兜底总公司", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		expectSubmitRefs(mock, 10, 5)
		mock.ExpectQuery(`SELECT cov.legal_entity_id`).
			WithArgs(int64(100)).
			WillReturnError(pgx.ErrNoRows)
		mock.ExpectQuery(`SELECT id, 'root' FROM legal_entities`).
			WillReturnRows(mock.NewRows([]string{"id", "path"}).AddRow(int64(9), "root"))
		mock.ExpectQuery(`SELECT 'ORD-'`).
			WithArgs(pgxmock.AnyArg()).
			WillReturnRows(mock.NewRows([]string{"order_no"}).AddRow("ORD-20250817-000001"))
		mock.ExpectQuery(`INSERT INTO orders`).
			WithArgs(pgxmock.AnyArg(), int64(1), int64(10), int64(100), int8(1), "PENDING", int64(5), int64(9), "root", "POSTPAID", 0, pgxmock.AnyArg()).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(7)))
		mock.ExpectExec(`INSERT INTO order_stages`).
			WithArgs(int64(7), int8(1), "DOING").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		s := NewPGStore(mock, stubExists{ok: true})
		o, err := s.Submit(context.Background(), SubmitReq{CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5})
		if err != nil {
			t.Fatalf("Submit: %v", err)
		}
		if o.LegalEntityID != 9 || o.RegionPath != "root" {
			t.Fatalf("want platform fallback(9/root), got %+v", o)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("平台总公司未配置", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		expectSubmitRefs(mock, 0, 5)
		mock.ExpectQuery(`SELECT cov.legal_entity_id`).
			WithArgs(int64(100)).
			WillReturnError(pgx.ErrNoRows)
		mock.ExpectQuery(`SELECT id, 'root' FROM legal_entities`).
			WillReturnError(pgx.ErrNoRows)

		s := NewPGStore(mock, stubExists{ok: true})
		_, err = s.Submit(context.Background(), SubmitReq{CustomerID: 1, AddressID: 100, ChannelID: 5})
		if !errors.Is(err, ErrPlatformMissing) {
			t.Fatalf("err=%v, want ErrPlatformMissing", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("归属冲突", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		expectSubmitRefs(mock, 0, 5)
		mock.ExpectQuery(`SELECT cov.legal_entity_id`).
			WithArgs(int64(100)).
			WillReturnRows(mock.NewRows([]string{"legal_entity_id", "path"}).AddRow(int64(2), "root.luzon"))

		s := NewPGStore(mock, stubExists{ok: true})
		_, err = s.Submit(context.Background(), SubmitReq{
			CustomerID: 1, AddressID: 100, ChannelID: 5, LegalEntityID: 1,
		})
		if !errors.Is(err, ErrOwnershipMismatch) {
			t.Fatalf("err=%v, want ErrOwnershipMismatch", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("客户不存在", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		s := NewPGStore(mock, stubExists{ok: false})
		_, err = s.Submit(context.Background(), SubmitReq{CustomerID: 99, ChannelID: 5})
		if err == nil {
			t.Fatal("want error for missing customer")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("缺渠道", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		s := NewPGStore(mock, stubExists{ok: true})
		if _, err := s.Submit(context.Background(), SubmitReq{CustomerID: 1}); err == nil {
			t.Fatal("want error for missing channel")
		}
	})
	t.Run("地址不存在", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM addresses`).
			WithArgs(int64(100)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))

		s := NewPGStore(mock, stubExists{ok: true})
		_, err = s.Submit(context.Background(), SubmitReq{CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5})
		if !errors.Is(err, ErrAddressNotFound) {
			t.Fatalf("err=%v, want ErrAddressNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("产品未上架", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		expectRefExists(mock, "addresses", 100)
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM product_offers`).
			WithArgs(int64(10)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))

		s := NewPGStore(mock, stubExists{ok: true})
		_, err = s.Submit(context.Background(), SubmitReq{CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5})
		if !errors.Is(err, ErrOfferNotOrderable) {
			t.Fatalf("err=%v, want ErrOfferNotOrderable", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("渠道停用", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		expectRefExists(mock, "addresses", 100)
		expectRefExists(mock, "product_offers", 10)
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM channels`).
			WithArgs(int64(5)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))

		s := NewPGStore(mock, stubExists{ok: true})
		_, err = s.Submit(context.Background(), SubmitReq{CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5})
		if !errors.Is(err, ErrChannelNotActive) {
			t.Fatalf("err=%v, want ErrChannelNotActive", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

func TestPGStore_Reserve(t *testing.T) {
	t.Run("成功", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectBegin() // advance 事务
		mock.ExpectQuery(`SELECT stage, status, order_no FROM orders`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows([]string{"stage", "status", "order_no"}).AddRow(int8(2), "PENDING", "ORD-1"))
		mock.ExpectQuery(`SELECT result FROM order_stages WHERE order_id=\$1 AND stage=\$2`).
			WithArgs(int64(1), int8(2)).
			WillReturnRows(mock.NewRows([]string{"result"}).AddRow("DONE"))
		mock.ExpectExec(`UPDATE orders SET stage`).
			WithArgs(int64(1), int8(3), "RESERVED").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`INSERT INTO order_stages`).
			WithArgs(int64(1), int8(3), "DONE").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectCommit()

		s := NewPGStore(mock, stubExists{})
		if err := s.Reserve(context.Background(), 1); err != nil {
			t.Fatalf("Reserve: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("非法流转", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectBegin() // advance 事务;非法流转拒绝后回滚
		mock.ExpectQuery(`SELECT stage, status, order_no FROM orders`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows([]string{"stage", "status", "order_no"}).AddRow(int8(2), "RESERVED", "ORD-1"))
		mock.ExpectRollback()

		s := NewPGStore(mock, stubExists{})
		err = s.Reserve(context.Background(), 1)
		if !errors.Is(err, ErrIllegalTransition) {
			t.Fatalf("err=%v, want ErrIllegalTransition", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

func TestPGStore_Track(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	orderCols := []string{"id", "order_no", "customer_id", "offer_id", "address_id", "stage", "status", "channel_id", "legal_entity_id", "region_path", "billing_mode", "buy_months", "gift_months", "created_at"}
	mock.ExpectQuery(`SELECT id, order_no, customer_id, offer_id, address_id, stage, status, channel_id, legal_entity_id, region_path, billing_mode, buy_months, gift_months, created_at FROM orders`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(orderCols).
			AddRow(int64(1), "ORD-20250817-001", int64(1), int64(10), int64(100), int8(3), "RESERVED", int64(5), int64(1), "root.luzon", "POSTPAID", 0, 0, ts))
	mock.ExpectQuery(`SELECT id, order_id, stage, result, retries, finished_at FROM order_stages`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"id", "order_id", "stage", "result", "retries", "finished_at"}).
			AddRow(int64(1), int64(1), int8(1), "DONE", int16(0), nil).
			AddRow(int64(2), int64(1), int8(3), "DOING", int16(0), nil))

	s := NewPGStore(mock, stubExists{})
	o, stages, err := s.Track(context.Background(), 1)
	if err != nil {
		t.Fatalf("Track: %v", err)
	}
	if o.Status != "RESERVED" || len(stages) != 2 || stages[0].Stage != 1 {
		t.Fatalf("o=%+v stages=%+v", o, stages)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_TrackNotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, order_no, customer_id`).
		WithArgs(int64(99)).
		WillReturnError(pgx.ErrNoRows)

	s := NewPGStore(mock, stubExists{})
	_, _, err = s.Track(context.Background(), 99)
	if !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("err=%v, want ErrOrderNotFound", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
