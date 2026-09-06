package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/domain/report"
	"github.com/ymm-001/boss/internal/domain/resource"
)

// mapParamReader biz_params 内存桩。
type mapParamReader map[string]string

func (m mapParamReader) GetParam(_ context.Context, key string) (string, error) {
	return m[key], nil
}

// fakeAuditRes 稽核执行桩(记录调用次数与入参阈值)。
type fakeAuditRes struct {
	calls      int
	staleHours int
	err        error
}

func (f *fakeAuditRes) AuditInventory(_ context.Context, opts resource.AuditOptions) (*resource.AuditReport, error) {
	f.calls++
	f.staleHours = opts.ReservedStaleHours
	if f.err != nil {
		return nil, f.err
	}
	return &resource.AuditReport{Total: 6, StaleHours: opts.ReservedStaleHours}, nil
}

// fakeOSSAppStore report.Store 全量桩(只关心 Upsert/Latest)。
type fakeOSSAppStore struct {
	upserts int
	last    *report.Snapshot
	latest  *report.Snapshot
}

func (f *fakeOSSAppStore) UpsertSnapshot(_ context.Context, s *report.Snapshot) error {
	f.upserts++
	f.last = s
	f.latest = s
	return nil
}

func (f *fakeOSSAppStore) LatestSnapshot(context.Context, string) (*report.Snapshot, error) {
	if f.latest == nil {
		return nil, report.ErrNoSnapshot
	}
	return f.latest, nil
}

func (f *fakeOSSAppStore) LatestSnapshots(context.Context, string, int) ([]report.Snapshot, error) {
	return nil, nil
}

func (f *fakeOSSAppStore) ListSnapshots(context.Context) ([]report.Snapshot, error) { return nil, nil }

func (f *fakeOSSAppStore) SnapshotByID(context.Context, int64) (*report.Snapshot, error) {
	return nil, report.ErrNoSnapshot
}

func ossAuditDepsFor(st *fakeOSSAppStore, res *fakeAuditRes) ossAuditDeps {
	return ossAuditDeps{res: res, rep: &report.ReportService{St: st}}
}

func TestRunOSSAuditIfDueBeforeHour(t *testing.T) {
	res := &fakeAuditRes{}
	st := &fakeOSSAppStore{}
	d := ossAuditDepsFor(st, res)
	runOSSAuditIfDue(context.Background(), d, time.Date(2026, 9, 6, 2, 0, 0, 0, time.Local))
	if res.calls != 0 || st.upserts != 0 {
		t.Fatalf("未到点不应执行: calls=%d upserts=%d", res.calls, st.upserts)
	}
}

func TestRunOSSAuditIfDueRunsOnceDaily(t *testing.T) {
	res := &fakeAuditRes{}
	st := &fakeOSSAppStore{}
	d := ossAuditDepsFor(st, res)
	now := time.Date(2026, 9, 6, 4, 0, 0, 0, time.Local)
	runOSSAuditIfDue(context.Background(), d, now)
	runOSSAuditIfDue(context.Background(), d, now.Add(30*time.Minute))
	if res.calls != 1 || st.upserts != 1 {
		t.Fatalf("当日应幂等单次: calls=%d upserts=%d", res.calls, st.upserts)
	}
	if st.last == nil || st.last.Period != report.PeriodOSSAuditDaily {
		t.Fatalf("快照周期 = %+v, want %s", st.last, report.PeriodOSSAuditDaily)
	}
}

func TestRunOSSAuditFailureNoSnapshot(t *testing.T) {
	res := &fakeAuditRes{err: errors.New("db down")}
	st := &fakeOSSAppStore{}
	d := ossAuditDepsFor(st, res)
	now := time.Date(2026, 9, 6, 4, 0, 0, 0, time.Local)
	runOSSAuditIfDue(context.Background(), d, now)
	if st.upserts != 0 {
		t.Fatalf("稽核失败不应落快照: upserts=%d", st.upserts)
	}
	// 失败不写当日快照 → 下一轮(未跳过)重试。
	if res.calls != 1 {
		t.Fatalf("calls = %d", res.calls)
	}
}

func TestOSSAuditStaleHoursFallback(t *testing.T) {
	if got := ossAuditStaleHours(context.Background(), nil); got != resource.AuditDefaultStaleHours {
		t.Fatalf("nil reader fallback = %d", got)
	}
	pg := mapParamReader{"resource.audit.reservedStaleHours": "72"}
	if got := ossAuditStaleHours(context.Background(), pg); got != 72 {
		t.Fatalf("param read = %d, want 72", got)
	}
	bad := mapParamReader{"resource.audit.reservedStaleHours": "abc"}
	if got := ossAuditStaleHours(context.Background(), bad); got != resource.AuditDefaultStaleHours {
		t.Fatalf("bad value fallback = %d", got)
	}
}
