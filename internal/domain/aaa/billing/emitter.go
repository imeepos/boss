package billing

// W6 话单投递落地实现:PGEmitter(落库 cdr_records,账务可查)与 KafkaEmitter(实时链路 → Flink)。
// 双写策略:先 Kafka(实时),失败兜底 PG;或仅 PG(未部署 Kafka 环境)。

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/ymm-001/boss/internal/domain/aaa"
)

// LoAccountWriter AAA 域最小写口(避免 PGEmitter 反向依赖整个 AaaService)。
type LoAccountWriter interface {
	AppendCdr(ctx context.Context, c aaa.CdrRecord) (int64, error)
}

// PGEmitter 话单落库实现:写 cdr_records(billing_status=UNBILLED,出账 GenerateBills 消费)。
type PGEmitter struct{ W LoAccountWriter }

// NewPGEmitter 构造 PG 话单投递器。
func NewPGEmitter(w LoAccountWriter) *PGEmitter { return &PGEmitter{W: w} }

// Emit 话单落库。
func (e *PGEmitter) Emit(ctx context.Context, cdr CDR) error {
	_, err := e.W.AppendCdr(ctx, aaa.CdrRecord{
		Loid: cdr.LOID, Username: cdr.Username, AcctStatus: int16(cdr.AcctStatus),
		SessionID: cdr.SessionID, SessionTime: int32(cdr.SessionTime),
		InputOctets: int64(cdr.InputOctets), OutputOctets: int64(cdr.OutputOctets),
		NasIP: cdr.NASIP, BillingStatus: "UNBILLED", StartedAt: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("aaa/billing: pg emit: %w", err)
	}
	return nil
}

// KafkaEmitter 话单实时投递(Kafka topic → Flink → Doris)。
type KafkaEmitter struct {
	w *kafka.Writer
}

// NewKafkaEmitter 构造 Kafka 话单投递器(brokers 如 192.168.0.102:29092,topic 如 boss-cdr)。
func NewKafkaEmitter(brokers []string, topic string) *KafkaEmitter {
	return &KafkaEmitter{w: &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		BatchTimeout: 10 * time.Millisecond,
	}}
}

// Emit JSON 序列化投递;value 即 CDR 全量。
func (e *KafkaEmitter) Emit(ctx context.Context, cdr CDR) error {
	b, err := json.Marshal(cdr)
	if err != nil {
		return fmt.Errorf("aaa/billing: kafka marshal: %w", err)
	}
	if err := e.w.WriteMessages(ctx, kafka.Message{Key: []byte(cdr.LOID), Value: b}); err != nil {
		return fmt.Errorf("aaa/billing: kafka emit: %w", err)
	}
	return nil
}

// Close 关闭底层连接。
func (e *KafkaEmitter) Close() error { return e.w.Close() }

// FanoutEmitter 双写:实时(Kafka) + 落库(PG);Kafka 失败仅记入错误不影响落库(最终一致由 PG 兜底)。
type FanoutEmitter struct {
	Realtime *KafkaEmitter
	Store    *PGEmitter
}

// Emit 先落库后实时;实时失败不阻塞计费链路。
func (e *FanoutEmitter) Emit(ctx context.Context, cdr CDR) error {
	if err := e.Store.Emit(ctx, cdr); err != nil {
		return err
	}
	if e.Realtime != nil {
		_ = e.Realtime.Emit(ctx, cdr) // 尽力而为,失败由对账任务补偿
	}
	return nil
}
