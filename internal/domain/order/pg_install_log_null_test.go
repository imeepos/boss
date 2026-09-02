package order

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// signed_at 可空回归:OPEN 态回单 signed_at 为 NULL,列表查询不得炸行
// (2026-09 审计:写路径 nullIfZero 允许 NULL,读路径扫非可空 time.Time)。
func TestListInstallLogs_NullSignedAt(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`FROM install_logs`).
		WithArgs(int64(2)).
		WillReturnRows(pgxmock.NewRows(
			[]string{"id", "ticket_id", "order_id", "worker_id", "worker_name", "photos",
				"sign_name", "sign_image_url", "signed_at", "note", "status", "created_at", "updated_at"},
		).AddRow(int64(1), int64(2), int64(3), int64(4), "张三", []byte("[]"),
			"", "", nil, "", "OPEN", time.Now().UTC(), time.Now().UTC()))

	s := NewPGStore(mock, stubExists{})
	logs, err := s.ListInstallLogs(context.Background(), 2)
	if err != nil {
		t.Fatalf("ListInstallLogs with NULL signed_at: %v", err)
	}
	if len(logs) != 1 || logs[0].SignedAt != nil {
		t.Fatalf("logs=%+v, want 1 行且 SignedAt=nil", logs)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
