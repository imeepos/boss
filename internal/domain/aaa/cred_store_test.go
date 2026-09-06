package aaa

// A1:ResetLoPassword 落库契约(密文写列 + 清防爆破计数 + 明文仅返回值可见)。

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"

	"github.com/ymm-001/boss/internal/domain/aaa/credential"
)

func TestPGStore_ResetLoPassword(t *testing.T) {
	codec, err := credential.New("unit-test-key-material")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("成功:明文一次性返回并清锁定", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE lo_accounts SET password_credential`).
			WithArgs("LOID-R1", pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`DELETE FROM lo_auth_lockouts`).
			WithArgs("LOID-R1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))
		s := NewPGStore(mock).WithCredentialCodec(codec)
		pw, err := s.ResetLoPassword(context.Background(), "LOID-R1")
		if err != nil {
			t.Fatalf("ResetLoPassword: %v", err)
		}
		if len(pw) != 16 || strings.ContainsAny(pw, "iIlLoO01") {
			t.Fatalf("pw=%q want 16 chars, no ambiguous chars", pw)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("未配置编解码器 报错", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		_, err := NewPGStore(mock).ResetLoPassword(context.Background(), "LOID-R2")
		if err == nil || !strings.Contains(err.Error(), "codec not configured") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("LOID 不存在 ErrNotFound", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE lo_accounts SET password_credential`).
			WithArgs("LOID-MISS", pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		s := NewPGStore(mock).WithCredentialCodec(codec)
		_, err := s.ResetLoPassword(context.Background(), "LOID-MISS")
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v want ErrNotFound", err)
		}
	})
}
