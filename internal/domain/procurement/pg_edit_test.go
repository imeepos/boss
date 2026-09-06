package procurement

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// P2-W2-T2 域层单测:A 供应商编辑 / B 供应商启用 / C 草稿编辑 / D 单详情 / E 入库驳回。
// mock 期望按调用顺序排列;ExpectationsWereMet 同时断言"不该发生的 SQL 未发生"。

func newMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { mock.Close() })
	return mock
}

func strPtr(s string) *string { return &s }

// orderHeadRow 16 列(orderCols)单头行。
func orderHeadRow(id int64, status string) *pgxmock.Rows {
	return pgxmock.NewRows([]string{"c1", "c2", "c3", "c4", "c5", "c6", "c7", "c8",
		"c9", "c10", "c11", "c12", "c13", "c14", "c15", "c16"}).
		AddRow(id, "PO-20260905-00001", int64(1), "总公司", int64(3), "上海供应商",
			status, 240.0, nil, "", int64(7), t0(), t0(), nil, nil, nil)
}

// ---------- A:供应商编辑 ----------

func TestUpdateSupplier_OK(t *testing.T) {
	mock := newMock(t)
	mock.ExpectExec("SET name = COALESCE").
		WithArgs(int64(5), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	s := NewPGStore(mock)
	err := s.UpdateSupplier(context.Background(), 5, SupplierUpdate{
		Name: strPtr("新名称"), ContactPhone: strPtr("13900000000")})
	if err != nil {
		t.Fatalf("update supplier: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestUpdateSupplier_EmptyNameRejected(t *testing.T) {
	mock := newMock(t)
	s := NewPGStore(mock)
	err := s.UpdateSupplier(context.Background(), 5, SupplierUpdate{Name: strPtr("")})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("got %v want ErrInvalidInput", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("空名必须零 SQL: %v", err)
	}
}

func TestUpdateSupplier_NotFound(t *testing.T) {
	mock := newMock(t)
	mock.ExpectExec("SET name = COALESCE").
		WithArgs(int64(404), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	s := NewPGStore(mock)
	if err := s.UpdateSupplier(context.Background(), 404, SupplierUpdate{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v want ErrNotFound", err)
	}
}

// 编辑 SQL 不允许出现 status 过滤:禁用态同样可改资料(结构性保证)。
func TestUpdateSupplier_SQLHasNoStatusFilter(t *testing.T) {
	mock := newMock(t)
	mock.ExpectExec("UPDATE procurement_suppliers[^']*WHERE id = [$]1").
		WithArgs(int64(5), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	s := NewPGStore(mock)
	if err := s.UpdateSupplier(context.Background(), 5, SupplierUpdate{Remark: strPtr("x")}); err != nil {
		t.Fatalf("disabled supplier edit: %v", err)
	}
}

// ---------- B:供应商启用 ----------

func TestEnableSupplier_FromDisabled(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery("SELECT status FROM procurement_suppliers").
		WithArgs(int64(5)).WillReturnRows(pgxmock.NewRows([]string{"status"}).AddRow("DISABLED"))
	mock.ExpectExec("UPDATE procurement_suppliers SET status=").
		WithArgs(int64(5)).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	s := NewPGStore(mock)
	if err := s.EnableSupplier(context.Background(), 5); err != nil {
		t.Fatalf("enable: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestEnableSupplier_Idempotent(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery("SELECT status FROM procurement_suppliers").
		WithArgs(int64(5)).WillReturnRows(pgxmock.NewRows([]string{"status"}).AddRow("ENABLED"))
	s := NewPGStore(mock)
	if err := s.EnableSupplier(context.Background(), 5); err != nil {
		t.Fatalf("idempotent enable: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("已启用不得再发 UPDATE: %v", err)
	}
}

func TestEnableSupplier_NotFound(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery("SELECT status FROM procurement_suppliers").
		WithArgs(int64(404)).WillReturnError(pgx.ErrNoRows)
	s := NewPGStore(mock)
	if err := s.EnableSupplier(context.Background(), 404); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v want ErrNotFound", err)
	}
}

// ---------- C:采购单草稿编辑 ----------

func TestUpdateOrderDraft_ReplaceItemsAndRetotal(t *testing.T) {
	mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT status FROM procurement_orders").
		WithArgs(int64(7)).WillReturnRows(pgxmock.NewRows([]string{"status"}).AddRow("DRAFT"))
	mock.ExpectExec("DELETE FROM procurement_order_items").
		WithArgs(int64(7)).WillReturnResult(pgxmock.NewResult("DELETE", 2))
	mock.ExpectExec("INSERT INTO procurement_order_items").
		WithArgs(int64(7), "ONU-X", "", int32(2), 100.0, "").WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec("INSERT INTO procurement_order_items").
		WithArgs(int64(7), "OND-Y", "", int32(1), 40.0, "").WillReturnResult(pgxmock.NewResult("INSERT", 1))
	// 总金额 = 2*100 + 1*40 = 240,随明细重算
	mock.ExpectExec("SET remark = COALESCE").
		WithArgs(int64(7), pgxmock.AnyArg(), pgxmock.AnyArg(), 240.0).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()
	s := NewPGStore(mock)
	err := s.UpdateOrderDraft(context.Background(), 7, OrderDraftUpdate{
		Remark: strPtr("加急"),
		Items: []OrderItem{
			{MaterialCode: "ONU-X", Quantity: 2, UnitAmount: 100.0},
			{MaterialCode: "OND-Y", Quantity: 1, UnitAmount: 40.0},
		}})
	if err != nil {
		t.Fatalf("update draft: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestUpdateOrderDraft_KeepItemsRecomputeTotal(t *testing.T) {
	mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT status FROM procurement_orders").
		WithArgs(int64(7)).WillReturnRows(pgxmock.NewRows([]string{"status"}).AddRow("DRAFT"))
	mock.ExpectQuery("SUM[(]quantity [*] unit_amount[)]").
		WithArgs(int64(7)).WillReturnRows(pgxmock.NewRows([]string{"sum"}).AddRow(350.5))
	mock.ExpectExec("SET remark = COALESCE").
		WithArgs(int64(7), pgxmock.AnyArg(), pgxmock.AnyArg(), 350.5).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()
	s := NewPGStore(mock)
	if err := s.UpdateOrderDraft(context.Background(), 7, OrderDraftUpdate{Remark: strPtr("改备注")}); err != nil {
		t.Fatalf("update draft keep items: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestUpdateOrderDraft_NonDraftConflict(t *testing.T) {
	mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT status FROM procurement_orders").
		WithArgs(int64(7)).WillReturnRows(pgxmock.NewRows([]string{"status"}).AddRow("SUBMITTED"))
	mock.ExpectRollback()
	s := NewPGStore(mock)
	err := s.UpdateOrderDraft(context.Background(), 7, OrderDraftUpdate{Items: []OrderItem{{MaterialCode: "X", Quantity: 1}}})
	if !errors.Is(err, ErrStateConflict) {
		t.Fatalf("got %v want ErrStateConflict(40900)", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestUpdateOrderDraft_ZeroQuantityRejected(t *testing.T) {
	mock := newMock(t)
	s := NewPGStore(mock)
	err := s.UpdateOrderDraft(context.Background(), 7, OrderDraftUpdate{
		Items: []OrderItem{{MaterialCode: "X", Quantity: 0}}})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("got %v want ErrInvalidInput", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("非法数量必须零 SQL: %v", err)
	}
}

func TestUpdateOrderDraft_NotFound(t *testing.T) {
	mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT status FROM procurement_orders").
		WithArgs(int64(404)).WillReturnError(pgx.ErrNoRows)
	mock.ExpectRollback()
	s := NewPGStore(mock)
	if err := s.UpdateOrderDraft(context.Background(), 404, OrderDraftUpdate{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v want ErrNotFound", err)
	}
}

// ---------- D:采购单详情 ----------

func TestGetOrderDetail_HeadPlusItems(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery("FROM procurement_orders WHERE id=").
		WithArgs(int64(7)).WillReturnRows(orderHeadRow(7, "DRAFT"))
	mock.ExpectQuery("FROM procurement_order_items WHERE order_id").
		WithArgs(int64(7)).WillReturnRows(pgxmock.NewRows([]string{"c1", "c2", "c3", "c4", "c5", "c6", "c7", "c8", "c9", "c10"}).
		AddRow(int64(1), int64(7), "ONU-X", "", int32(2), int32(0), 100.0, "", t0(), t0()).
		AddRow(int64(2), int64(7), "OND-Y", "白", int32(1), int32(0), 40.0, "", t0(), t0()))
	s := NewPGStore(mock)
	o, err := s.GetOrderDetail(context.Background(), 7)
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if o.Status != "DRAFT" || len(o.Items) != 2 || o.Items[1].MaterialCode != "OND-Y" {
		t.Fatalf("o=%+v items=%+v", o, o.Items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestGetOrderDetail_NotFound(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery("FROM procurement_orders WHERE id=").
		WithArgs(int64(404)).WillReturnError(pgx.ErrNoRows)
	s := NewPGStore(mock)
	if _, err := s.GetOrderDetail(context.Background(), 404); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v want ErrNotFound", err)
	}
}

// ---------- E:入库驳回 ----------

func TestRejectReceipt_DraftToRejected(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery("SELECT status FROM procurement_receipts").
		WithArgs(int64(9)).WillReturnRows(pgxmock.NewRows([]string{"status"}).AddRow("DRAFT"))
	mock.ExpectExec("UPDATE procurement_receipts SET status=").
		WithArgs(int64(9), "质检破损退回").WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	s := NewPGStore(mock)
	if err := s.RejectReceipt(context.Background(), 9, "质检破损退回"); err != nil {
		t.Fatalf("reject: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestRejectReceipt_ConfirmedConflict(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery("SELECT status FROM procurement_receipts").
		WithArgs(int64(9)).WillReturnRows(pgxmock.NewRows([]string{"status"}).AddRow("CONFIRMED"))
	s := NewPGStore(mock)
	if err := s.RejectReceipt(context.Background(), 9, "x"); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("got %v want ErrStateConflict(40900)", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("CONFIRMED 不得发 UPDATE: %v", err)
	}
}

func TestRejectReceipt_NotFound(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery("SELECT status FROM procurement_receipts").
		WithArgs(int64(404)).WillReturnError(pgx.ErrNoRows)
	s := NewPGStore(mock)
	if err := s.RejectReceipt(context.Background(), 404, "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v want ErrNotFound", err)
	}
}

func TestRejectReceipt_ReasonTooLong(t *testing.T) {
	mock := newMock(t)
	s := NewPGStore(mock)
	long := make([]byte, 256)
	for i := range long {
		long[i] = 'a'
	}
	if err := s.RejectReceipt(context.Background(), 9, string(long)); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("got %v want ErrInvalidInput", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("超长原因必须零 SQL: %v", err)
	}
}
