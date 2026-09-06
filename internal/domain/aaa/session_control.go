package aaa

import (
	"context"
	"fmt"
	"log"
	"time"
)

// SessionControlRepo 会话控制读口(收窄自 PGStore,单测以假仓储注入)。
type SessionControlRepo interface {
	GetLoAccountByID(ctx context.Context, id int64) (*LoAccount, error)
	SuspendLoAccount(ctx context.Context, id int64) error
	GetSessionByID(ctx context.Context, id int64) (*SessionRecord, error)
	ListOnlineSessions(ctx context.Context, loid string) ([]SessionRecord, error)
	ListPendingOfflineSessions(ctx context.Context, limit int) ([]SessionRecord, error)
	ListStaleOnlineSessions(ctx context.Context, staleBefore time.Time, limit int) ([]SessionRecord, error)
	MarkSessionOffline(ctx context.Context, id int64, closeReason string) error
	MarkSessionPendingOffline(ctx context.Context, id int64) error
	MarkSessionOfflineFailed(ctx context.Context, id int64, closeReason string) error
	IncrementSessionAttempts(ctx context.Context, id int64) (int, error)
	ReapSessionZombie(ctx context.Context, id int64) (bool, error)
	AppendCdr(ctx context.Context, c CdrRecord) (int64, error)
}

// SessionControlService 在线会话控制:CoA 强制下线/后台重试/僵尸清理/停复机联动。
// 失败路径一律落 [aaa] 前缀可 grep 日志;重试耗尽输出 [aaa] ALERT。
type SessionControlService struct {
	repo     SessionControlRepo
	sender   DisconnectSender
	retryMax int // Disconnect 重试上限,耗尽转 OFFLINE_FAILED
}

// NewSessionControlService 构造;retryMax<1 回退 1。
func NewSessionControlService(repo SessionControlRepo, sender DisconnectSender, retryMax int) *SessionControlService {
	if retryMax < 1 {
		retryMax = 1
	}
	return &SessionControlService{repo: repo, sender: sender, retryMax: retryMax}
}

// ForceOfflineSessionByID 单会话强制下线(admin 路由入口);非在途会话拒绝。
func (s *SessionControlService) ForceOfflineSessionByID(ctx context.Context, sessionID int64, reason string) error {
	sess, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if sess.Status != SessionOnline && sess.Status != SessionPendingOffline {
		return ErrIllegalTransition
	}
	s.dispatch(ctx, *sess, reason)
	return nil
}

// ForceOfflineLoid 对 LOID 全部 ONLINE 会话下发 Disconnect,返回处理条数。
func (s *SessionControlService) ForceOfflineLoid(ctx context.Context, loid, reason string) (int, error) {
	sessions, err := s.repo.ListOnlineSessions(ctx, loid)
	if err != nil {
		return 0, err
	}
	for _, sess := range sessions {
		s.dispatch(ctx, sess, reason)
	}
	return len(sessions), nil
}

// dispatch 下发 Disconnect:ACK→OFFLINE;失败→PENDING_OFFLINE(重试循环接管)。
// 下发结果只落状态与日志,不向调用方报错(RADIUS 侧异步语义)。
func (s *SessionControlService) dispatch(ctx context.Context, sess SessionRecord, reason string) {
	if err := s.sender.SendDisconnect(ctx, sess.NasIP, sess.Loid, sess.SessionID); err != nil {
		log.Printf("[aaa] disconnect to %s FAILED loid=%s session_id=%s: %v", sess.NasIP, sess.Loid, sess.SessionID, err)
		if merr := s.repo.MarkSessionPendingOffline(ctx, sess.ID); merr != nil {
			log.Printf("[aaa] session pending-offline mark FAILED id=%d: %v", sess.ID, merr)
		}
		return
	}
	if err := s.repo.MarkSessionOffline(ctx, sess.ID, reason); err != nil {
		log.Printf("[aaa] session offline mark FAILED id=%d: %v", sess.ID, err)
	}
}

// RetryPendingOffline 一轮重试:PENDING_OFFLINE 重新下发;已达上限转 OFFLINE_FAILED 并 ALERT。
// 返回本轮实际重发条数。
func (s *SessionControlService) RetryPendingOffline(ctx context.Context, limit int) int {
	pending, err := s.repo.ListPendingOfflineSessions(ctx, limit)
	if err != nil {
		log.Printf("[aaa] retry pending list FAILED: %v", err)
		return 0
	}
	retried := 0
	for _, sess := range pending {
		if sess.DisconnectAttempts >= s.retryMax {
			s.failSession(ctx, sess)
			continue
		}
		if _, err := s.repo.IncrementSessionAttempts(ctx, sess.ID); err != nil {
			log.Printf("[aaa] retry attempts bump FAILED id=%d: %v", sess.ID, err)
			continue
		}
		retried++
		s.dispatch(ctx, sess, CloseReasonCoA)
	}
	return retried
}

// failSession 重试耗尽:OFFLINE_FAILED + [aaa] ALERT(可 grep 告警)。
func (s *SessionControlService) failSession(ctx context.Context, sess SessionRecord) {
	if err := s.repo.MarkSessionOfflineFailed(ctx, sess.ID, CloseReasonOfflineFailed); err != nil {
		log.Printf("[aaa] session offline-failed mark FAILED id=%d: %v", sess.ID, err)
		return
	}
	log.Printf("[aaa] ALERT session offline FAILED after %d attempts: loid=%s session_id=%s nas=%s",
		sess.DisconnectAttempts, sess.Loid, sess.SessionID, sess.NasIP)
}

// ReapZombieSessions 僵尸清理:last_update 早于 staleBefore 的 ONLINE 会话关闭,
// 并补录一条带 ZOMBIE_REAP 标记的本地 Stop 话单;返回清理条数。
func (s *SessionControlService) ReapZombieSessions(ctx context.Context, staleBefore time.Time, limit int) (int, error) {
	stale, err := s.repo.ListStaleOnlineSessions(ctx, staleBefore, limit)
	if err != nil {
		return 0, fmt.Errorf("aaa: zombie list: %w", err)
	}
	n := 0
	for _, sess := range stale {
		ok, err := s.repo.ReapSessionZombie(ctx, sess.ID)
		if err != nil {
			log.Printf("[aaa] zombie reap FAILED id=%d: %v", sess.ID, err)
			continue
		}
		if !ok { // 并发已 Stop/迁移,跳过补录避免双话单
			continue
		}
		s.appendZombieStopCdr(ctx, sess)
		n++
	}
	return n, nil
}

// appendZombieStopCdr 补录带原因标记的本地 Stop 话单;失败留 ALERT 不回滚会话关闭。
func (s *SessionControlService) appendZombieStopCdr(ctx context.Context, sess SessionRecord) {
	if _, err := s.repo.AppendCdr(ctx, zombieStopCdr(sess)); err != nil {
		log.Printf("[aaa] ALERT zombie stop cdr FAILED loid=%s session_id=%s: %v", sess.Loid, sess.SessionID, err)
	}
}

// zombieStopCdr 构造补录 Stop 话单(时长=now-started_at,流量取会话累计)。
func zombieStopCdr(sess SessionRecord) CdrRecord {
	now := time.Now()
	sessionTime := int32(now.Sub(sess.StartedAt).Seconds())
	if sessionTime < 0 {
		sessionTime = 0
	}
	return CdrRecord{
		Loid: sess.Loid, AcctStatus: AcctStatusStop, SessionID: sess.SessionID,
		SessionTime: sessionTime, InputOctets: sess.InputOctets, OutputOctets: sess.OutputOctets,
		NasIP: sess.NasIP, BillingStatus: "UNBILLED", StartedAt: now, CloseReason: CloseReasonZombie,
	}
}

// SuspendWithOffline 停复机联动:停机成功即对该 LOID 全部在线会话强制下线(即时生效)。
func (s *SessionControlService) SuspendWithOffline(ctx context.Context, loAccountID int64) error {
	if err := s.repo.SuspendLoAccount(ctx, loAccountID); err != nil {
		return err
	}
	lo, err := s.repo.GetLoAccountByID(ctx, loAccountID)
	if err != nil {
		return fmt.Errorf("aaa: suspend lookup %d: %w", loAccountID, err)
	}
	if _, err := s.ForceOfflineLoid(ctx, lo.Loid, CloseReasonCoA); err != nil {
		return err
	}
	return nil
}
