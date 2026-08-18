package gis

// GIS 域单测:八级下钻层级分发与非法层级,详情合并(指标/占用/关联用户)。

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

var ts = time.Date(2025, 8, 17, 10, 0, 0, 0, time.UTC)

func TestDrill_LevelDispatch(t *testing.T) {
	ctx := context.Background()

	t.Run("非法层级拒", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		s := NewPGStore(mock)
		if _, err := s.Drill(ctx, 0, 0); !errors.Is(err, errLevelInvalid) {
			t.Fatalf("err=%v, want errLevelInvalid", err)
		}
		if _, err := s.Drill(ctx, 9, 0); !errors.Is(err, errLevelInvalid) {
			t.Fatalf("err=%v, want errLevelInvalid", err)
		}
	})

	t.Run("地址层级带子级计数", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT a.id, a.name`).
			WithArgs(int16(1), int64(0)).
			WillReturnRows(mock.NewRows([]string{"id", "name", "cnt"}).
				AddRow(int64(1), "马尼拉市", int64(3)))
		s := NewPGStore(mock)
		got, err := s.Drill(ctx, 1, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Level != 1 || got[0].Count != 3 {
			t.Fatalf("got=%+v", got)
		}
	})

	t.Run("层6 OLT 弱电井", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`FROM resources r`).
			WithArgs(int64(5)).
			WillReturnRows(mock.NewRows([]string{"id", "name", "cnt"}).
				AddRow(int64(10), "OLT-01", int64(2)))
		s := NewPGStore(mock)
		got, _ := s.Drill(ctx, 6, 5)
		if len(got) != 1 || got[0].Level != 6 {
			t.Fatalf("got=%+v", got)
		}
	})
}

func TestResourceDetail(t *testing.T) {
	ctx := context.Background()

	t.Run("合并指标与占用与用户", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT id, code, name, type, status, address_id`).
			WithArgs(int64(10)).
			WillReturnRows(mock.NewRows([]string{"id", "code", "name", "type", "status", "address_id"}).
				AddRow(int64(10), "OLT-01", "OLT一", "OLT", "ONLINE", int64(5)))
		mock.ExpectQuery(`SELECT optical_power, packet_loss, collected_at`).
			WithArgs(int64(10)).
			WillReturnRows(mock.NewRows([]string{"optical_power", "packet_loss", "collected_at"}).
				AddRow(-24.5, 12.0, ts))
		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(10)).
			WillReturnRows(mock.NewRows([]string{"total", "used", "cust"}).
				AddRow(int64(16), int64(4), "张三"))

		s := NewPGStore(mock)
		d, err := s.ResourceDetail(ctx, 10)
		if err != nil {
			t.Fatal(err)
		}
		if d.Status != "ONLINE" || d.PortsTotal != 16 || d.PortsUsed != 4 || d.CustomerName != "张三" {
			t.Fatalf("d=%+v", d)
		}
		if d.OpticalPower == nil || *d.OpticalPower != -24.5 {
			t.Fatalf("optical=%v", d.OpticalPower)
		}
	})

	t.Run("不存在", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT id, code, name, type, status, address_id`).
			WithArgs(int64(99)).
			WillReturnError(pgx.ErrNoRows)
		s := NewPGStore(mock)
		if _, err := s.ResourceDetail(ctx, 99); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
	})
}
