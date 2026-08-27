// 事件目录与事件清单规整的回归:目录必须包含发射方实际广播的事件类型。
package openplat

import (
	"slices"
	"testing"

	"github.com/ymm-001/boss/internal/domain/order"
)

func TestEventCatalogCoversEmittedEvents(t *testing.T) {
	catalog := EventCatalog()
	types := make([]string, 0, len(catalog))
	for _, e := range catalog {
		if e.Type == "" || e.Description == "" {
			t.Fatalf("catalog entry missing fields: %+v", e)
		}
		types = append(types, e.Type)
	}
	if !slices.Contains(types, order.StageEventType) {
		t.Fatalf("catalog %v does not cover emitted event %s", types, order.StageEventType)
	}
	if len(catalog) != len(eventCatalog) {
		t.Fatalf("EventCatalog must return a copy, got len=%d want=%d", len(catalog), len(eventCatalog))
	}
}

func TestNormalizeEventTypes(t *testing.T) {
	got := NormalizeEventTypes([]string{" a.b ", "", "c.d", "a.b", "  "})
	want := []string{"a.b", "c.d"}
	if !slices.Equal(got, want) {
		t.Fatalf("NormalizeEventTypes = %v, want %v", got, want)
	}
	if got := NormalizeEventTypes(nil); len(got) != 0 {
		t.Fatalf("NormalizeEventTypes(nil) = %v, want empty", got)
	}
}
