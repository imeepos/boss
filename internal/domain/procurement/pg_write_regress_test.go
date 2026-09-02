package procurement

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func t0() time.Time { return time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC) }

// 批次码列宽回归:RK-YYYYMMDD-NNNNN 恒 17 字符,满足 asset_batches.code VARCHAR(32)。
// 2026-09 审计:原实现拼 orderNo,超 20 字符即超列宽,ConfirmReceipt 必炸。
func TestNextBatchCode_FitsColumn(t *testing.T) {
	got := nextBatchCode()
	if len(got) != 17 {
		t.Fatalf("len=%d, want 17: %q", len(got), got)
	}
	if got[:3] != "RK-" {
		t.Fatalf("prefix=%q, want RK-", got[:3])
	}
}

// 资产码列宽/唯一性回归:A-{batchID 8 位}-{seq 5 位} 恒 16 字符,永不超
// assets.asset_code VARCHAR(32);batchID 唯一保证全局不撞。
// 2026-09 审计:原实现拼 batchCode+materialCode,最短 39 字符必超列宽。
func TestNextAssetCode_FitsColumnAndUnique(t *testing.T) {
	c1 := nextAssetCode(1, 1)
	if len(c1) != 16 {
		t.Fatalf("len=%d, want 16: %q", len(c1), c1)
	}
	c2 := nextAssetCode(99999999, 99999)
	if len(c2) != 16 {
		t.Fatalf("max len=%d, want 16: %q", len(c2), c2)
	}
	if c1 == c2 || c1 == nextAssetCode(1, 2) {
		t.Fatalf("collision: %q %q", c1, c2)
	}
}

// 入库单列表空值回归:DRAFT 行 batch_id/received_by/received_at 为 NULL,
// 列表查询不得炸行(2026-09 审计:非指针 Scan 遇 NULL 即错)。
func TestListReceipts_NullBackfillCols(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery("FROM procurement_receipts").
		WillReturnRows(mock.NewRows(
			[]string{"id", "receipt_no", "order_id", "order_no", "batch_id",
				"legal_entity_id", "legal_entity_name", "received_by", "received_at",
				"status", "remark", "created_at", "updated_at"},
		).AddRow(int64(9), "RC-20260902-00001", int64(3), "PO-20260901-001", nil,
			int64(1), "某某实业", nil, nil,
			"DRAFT", "", t0(), t0()))

	s := NewPGStore(mock)
	rs, err := s.ListReceipts(context.Background(), 0)
	if err != nil {
		t.Fatalf("ListReceipts with NULL backfill cols: %v", err)
	}
	if len(rs) != 1 || rs[0].BatchID != 0 || rs[0].ReceivedBy != 0 || !rs[0].ReceivedAt.IsZero() {
		t.Fatalf("rs=%+v, want 零值回填", rs)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
