package app

import (
	"context"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/domain/user"
)

// fakeReserveReleaser 超时释放桩:记录 cutoff,返回预置 id。
type fakeReserveReleaser struct {
	ids     []int64
	err     error
	cutoffs []time.Time
}

func (f *fakeReserveReleaser) ReleaseExpiredReserves(_ context.Context, cutoff time.Time) ([]int64, error) {
	f.cutoffs = append(f.cutoffs, cutoff)
	return f.ids, f.err
}

// fakeParamLister biz_params 桩。
type fakeParamLister struct{ params []user.Param }

func (f *fakeParamLister) ListParams(context.Context) ([]user.Param, error) {
	return f.params, nil
}

// fakeNotify 提醒中心桩:只记 Emit。
type fakeNotify struct{ inputs []notify.Input }

func (f *fakeNotify) Emit(_ context.Context, in notify.Input) error {
	f.inputs = append(f.inputs, in)
	return nil
}
func (f *fakeNotify) Resolve(context.Context, string, string) error { return nil }
func (f *fakeNotify) List(context.Context, string, int64, notify.Filter) ([]notify.Item, int, error) {
	return nil, 0, nil
}
func (f *fakeNotify) UnreadCount(context.Context, string, int64) (int, error) {
	return 0, nil
}
func (f *fakeNotify) MarkRead(context.Context, string, int64, []int64) error { return nil }

func TestReserveTimeoutMinutes(t *testing.T) {
	if got := reserveTimeoutMinutes(context.Background(), nil); got != reserveTimeoutDefault {
		t.Fatalf("nil lister: got %d want default %d", got, reserveTimeoutDefault)
	}
	cases := []struct {
		value string
		want  int
	}{
		{"", reserveTimeoutDefault},
		{"60", 60},
		{"abc", reserveTimeoutDefault},
		{"0", reserveTimeoutDefault},
		{"99999", reserveTimeoutMax},
	}
	for _, c := range cases {
		l := &fakeParamLister{}
		if c.value != "" {
			l.params = []user.Param{{Key: reserveTimeoutParamKey, Value: c.value}}
		}
		if got := reserveTimeoutMinutes(context.Background(), l); got != c.want {
			t.Fatalf("value %q: got %d want %d", c.value, got, c.want)
		}
	}
}

func TestRunReserveTimeoutOnce(t *testing.T) {
	t.Run("释放后逐单发 task 通知", func(t *testing.T) {
		rel := &fakeReserveReleaser{ids: []int64{11, 12}}
		n := &fakeNotify{}
		runReserveTimeoutOnce(context.Background(), reserveTimeoutDeps{rel: rel, l: &fakeParamLister{}, n: n})
		if len(rel.cutoffs) != 1 {
			t.Fatalf("rounds=%d", len(rel.cutoffs))
		}
		drift := time.Since(rel.cutoffs[0]) - reserveTimeoutDefault*time.Minute
		if drift < -time.Minute || drift > time.Minute {
			t.Fatalf("cutoff drift: %v", drift)
		}
		if len(n.inputs) != 2 {
			t.Fatalf("notices=%d", len(n.inputs))
		}
		first := n.inputs[0]
		if first.Category != notify.CategoryTask || first.RefType != "order_reserve_timeout" || first.RefID != "11" {
			t.Fatalf("notice=%+v", first)
		}
	})
	t.Run("阈值 60 分钟生效", func(t *testing.T) {
		rel := &fakeReserveReleaser{}
		l := &fakeParamLister{params: []user.Param{{Key: reserveTimeoutParamKey, Value: "60"}}}
		runReserveTimeoutOnce(context.Background(), reserveTimeoutDeps{rel: rel, l: l})
		drift := time.Since(rel.cutoffs[0]) - 60*time.Minute
		if drift < -time.Minute || drift > time.Minute {
			t.Fatalf("cutoff drift: %v", drift)
		}
	})
	t.Run("释放失败不通知不中断", func(t *testing.T) {
		rel := &fakeReserveReleaser{err: context.DeadlineExceeded}
		n := &fakeNotify{}
		runReserveTimeoutOnce(context.Background(), reserveTimeoutDeps{rel: rel, l: &fakeParamLister{}, n: n})
		if len(n.inputs) != 0 {
			t.Fatalf("notices=%d, want 0", len(n.inputs))
		}
	})
	t.Run("无 Notify 装配也安全", func(t *testing.T) {
		rel := &fakeReserveReleaser{ids: []int64{1}}
		runReserveTimeoutOnce(context.Background(), reserveTimeoutDeps{rel: rel, l: &fakeParamLister{}})
	})
}

func TestStartReserveTimeoutLoop(t *testing.T) {
	t.Run("Order 无能力时空操作", func(t *testing.T) {
		a := &Application{}
		stop := startReserveTimeoutLoop(a)
		stop()
	})
	t.Run("stop 幂等且可重复", func(t *testing.T) {
		d := reserveTimeoutDeps{rel: &fakeReserveReleaser{}}
		stop := runReserveTimeoutLoop(d)
		stop()
		stop()
	})
}
