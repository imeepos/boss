package events

// Kafka 实链路集成测试:发布 → 消费 round-trip。
// 运行: BOSS_KAFKA_TEST_BROKERS="192.168.0.102:29092" go test ./internal/pkg/events/ -run TestKafkaPublishConsume -v -count=1

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

func TestKafkaPublishConsume(t *testing.T) {
	brokers := os.Getenv("BOSS_KAFKA_TEST_BROKERS")
	if brokers == "" {
		t.Skip("BOSS_KAFKA_TEST_BROKERS 未设置,跳过 Kafka 实链路测试")
	}
	b := []string{brokers}
	topic := "boss-order-events"

	pub := NewKafkaPublisher(b, topic)
	defer pub.Close()

	want := Event{
		Type: "order.stage.changed", OrderID: 4321, Stage: 9, Status: "DOING",
		Timestamp: time.Now().Truncate(time.Millisecond), // Kafka 往返不承诺纳秒精度
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := pub.Publish(ctx, "ORD-KAFKA-1", want); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: b, Topic: topic, GroupID: "e2e-kafka-test",
		MinBytes: 1, MaxBytes: 1e6,
	})
	defer r.Close()
	// 轮询直到读到本测试 key(消费组从最新起,容忍历史消息)。
	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			t.Fatalf("ReadMessage: %v", err)
		}
		if string(m.Key) != "ORD-KAFKA-1" {
			continue
		}
		var got Event
		if err := json.Unmarshal(m.Value, &got); err != nil {
			t.Fatalf("Unmarshal: %v (%s)", err, m.Value)
		}
		if got.Type != want.Type || got.OrderID != want.OrderID ||
			got.Stage != want.Stage || got.Status != want.Status {
			t.Fatalf("got=%+v want=%+v", got, want)
		}
		return
	}
}
