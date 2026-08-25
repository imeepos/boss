package gis

// ODN 点位(ODNPoints)单测:实体分发、bbox 解析、无坐标过滤、非法实体拒。

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestODNPoints_EntityDispatch(t *testing.T) {
	ctx := context.Background()

	t.Run("非法实体拒", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		s := NewPGStore(mock)
		if _, err := s.ODNPoints(ctx, "pole", ""); !errors.Is(err, ErrODNEntityInvalid) {
			t.Fatalf("err=%v, want ErrODNEntityInvalid", err)
		}
	})

	t.Run("设施点位过滤无坐标行", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`FROM odn_facility f`).
			WillReturnRows(mock.NewRows([]string{"id", "level", "name", "lng", "lat", "status", "count", "parent_id"}).
				AddRow(int64(1), int16(9), "P01001", 120.98, 14.55, "IN_USE", int64(0), int64(0)))
		s := NewPGStore(mock)
		got, err := s.ODNPoints(ctx, "facility", "")
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Level != 9 || got[0].Name != "P01001" {
			t.Fatalf("got=%+v", got)
		}
	})

	t.Run("局点 NodeCode 合成", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`FROM odn_site s`).
			WillReturnRows(mock.NewRows([]string{"id", "level", "name", "lng", "lat", "status", "count", "parent_id"}).
				AddRow(int64(1), int16(10), "MNL001", 120.98, 14.55, "ACTIVE", int64(0), int64(0)))
		s := NewPGStore(mock)
		got, err := s.ODNPoints(ctx, "site", "")
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Level != 10 || got[0].Name != "MNL001" {
			t.Fatalf("got=%+v", got)
		}
	})

	t.Run("设备点位带真实 id", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`FROM odn_device d`).
			WillReturnRows(mock.NewRows([]string{"id", "level", "name", "lng", "lat", "status", "count", "parent_id"}).
				AddRow(int64(42), int16(11), "OLT001", 120.98, 14.55, "IN_USE", int64(0), int64(1)))
		s := NewPGStore(mock)
		got, err := s.ODNPoints(ctx, "device", "")
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].ID != 42 || got[0].Level != 11 || got[0].ParentID != 1 {
			t.Fatalf("got=%+v", got)
		}
	})
}

func TestODNPoints_Bbox(t *testing.T) {
	t.Run("bbox 空串不限", func(t *testing.T) {
		sql, args, err := odnPointsQuery("facility", "")
		if err != nil {
			t.Fatal(err)
		}
		if len(args) != 0 || sql == "" {
			t.Fatalf("sql=%q args=%v", sql, args)
		}
	})

	t.Run("bbox 四段数值过滤", func(t *testing.T) {
		sql, args, err := odnPointsQuery("facility", "120,14,121,15")
		if err != nil {
			t.Fatal(err)
		}
		if len(args) != 4 || args[0] != 120.0 || args[1] != 14.0 || args[2] != 121.0 || args[3] != 15.0 {
			t.Fatalf("args=%v", args)
		}
		if sql == "" {
			t.Fatal("bbox SQL 不应为空")
		}
	})

	t.Run("bbox 段数非法", func(t *testing.T) {
		if _, _, err := odnPointsQuery("facility", "120,14,121"); err == nil {
			t.Fatal("want bbox error")
		}
	})
}
