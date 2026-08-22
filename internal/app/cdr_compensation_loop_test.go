package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/domain/aaa"
	aaability "github.com/ymm-001/boss/internal/domain/aaa/billing"
)

// fakeCdrCompStore 话单补偿存桩。
type fakeCdrCompStore struct {
	cdrs     []aaa.CdrRecord
	listErr  error
	marked   map[int64]string
	markErr  error
}

func (f *fakeCdrCompStore) ListUnsentCdrs(context.Context, int) ([]aaa.CdrRecord, error) {
	return f.cdrs, f.listErr
}
func (f *fakeCdrCompStore) MarkCdrsKafkaStatus(_ context.Context, ids []int64, status string) error {
	if f.markErr != nil {
		return f.markErr
	}
	if f.marked == nil {
		f.marked = map[int64]string{}
	}
	for _, id := range ids {
		f.marked[id] = status
	}
	return nil
}

// fakeCdrEmitter 投递桩。
type fakeCdrEmitter struct {
	failIDs map[int64]bool
	got     []aaability.CDR
}

func (f *fakeCdrEmitter) Emit(_ context.Context, c aaability.CDR) error {
	f.got = append(f.got, c)
	if c.SessionID == "fail" {
		return errors.New("kafka down")
	}
	return nil
}

func TestRunCdrCompOnce(t *testing.T) {
	mk := func(id int64, sid string) aaa.CdrRecord {
		return aaa.CdrRecord{ID: id, Loid: "L", SessionID: sid, AcctStatus: 1}
	}
	t.Run("补投成功回写 SENT", func(t *testing.T) {
		st := &fakeCdrCompStore{cdrs: []aaa.CdrRecord{mk(1, "s1"), mk(2, "s2")}}
		em := &fakeCdrEmitter{}
		runCdrCompOnce(context.Background(), cdrCompDeps{store: st, emit: em, batch: 100, nowFunc: time.Now})
		if len(em.got) != 2 {
			t.Fatalf("emitted=%d", len(em.got))
		}
		if st.marked[1] != aaa.CdrKafkaSent || st.marked[2] != aaa.CdrKafkaSent {
			t.Fatalf("marked=%v", st.marked)
		}
	})
	t.Run("失败回写 FAILED 并发通知", func(t *testing.T) {
		st := &fakeCdrCompStore{cdrs: []aaa.CdrRecord{mk(1, "ok"), mk(2, "fail")}}
		em := &fakeCdrEmitter{}
		n := &fakeNotify{}
		runCdrCompOnce(context.Background(), cdrCompDeps{store: st, emit: em, n: n, batch: 100, nowFunc: time.Now})
		if st.marked[1] != aaa.CdrKafkaSent || st.marked[2] != aaa.CdrKafkaFailed {
			t.Fatalf("marked=%v", st.marked)
		}
		if len(n.inputs) != 1 || n.inputs[0].RefType != "cdr_compensation" {
			t.Fatalf("notices=%+v", n.inputs)
		}
	})
	t.Run("全部成功不发通知", func(t *testing.T) {
		st := &fakeCdrCompStore{cdrs: []aaa.CdrRecord{mk(1, "ok")}}
		n := &fakeNotify{}
		runCdrCompOnce(context.Background(), cdrCompDeps{store: st, emit: &fakeCdrEmitter{}, n: n, batch: 100, nowFunc: time.Now})
		if len(n.inputs) != 0 {
			t.Fatalf("notices=%d", len(n.inputs))
		}
	})
	t.Run("读库失败安全返回", func(t *testing.T) {
		st := &fakeCdrCompStore{listErr: errors.New("db down")}
		runCdrCompOnce(context.Background(), cdrCompDeps{store: st, emit: &fakeCdrEmitter{}, batch: 100, nowFunc: time.Now})
	})
	t.Run("字段映射无损", func(t *testing.T) {
		em := &fakeCdrEmitter{}
		got := cdrRecordToCDR(aaa.CdrRecord{
			ID: 1, Loid: "LOID", Username: "u", AcctStatus: 2, SessionID: "sid",
			SessionTime: 90, InputOctets: 11, OutputOctets: 22, NasIP: "1.2.3.4",
		})
		_ = em
		if got.LOID != "LOID" || got.AcctStatus != 2 || got.SessionTime != 90 ||
			got.InputOctets != 11 || got.OutputOctets != 22 || got.NASIP != "1.2.3.4" || got.Username != "u" || got.SessionID != "sid" {
			t.Fatalf("map=%+v", got)
		}
	})
}

func TestStartCdrCompensationLoop(t *testing.T) {
	t.Run("依赖缺失空操作", func(t *testing.T) {
		startCdrCompensationLoop(nil, nil, nil)()
		startCdrCompensationLoop(&fakeCdrCompStore{}, nil, nil)()
	})
	t.Run("stop 幂等", func(t *testing.T) {
		stop := startCdrCompensationLoop(&fakeCdrCompStore{}, &fakeCdrEmitter{}, nil)
		stop()
		stop()
	})
}
