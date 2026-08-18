package report

// report 推送 Kafka 实链路集成测试:信封发布 → 消费 round-trip。
// 运行: BOSS_KAFKA_TEST_BROKERS="192.168.0.102:29092" go test ./internal/domain/report/ -run TestKafkaNotifyConsume -v -count=1

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

func TestKafkaNotifyConsume(t *testing.T) {
	brokers := os.Getenv("BOSS_KAFKA_TEST_BROKERS")
	if brokers == "" {
		t.Skip("BOSS_KAFKA_TEST_BROKERS 未设置,跳过 Kafka 实链路测试")
	}
	b := []string{brokers}
	topic := "boss-report-snapshots"

	// 主题不存在则建(分区1/保留24h,与既有事件主题同规格)。
	conn, err := kafka.Dial("tcp", brokers)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	_ = conn.CreateTopics(kafka.TopicConfig{Topic: topic, NumPartitions: 1, ReplicationFactor: 1})

	nt := NewKafkaNotifier(b, topic)
	defer nt.Close()

	at := time.Now().Truncate(time.Millisecond)
	want := &Snapshot{ID: 99, Period: "daily", WindowStart: at.Add(-24 * time.Hour),
		WindowEnd: at, Payload: []byte(`{"conclusions":["ok"]}`), CreatedAt: at}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := nt.Notify(ctx, "daily", want); err != nil {
		t.Fatalf("Notify: %v", err)
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: b, Topic: topic, GroupID: "e2e-report-push-test",
		MinBytes: 1, MaxBytes: 1e6,
	})
	defer r.Close()
	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			t.Fatalf("ReadMessage: %v", err)
		}
		if string(m.Key) != "daily" {
			continue
		}
		var got Envelope
		if err := json.Unmarshal(m.Value, &got); err != nil {
			t.Fatalf("Unmarshal: %v (%s)", err, m.Value)
		}
		if got.Type != "report.snapshot" || got.Period != "daily" || got.SnapshotID != 99 ||
			string(got.Payload) != string(want.Payload) {
			t.Fatalf("got=%+v", got)
		}
		return
	}
}
