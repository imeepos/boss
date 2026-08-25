package crash

import (
	"context"
	"strings"
	"testing"
)

func TestMemStoreInsertAndList(t *testing.T) {
	s := NewMemStore()
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if err := s.Insert(ctx, Log{SubjectID: int64(i), App: "test", Log: "boom"}); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}
	logs, err := s.ListRecent(ctx, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(logs) != 5 {
		t.Fatalf("want 5, got %d", len(logs))
	}
	for i, l := range logs {
		if l.SubjectType != "worker" {
			t.Fatalf("log[%d] subjectType=%q", i, l.SubjectType)
		}
		if l.ID == 0 {
			t.Fatalf("log[%d] id not assigned", i)
		}
	}
}

func TestMemStoreLimit(t *testing.T) {
	s := NewMemStore()
	ctx := context.Background()
	for i := 0; i < 250; i++ {
		_ = s.Insert(ctx, Log{App: "test", Log: "x"})
	}
	got, _ := s.ListRecent(ctx, 10)
	if len(got) != 10 {
		t.Fatalf("limit not applied, got %d", len(got))
	}
}

func TestClipTails(t *testing.T) {
	long := strings.Repeat("z", MaxLogBytes+100)
	got := Clip(long)
	if len(got) > MaxLogBytes+32 {
		t.Fatalf("clip len=%d", len(got))
	}
	if !strings.HasSuffix(got, strings.Repeat("z", MaxLogBytes)) {
		t.Fatalf("clip should keep tail")
	}
}
