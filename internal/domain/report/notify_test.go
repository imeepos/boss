package report

// 推送通道单测:快照生成后经 Notifier 投递,key=period,value=快照信封。

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type stubNotifier struct {
	keys  []string
	snaps []*Snapshot
	err   error
}

func (s *stubNotifier) Notify(_ context.Context, key string, snap *Snapshot) error {
	if s.err != nil {
		return s.err
	}
	s.keys = append(s.keys, key)
	s.snaps = append(s.snaps, snap)
	return nil
}

func TestPushDelegates(t *testing.T) {
	nt := &stubNotifier{}
	svc := &ReportService{Ana: stubAna{ind: sampleIndicators()}, St: &stubStore{}, Nt: nt}
	snap, err := svc.Generate(context.Background(), "daily", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Push(context.Background(), snap); err != nil {
		t.Fatal(err)
	}
	if len(nt.keys) != 1 || nt.keys[0] != "daily" || nt.snaps[0] != snap {
		t.Fatalf("pushed keys=%v snaps=%v", nt.keys, nt.snaps)
	}
}

func TestPushNilNotifier(t *testing.T) {
	svc := &ReportService{Ana: stubAna{ind: sampleIndicators()}, St: &stubStore{}}
	snap, err := svc.Generate(context.Background(), "daily", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Push(context.Background(), snap); !errors.Is(err, ErrNoNotifier) {
		t.Fatalf("err=%v want ErrNoNotifier", err)
	}
}

func TestPushErrorPropagates(t *testing.T) {
	boom := errors.New("boom")
	svc := &ReportService{Nt: &stubNotifier{err: boom}}
	if err := svc.Push(context.Background(), &Snapshot{}); !errors.Is(err, boom) {
		t.Fatalf("err=%v want boom", err)
	}
}

func TestNoopNotifier(t *testing.T) {
	if err := (NoopNotifier{}).Notify(context.Background(), "daily", &Snapshot{}); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotEnvelopeJSON(t *testing.T) {
	// 信封结构稳定:type/period/window/payload,消费方按此契约解析。
	at := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	snap := &Snapshot{ID: 7, Period: "weekly", WindowStart: at.Add(-7 * 24 * time.Hour),
		WindowEnd: at, Payload: []byte(`{"conclusions":["x"]}`), CreatedAt: at}
	env := newEnvelope(snap)
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got["type"] != "report.snapshot" || got["period"] != "weekly" || got["snapshotId"] != float64(7) {
		t.Fatalf("envelope=%v", got)
	}
}
