package customer

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

var fixedTimeTo = fixedTime.Add(24 * time.Hour)

// TestPGStore_ListCustomerHistories 契约:按客户过滤归属台账;customerID=0 返回全部。
func TestPGStore_ListCustomerHistories(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, customer_id, legal_entity_id, legal_entity_name, address_id, address_name, region_id, region_name, COALESCE\(reason, ''\), COALESCE\(operator_account_id, 0\), effective_from, effective_to`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{
			"id", "customer_id", "legal_entity_id", "legal_entity_name", "address_id", "address_name",
			"region_id", "region_name", "reason", "operator_account_id", "effective_from", "effective_to",
		}).AddRow(int64(1), int64(9), int64(1), "Acme Ltd", int64(100), "Manila", int64(11), "root.luzon.ncr.manila",
			"搬家", int64(3), fixedTime, fixedTimeTo))

	s := NewPGStore(mock)
	got, err := s.ListCustomerHistories(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListCustomerHistories: %v", err)
	}
	if len(got) != 1 || got[0].LegalEntityName != "Acme Ltd" || got[0].EffectiveTo == nil || !got[0].EffectiveTo.Equal(fixedTimeTo) {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_AppendCustomerHistory 契约:追加归属台账并返回自增 id;operator=0 写 NULL。
func TestPGStore_AppendCustomerHistory(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO customer_histories`).
		WithArgs(int64(9), int64(1), "Acme Ltd", int64(100), "Manila", int64(11), "root.luzon.ncr.manila", "搬家", nil, fixedTime, &fixedTimeTo).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(7)))

	s := NewPGStore(mock)
	id, err := s.AppendCustomerHistory(context.Background(), CustomerHistory{
		CustomerID: 9, LegalEntityID: 1, LegalEntityName: "Acme Ltd", AddressID: 100, AddressName: "Manila",
		RegionID: 11, RegionName: "root.luzon.ncr.manila", Reason: "搬家",
		EffectiveFrom: fixedTime, EffectiveTo: &fixedTimeTo,
	})
	if err != nil {
		t.Fatalf("AppendCustomerHistory: %v", err)
	}
	if id != 7 {
		t.Fatalf("id=%d, want 7", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_ListProductPriceHistories 契约:按产品过滤调价台账;offerID=0 返回全部。
func TestPGStore_ListProductPriceHistories(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, offer_id, old_monthly_fee, new_monthly_fee, effective_at, COALESCE\(reason, ''\), COALESCE\(operator_account_id, 0\)`).
		WithArgs(int64(2)).
		WillReturnRows(mock.NewRows([]string{
			"id", "offer_id", "old_monthly_fee", "new_monthly_fee", "effective_at", "reason", "operator_account_id",
		}).AddRow(int64(1), int64(2), 99.00, 129.00, fixedTime, "涨价", int64(3)))

	s := NewPGStore(mock)
	got, err := s.ListProductPriceHistories(context.Background(), 2)
	if err != nil {
		t.Fatalf("ListProductPriceHistories: %v", err)
	}
	if len(got) != 1 || got[0].NewMonthlyFee != 129.00 || got[0].Reason != "涨价" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_AppendProductPriceHistory 契约:追加产品调价台账并返回自增 id。
func TestPGStore_AppendProductPriceHistory(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO product_price_histories`).
		WithArgs(int64(2), 99.00, 129.00, fixedTime, "涨价", int64(3)).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))

	s := NewPGStore(mock)
	id, err := s.AppendProductPriceHistory(context.Background(), ProductPriceHistory{
		OfferID: 2, OldMonthlyFee: 99.00, NewMonthlyFee: 129.00, EffectiveAt: fixedTime,
		Reason: "涨价", OperatorAccountID: 3,
	})
	if err != nil {
		t.Fatalf("AppendProductPriceHistory: %v", err)
	}
	if id != 1 {
		t.Fatalf("id=%d, want 1", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_ListRegionPriceHistories 契约:按区域产品过滤调价台账;regionOfferID=0 返回全部。
func TestPGStore_ListRegionPriceHistories(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, region_offer_id, old_monthly_fee, new_monthly_fee, effective_at, COALESCE\(reason, ''\), COALESCE\(operator_account_id, 0\)`).
		WithArgs(int64(5)).
		WillReturnRows(mock.NewRows([]string{
			"id", "region_offer_id", "old_monthly_fee", "new_monthly_fee", "effective_at", "reason", "operator_account_id",
		}).AddRow(int64(1), int64(5), 79.00, 89.00, fixedTime, "区域调价", int64(3)))

	s := NewPGStore(mock)
	got, err := s.ListRegionPriceHistories(context.Background(), 5)
	if err != nil {
		t.Fatalf("ListRegionPriceHistories: %v", err)
	}
	if len(got) != 1 || got[0].OldMonthlyFee != 79.00 || got[0].Reason != "区域调价" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_AppendRegionPriceHistory 契约:追加区域调价台账并返回自增 id。
func TestPGStore_AppendRegionPriceHistory(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO region_price_histories`).
		WithArgs(int64(5), 79.00, 89.00, fixedTime, "区域调价", nil).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))

	s := NewPGStore(mock)
	id, err := s.AppendRegionPriceHistory(context.Background(), RegionPriceHistory{
		RegionOfferID: 5, OldMonthlyFee: 79.00, NewMonthlyFee: 89.00, EffectiveAt: fixedTime,
		Reason: "区域调价",
	})
	if err != nil {
		t.Fatalf("AppendRegionPriceHistory: %v", err)
	}
	if id != 1 {
		t.Fatalf("id=%d, want 1", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
