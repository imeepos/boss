package main

// Syncer 单测:仅环节12(updateMap)触发地图同步;其它环节不回查。

import (
	"context"
	"testing"

	"github.com/ymm-001/boss/internal/domain/gis"
)

type stubGIS struct {
	gis.GISService
	countCalls int
}

func (s *stubGIS) LevelCounts(context.Context) ([]gis.Node, error) {
	s.countCalls++
	return []gis.Node{{Level: 1, Count: 1}}, nil
}

func TestSyncer_Handle(t *testing.T) {
	t.Run("环节12触发同步", func(t *testing.T) {
		g := &stubGIS{}
		s := &Syncer{Gis: g}
		s.Handle(context.Background(), orderEvent{Type: "order.stage.changed", OrderID: 1, Stage: 12, Status: "DONE"})
		if g.countCalls != 1 {
			t.Fatalf("countCalls=%d, want 1", g.countCalls)
		}
	})

	t.Run("其它环节不同步", func(t *testing.T) {
		g := &stubGIS{}
		s := &Syncer{Gis: g}
		s.Handle(context.Background(), orderEvent{OrderID: 1, Stage: 9, Status: "INSTALLING"})
		if g.countCalls != 0 {
			t.Fatalf("countCalls=%d, want 0", g.countCalls)
		}
	})
}
