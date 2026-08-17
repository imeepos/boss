package order

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// TestPGStore_ListChannels 契约:列出全部渠道。
func TestPGStore_ListChannels(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, code, name, status FROM channels`).
		WillReturnRows(mock.NewRows([]string{"id", "code", "name", "status"}).
			AddRow(int64(1), "HALL", "营业厅", "ACTIVE").
			AddRow(int64(2), "ONLINE", "线上", "ACTIVE"))

	s := NewPGStore(mock, stubExists{})
	got, err := s.ListChannels(context.Background())
	if err != nil {
		t.Fatalf("ListChannels: %v", err)
	}
	if len(got) != 2 || got[0].Code != "HALL" || got[1].Name != "线上" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_GetChannel 契约:按 id 查渠道;未命中返回 ErrOrderNotFound。
func TestPGStore_GetChannel(t *testing.T) {
	t.Run("命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT id, code, name, status FROM channels`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows([]string{"id", "code", "name", "status"}).
				AddRow(int64(1), "HALL", "营业厅", "ACTIVE"))

		s := NewPGStore(mock, stubExists{})
		c, err := s.GetChannel(context.Background(), 1)
		if err != nil {
			t.Fatalf("GetChannel: %v", err)
		}
		if c.Code != "HALL" || c.Name != "营业厅" {
			t.Fatalf("c=%+v", c)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
	t.Run("未命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT id, code, name, status FROM channels`).
			WithArgs(int64(99)).
			WillReturnError(pgx.ErrNoRows)

		s := NewPGStore(mock, stubExists{})
		_, err = s.GetChannel(context.Background(), 99)
		if !errors.Is(err, ErrOrderNotFound) {
			t.Fatalf("err=%v, want ErrOrderNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

// TestPGStore_CreateChannel 契约:新增渠道并返回自增 id。
func TestPGStore_CreateChannel(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO channels`).
		WithArgs("HALL", "营业厅", "ACTIVE").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))

	s := NewPGStore(mock, stubExists{})
	id, err := s.CreateChannel(context.Background(), Channel{Code: "HALL", Name: "营业厅", Status: "ACTIVE"})
	if err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}
	if id != 1 {
		t.Fatalf("id=%d, want 1", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
