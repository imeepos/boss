package app

import (
	"context"
	"testing"

	"github.com/ymm-001/boss/internal/domain/metric"
	"github.com/ymm-001/boss/internal/domain/report"
)

type fakeETLScannerStore struct{ freshness []metric.Freshness }

func (f *fakeETLScannerStore) ListJobs(context.Context) ([]metric.ETLJob, error) { return nil, nil }
func (f *fakeETLScannerStore) GetJob(context.Context, string) (*metric.ETLJob, error) {
	return nil, nil
}
func (f *fakeETLScannerStore) UpsertJob(context.Context, metric.ETLJob) (*metric.ETLJob, error) {
	return nil, nil
}
func (f *fakeETLScannerStore) DisableJob(context.Context, string) error          { return nil }
func (f *fakeETLScannerStore) RecordRun(context.Context, metric.ETLJobRun) error { return nil }
func (f *fakeETLScannerStore) ListFreshness(context.Context) ([]metric.Freshness, error) {
	return f.freshness, nil
}
func (f *fakeETLScannerStore) ListRuns(context.Context, string) ([]metric.ETLJobRun, error) {
	return nil, nil
}

type fakeETLCompStore struct{ inserted []report.CompTask }

func (f *fakeETLCompStore) ListCompTasks(_ context.Context, filter report.CompTaskFilter) ([]report.CompTask, int, error) {
	out := make([]report.CompTask, 0)
	for _, t := range f.inserted {
		if filter.Status != "" && t.Status != filter.Status || filter.BizType != "" && t.BizType != filter.BizType {
			continue
		}
		out = append(out, t)
	}
	return out, len(out), nil
}
func (f *fakeETLCompStore) GetCompTask(context.Context, int64) (*report.CompTask, error) {
	return nil, report.ErrCompTaskNotFound
}
func (f *fakeETLCompStore) CreateCompTask(context.Context, *report.CompTask) error    { return nil }
func (f *fakeETLCompStore) UpdateCompTask(context.Context, *report.CompTask) error    { return nil }
func (f *fakeETLCompStore) ClaimCompTask(context.Context, int64, int64, string) error { return nil }
func (f *fakeETLCompStore) TransferCompTask(context.Context, int64, int64, int64, string, string) error {
	return nil
}
func (f *fakeETLCompStore) CloseCompTask(context.Context, int64, int64, string, string) error {
	return nil
}
func (f *fakeETLCompStore) IncrementRetry(context.Context, int64) error { return nil }
func (f *fakeETLCompStore) BatchInsertCompTasks(_ context.Context, tasks []report.CompTask) error {
	f.inserted = append(f.inserted, tasks...)
	return nil
}

func TestETLScannerDispatchesOverdueIdempotently(t *testing.T) {
	etl := &fakeETLScannerStore{freshness: []metric.Freshness{
		{JobKey: "orders", Name: "订单投影", Status: metric.Overdue, LatenessThreshold: 15},
		{JobKey: "fresh", Name: "实时投影", Status: metric.Fresh, LatenessThreshold: 15},
	}}
	comp := &fakeETLCompStore{}
	scanner := NewETLScanner(etl, report.NewCompTaskService(comp))
	got, err := scanner.Scan(context.Background())
	if err != nil || got.Checked != 2 || got.Overdue != 1 || got.Dispatched != 1 {
		t.Fatalf("result=%+v err=%v", got, err)
	}
	got, err = scanner.Scan(context.Background())
	if err != nil || got.Dispatched != 0 {
		t.Fatalf("second result=%+v err=%v", got, err)
	}
	if latest := scanner.Latest(); latest.Overdue != 1 {
		t.Fatalf("latest=%+v", latest)
	}
}
