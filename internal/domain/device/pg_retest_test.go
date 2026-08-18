package device

import (
	"context"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// TestPGStore_AppendRetestTask 契约:批量复测落任务行,任务号 RT-YYYYMMDD-NNNN。
func TestPGStore_AppendRetestTask(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery(`INSERT INTO alarm_retest_tasks`).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(42)))
	mock.ExpectExec(`UPDATE alarm_retest_tasks SET task_no=\$1 WHERE id=\$2`).
		WithArgs(pgxmock.AnyArg(), int64(42)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	taskNo, err := NewPGStore(mock).AppendRetestTask(context.Background(), "马尼拉东区")
	if err != nil {
		t.Fatalf("AppendRetestTask: %v", err)
	}
	if !strings.HasPrefix(taskNo, "RT-") || !strings.HasSuffix(taskNo, "-0042") {
		t.Fatalf("taskNo=%q", taskNo)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
