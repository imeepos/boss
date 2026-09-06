package aaa

import (
	"bytes"
	"context"
	"errors"
	"log"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// 在线会话存储形状回归(pgxmock):Start 幂等去重 / Interim 覆盖累计 / Stop 关闭 / 并发闸口。

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

// A4 Interim 语义回归:覆盖式更新(RFC 2866 累计口径),QuoteMeta 精确匹配实现 SQL——
// SET 为 CASE 取大形态即结构上不回退,回写值=上报累计值(非累加);回退护栏留 [aaa] ALERT;
// 孤儿 Interim no-op。
func TestPGStore_TouchSessionTraffic(t *testing.T) {
	var logBuf bytes.Buffer
	origOut := log.Writer()
	log.SetOutput(&logBuf)
	t.Cleanup(func() { log.SetOutput(origOut) })
	mockExpect := func(mock pgxmock.PgxPoolIface, loid, session string, in, out, keptIn, keptOut int64) {
		mock.ExpectQuery(regexp.QuoteMeta(interimTouchSQL)).
			WithArgs(loid, session, in, out, SessionOnline).
			WillReturnRows(mock.NewRows([]string{"input_octets", "output_octets"}).AddRow(keptIn, keptOut))
	}
	t.Run("累计值覆盖更新", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		// 已存 100/200 < 上报 300/400:回写值=上报累计值 300/400(CASE 取大),而非 100+300 累加。
		mockExpect(mock, "LOID-1", "S-1", 300, 400, 300, 400)
		if err := NewPGStore(mock).TouchSessionTraffic(context.Background(), "LOID-1", "S-1", 300, 400); err != nil {
			t.Fatalf("TouchSessionTraffic: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("等值与零值仅刷新时间不写流量", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		n := logBuf.Len()
		mockExpect(mock, "LOID-1", "S-1", 100, 200, 100, 200) // 等值:kept==reported 无告警
		if err := NewPGStore(mock).TouchSessionTraffic(context.Background(), "LOID-1", "S-1", 100, 200); err != nil {
			t.Fatalf("TouchSessionTraffic: %v", err)
		}
		mockExpect(mock, "LOID-2", "S-2", 0, 0, 0, 0) // 零值:不把累计清成 0,仅刷 last_update
		if err := NewPGStore(mock).TouchSessionTraffic(context.Background(), "LOID-2", "S-2", 0, 0); err != nil {
			t.Fatalf("TouchSessionTraffic: %v", err)
		}
		if logBuf.Len() != n {
			t.Fatalf("等值/零值不得触发回退告警: %s", logBuf.String()[n:])
		}
	})
	t.Run("回退值触发护栏不回写且日志含[aaa]", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		n := logBuf.Len()
		// 已存 300/400 > 上报 50/60:SQL CASE 取大保留已存值(结构上不回写),留 ALERT 供 grep。
		mockExpect(mock, "LOID-1", "S-1", 50, 60, 300, 400)
		if err := NewPGStore(mock).TouchSessionTraffic(context.Background(), "LOID-1", "S-1", 50, 60); err != nil {
			t.Fatalf("护栏是保序而非报错: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
		line := logBuf.String()[n:]
		for _, want := range []string{"[aaa]", "ALERT", "loid=LOID-1", "session=S-1", "kept_in=300", "reported_in=50"} {
			if !strings.Contains(line, want) {
				t.Fatalf("回退告警缺 %q: %s", want, line)
			}
		}
	})
	t.Run("孤儿 Interim no-op", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(regexp.QuoteMeta(interimTouchSQL)).
			WithArgs("LOID-X", "S-404", int64(1), int64(2), SessionOnline).
			WillReturnError(pgx.ErrNoRows)
		if err := NewPGStore(mock).TouchSessionTraffic(context.Background(), "LOID-X", "S-404", 1, 2); err != nil {
			t.Fatalf("孤儿 Interim 属协议语义不得报错: %v", err)
		}
	})
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
