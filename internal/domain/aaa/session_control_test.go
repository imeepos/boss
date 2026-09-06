package aaa

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// 会话控制编排回归(假仓储+假 sender):强制下线/重试耗尽 ALERT/僵尸清理补录话单/停复机联动。

type fakeSessionRepo struct {
	account      *LoAccount
	session      *SessionRecord
	online       []SessionRecord
	pending      []SessionRecord
	stale        []SessionRecord
	offlineMarks []string // 记录 MarkSessionOffline(id, reason)
	pendingMarks []int64
	failMarks    []int64
	bumped       []int64
	reaped       []int64
	cdrs         []CdrRecord
	suspendedID  int64
	staleBefore  time.Time
}

func (f *fakeSessionRepo) GetLoAccountByID(_ context.Context, id int64) (*LoAccount, error) {
	if f.account == nil || f.account.ID != id {
		return nil, ErrNotFound
	}
	return f.account, nil
}

func (f *fakeSessionRepo) SuspendLoAccount(_ context.Context, id int64) error {
	f.suspendedID = id
	return nil
}

func (f *fakeSessionRepo) GetSessionByID(_ context.Context, id int64) (*SessionRecord, error) {
	if f.session == nil || f.session.ID != id {
		return nil, ErrSessionNotFound
	}
	return f.session, nil
}

func (f *fakeSessionRepo) ListOnlineSessions(_ context.Context, _ string) ([]SessionRecord, error) {
	return f.online, nil
}

func (f *fakeSessionRepo) ListPendingOfflineSessions(_ context.Context, _ int) ([]SessionRecord, error) {
	return f.pending, nil
}

func (f *fakeSessionRepo) ListStaleOnlineSessions(_ context.Context, before time.Time, _ int) ([]SessionRecord, error) {
	f.staleBefore = before
	return f.stale, nil
}

func (f *fakeSessionRepo) MarkSessionOffline(_ context.Context, id int64, reason string) error {
	f.offlineMarks = append(f.offlineMarks, reason+":"+strconv.FormatInt(id, 10))
	return nil
}

func (f *fakeSessionRepo) MarkSessionPendingOffline(_ context.Context, id int64) error {
	f.pendingMarks = append(f.pendingMarks, id)
	return nil
}

func (f *fakeSessionRepo) MarkSessionOfflineFailed(_ context.Context, id int64, _ string) error {
	f.failMarks = append(f.failMarks, id)
	return nil
}

func (f *fakeSessionRepo) IncrementSessionAttempts(_ context.Context, id int64) (int, error) {
	f.bumped = append(f.bumped, id)
	return 1, nil
}

func (f *fakeSessionRepo) ReapSessionZombie(_ context.Context, id int64) (bool, error) {
	f.reaped = append(f.reaped, id)
	return true, nil
}

func (f *fakeSessionRepo) AppendCdr(_ context.Context, c CdrRecord) (int64, error) {
	f.cdrs = append(f.cdrs, c)
	return int64(len(f.cdrs)), nil
}

type fakeSender struct {
	err  error
	sent []string
}

func (s *fakeSender) SendDisconnect(_ context.Context, nasIP, loid, sessionID string) error {
	s.sent = append(s.sent, nasIP+"|"+loid+"|"+sessionID)
	return s.err
}

// captureLog 捕获标准 log 输出(断言 [aaa] ALERT 等 grep 信号)。
func captureLog(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer log.SetOutput(os.Stderr)
	fn()
	return buf.String()
}

func sess(id int64, loid, nas string, attempts int, status string) SessionRecord {
	return SessionRecord{ID: id, Loid: loid, SessionID: "S-1", NasIP: nas, Status: status, DisconnectAttempts: attempts}
}

func TestForceOfflineLoid(t *testing.T) {
	t.Run("ACK 关闭会话", func(t *testing.T) {
		repo := &fakeSessionRepo{online: []SessionRecord{sess(1, "LOID-1", "10.0.0.1", 0, SessionOnline)}}
		sender := &fakeSender{}
		svc := NewSessionControlService(repo, sender, 3)
		n, err := svc.ForceOfflineLoid(context.Background(), "LOID-1", CloseReasonCoA)
		if err != nil || n != 1 || len(repo.offlineMarks) != 1 || repo.offlineMarks[0] != CloseReasonCoA+":"+"1" {
			t.Fatalf("n=%d err=%v marks=%v", n, err, repo.offlineMarks)
		}
		if sender.sent[0] != "10.0.0.1|LOID-1|S-1" {
			t.Fatalf("sent=%v", sender.sent)
		}
	})
	t.Run("不可达转 PENDING_OFFLINE", func(t *testing.T) {
		repo := &fakeSessionRepo{online: []SessionRecord{sess(1, "LOID-1", "10.0.0.9", 0, SessionOnline)}}
		logText := captureLog(t, func() {
			svc := NewSessionControlService(repo, &fakeSender{err: errors.New("timeout")}, 3)
			if _, err := svc.ForceOfflineLoid(context.Background(), "LOID-1", CloseReasonCoA); err != nil {
				t.Fatalf("err=%v", err)
			}
		})
		if len(repo.pendingMarks) != 1 || repo.pendingMarks[0] != 1 {
			t.Fatalf("pendingMarks=%v", repo.pendingMarks)
		}
		if !strings.Contains(logText, "[aaa] disconnect to 10.0.0.9 FAILED") {
			t.Fatalf("缺可 grep 留痕: %s", logText)
		}
	})
}

func TestRetryPendingOffline_ExhaustedAlert(t *testing.T) {
	repo := &fakeSessionRepo{pending: []SessionRecord{sess(2, "LOID-2", "10.0.0.2", 3, SessionPendingOffline)}}
	sender := &fakeSender{err: errors.New("unreachable")}
	svc := NewSessionControlService(repo, sender, 3)
	logText := captureLog(t, func() {
		if n := svc.RetryPendingOffline(context.Background(), 100); n != 0 {
			t.Fatalf("达上限不应再重发, retried=%d", n)
		}
	})
	// attempts 已达上限(3>=3):不再重发,直接 OFFLINE_FAILED + [aaa] ALERT。
	if len(repo.failMarks) != 1 || repo.failMarks[0] != 2 || len(sender.sent) != 0 {
		t.Fatalf("failMarks=%v sent=%v", repo.failMarks, sender.sent)
	}
	if !strings.Contains(logText, "[aaa] ALERT session offline FAILED after 3 attempts") ||
		!strings.Contains(logText, "loid=LOID-2") {
		t.Fatalf("缺 ALERT: %s", logText)
	}
}

func TestRetryPendingOffline_Resends(t *testing.T) {
	repo := &fakeSessionRepo{pending: []SessionRecord{sess(3, "LOID-3", "10.0.0.3", 0, SessionPendingOffline)}}
	sender := &fakeSender{}
	svc := NewSessionControlService(repo, sender, 3)
	n := svc.RetryPendingOffline(context.Background(), 100)
	if n != 1 || len(repo.bumped) != 1 || len(repo.offlineMarks) != 1 || len(sender.sent) != 1 {
		t.Fatalf("n=%d bumped=%v offline=%v sent=%v", n, repo.bumped, repo.offlineMarks, sender.sent)
	}
}

func TestReapZombieSessions(t *testing.T) {
	stale := sess(4, "LOID-4", "10.0.0.4", 0, SessionOnline)
	stale.StartedAt = time.Now().Add(-3 * time.Hour)
	stale.InputOctets, stale.OutputOctets = 11, 22
	repo := &fakeSessionRepo{stale: []SessionRecord{stale}}
	svc := NewSessionControlService(repo, &fakeSender{}, 3)
	n, err := svc.ReapZombieSessions(context.Background(), time.Now().Add(-2*time.Hour), 100)
	if err != nil || n != 1 || len(repo.reaped) != 1 {
		t.Fatalf("n=%d err=%v reaped=%v", n, err, repo.reaped)
	}
	if len(repo.cdrs) != 1 {
		t.Fatalf("僵尸清理必须补录 Stop 话单: %v", repo.cdrs)
	}
	c := repo.cdrs[0]
	if c.AcctStatus != AcctStatusStop || c.CloseReason != CloseReasonZombie || c.Loid != "LOID-4" ||
		c.InputOctets != 11 || c.OutputOctets != 22 || c.BillingStatus != "UNBILLED" {
		t.Fatalf("补录话单形状不符: %+v", c)
	}
}

func TestSuspendWithOffline(t *testing.T) {
	repo := &fakeSessionRepo{
		account: &LoAccount{ID: 9, Loid: "LOID-9"},
		online:  []SessionRecord{sess(5, "LOID-9", "10.0.0.5", 0, SessionOnline)},
	}
	sender := &fakeSender{}
	svc := NewSessionControlService(repo, sender, 3)
	if err := svc.SuspendWithOffline(context.Background(), 9); err != nil {
		t.Fatalf("SuspendWithOffline: %v", err)
	}
	if repo.suspendedID != 9 || len(repo.offlineMarks) != 1 || len(sender.sent) != 1 {
		t.Fatalf("停机后必须联动下线: suspended=%d offline=%v sent=%v", repo.suspendedID, repo.offlineMarks, sender.sent)
	}
}

func TestForceOfflineSessionByID_Guards(t *testing.T) {
	t.Run("未命中 ErrSessionNotFound", func(t *testing.T) {
		svc := NewSessionControlService(&fakeSessionRepo{}, &fakeSender{}, 3)
		if err := svc.ForceOfflineSessionByID(context.Background(), 404, CloseReasonCoA); !errors.Is(err, ErrSessionNotFound) {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("终态会话拒绝", func(t *testing.T) {
		repo := &fakeSessionRepo{session: &SessionRecord{ID: 6, Status: SessionOffline}}
		svc := NewSessionControlService(repo, &fakeSender{}, 3)
		if err := svc.ForceOfflineSessionByID(context.Background(), 6, CloseReasonCoA); !errors.Is(err, ErrIllegalTransition) {
			t.Fatalf("err=%v", err)
		}
	})
}