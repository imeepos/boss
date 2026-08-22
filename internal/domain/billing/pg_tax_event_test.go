package billing

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// TestPGStore_AppendTaxEvent 契约:轨迹行落库含操作人。
func TestPGStore_AppendTaxEvent(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery(`INSERT INTO invoice_tax_events`).
		WithArgs(int64(7), TaxEventReceipt, TaxStatusIssued, "24122000000012345678", "", int64(103)).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))
	id, err := NewPGStore(mock).AppendTaxEvent(context.Background(), TaxEvent{
		InvoiceID: 7, Event: TaxEventReceipt, TaxStatusAfter: TaxStatusIssued,
		TaxNo: "24122000000012345678", OperatorAccountID: 103,
	})
	if err != nil || id != 1 {
		t.Fatalf("id=%d err=%v", id, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestPGStore_ListTaxEvents 契约:按发票回放,时间正序。
func TestPGStore_ListTaxEvents(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	t0 := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(time.Hour)
	mock.ExpectQuery(`FROM invoice_tax_events`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{
			"id", "invoice_id", "event", "tax_status_after", "tax_no", "fail_reason", "operator_account_id", "created_at",
		}).
			AddRow(int64(1), int64(7), TaxEventReceipt, TaxStatusFailed, "", "signature invalid", int64(103), t0).
			AddRow(int64(2), int64(7), TaxEventReceipt, TaxStatusIssued, "24122000000012345678", "", int64(103), t1))
	events, err := NewPGStore(mock).ListTaxEvents(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Event != TaxEventReceipt || events[0].FailReason != "signature invalid" ||
		events[1].TaxNo != "24122000000012345678" || events[1].CreatedAt.Before(events[0].CreatedAt) {
		t.Fatalf("events=%+v", events)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
