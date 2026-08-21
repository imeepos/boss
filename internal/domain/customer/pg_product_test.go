package customer

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// TestPGStore_ListProducts 契约:返回产品,按公司过滤。
func TestPGStore_ListProducts(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, legal_entity_id, name, bandwidth, monthly_fee, category, effective_at, status FROM product_offers`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"id", "legal_entity_id", "name", "bandwidth", "monthly_fee", "category", "effective_at", "status"}).
			AddRow(int64(1), int64(1), "300M 畅享宽带", "300M", 99.0, "broadband", fixedTime, "PUBLISHED").
			AddRow(int64(2), int64(1), "1000M 极速宽带", "1000M", 199.0, "broadband", fixedTime, "PUBLISHED"))

	s := NewPGStore(mock)
	got, err := s.ListProducts(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if len(got) != 2 || got[0].Name != "300M 畅享宽带" || got[1].MonthlyFee != 199.0 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_CreateProduct 契约:新建产品返回自增 id。
func TestPGStore_CreateProduct(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// FK validation: legal entity exists
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`INSERT INTO product_offers`).
		WithArgs(int64(1), "500M 畅享宽带", "500M", 129.0, "broadband", fixedTime, "DRAFT").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(3)))

	s := NewPGStore(mock)
	id, err := s.CreateProduct(context.Background(), ProductOffer{
		LegalEntityID: 1, Name: "500M 畅享宽带", Bandwidth: "500M",
		MonthlyFee: 129.0, EffectiveAt: fixedTime, Status: "DRAFT",
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	if id != 3 {
		t.Fatalf("id=%d, want 3", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_ListRegionOffers 契约:返回区域运营包,空名/空原因归一为 ""。
func TestPGStore_ListRegionOffers(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, offer_id, region_path, COALESCE\(name, ''\), monthly_fee, COALESCE\(reason, ''\) FROM region_offers`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"id", "offer_id", "region_path", "name", "monthly_fee", "reason"}).
			AddRow(int64(1), int64(1), "root.luzon", "", 89.0, "").
			AddRow(int64(2), int64(1), "root.visayas", "促销价", 79.0, "新装立减"))

	s := NewPGStore(mock)
	got, err := s.ListRegionOffers(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListRegionOffers: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	if got[0].Name != "" {
		t.Fatalf("got[0].Name=%q, want empty", got[0].Name)
	}
	if got[1].Name != "促销价" || got[1].Reason != "新装立减" {
		t.Fatalf("got[1]=%+v", got[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_CreateRegionOffer 契约:新建区域运营包返回自增 id。
func TestPGStore_CreateRegionOffer(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// FK validation: offer exists
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`INSERT INTO region_offers`).
		WithArgs(int64(1), "root.luzon.ncr", "区域特惠", 69.0, "").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(4)))

	s := NewPGStore(mock)
	id, err := s.CreateRegionOffer(context.Background(), RegionOffer{
		OfferID: 1, RegionPath: "root.luzon.ncr", Name: "区域特惠", MonthlyFee: 69.0,
	})
	if err != nil {
		t.Fatalf("CreateRegionOffer: %v", err)
	}
	if id != 4 {
		t.Fatalf("id=%d, want 4", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_ChangeProductPrice 契约:产品调价 事务内 更新月费并追加台账,返回台账 id。
func TestPGStore_ChangeProductPrice(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT monthly_fee FROM product_offers WHERE id=\$1 FOR UPDATE`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"monthly_fee"}).AddRow(99.0))
	mock.ExpectExec(`UPDATE product_offers SET monthly_fee=\$2, effective_at=\$3`).
		WithArgs(int64(1), 129.0, fixedTime).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectQuery(`INSERT INTO product_price_histories`).
		WithArgs(int64(1), 99.0, 129.0, fixedTime, "产品调价", nil).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(11)))
	mock.ExpectCommit()

	s := NewPGStore(mock)
	historyID, err := s.ChangeProductPrice(context.Background(), 1, 129.0, fixedTime, "产品调价", 0)
	if err != nil {
		t.Fatalf("ChangeProductPrice: %v", err)
	}
	if historyID != 11 {
		t.Fatalf("historyID=%d, want 11", historyID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_ChangeProductPriceNotFound 契约:产品不存在返回 ErrProductNotFound,事务回滚。
func TestPGStore_ChangeProductPriceNotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT monthly_fee FROM product_offers WHERE id=\$1 FOR UPDATE`).
		WithArgs(int64(99)).
		WillReturnError(pgx.ErrNoRows)
	mock.ExpectRollback()

	_, err = NewPGStore(mock).ChangeProductPrice(context.Background(), 99, 129.0, fixedTime, "产品调价", 0)
	if !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("err=%v, want ErrProductNotFound", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
