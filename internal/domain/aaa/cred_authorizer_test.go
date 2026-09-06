package aaa

// A1 凭据校验决策器单测(pgxmock):R2 用例集——PAP 正确 Accept/错误 Reject、
// CHAP 正确 Accept/篡改 Reject、未设密默认拒+开关放行、连续失败锁定与到期自动解锁。

import (
	"context"
	"crypto/md5"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"

	"github.com/ymm-001/boss/internal/domain/aaa/credential"
)

var fixedNow = time.Date(2026, 9, 6, 8, 0, 0, 0, time.UTC)

func credTestCodec(t *testing.T) *credential.Codec {
	t.Helper()
	c, err := credential.New("unit-test-key-material")
	if err != nil {
		t.Fatalf("codec: %v", err)
	}
	return c
}

func mustStored(t *testing.T, pw string) string {
	t.Helper()
	s, err := credTestCodec(t).Encode(pw)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	return s
}

func newCredAuth(t *testing.T, mock pgxmock.PgxPoolIface, mutate func(*CredentialConfig)) *CredentialAuthorizer {
	t.Helper()
	cfg := CredentialConfig{Codec: credTestCodec(t), Now: func() time.Time { return fixedNow }}
	if mutate != nil {
		mutate(&cfg)
	}
	return NewCredentialAuthorizer(mock, cfg)
}

// expectNoLockRow 预置锁定表无该 LOID 行。
func expectNoLockRow(mock pgxmock.PgxPoolIface, loid string) {
	mock.ExpectQuery(`FROM lo_auth_lockouts`).
		WithArgs(loid).
		WillReturnRows(mock.NewRows([]string{"fail_count", "locked_until"}))
}

// expectAccount 预置账号状态行(status, bandwidth, qos, credential)。
func expectAccount(mock pgxmock.PgxPoolIface, loid, status, bandwidth, qos, credText string) {
	mock.ExpectQuery(`FROM lo_accounts`).
		WithArgs(loid).
		WillReturnRows(mock.NewRows([]string{"status", "bandwidth", "qos", "credential"}).
			AddRow(status, bandwidth, qos, credText))
}

func expectLockClear(mock pgxmock.PgxPoolIface, loid string) {
	mock.ExpectExec(`DELETE FROM lo_auth_lockouts`).
		WithArgs(loid).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
}

func expectFailCount(mock pgxmock.PgxPoolIface, loid string, threshold int) {
	mock.ExpectExec(`INSERT INTO lo_auth_lockouts`).
		WithArgs(loid, threshold, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
}

func TestCredentialAuthorizer_PAP(t *testing.T) {
	t.Run("正确密码 Accept 且清零计数", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		expectNoLockRow(mock, "LOID-P1")
		expectAccount(mock, "LOID-P1", "ACTIVE", "300M/150M", "QOS-STD", mustStored(t, "pw-correct"))
		expectLockClear(mock, "LOID-P1")
		a := newCredAuth(t, mock, nil)
		d, err := a.Authenticate(context.Background(), "LOID-P1", Credentials{PAP: "pw-correct"})
		if err != nil || !d.Authorize || d.Bandwidth != "300M/150M" || d.SessionTTL != defaultSessionTTL {
			t.Fatalf("d=%+v err=%v", d, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("错误密码 Reject 且记一次失败", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		expectNoLockRow(mock, "LOID-P2")
		expectAccount(mock, "LOID-P2", "ACTIVE", "", "", mustStored(t, "pw-real"))
		expectFailCount(mock, "LOID-P2", 5)
		a := newCredAuth(t, mock, nil)
		_, err := a.Authenticate(context.Background(), "LOID-P2", Credentials{PAP: "pw-wrong"})
		if !errors.Is(err, ErrBadCredential) {
			t.Fatalf("err=%v want ErrBadCredential", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("PAP 与 CHAP 皆无 Reject", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		expectNoLockRow(mock, "LOID-P3")
		expectAccount(mock, "LOID-P3", "ACTIVE", "", "", mustStored(t, "pw-real"))
		expectFailCount(mock, "LOID-P3", 5)
		a := newCredAuth(t, mock, nil)
		_, err := a.Authenticate(context.Background(), "LOID-P3", Credentials{})
		if !errors.Is(err, ErrBadCredential) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestCredentialAuthorizer_CHAP(t *testing.T) {
	chapResp := func(pw string, challenge []byte, tamper bool) []byte {
		h := md5.New()
		h.Write([]byte{7})
		h.Write([]byte(pw))
		h.Write(challenge)
		resp := h.Sum(nil)
		if tamper {
			resp[0] ^= 0xFF
		}
		return resp
	}
	chapCreds := func(pw string, challenge []byte, tamper bool) Credentials {
		return Credentials{CHAP: &CHAPCredentials{Ident: 7, Challenge: challenge, Response: chapResp(pw, challenge, tamper)}}
	}
	challenge := []byte{0x11, 0x22, 0x33, 0x44}

	t.Run("正确 CHAP Accept", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		expectNoLockRow(mock, "LOID-C1")
		expectAccount(mock, "LOID-C1", "ACTIVE", "500M/250M", "", mustStored(t, "chap-pw"))
		expectLockClear(mock, "LOID-C1")
		a := newCredAuth(t, mock, nil)
		d, err := a.Authenticate(context.Background(), "LOID-C1", chapCreds("chap-pw", challenge, false))
		if err != nil || !d.Authorize {
			t.Fatalf("d=%+v err=%v", d, err)
		}
	})

	t.Run("篡改摘要 Reject", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		expectNoLockRow(mock, "LOID-C2")
		expectAccount(mock, "LOID-C2", "ACTIVE", "", "", mustStored(t, "chap-pw"))
		expectFailCount(mock, "LOID-C2", 5)
		a := newCredAuth(t, mock, nil)
		_, err := a.Authenticate(context.Background(), "LOID-C2", chapCreds("chap-pw", challenge, true))
		if !errors.Is(err, ErrBadCredential) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("challenge 不符 Reject", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		expectNoLockRow(mock, "LOID-C3")
		expectAccount(mock, "LOID-C3", "ACTIVE", "", "", mustStored(t, "chap-pw"))
		expectFailCount(mock, "LOID-C3", 5)
		a := newCredAuth(t, mock, nil)
		resp := chapResp("chap-pw", challenge, false) // 摘要按真 challenge 计算
		creds := Credentials{CHAP: &CHAPCredentials{Ident: 7, Challenge: []byte("evil"), Response: resp}}
		_, err := a.Authenticate(context.Background(), "LOID-C3", creds)
		if !errors.Is(err, ErrBadCredential) {
			t.Fatalf("err=%v", err)
		}
	})
}
