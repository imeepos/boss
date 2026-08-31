package worker

// 师傅登录密码 PG 测试:录入(带密码)/重置/校验 + 唯一约束冲突映射。

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"golang.org/x/crypto/bcrypt"
)

func TestPGStore_CreateWorkerWithPassword(t *testing.T) {
	t.Run("带密码落库", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(11)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`INSERT INTO workers`).
			WithArgs("WK-2001", "赵师傅", int64(1), int64(11), "13900002233", int16(1), ts, (*time.Time)(nil), pgxmock.AnyArg()).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(9)))

		s := NewPGStore(mock)
		id, err := s.CreateWorkerWithPassword(context.Background(), Worker{
			StaffNo: "WK-2001", Name: "赵师傅", GroupID: 1, RegionID: 11, Phone: "13900002233", Status: 1, JoinedAt: ts,
		}, "secret-6")
		if err != nil {
			t.Fatalf("CreateWorkerWithPassword: %v", err)
		}
		if id != 9 {
			t.Fatalf("id=%d, want 9", id)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("密码过短拒绝", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		s := NewPGStore(mock)
		if _, err := s.CreateWorkerWithPassword(context.Background(),
			Worker{StaffNo: "WK-2002", Name: "钱师傅"}, "12345"); !errors.Is(err, ErrInvalidPassword) {
			t.Fatalf("err=%v, want ErrInvalidPassword", err)
		}
	})
	t.Run("工号冲突映射 ErrDuplicate", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(11)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`INSERT INTO workers`).
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
				pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: "workers_staff_no_key"})

		s := NewPGStore(mock)
		if _, err := s.CreateWorkerWithPassword(context.Background(), Worker{
			StaffNo: "WK-2001", Name: "赵师傅", GroupID: 1, RegionID: 11, Phone: "13900002233", Status: 1, JoinedAt: ts,
		}, "secret-6"); !errors.Is(err, ErrDuplicate) {
			t.Fatalf("err=%v, want ErrDuplicate", err)
		}
	})
}

func TestPGStore_SetPassword(t *testing.T) {
	t.Run("重置成功", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectExec(`UPDATE workers SET password_hash`).
			WithArgs(int64(9), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		s := NewPGStore(mock)
		if err := s.SetPassword(context.Background(), 9, "new-pass-9"); err != nil {
			t.Fatalf("SetPassword: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("未命中返回 ErrNotFound", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectExec(`UPDATE workers SET password_hash`).
			WithArgs(int64(99), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		s := NewPGStore(mock)
		if err := s.SetPassword(context.Background(), 99, "new-pass-9"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
	})
	t.Run("密码过短拒绝", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		s := NewPGStore(mock)
		if err := s.SetPassword(context.Background(), 9, "12345"); !errors.Is(err, ErrInvalidPassword) {
			t.Fatalf("err=%v, want ErrInvalidPassword", err)
		}
	})
}

func TestPGStore_VerifyPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("right-pass-6"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		hash string
		pwd  string
		want bool
	}{
		{"匹配", string(hash), "right-pass-6", true},
		{"不匹配", string(hash), "wrong-pass-6", false},
		{"未设置密码", "", "right-pass-6", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatal(err)
			}
			defer mock.Close()
			mock.ExpectQuery(`SELECT COALESCE\(password_hash`).
				WithArgs(int64(9)).
				WillReturnRows(mock.NewRows([]string{"password_hash"}).AddRow(tc.hash))

			s := NewPGStore(mock)
			got, err := s.VerifyPassword(context.Background(), 9, tc.pwd)
			if err != nil {
				t.Fatalf("VerifyPassword: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got=%v, want %v", got, tc.want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet: %v", err)
			}
		})
	}
}
