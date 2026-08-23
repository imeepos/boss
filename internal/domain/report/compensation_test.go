package report

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// CompensationTasks(补偿任务中心)形状回归:九类查询全执行,
// retryPath 按 refID 拼装,cdrKafka/webhookDelivery/couponRecon 聚合 0 不出条目。
func TestPGStore_CompensationTasks(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`FROM provision_tasks`).
		WillReturnRows(pgxmock.NewRows([]string{"task_no"}).AddRow("TASK-9"))
	mock.ExpectQuery(`FROM stop_resume_tasks`).
		WillReturnRows(pgxmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`FROM activation_callbacks`).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("7"))
	mock.ExpectQuery(`FROM invoices`).
		WillReturnRows(pgxmock.NewRows([]string{"invoice_no"}).AddRow("INV-00000001"))
	mock.ExpectQuery(`FROM cdrs`).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow("0"))
	mock.ExpectQuery(`FROM reconciliation_batches`).
		WillReturnRows(pgxmock.NewRows([]string{"batch_no"}).AddRow("PC-20260826-01"))
	mock.ExpectQuery(`FROM open_webhook_deliveries`).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow("0"))
	mock.ExpectQuery(`FROM coupons`).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow("0"))
	mock.ExpectQuery(`FROM loy_entries`).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow("0"))

	s := NewPGStore(mock)
	tasks, err := s.CompensationTasks(context.Background())
	if err != nil {
		t.Fatalf("CompensationTasks: %v", err)
	}
	if len(tasks) != 4 {
		t.Fatalf("tasks=%d, want 4(cdrKafka/webhookDelivery/couponRecon/pointsFailed=0 不出条目): %+v", len(tasks), tasks)
	}
	byType := map[string]CompTaskView{}
	for _, t := range tasks {
		byType[t.Type] = t
	}
	if byType["provisionTask"].RetryPath != "/api/admin/v1/provision-tasks/TASK-9/retry" {
		t.Fatalf("provision retryPath=%q", byType["provisionTask"].RetryPath)
	}
	if byType["activationCallback"].RetryPath != "/api/admin/v1/activation-callbacks/7/retry" {
		t.Fatalf("callback retryPath=%q", byType["activationCallback"].RetryPath)
	}
	if byType["taxInvoice"].RetryPath != "" {
		t.Fatalf("taxInvoice retryPath should be empty: %q", byType["taxInvoice"].RetryPath)
	}
	if byType["reconBatch"].RetryPath != "/api/admin/v1/reconciliations/PC-20260826-01/settle" {
		t.Fatalf("reconBatch retryPath=%q", byType["reconBatch"].RetryPath)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CompensationTasks_CdrAggregate(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	for _, q := range []string{`FROM provision_tasks`, `FROM stop_resume_tasks`,
		`FROM activation_callbacks`, `FROM invoices`} {
		mock.ExpectQuery(q).WillReturnRows(pgxmock.NewRows([]string{"x"}))
	}
	mock.ExpectQuery(`FROM cdrs`).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow("12"))
	mock.ExpectQuery(`FROM reconciliation_batches`).
		WillReturnRows(pgxmock.NewRows([]string{"x"}))
	mock.ExpectQuery(`FROM open_webhook_deliveries`).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow("0"))
	mock.ExpectQuery(`FROM coupons`).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow("0"))
	mock.ExpectQuery(`FROM loy_entries`).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow("0"))

	s := NewPGStore(mock)
	tasks, err := s.CompensationTasks(context.Background())
	if err != nil {
		t.Fatalf("CompensationTasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Type != "cdrKafka" || tasks[0].RefID != "" {
		t.Fatalf("tasks=%+v", tasks)
	}
	if tasks[0].Detail == "" || tasks[0].RetryPath != "" {
		t.Fatalf("cdrKafka task=%+v", tasks[0])
	}
}

// fakeCompStore 基于 fakeReconStore(已实现 Store),补 CompTaskLister。
type fakeCompStore struct {
	fakeReconStore
	tasks []CompTaskView
}

func (f *fakeCompStore) CompensationTasks(context.Context) ([]CompTaskView, error) {
	return f.tasks, nil
}

func TestReportService_CompensationTasks(t *testing.T) {
	r := &ReportService{St: &fakeCompStore{tasks: []CompTaskView{{Domain: "order", Type: "activationCallback"}}}}
	got, err := r.CompensationTasks(context.Background())
	if err != nil || len(got) != 1 {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	r2 := &ReportService{St: &fakePatrolStore{}}
	if _, err := r2.CompensationTasks(context.Background()); err != ErrCompUnsupported {
		t.Fatalf("err=%v, want ErrCompUnsupported", err)
	}
}
