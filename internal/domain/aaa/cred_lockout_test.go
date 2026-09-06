package aaa

// A1 防爆破锁定用例:未设密开关/状态原因码/连续失败锁定与窗口到期自动解锁。

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func TestCredentialAuthorizer_NoCredential(t *testing.T) {
	t.Run("未设密 默认 Reject(记失败)", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		expectNoLockRow(mock, "LOID-N1")
		expectAccount(mock, "LOID-N1", "ACTIVE", "", "", "")
		expectFailCount(mock, "LOID-N1", 5)
		a := newCredAuth(t, mock, nil)
		_, err := a.Authenticate(context.Background(), "LOID-N1", Credentials{})
		if !errors.Is(err, ErrBadCredential) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("开关开启 未设密放行(迁移缓冲)", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		expectNoLockRow(mock, "LOID-N2")
		expectAccount(mock, "LOID-N2", "ACTIVE", "100M/50M", "", "")
		expectLockClear(mock, "LOID-N2")
		a := newCredAuth(t, mock, func(c *CredentialConfig) { c.AllowNoCred = true })
		d, err := a.Authenticate(context.Background(), "LOID-N2", Credentials{})
		if err != nil || !d.Authorize {
			t.Fatalf("d=%+v err=%v", d, err)
		}
	})

	t.Run("开关开启但已设密仍须校验", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		expectNoLockRow(mock, "LOID-N3")
		expectAccount(mock, "LOID-N3", "ACTIVE", "", "", mustStored(t, "pw-real"))
		expectFailCount(mock, "LOID-N3", 5)
		a := newCredAuth(t, mock, func(c *CredentialConfig) { c.AllowNoCred = true })
		_, err := a.Authenticate(context.Background(), "LOID-N3", Credentials{PAP: "wrong"})
		if !errors.Is(err, ErrBadCredential) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestCredentialAuthorizer_StatusMapping(t *testing.T) {
	cases := []struct {
		status string
		want   error
	}{
		{"SUSPENDED", ErrSuspended},
		{"CLOSED", ErrClosed},
	}
	for _, tc := range cases {
		t.Run(tc.status, func(t *testing.T) {
			mock, _ := pgxmock.NewPool()
			defer mock.Close()
			expectNoLockRow(mock, "LOID-S")
			expectAccount(mock, "LOID-S", tc.status, "", "", mustStored(t, "pw"))
			a := newCredAuth(t, mock, nil)
			_, err := a.Authenticate(context.Background(), "LOID-S", Credentials{PAP: "pw"})
			if !errors.Is(err, tc.want) {
				t.Fatalf("err=%v want %v", err, tc.want)
			}
		})
	}
	t.Run("NOT_FOUND", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		expectNoLockRow(mock, "LOID-X")
		mock.ExpectQuery(`FROM lo_accounts`).
			WithArgs("LOID-X").
			WillReturnRows(mock.NewRows([]string{"status", "bandwidth", "qos", "credential"}))
		a := newCredAuth(t, mock, nil)
		_, err := a.Authenticate(context.Background(), "LOID-X", Credentials{PAP: "pw"})
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v", err)
		}
	})
}
func TestCredentialAuthorizer_Lockout(t *testing.T) {
	t.Run("锁定窗口内 一律 Reject(正确密码也不放行)", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		future := fixedNow.Add(10 * time.Minute)
		mock.ExpectQuery(`FROM lo_auth_lockouts`).
			WithArgs("LOID-L1").
			WillReturnRows(mock.NewRows([]string{"fail_count", "locked_until"}).AddRow(5, future))
		a := newCredAuth(t, mock, nil)
		_, err := a.Authenticate(context.Background(), "LOID-L1", Credentials{PAP: "pw-correct"})
		if !errors.Is(err, ErrLocked) {
			t.Fatalf("err=%v want ErrLocked", err)
		}
	})

	t.Run("连续失败达阈值记锁定参数", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		expectNoLockRow(mock, "LOID-L2")
		expectAccount(mock, "LOID-L2", "ACTIVE", "", "", mustStored(t, "pw"))
		mock.ExpectExec(`INSERT INTO lo_auth_lockouts`).
			WithArgs("LOID-L2", 7, fixedNow.Add(30*time.Minute)).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		a := newCredAuth(t, mock, func(c *CredentialConfig) { c.LockThreshold = 7; c.LockWindow = 30 * time.Minute })
		_, err := a.Authenticate(context.Background(), "LOID-L2", Credentials{PAP: "bad"})
		if !errors.Is(err, ErrBadCredential) {
			t.Fatalf("err=%v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("锁定窗口到期 自动解锁并放行正确密码", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		expired := fixedNow.Add(-1 * time.Minute)
		mock.ExpectQuery(`FROM lo_auth_lockouts`).
			WithArgs("LOID-L3").
			WillReturnRows(mock.NewRows([]string{"fail_count", "locked_until"}).AddRow(5, expired))
		expectLockClear(mock, "LOID-L3") // 到期清行,计数从零重开
		expectAccount(mock, "LOID-L3", "ACTIVE", "", "", mustStored(t, "pw-correct"))
		expectLockClear(mock, "LOID-L3") // 认证成功清零
		a := newCredAuth(t, mock, nil)
		d, err := a.Authenticate(context.Background(), "LOID-L3", Credentials{PAP: "pw-correct"})
		if err != nil || !d.Authorize {
			t.Fatalf("d=%+v err=%v", d, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("计数存在但未锁定 放行正确密码", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`FROM lo_auth_lockouts`).
			WithArgs("LOID-L4").
			WillReturnRows(mock.NewRows([]string{"fail_count", "locked_until"}).AddRow(3, nil))
		expectAccount(mock, "LOID-L4", "ACTIVE", "", "", mustStored(t, "pw"))
		expectLockClear(mock, "LOID-L4")
		a := newCredAuth(t, mock, nil)
		d, err := a.Authenticate(context.Background(), "LOID-L4", Credentials{PAP: "pw"})
		if err != nil || !d.Authorize {
			t.Fatalf("d=%+v err=%v", d, err)
		}
	})
}

func TestCredentialAuthorizer_Defaults(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	a := NewCredentialAuthorizer(mock, CredentialConfig{})
	if a.cfg.LockThreshold != 5 || a.cfg.LockWindow != 15*time.Minute || a.cfg.Now == nil {
		t.Fatalf("defaults: %+v", a.cfg)
	}
}
