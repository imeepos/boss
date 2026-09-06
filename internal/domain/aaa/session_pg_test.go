package aaa

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// 在线会话存储形状回归(pgxmock):Start 幂等去重 / Interim 累加 / Stop 关闭 / 并发闸口。

func TestPGStore_StartSession(t *testing.T) {
	t.Run("新建返回 created", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`INSERT INTO aaa_online_sessions`).
			WithArgs("LOID-1", "S-1", "10.0.0.1", int64(11), int64(22)).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		created, err := NewPGStore(mock).StartSession(context.Background(), SessionRecord{
			Loid: "LOID-1", SessionID: "S-1", NasIP: "10.0.0.1", InputOctets: 11, OutputOctets: 22,
		})
		if err != nil || !created {
			t.Fatalf("created=%v err=%v", created, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("重复 Start 去重", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`ON CONFLICT \(loid, session_id\) DO NOTHING`).
			WithArgs("LOID-1", "S-1", "", int64(0), int64(0)).
			WillReturnResult(pgxmock.NewResult("INSERT", 0))
		created, err := NewPGStore(mock).StartSession(context.Background(), SessionRecord{Loid: "LOID-1", SessionID: "S-1"})
		if err != nil || created {
			t.Fatalf("重复 Start 应幂等去重: created=%v err=%v", created, err)
		}
	})
}

func TestPGStore_TouchSessionTraffic(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	// Interim 语义:在既有累计上做累加(SQL 侧 input_octets = input_octets + $3),仅 ONLINE 生效。
	mock.ExpectExec(`input_octets = input_octets \+ \$3, output_octets = output_octets \+ \$4`).
		WithArgs("LOID-1", "S-1", int64(100), int64(200), SessionOnline).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	if err := NewPGStore(mock).TouchSessionTraffic(context.Background(), "LOID-1", "S-1", 100, 200); err != nil {
		t.Fatalf("TouchSessionTraffic: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPGStore_StopSession(t *testing.T) {
	t.Run("命中关闭", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE aaa_online_sessions`).
			WithArgs("LOID-1", "S-1", SessionOffline, CloseReasonAcctStop, []string{SessionOnline, SessionPendingOffline}).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		found, err := NewPGStore(mock).StopSession(context.Background(), "LOID-1", "S-1", CloseReasonAcctStop)
		if err != nil || !found {
			t.Fatalf("found=%v err=%v", found, err)
		}
	})
	t.Run("孤儿 Stop 不存在", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE aaa_online_sessions`).
			WithArgs("LOID-X", "S-404", SessionOffline, CloseReasonAcctStop, []string{SessionOnline, SessionPendingOffline}).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		found, err := NewPGStore(mock).StopSession(context.Background(), "LOID-X", "S-404", CloseReasonAcctStop)
		if err != nil || found {
			t.Fatalf("孤儿 Stop 只记话单不报错: found=%v err=%v", found, err)
		}
	})
}

func TestPGStore_AllowNewSession(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`COUNT\(\*\) FROM aaa_online_sessions`).
		WithArgs("LOID-1", []string{SessionOnline, SessionPendingOffline}).
		WillReturnRows(mock.NewRows([]string{"count"}).AddRow(1))
	allow, current, err := NewPGStore(mock).AllowNewSession(context.Background(), "LOID-1", 1)
	if err != nil || allow || current != 1 {
		t.Fatalf("allow=%v current=%d err=%v, want 拒绝", allow, current, err)
	}
}

func TestPGStore_GetSessionByID(t *testing.T) {
	t.Run("命中", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		closed := time.Date(2025, 9, 1, 12, 0, 0, 0, time.UTC)
		mock.ExpectQuery(`FROM aaa_online_sessions WHERE id = \$1`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"id", "loid", "session_id", "nas_ip", "started_at", "last_update", "input_octets", "output_octets", "status", "disconnect_attempts", "close_reason", "closed_at"}).
				AddRow(int64(7), "LOID-1", "S-1", "10.0.0.1", ts, ts, int64(1), int64(2), SessionPendingOffline, 2, "", &closed))
		sess, err := NewPGStore(mock).GetSessionByID(context.Background(), 7)
		if err != nil || sess.Status != SessionPendingOffline || sess.DisconnectAttempts != 2 || sess.ClosedAt == nil {
			t.Fatalf("sess=%+v err=%v", sess, err)
		}
	})
	t.Run("未命中 ErrSessionNotFound", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`FROM aaa_online_sessions WHERE id = \$1`).
			WithArgs(int64(404)).
			WillReturnError(pgx.ErrNoRows)
		if _, err := NewPGStore(mock).GetSessionByID(context.Background(), 404); !errors.Is(err, ErrSessionNotFound) {
			t.Fatalf("err=%v, want ErrSessionNotFound", err)
		}
	})
}