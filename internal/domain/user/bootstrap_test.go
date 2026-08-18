package user

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// TestPGStore_EnsureSuperAdmin 契约:插入命中返回 created=true;冲突时跳过不覆盖;弱口令拒绝。
func TestPGStore_EnsureSuperAdmin(t *testing.T) {
	t.Run("新建", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`INSERT INTO accounts`).
			WithArgs("root", pgxmock.AnyArg(), "超级管理员").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		created, err := NewPGStore(mock).EnsureSuperAdmin(context.Background(), "root", "s3cret!", "超级管理员")
		if err != nil || !created {
			t.Fatalf("created=%v err=%v", created, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("已存在跳过", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`INSERT INTO accounts`).
			WithArgs("root", pgxmock.AnyArg(), "超级管理员").
			WillReturnResult(pgxmock.NewResult("INSERT", 0))
		created, err := NewPGStore(mock).EnsureSuperAdmin(context.Background(), "root", "s3cret!", "超级管理员")
		if err != nil || created {
			t.Fatalf("created=%v err=%v", created, err)
		}
	})
	t.Run("弱口令", func(t *testing.T) {
		_, err := NewPGStore(nil).EnsureSuperAdmin(context.Background(), "root", "123", "超级管理员")
		if !errors.Is(err, ErrBootstrapPassword) {
			t.Fatalf("want ErrBootstrapPassword, got %v", err)
		}
	})
}
