package resource

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// TestPGStore_Check 契约:按地址核查空闲端口;有则 available=true 并返回端口码。
func TestPGStore_Check(t *testing.T) {
	t.Run("有空闲", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT p.port_code`).
			WithArgs(int64(100)).
			WillReturnRows(mock.NewRows([]string{"port_code"}).
				AddRow("P-SPL01-01").AddRow("P-SPL01-02"))

		s := NewPGStore(mock)
		avail, options, err := s.Check(context.Background(), 100)
		if err != nil {
			t.Fatalf("Check: %v", err)
		}
		if !avail || len(options) != 2 || options[0] != "P-SPL01-01" {
			t.Fatalf("avail=%v options=%v", avail, options)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
	t.Run("无空闲", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT p.port_code`).
			WithArgs(int64(200)).
			WillReturnRows(mock.NewRows([]string{"port_code"}))

		s := NewPGStore(mock)
		avail, options, err := s.Check(context.Background(), 200)
		if err != nil {
			t.Fatalf("Check: %v", err)
		}
		if avail || len(options) != 0 {
			t.Fatalf("avail=%v options=%v", avail, options)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}
