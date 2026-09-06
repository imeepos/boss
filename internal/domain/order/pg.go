package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// dbtx 是 PGStore 依赖的最小数据库接口;*pgxpool.Pool 天然满足,单测用 pgxmock 注入。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type transactionalDB interface {
	dbtx
	Begin(ctx context.Context) (pgx.Tx, error)
}

// PGStore 是 OrderService 接口的 PostgreSQL 实现(阶段5)。
type PGStore struct {
	db         dbtx
	cust       CustomerLookup    // 跨域:客户存在性校验
	checker    ResourceChecker   // 跨域:资源核查(环节2)
	reserve    PortReserver      // 跨域:端口预占(环节3/5)
	quad       QuadLinkPrebinder // 跨域:四码预绑定(环节5)
	prepaid    PrepaidCollector  // 跨域:预付费当场收款(环节4)
	commission PartnerCommissionAccrual
	notifier   StageNotifier // 可选:环节推进广播(开放平台 Webhook,nil=未启用)
	params     PartnerCommissionRate
	prof       UserProfileCreator      // 跨域:创建认证账号(环节6)
	prov       ProvisionTaskCreator    // 跨域:创建下发任务(环节7)
	tpl        ProvisionTemplateFinder // 跨域:套餐→下发模板解析(环节7)
	odnGate    CoverageGate            // 可选:ODN 覆盖门控(P2 下单硬校验,未注入=跳过;灰度 BOSS_ODN_COVERAGE_GATE)
}

// NewPGStore 构造 PGStore;cust 由 app 装配层注入 customer 域实现。
// checker/reserve/quad 可选(各环节依赖,未注入时对应环节返回错误)。
func NewPGStore(db dbtx, cust CustomerLookup, extras ...any) *PGStore {
	s := &PGStore{db: db, cust: cust}
	for _, e := range extras {
		switch v := e.(type) {
		case ResourceChecker:
			s.checker = v
		case PortReserver:
			s.reserve = v
		case QuadLinkPrebinder:
			s.quad = v
		case PrepaidCollector:
			s.prepaid = v
		case PartnerCommissionAccrual:
			s.commission = v
		case PartnerCommissionRate:
			s.params = v
		case UserProfileCreator:
			s.prof = v
		case ProvisionTaskCreator:
			s.prov = v
			if f, ok := v.(ProvisionTemplateFinder); ok { // 同一适配器可同时实现两口
				s.tpl = f
			}
		case CoverageGate:
			s.odnGate = v
		}
	}
	return s
}

// SetStageNotifier 注入环节推进广播钩子(app 装配层,wireAAAInfra 之后调用);
// 广播尽力而为,失败不影响环节推进本身。
func (s *PGStore) SetStageNotifier(n StageNotifier) { s.notifier = n }

// SetCoverageGate 注入 ODN 覆盖门控(app 装配层,wireGeoServices 之后;灰度 BOSS_ODN_COVERAGE_GATE=on)。
// nil=门控关闭(下单不校验覆盖)。
func (s *PGStore) SetCoverageGate(g CoverageGate) { s.odnGate = g }

const orderCols = `id, order_no, customer_id, offer_id, address_id, stage, status, channel_id, legal_entity_id, region_path, billing_mode, buy_months, gift_months, created_at`

// ErrAddressNotFound 安装地址不存在(orders.address_id 无外键,应用层校验)。
var ErrAddressNotFound = errors.New("order: address not found")

// ErrOfferNotOrderable 产品不存在或非 PUBLISHED(草稿/下架不可下单)。
var ErrOfferNotOrderable = errors.New("order: offer not found or not published")

// ErrChannelNotActive 渠道不存在或已停用。
var ErrChannelNotActive = errors.New("order: channel not found or disabled")

// ErrInvalidInput 请求参数超出约束(如 buyMonths 越界)。
var ErrInvalidInput = errors.New("order: invalid input")

// ErrForeignKeyViolation 关联实体不存在(投诉/工单等子表写入口的孤儿数据防护)。
var ErrForeignKeyViolation = errors.New("order: referenced entity not found")

// exists 校验单表存在性(orders 无外键约束,关联完整性由本域应用层保证)。
func (s *PGStore) exists(ctx context.Context, table string, id int64, extra string) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM `+table+` WHERE id = $1`+extra+`)`, id).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("order: check %s %d: %w", table, id, err)
	}
	return ok, nil
}

// Reserve 端口预占(环节3):经工作流推进 stage=3 且 status PENDING→RESERVED。
func (s *PGStore) Reserve(ctx context.Context, orderID int64) error {
	return s.advance(ctx, orderID, "reservePort")
}

// Track 跟踪:返回订单 + 环节时间轴。
func (s *PGStore) Track(ctx context.Context, orderID int64) (*Order, []StageLog, error) {
	var o Order
	err := s.db.QueryRow(ctx, `SELECT `+orderCols+` FROM orders WHERE id = $1`, orderID).
		Scan(&o.ID, &o.OrderNo, &o.CustomerID, &o.OfferID, &o.AddressID, &o.Stage, &o.Status,
			&o.ChannelID, &o.LegalEntityID, &o.RegionPath, &o.BillingMode, &o.BuyMonths, &o.GiftMonths, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("order: track order: %w", err)
	}

	rows, err := s.db.Query(ctx,
		`SELECT id, order_id, stage, result, retries, finished_at FROM order_stages WHERE order_id = $1 ORDER BY stage`, orderID)
	if err != nil {
		return nil, nil, fmt.Errorf("order: track stages: %w", err)
	}
	defer rows.Close()
	stages := make([]StageLog, 0)
	for rows.Next() {
		var lg StageLog
		var fin pgtype.Timestamptz
		if err := rows.Scan(&lg.ID, &lg.OrderID, &lg.Stage, &lg.Result, &lg.Retries, &fin); err != nil {
			return nil, nil, fmt.Errorf("order: scan stage: %w", err)
		}
		if fin.Valid {
			t := fin.Time
			lg.FinishedAt = &t
		}
		stages = append(stages, lg)
	}
	return &o, stages, rows.Err()
}

// resolveOwnership 地址→经营区域→最近覆盖祖先的运营主体(migrations/000076)。
// 未匹配到子公司覆盖时兜底平台总公司(migrations/000077):root 挂总公司,天然最近祖先兜底;
// 地址连区域都未挂时直接返回总公司。仅平台总公司未配置才报错。
func (s *PGStore) resolveOwnership(ctx context.Context, addressID int64) (AddressOwnership, error) {
	var own AddressOwnership
	err := s.db.QueryRow(ctx, `
		SELECT cov.legal_entity_id, cov.path::text
		FROM addresses a
		JOIN LATERAL (
			SELECT r.legal_entity_id, r.path
			FROM regions r
			WHERE r.path <@ (SELECT path FROM regions WHERE id = a.region_id)
			  AND r.legal_entity_id IS NOT NULL
			ORDER BY r.path DESC
			LIMIT 1
		) cov ON TRUE
		WHERE a.id = $1`, addressID).Scan(&own.LegalEntityID, &own.RegionPath)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.platformFallback(ctx)
	}
	if err != nil {
		return AddressOwnership{}, fmt.Errorf("order: resolve ownership: %w", err)
	}
	return own, nil
}

// platformFallback 兜底平台总公司(is_platform 唯一),RegionPath 取 root。
func (s *PGStore) platformFallback(ctx context.Context) (AddressOwnership, error) {
	var own AddressOwnership
	err := s.db.QueryRow(ctx,
		`SELECT id, 'root' FROM legal_entities WHERE is_platform LIMIT 1`).
		Scan(&own.LegalEntityID, &own.RegionPath)
	if errors.Is(err, pgx.ErrNoRows) {
		return AddressOwnership{}, fmt.Errorf("order: submit needs platform entity: %w", ErrPlatformMissing)
	}
	if err != nil {
		return AddressOwnership{}, fmt.Errorf("order: platform fallback: %w", err)
	}
	return own, nil
}

// appendStage 在默认连接上写环节日志(事务内写入走 advanceGuarded/appendStage 自由函数)。
func (s *PGStore) appendStage(ctx context.Context, orderID int64, stage int8, result string) error {
	return appendStage(ctx, s.db, orderID, stage, result)
}
