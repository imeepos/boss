package report

// 推送通道:快照生成留痕(PG)后投递 Kafka,消费方(邮件网关/IM 机器人/下游看板)订阅 boss-report-snapshots。
// 语义与订单事件一致:留痕为权威,推送尽力而为。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// ErrNoNotifier 未配置推送通道时 Push 返回。
var ErrNoNotifier = errors.New("report: no notifier configured")

// Envelope 推送信封(契约:消费方按 type/period/window 解析)。
type Envelope struct {
	Type        string    `json:"type"` // 固定 report.snapshot
	SnapshotID  int64     `json:"snapshotId"`
	Period      string    `json:"period"`
	WindowStart time.Time `json:"windowStart"`
	WindowEnd   time.Time `json:"windowEnd"`
	Payload     []byte    `json:"payload"` // Payload 原文 JSON(指标/ROI/热力/维护/结论)
}

func newEnvelope(s *Snapshot) Envelope {
	return Envelope{Type: "report.snapshot", SnapshotID: s.ID, Period: s.Period,
		WindowStart: s.WindowStart, WindowEnd: s.WindowEnd, Payload: s.Payload}
}

// Notifier 推送口(key=period 保序)。
type Notifier interface {
	Notify(ctx context.Context, key string, snap *Snapshot) error
}

// NoopNotifier 无操作实现(未部署 Kafka 环境/单测默认)。
type NoopNotifier struct{}

// Notify 丢弃。
func (NoopNotifier) Notify(context.Context, string, *Snapshot) error { return nil }

// Push 推送一份快照(留痕之后的旁路动作,失败不回滚快照)。
func (r *ReportService) Push(ctx context.Context, snap *Snapshot) error {
	if r.Nt == nil {
		return ErrNoNotifier
	}
	return r.Nt.Notify(ctx, snap.Period, snap)
}

// KafkaNotifier Kafka 实现。
type KafkaNotifier struct {
	w     *kafka.Writer
	topic string
}

// NewKafkaNotifier 构造(brokers 如 192.168.0.102:29092,topic 如 boss-report-snapshots)。
func NewKafkaNotifier(brokers []string, topic string) *KafkaNotifier {
	return &KafkaNotifier{w: &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.Hash{}, // key=period 分区保序
	}, topic: topic}
}

// Notify 信封序列化投递。
func (n *KafkaNotifier) Notify(ctx context.Context, key string, snap *Snapshot) error {
	b, err := json.Marshal(newEnvelope(snap))
	if err != nil {
		return fmt.Errorf("report: marshal envelope: %w", err)
	}
	if err := n.w.WriteMessages(ctx, kafka.Message{Key: []byte(key), Value: b}); err != nil {
		return fmt.Errorf("report: kafka notify: %w", err)
	}
	return nil
}

// Close 关闭底层连接。
func (n *KafkaNotifier) Close() error { return n.w.Close() }
