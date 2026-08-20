package events

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

func TestNoopPublish(t *testing.T) {
	if err := (Noop{}).Publish(context.Background(), "k", Event{}); err != nil {
		t.Fatalf("Noop.Publish: %v", err)
	}
}

func TestNewKafkaPublisher(t *testing.T) {
	p := NewKafkaPublisher([]string{"127.0.0.1:9092"}, "boss-order-events")
	if p.topic != "boss-order-events" {
		t.Fatalf("topic=%q", p.topic)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestKafkaPublisherPublish(t *testing.T) {
	want := Event{
		Type: "order.status.changed", OrderID: 7, Stage: 3, Status: "DOING",
		Timestamp: time.Unix(1700000000, 0).UTC(),
	}
	var gotKey string
	var gotVal []byte
	p := NewKafkaPublisher([]string{"127.0.0.1:9092"}, "t")
	p.write = func(ctx context.Context, msgs ...kafka.Message) error {
		gotKey, gotVal = string(msgs[0].Key), msgs[0].Value
		return nil
	}
	if err := p.Publish(context.Background(), "ORD-7", want); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if gotKey != "ORD-7" {
		t.Fatalf("key=%q", gotKey)
	}
	var got Event
	if err := json.Unmarshal(gotVal, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got != want {
		t.Fatalf("got=%+v want=%+v", got, want)
	}
}

func TestKafkaPublisherPublishWriteError(t *testing.T) {
	p := NewKafkaPublisher([]string{"127.0.0.1:9092"}, "t")
	sentinel := errors.New("boom")
	p.write = func(ctx context.Context, msgs ...kafka.Message) error { return sentinel }
	err := p.Publish(context.Background(), "k", Event{})
	if err == nil || !errors.Is(err, sentinel) {
		t.Fatalf("err=%v", err)
	}
}
