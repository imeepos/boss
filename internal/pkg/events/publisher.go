package events

// Package events 状态变更事件发布(W8:Kafka 状态变更链路)。
// 语义:订单环节/状态每次迁移发布一条 JSON 事件;实时链路尽力而为,审计留痕仍走 PG。

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// Event 一条状态变更事件(跨服务消费:APISIX 推送/Flink 大盘/工单联动)。
type Event struct {
	Type      string    `json:"type"` // order.stage.changed / order.status.changed
	OrderID   int64     `json:"orderId"`
	Stage     int8      `json:"stage"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// Publisher 事件发布口。
type Publisher interface {
	Publish(ctx context.Context, key string, e Event) error
}

// Noop 无操作实现(未部署 Kafka 环境/测试默认)。
type Noop struct{}

// Publish 丢弃事件。
func (Noop) Publish(context.Context, string, Event) error { return nil }

// KafkaPublisher Kafka 实现(topic 固定由构造传入,如 boss-order-events)。
type KafkaPublisher struct {
	w     *kafka.Writer
	topic string
}

// NewKafkaPublisher 构造(brokers 如 192.168.0.102:29092)。
func NewKafkaPublisher(brokers []string, topic string) *KafkaPublisher {
	return &KafkaPublisher{w: &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}, topic: topic}
}

// Publish JSON 序列化投递,key=订单号。
func (p *KafkaPublisher) Publish(ctx context.Context, key string, e Event) error {
	b, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("events: marshal: %w", err)
	}
	if err := p.w.WriteMessages(ctx, kafka.Message{Key: []byte(key), Value: b}); err != nil {
		return fmt.Errorf("events: kafka publish: %w", err)
	}
	return nil
}

// Close 关闭底层连接。
func (p *KafkaPublisher) Close() error { return p.w.Close() }
