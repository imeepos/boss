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

// PGStore 是 OrderService 接口的 PostgreSQL 实现(阶段5)。
type PGStore struct {
	db      dbtx
	cust    CustomerLookup   // 跨域:客户存在性校验
	checker ResourceChecker  // 跨域:资源核查(环节2)
	reserve PortReserver     // 跨域:端口预占(环节3/5)
	quad    QuadLinkPrebinder // 跨域:四码预绑定(环节5)
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
		}
	}
	return s
}

const orderCols = `id, order_no, customer_id, offer_id, address_id, stage, status, channel_id, legal_entity_id, region_path, created_at`

// ErrAddressNotFound 安装地址不存在(orders.address_id 无外键,应用层校验)。
var ErrAddressNotFound = errors.New("order: address not found")

// ErrOfferNotOrderable 产品不存在或非 PUBLISHED(草稿/下架不可下单)。
var ErrOfferNotOrderable = errors.New("order: offer not found or not published")

// ErrChannelNotActive 渠道不存在或已停用。
var ErrChannelNotActive = errors.New("order: channel not found or disabled")

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

// Submit 下单(环节1):校验渠道/客户/产品/地址关联完整性后建单,status=PENDING、stage=1,写环节日志。
func (s *PGStore) Submit(ctx context.Context, req SubmitReq) (*Order, error) {
	if req.ChannelID == 0 {
		return nil, errors.New("order: channel_id required")
	}
	ok, err := s.cust.Exists(ctx, req.CustomerID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("order: customer %d not found", req.CustomerID)
	}
	// 关联完整性(orders 表无外键,这里拒掉孤儿引用):
	// 地址必须存在(否则归属推导会误走平台兜底放行);产品必须 PUBLISHED;渠道必须 ACTIVE。
	addrOK, err := s.exists(ctx, "addresses", req.AddressID, "")
	if err != nil {
		return nil, err
	}
	if !addrOK {
		return nil, fmt.Errorf("order: address %d: %w", req.AddressID, ErrAddressNotFound)
	}
	offerOK, err := s.exists(ctx, "product_offers", req.OfferID, ` AND status = 'PUBLISHED'`)
	if err != nil {
		return nil, err
	}
	if !offerOK {
		return nil, fmt.Errorf("order: offer %d: %w", req.OfferID, ErrOfferNotOrderable)
	}
	chOK, err := s.exists(ctx, "channels", req.ChannelID, ` AND status = 'ACTIVE'`)
	if err != nil {
		return nil, err
	}
	if !chOK {
		return nil, fmt.Errorf("order: channel %d: %w", req.ChannelID, ErrChannelNotActive)
	}
	own, err := s.resolveOwnership(ctx, req.AddressID)
	if err != nil {
		return nil, err
	}
	if err := checkOwnershipConflict(req, own); err != nil {
		return nil, err
	}

	// 订单号由数据库序列发号(migrations/000031):跨进程/重启不重复。
	var orderNo string
	if err := s.db.QueryRow(ctx,
		`SELECT 'ORD-' || to_char(now(), 'YYYYMMDD') || '-' || lpad(nextval('order_no_seq')::text, 6, '0')`,
	).Scan(&orderNo); err != nil {
		return nil, fmt.Errorf("order: next order_no: %w", err)
	}
	o := &Order{
		OrderNo:       orderNo,
		CustomerID:    req.CustomerID,
		OfferID:       req.OfferID,
		AddressID:     req.AddressID,
		Stage:         1,
		Status:        "PENDING",
		ChannelID:     req.ChannelID,
		LegalEntityID: own.LegalEntityID,
		RegionPath:    own.RegionPath,
	}
	err = s.db.QueryRow(ctx, `
		INSERT INTO orders(order_no, customer_id, offer_id, address_id, stage, status, channel_id, legal_entity_id, region_path)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
		o.OrderNo, o.CustomerID, o.OfferID, o.AddressID, o.Stage, o.Status, o.ChannelID, o.LegalEntityID, o.RegionPath).Scan(&o.ID)
	if err != nil {
		return nil, fmt.Errorf("order: submit insert: %w", err)
	}
	if err := s.appendStage(ctx, o.ID, 1, "DOING"); err != nil {
		return nil, err
	}
	return o, nil
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
			&o.ChannelID, &o.LegalEntityID, &o.RegionPath, &o.CreatedAt)
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

// appendStage 写环节日志。
func (s *PGStore) appendStage(ctx context.Context, orderID int64, stage int8, result string) error {
	if _, err := s.db.Exec(ctx,
		`INSERT INTO order_stages(order_id, stage, result) VALUES($1,$2,$3)`, orderID, stage, result); err != nil {
		return fmt.Errorf("order: append stage: %w", err)
	}
	return nil
}
