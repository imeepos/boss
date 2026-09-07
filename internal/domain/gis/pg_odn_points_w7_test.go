package gis

// W7 勘测打点/施工进度点位单测:实体分发与 level/状态语义。

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestODNPoints_SurveyProgress(t *testing.T) {
	ctx := context.Background()

	t.Run("勘测打点 level12 建议作状态", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`FROM survey_task_reports r`).
			WillReturnRows(mock.NewRows([]string{"id", "level", "name", "lng", "lat", "status", "count", "parent_id"}).
				AddRow(int64(1), int16(12), "SV-20260907-00001", 120.98, 14.55, "CAN_INSTALL", int64(0), int64(9)))
		s := NewPGStore(mock)
		got, err := s.ODNPoints(ctx, "survey", "")
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Level != 12 || got[0].Status != "CAN_INSTALL" || got[0].ParentID != 9 {
			t.Fatalf("got=%+v", got)
		}
	})

	t.Run("施工进度点 level13 项目状态作状态", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`FROM construction_progress p`).
			WillReturnRows(mock.NewRows([]string{"id", "level", "name", "lng", "lat", "status", "count", "parent_id"}).
				AddRow(int64(2), int16(13), "P01001", 120.99, 14.56, "BUILDING", int64(0), int64(4)))
		s := NewPGStore(mock)
		got, err := s.ODNPoints(ctx, "progress", "")
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Level != 13 || got[0].Status != "BUILDING" || got[0].ParentID != 4 {
			t.Fatalf("got=%+v", got)
		}
	})
}
