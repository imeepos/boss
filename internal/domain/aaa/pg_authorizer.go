package aaa

// W6:PG 授权器 + 停复机即时生效(状态迁移)。
// 授权链路:lo_accounts JOIN product_offers(带宽)/qos_templates(QoS) → Decision;
// 停机(SUSPENDED)→ 拒绝接入,复机(ACTIVE)→ 放行。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ErrIllegalTransition 非法状态迁移(如重复停机/复机非停机账号)。
var ErrIllegalTransition = errors.New("aaa: illegal transition")

// defaultSessionTTL RADIUS Session-Timeout 缺省(秒);阶段7按套餐覆盖。
const defaultSessionTTL = 3600

// PGAuthorizer PG 授权器:权威状态来自 lo_accounts(技术栈方案 3.4:前置 Redis 缓存 TTL 60s 可选)。
type PGAuthorizer struct {
	db dbtx
}

// NewPGAuthorizer 构造 PG 授权器。
func NewPGAuthorizer(db dbtx) *PGAuthorizer { return &PGAuthorizer{db: db} }

// Decide 判定 LOID 是否可接入:ACTIVE 放行(带宽=套餐带宽,QoS 码兜底),否则拒。
func (s *PGAuthorizer) Decide(ctx context.Context, loid string) (Decision, error) {
	var status, bandwidth, qos string
	err := s.db.QueryRow(ctx, `
		SELECT a.status, COALESCE(o.bandwidth, ''), COALESCE(q.code, '')
		FROM lo_accounts a
		LEFT JOIN product_offers o ON o.id = a.offer_id
		LEFT JOIN qos_templates q ON q.id = a.qos_template_id
		WHERE a.loid = $1`, loid).Scan(&status, &bandwidth, &qos)
	if errors.Is(err, pgx.ErrNoRows) {
		return Decision{}, ErrNotFound
	}
	if err != nil {
		return Decision{}, fmt.Errorf("aaa: decide: %w", err)
	}
	if Status(status) != StatusActive {
		return Decision{LOID: loid, Authorize: false}, ErrSuspended
	}
	if bandwidth == "" {
		bandwidth = qos
	}
	return Decision{LOID: loid, Authorize: true, Bandwidth: bandwidth, SessionTTL: defaultSessionTTL}, nil
}

// SuspendLoAccount 停机(欠费/人工):ACTIVE→SUSPENDED;仅 ACTIVE 可停。
func (s *PGStore) SuspendLoAccount(ctx context.Context, loAccountID int64) error {
	return s.transitLoStatus(ctx, loAccountID, "ACTIVE", "SUSPENDED")
}

// ResumeLoAccount 复机(缴费):SUSPENDED→ACTIVE;仅 SUSPENDED 可复。
func (s *PGStore) ResumeLoAccount(ctx context.Context, loAccountID int64) error {
	return s.transitLoStatus(ctx, loAccountID, "SUSPENDED", "ACTIVE")
}

func (s *PGStore) transitLoStatus(ctx context.Context, id int64, from, to string) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE lo_accounts SET status = $2 WHERE id = $1 AND status = $3`, id, to, from)
	if err != nil {
		return fmt.Errorf("aaa: transit lo status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrIllegalTransition
	}
	return nil
}
