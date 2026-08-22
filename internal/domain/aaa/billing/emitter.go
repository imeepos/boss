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
	_, err := e.EmitID(ctx, cdr)
	return err
}

// EmitID 话单落库并返回自增 id(补偿侧按 id 回写 Kafka 投递状态)。
func (e *PGEmitter) EmitID(ctx context.Context, cdr CDR) (int64, error) {
	var id int64
	id, err := e.W.AppendCdr(ctx, aaa.CdrRecord{
		Loid: cdr.LOID, Username: cdr.Username, AcctStatus: int16(cdr.AcctStatus),
		SessionID: cdr.SessionID, SessionTime: int32(cdr.SessionTime),
		InputOctets: int64(cdr.InputOctets), OutputOctets: int64(cdr.OutputOctets),
		NasIP: cdr.NASIP, BillingStatus: "UNBILLED", StartedAt: time.Now(),
	})
	if err != nil {
		return 0, fmt.Errorf("aaa/billing: pg emit: %w", err)
	}
	return id, nil
}

// kafkaWriter kafka.Writer 最小写口(测试可替身)。
type kafkaWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

// KafkaEmitter 话单实时投递(Kafka topic → Flink → Doris)。
type KafkaEmitter struct {
	w kafkaWriter
}

// NewKafkaEmitter 构造 Kafka 话单投递器(brokers 如 192.168.0.102:29092,topic 如 boss-cdr)。
func NewKafkaEmitter(brokers []string, topic string) *KafkaEmitter {
	return &KafkaEmitter{w: &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{}, // 按 key(LOID/订单号)分区,保序
		RequiredAcks: kafka.RequireOne,
		BatchTimeout: 10 * time.Millisecond,
	}}
}

// marshalCDR 测试可替换的序列化口(CDR 全基本字段,正常路径不可能失败)。
var marshalCDR = json.Marshal

// Emit JSON 序列化投递;value 即 CDR 全量。
func (e *KafkaEmitter) Emit(ctx context.Context, cdr CDR) error {
	b, err := marshalCDR(cdr)
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

// CdrKafkaMarker Kafka 投递状态回写口(实现挂 aaa.PGStore,迁移 000109)。
type CdrKafkaMarker interface {
	MarkCdrsKafkaStatus(ctx context.Context, ids []int64, status string) error
}

// FanoutEmitter 双写:实时(Kafka) + 落库(PG);Kafka 失败仅记入错误不影响落库
// (投递状态留痕,由补偿循环从 PG 权威侧补投,最终一致)。
type FanoutEmitter struct {
	Realtime *KafkaEmitter
	Store    *PGEmitter
	Marker   CdrKafkaMarker // 可空:无 000109 列的降级部署
}

// Emit 先落库后实时;实时失败不阻塞计费链路,状态回写尽力而为。
func (e *FanoutEmitter) Emit(ctx context.Context, cdr CDR) error {
	id, err := e.Store.EmitID(ctx, cdr)
	if err != nil {
		return err
	}
	if e.Realtime == nil {
		return nil
	}
	if err := e.Realtime.Emit(ctx, cdr); err != nil {
		e.markKafka(ctx, id, aaa.CdrKafkaFailed)
		return nil
	}
	e.markKafka(ctx, id, aaa.CdrKafkaSent)
	return nil
}

// markKafka 投递状态回写,尽力而为。
func (e *FanoutEmitter) markKafka(ctx context.Context, id int64, status string) {
	if e.Marker == nil || id == 0 {
		return
	}
	_ = e.Marker.MarkCdrsKafkaStatus(ctx, []int64{id}, status)
}
