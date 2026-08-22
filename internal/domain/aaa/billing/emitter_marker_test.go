package billing

import (
	"context"
	"errors"
	"testing"

	"github.com/segmentio/kafka-go"

	"github.com/ymm-001/boss/internal/domain/aaa"
)

// FanoutEmitter 投递状态回写(Q2 话单补偿,000109)单测。

// fakeLoWriter 记 AppendCdr 返回自增 id。
type fakeLoWriter struct{ nextID int64 }

func (f *fakeLoWriter) AppendCdr(_ context.Context, _ aaa.CdrRecord) (int64, error) {
	f.nextID++
	return f.nextID, nil
}

// markerOKWriter 成功路径 kafka 替身。
type markerOKWriter struct{}

func (markerOKWriter) WriteMessages(context.Context, ...kafka.Message) error { return nil }
func (markerOKWriter) Close() error                                          { return nil }

// fakeMarker 记回写。
type fakeMarker struct {
	calls []struct {
		ids    []int64
		status string
	}
}

func (f *fakeMarker) MarkCdrsKafkaStatus(_ context.Context, ids []int64, status string) error {
	f.calls = append(f.calls, struct {
		ids    []int64
		status string
	}{ids, status})
	return nil
}

func TestFanoutEmitter_KafkaStatusMarking(t *testing.T) {
	t.Run("实时成功置 SENT", func(t *testing.T) {
		w := &fakeLoWriter{}
		m := &fakeMarker{}
		fe := &FanoutEmitter{Realtime: &KafkaEmitter{w: markerOKWriter{}}, Store: NewPGEmitter(w), Marker: m}
		if err := fe.Emit(context.Background(), CDR{LOID: "L1"}); err != nil {
			t.Fatalf("Emit: %v", err)
		}
		if len(m.calls) != 1 || m.calls[0].status != aaa.CdrKafkaSent || m.calls[0].ids[0] != 1 {
			t.Fatalf("calls=%+v", m.calls)
		}
	})
	t.Run("实时失败置 FAILED 且不阻塞落库", func(t *testing.T) {
		w := &fakeLoWriter{}
		m := &fakeMarker{}
		ke := &KafkaEmitter{w: failWriter{}}
		fe := &FanoutEmitter{Realtime: ke, Store: NewPGEmitter(w), Marker: m}
		if err := fe.Emit(context.Background(), CDR{LOID: "L1"}); err != nil {
			t.Fatalf("Kafka 失败不应阻塞: %v", err)
		}
		if w.nextID != 1 {
			t.Fatalf("落库应成功 nextID=%d", w.nextID)
		}
		if len(m.calls) != 1 || m.calls[0].status != aaa.CdrKafkaFailed {
			t.Fatalf("calls=%+v", m.calls)
		}
	})
	t.Run("无 Marker 降级安全", func(t *testing.T) {
		fe := &FanoutEmitter{Realtime: &KafkaEmitter{w: failWriter{}}, Store: NewPGEmitter(&fakeLoWriter{})}
		if err := fe.Emit(context.Background(), CDR{LOID: "L1"}); err != nil {
			t.Fatalf("Emit: %v", err)
		}
	})
}

type failWriter struct{}

func (failWriter) WriteMessages(context.Context, ...kafka.Message) error { return errors.New("kafka down") }
func (failWriter) Close() error                                          { return nil }
