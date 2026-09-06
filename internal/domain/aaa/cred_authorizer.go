package aaa

// A1(认证凭证与防爆破):凭据校验决策器(RADIUS Access-Request 专用)。
// 链路:锁定检查(LOCKED)→ 账号状态(NOT_FOUND/SUSPENDED/CLOSED)→ 凭据校验(PAP/CHAP);
// BAD_CREDENTIAL 记一次连续失败,达阈值即锁定;成功清零计数。
// 未设密账号默认拒绝;AllowNoCred 为迁移缓冲开关(存量账号过渡,默认关)。

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ymm-001/boss/internal/domain/aaa/credential"
)

// CredentialConfig 凭据校验配置;零值项取缺省(阈值 5 次/窗口 15 分钟/系统时钟)。
type CredentialConfig struct {
	Codec         *credential.Codec
	AllowNoCred   bool             // 未设密账号放行(迁移缓冲,默认 false)
	LockThreshold int              // 连续失败锁定阈值(默认 5)
	LockWindow    time.Duration    // 锁定时长(默认 15 分钟)
	Now           func() time.Time // 可注入时钟(测试);nil=time.Now
}

// CredentialAuthorizer CredentialAuthenticator 的 PG 实现(权威状态=lo_accounts)。
type CredentialAuthorizer struct {
	db  dbtx
	cfg CredentialConfig
}

// NewCredentialAuthorizer 构造并补齐缺省配置。
func NewCredentialAuthorizer(db dbtx, cfg CredentialConfig) *CredentialAuthorizer {
	if cfg.LockThreshold <= 0 {
		cfg.LockThreshold = 5
	}
	if cfg.LockWindow <= 0 {
		cfg.LockWindow = 15 * time.Minute
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &CredentialAuthorizer{db: db, cfg: cfg}
}

const credAccountSQL = `SELECT a.status, COALESCE(o.bandwidth, ''), COALESCE(q.code, ''), COALESCE(a.password_credential, '')
	FROM lo_accounts a
	LEFT JOIN product_offers o ON o.id = a.offer_id
	LEFT JOIN qos_templates q ON q.id = a.qos_template_id
	WHERE a.loid = $1`

// Authenticate RADIUS Access-Request 全链路校验;error 即失败原因码来源。
func (a *CredentialAuthorizer) Authenticate(ctx context.Context, loid string, creds Credentials) (Decision, error) {
	if err := a.checkLock(ctx, loid); err != nil {
		return Decision{}, err
	}
	var status, bandwidth, qos, credText string
	err := a.db.QueryRow(ctx, credAccountSQL, loid).Scan(&status, &bandwidth, &qos, &credText)
	if errors.Is(err, pgx.ErrNoRows) {
		return Decision{}, ErrNotFound
	}
	if err != nil {
		return Decision{}, fmt.Errorf("aaa: cred load: %w", err)
	}
	switch Status(status) {
	case StatusClosed:
		return Decision{LOID: loid}, ErrClosed
	case StatusActive:
		return a.verifyCredential(ctx, loid, bandwidth, qos, credText, creds)
	default: // SUSPENDED 及其余非 ACTIVE 视为停服(与 PGAuthorizer 口径一致)
		return Decision{LOID: loid}, ErrSuspended
	}
}

// verifyCredential 凭据校验:未设密且开关开 → 放行;皆无/不符 → 记失败并拒;成功 → 清零。
func (a *CredentialAuthorizer) verifyCredential(ctx context.Context, loid, bandwidth, qos, credText string, creds Credentials) (Decision, error) {
	if credText == "" {
		if a.cfg.AllowNoCred {
			if err := a.clearLock(ctx, loid); err != nil {
				return Decision{}, err
			}
			return authorizedDecision(loid, bandwidth, qos), nil
		}
		return a.rejectAttempt(ctx, loid)
	}
	if creds.PAP == "" && creds.CHAP == nil { // PAP/CHAP 皆无一律拒
		return a.rejectAttempt(ctx, loid)
	}
	if !a.match(credText, creds) {
		return a.rejectAttempt(ctx, loid)
	}
	if err := a.clearLock(ctx, loid); err != nil { // 认证成功清零计数
		return Decision{}, err
	}
	return authorizedDecision(loid, bandwidth, qos), nil
}

// match 按请求形态分流校验:CHAP 优先(RADIUS 客户端 PAP/CHAP 不同时上送)。
func (a *CredentialAuthorizer) match(credText string, creds Credentials) bool {
	if creds.CHAP != nil {
		return a.cfg.Codec.CHAPOK(credText, creds.CHAP.Ident, creds.CHAP.Challenge, creds.CHAP.Response)
	}
	return a.cfg.Codec.Equal(credText, creds.PAP)
}

// rejectAttempt 记一次连续失败(达阈值即锁定)并返回 BAD_CREDENTIAL。
func (a *CredentialAuthorizer) rejectAttempt(ctx context.Context, loid string) (Decision, error) {
	if err := a.recordFail(ctx, loid); err != nil {
		return Decision{}, err
	}
	return Decision{LOID: loid}, ErrBadCredential
}

// authorizedDecision 组装放行决策:带宽=套餐带宽,QoS 码兜底,TTL 缺省(同 PGAuthorizer 口径)。
func authorizedDecision(loid, bandwidth, qos string) Decision {
	if bandwidth == "" {
		bandwidth = qos
	}
	return Decision{LOID: loid, Authorize: true, Bandwidth: bandwidth, QosTemplate: qos, SessionTTL: defaultSessionTTL}
}
