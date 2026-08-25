package aaa

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// W6(TDD 先行):PG 授权器(LOID→带宽/状态) + 停复机即时生效。

func TestPGAuthorizer_Decide(t *testing.T) {
	cols := []string{"status", "bandwidth", "qos"}

	t.Run("ACTIVE → 放行并下发带宽", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`FROM lo_accounts`).
			WithArgs("LOID-1").
			WillReturnRows(mock.NewRows(cols).AddRow("ACTIVE", "300M/150M", "QoS-STD"))

		s := NewPGAuthorizer(mock)
		d, err := s.Decide(context.Background(), "LOID-1")
		if err != nil {
			t.Fatalf("Decide: %v", err)
		}
		if !d.Authorize || d.Bandwidth != "300M/150M" || d.SessionTTL != defaultSessionTTL {
			t.Fatalf("d=%+v", d)
		}
	})

	t.Run("SUSPENDED → 拒(停机在线无网)", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`FROM lo_accounts`).
			WithArgs("LOID-1").
			WillReturnRows(mock.NewRows(cols).AddRow("SUSPENDED", "300M/150M", ""))

		s := NewPGAuthorizer(mock)
		d, err := s.Decide(context.Background(), "LOID-1")
		if !errors.Is(err, ErrSuspended) || d.Authorize {
			t.Fatalf("d=%+v err=%v", d, err)
		}
	})

	t.Run("CLOSED → ErrClosed(注销)", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`FROM lo_accounts`).
			WithArgs("LOID-C").
			WillReturnRows(mock.NewRows(cols).AddRow("CLOSED", "", ""))

		s := NewPGAuthorizer(mock)
		d, err := s.Decide(context.Background(), "LOID-C")
		if !errors.Is(err, ErrClosed) || d.Authorize {
			t.Fatalf("d=%+v err=%v, want ErrClosed", d, err)
		}
	})

	t.Run("不存在 → ErrNotFound", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`FROM lo_accounts`).
			WithArgs("LOID-X").
			WillReturnRows(mock.NewRows(cols))

		s := NewPGAuthorizer(mock)
		_, err := s.Decide(context.Background(), "LOID-X")
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestPGStore_SuspendResume(t *testing.T) {
	t.Run("Suspend ACTIVE→SUSPENDED", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE lo_accounts SET status`).
			WithArgs(int64(5), "SUSPENDED", "ACTIVE").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		s := NewPGStore(mock)
		if err := s.SuspendLoAccount(context.Background(), 5); err != nil {
			t.Fatalf("Suspend: %v", err)
		}
	})

	t.Run("Resume SUSPENDED→ACTIVE", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE lo_accounts SET status`).
			WithArgs(int64(5), "ACTIVE", "SUSPENDED").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		s := NewPGStore(mock)
		if err := s.ResumeLoAccount(context.Background(), 5); err != nil {
			t.Fatalf("Resume: %v", err)
		}
	})

	t.Run("非法迁移 → ErrIllegalTransition", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE lo_accounts SET status`).
			WithArgs(int64(6), "SUSPENDED", "ACTIVE").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		s := NewPGStore(mock)
		if err := s.SuspendLoAccount(context.Background(), 6); !errors.Is(err, ErrIllegalTransition) {
			t.Fatalf("err=%v", err)
		}
	})
}
