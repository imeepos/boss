package billing

import (
	"context"
	"errors"
	"testing"

	"github.com/segmentio/kafka-go"

	"github.com/ymm-001/boss/internal/domain/aaa"
)

// PGEmitter 话单落库单测:字段映射 + UNBILLED 状态。
func TestPGEmitter_Emit(t *testing.T) {
	svc := &fakeAaaWriter{}
	e := NewPGEmitter(svc)
	if err := e.Emit(context.Background(), CDR{
		LOID: "LOID-1", Username: "u1", AcctStatus: 2, SessionID: "S-1",
		SessionTime: 3600, InputOctets: 1024, OutputOctets: 2048, NASIP: "10.0.0.1",
	}); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	c := svc.captured
	if c.Loid != "LOID-1" || c.AcctStatus != 2 || c.SessionTime != 3600 ||
		c.InputOctets != 1024 || c.OutputOctets != 2048 || c.NasIP != "10.0.0.1" {
		t.Fatalf("captured=%+v", c)
	}
	if c.BillingStatus != "UNBILLED" {
		t.Fatalf("billingStatus=%s", c.BillingStatus)
	}
}

type fakeAaaWriter struct{ captured aaa.CdrRecord }

func (f *fakeAaaWriter) AppendCdr(_ context.Context, c aaa.CdrRecord) (int64, error) {
	f.captured = c
	return 1, nil
}

// PGEmitter 落库失败分支。
func TestPGEmitter_EmitError(t *testing.T) {
	want := errors.New("db down")
	e := NewPGEmitter(&errAaaWriter{err: want})
	if err := e.Emit(context.Background(), CDR{}); !errors.Is(err, want) {
		t.Fatalf("err=%v, want wrap %v", err, want)
	}
}

type errAaaWriter struct{ err error }

func (f *errAaaWriter) AppendCdr(_ context.Context, _ aaa.CdrRecord) (int64, error) {
	return 0, f.err
}

// fakeKafkaWriter kafka 替身:可控返回值,捕获消息。
type fakeKafkaWriter struct {
	msgs []kafka.Message
	err  error
}

func (f *fakeKafkaWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	if f.err != nil {
		return f.err
	}
	f.msgs = append(f.msgs, msgs...)
	return nil
}

func (f *fakeKafkaWriter) Close() error { return nil }

// KafkaEmitter 成功投递:JSON 全量 + Key=LOID。
func TestKafkaEmitter_Emit(t *testing.T) {
	f := &fakeKafkaWriter{}
	e := &KafkaEmitter{w: f}
	cdr := CDR{LOID: "LOID-K", Username: "u", AcctStatus: 1, SessionID: "S",
		SessionTime: 60, InputOctets: 1, OutputOctets: 2, NASIP: "10.0.0.1"}
	if err := e.Emit(context.Background(), cdr); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	if len(f.msgs) != 1 || string(f.msgs[0].Key) != "LOID-K" {
		t.Fatalf("msgs=%+v", f.msgs)
	}
}

// KafkaEmitter 投递失败分支。
func TestKafkaEmitter_EmitError(t *testing.T) {
	want := errors.New("kafka down")
	e := &KafkaEmitter{w: &fakeKafkaWriter{err: want}}
	if err := e.Emit(context.Background(), CDR{}); !errors.Is(err, want) {
		t.Fatalf("err=%v, want wrap %v", err, want)
	}
}

// KafkaEmitter 序列化失败分支(测试口注入,正常 CDR 不会触发)。
func TestKafkaEmitter_EmitMarshalError(t *testing.T) {
	old := marshalCDR
	defer func() { marshalCDR = old }()
	marshalCDR = func(any) ([]byte, error) { return nil, errors.New("marshal") }
	e := &KafkaEmitter{w: &fakeKafkaWriter{}}
	if err := e.Emit(context.Background(), CDR{}); err == nil {
		t.Fatal("want marshal error")
	}
}

// NewKafkaEmitter 构造 + Close(不拨号,离线可关)。
func TestNewKafkaEmitter(t *testing.T) {
	e := NewKafkaEmitter([]string{"127.0.0.1:1"}, "boss-cdr")
	if e.w == nil {
		t.Fatal("writer nil")
	}
	if err := e.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// FanoutEmitter:落库成功+实时成功 / 落库失败 / 实时失败不阻塞 / Realtime 为 nil。
func TestFanoutEmitter_Emit(t *testing.T) {
	pgOK := NewPGEmitter(&fakeAaaWriter{})
	cdr := CDR{LOID: "LOID-F"}

	// PG 成功 + Kafka 成功。
	kw := &fakeKafkaWriter{}
	f := &FanoutEmitter{Realtime: &KafkaEmitter{w: kw}, Store: pgOK}
	if err := f.Emit(context.Background(), cdr); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	if len(kw.msgs) != 1 {
		t.Fatalf("kafka msgs=%d", len(kw.msgs))
	}

	// PG 失败直接返回,不触达 Kafka。
	want := errors.New("pg down")
	f = &FanoutEmitter{Realtime: &KafkaEmitter{w: kw}, Store: NewPGEmitter(&errAaaWriter{err: want})}
	if err := f.Emit(context.Background(), cdr); !errors.Is(err, want) {
		t.Fatalf("err=%v, want wrap %v", err, want)
	}
	if len(kw.msgs) != 1 {
		t.Fatalf("kafka should not be touched, msgs=%d", len(kw.msgs))
	}

	// Kafka 失败仅吞掉,不影响结果。
	f = &FanoutEmitter{Realtime: &KafkaEmitter{w: &fakeKafkaWriter{err: errors.New("boom")}}, Store: pgOK}
	if err := f.Emit(context.Background(), cdr); err != nil {
		t.Fatalf("kafka failure should not propagate: %v", err)
	}

	// Realtime 为 nil(未部署 Kafka 环境)。
	f = &FanoutEmitter{Store: pgOK}
	if err := f.Emit(context.Background(), cdr); err != nil {
		t.Fatalf("Emit without realtime: %v", err)
	}
}
