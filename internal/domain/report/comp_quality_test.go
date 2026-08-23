package report

import (
	"context"
	"testing"
)

func TestSubmitQualityViolationsEmpty(t *testing.T) {
	svc := NewCompTaskService(&fakeQualityCompStore{})
	n, err := svc.SubmitQualityViolations(context.Background(), nil, "ops")
	if err != nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func TestSubmitQualityViolationsCreatesTasks(t *testing.T) {
	store := &fakeQualityCompStore{}
	svc := NewCompTaskService(store)
	violations := []QualityViolationInput{
		{RuleKey: "completeness_orders", Severity: "CRITICAL", Scope: "orders", Detail: "count=0", Observed: 0},
		{RuleKey: "uniqueness_customers", Severity: "WARN", Scope: "customers", Detail: "dup found", Observed: 2},
	}
	n, err := svc.SubmitQualityViolations(context.Background(), violations, "ops")
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("n=%d, want 2", n)
	}
	if len(store.inserted) != 2 {
		t.Fatalf("inserted=%d, want 2", len(store.inserted))
	}
	if store.inserted[0].Source != "QUALITY" || store.inserted[0].BizType != "metric_quality" {
		t.Fatalf("first=%+v", store.inserted[0])
	}
	if store.inserted[0].Priority != PriorityUrgent {
		t.Fatalf("priority=%s, want URGENT", store.inserted[0].Priority)
	}
	if store.inserted[1].Priority != PriorityHigh {
		t.Fatalf("priority=%s, want HIGH", store.inserted[1].Priority)
	}
}

func TestSubmitQualityViolationsDeduplicatesExisting(t *testing.T) {
	store := &fakeQualityCompStore{
		existing: []CompTask{
			{BizType: "metric_quality", BizID: "completeness_orders", Status: TaskStatusOpen},
		},
	}
	svc := NewCompTaskService(store)
	violations := []QualityViolationInput{
		{RuleKey: "completeness_orders", Severity: "CRITICAL", Scope: "orders", Detail: "count=0"},
		{RuleKey: "new_rule", Severity: "WARN", Scope: "payments", Detail: "slow"},
	}
	n, err := svc.SubmitQualityViolations(context.Background(), violations, "ops")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("n=%d, want 1 (dedup existing)", n)
	}
}

// fakeQualityCompStore 是 CompTaskStore 的内存测试桩（质量派单专用）。
type fakeQualityCompStore struct {
	existing []CompTask
	inserted []CompTask
}

func (f *fakeQualityCompStore) ListCompTasks(_ context.Context, filter CompTaskFilter) ([]CompTask, int, error) {
	out := make([]CompTask, 0)
	for _, t := range f.existing {
		if filter.Status != "" && t.Status != filter.Status {
			continue
		}
		if filter.BizType != "" && t.BizType != filter.BizType {
			continue
		}
		out = append(out, t)
	}
	return out, len(out), nil
}

func (f *fakeQualityCompStore) GetCompTask(_ context.Context, _ int64) (*CompTask, error) {
	return nil, ErrCompTaskNotFound
}

func (f *fakeQualityCompStore) BatchInsertCompTasks(_ context.Context, tasks []CompTask) error {
	f.inserted = append(f.inserted, tasks...)
	return nil
}

func (f *fakeQualityCompStore) UpdateCompTask(_ context.Context, _ *CompTask) error {
	return nil
}

func (f *fakeQualityCompStore) CreateCompTask(_ context.Context, _ *CompTask) error {
	return nil
}

func (f *fakeQualityCompStore) ClaimCompTask(_ context.Context, _, _ int64, _ string) error {
	return nil
}

func (f *fakeQualityCompStore) TransferCompTask(_ context.Context, _, _, _ int64, _, _ string) error {
	return nil
}

func (f *fakeQualityCompStore) CloseCompTask(_ context.Context, _, _ int64, _, _ string) error {
	return nil
}

func (f *fakeQualityCompStore) IncrementRetry(_ context.Context, _ int64) error {
	return nil
}
