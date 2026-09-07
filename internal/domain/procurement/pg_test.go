package procurement

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// stubExists 桩 exists() 接口,避免单测 mock 这条 SQL。
// 当前实现走 db.QueryRow + 真实表;pgxmock 通过 WithArgs + WillReturnRows 即可。

func TestPGStore_CreateSupplier_OK(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`INSERT INTO procurement_suppliers`).
		WithArgs("S-01", "上海供应商", "张三", "13800000000", int64(1), "ENABLED", "", "MATERIAL", "").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(42)))

	s := NewPGStore(mock)
	id, err := s.CreateSupplier(context.Background(), Supplier{
		Code: "S-01", Name: "上海供应商",
		LegalEntityID: 1, ContactName: "张三", ContactPhone: "13800000000",
	})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}
	if id != 42 {
		t.Fatalf("got id=%d want 42", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_SubmitOrder_InvalidTransition(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectExec(`UPDATE procurement_orders SET status='SUBMITTED'`).
		WithArgs(int64(99)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	s := NewPGStore(mock)
	err = s.SubmitOrder(context.Background(), 99)
	if err != ErrInvalidTransition {
		t.Fatalf("got %v want ErrInvalidTransition", err)
	}
}

func TestPGStore_ListInventory_OK(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`FROM assets a`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"code", "batch_id", "in_stock_qty"}).
			AddRow("RK-20260828-PO-12345", int64(7), int32(20)).
			AddRow("RK-20260828-PO-67890", int64(8), int32(5)))

	s := NewPGStore(mock)
	list, err := s.ListInventory(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("list inventory: %v", err)
	}
	if len(list) != 2 || list[0].InStockQty != 20 {
		t.Fatalf("got=%+v", list)
	}
}

// TestTimeNow_Injection 测试 timeNow 可被替换(避免真实时间干扰断言)。
func TestTimeNow_Injection(t *testing.T) {
	frozen := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	orig := timeNow
	timeNow = func() time.Time { return frozen }
	defer func() { timeNow = orig }()
	if got := nextProcurementNo(); got[:11] != "PO-20260828" {
		t.Fatalf("nextProcurementNo got=%s want prefix PO-20260828", got)
	}
	if got := nextReceiptNo(); got[:11] != "RC-20260828" {
		t.Fatalf("nextReceiptNo got=%s want prefix RC-20260828", got)
	}
}
