package billing

// Kafka 话单实链路集成测试:KafkaEmitter 投递 → 消费还原 CDR。
// 运行: BOSS_KAFKA_TEST_BROKERS="192.168.0.102:29092" go test ./internal/domain/aaa/billing/ -run TestKafkaEmitterRoundTrip -v -count=1

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

func TestKafkaEmitterRoundTrip(t *testing.T) {
	brokers := os.Getenv("BOSS_KAFKA_TEST_BROKERS")
	if brokers == "" {
		t.Skip("BOSS_KAFKA_TEST_BROKERS 未设置,跳过 Kafka 实链路测试")
	}
	b := []string{brokers}
	em := NewKafkaEmitter(b, "boss-cdr")
	defer em.Close()

	want := CDR{
		LOID: "LOID-KAFKA-1", Username: "u", AcctStatus: 2, SessionID: "S-1",
		SessionTime: 600, InputOctets: 1024, OutputOctets: 2048, NASIP: "10.0.0.1",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := em.Emit(ctx, want); err != nil {
		t.Fatalf("Emit: %v", err)
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: b, Topic: "boss-cdr", GroupID: "e2e-cdr-test",
		MinBytes: 1, MaxBytes: 1e6,
	})
	defer r.Close()
	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			t.Fatalf("ReadMessage: %v", err)
		}
		if string(m.Key) != "LOID-KAFKA-1" {
			continue
		}
		var got CDR
		if err := json.Unmarshal(m.Value, &got); err != nil {
			t.Fatalf("Unmarshal: %v (%s)", err, m.Value)
		}
		if got != want {
			t.Fatalf("got=%+v want=%+v", got, want)
		}
		return
	}
}
