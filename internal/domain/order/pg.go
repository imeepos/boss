package order

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

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
	cust    CustomerLookup  // 跨域:客户存在性校验
	checker ResourceChecker // 跨域:资源核查(环节2)
	seq     atomic.Int64    // 订单号序号(单进程内)
}

// NewPGStore 构造 PGStore;cust 由 app 装配层注入 customer 域实现。
// checker 可选(环节2 资源核查依赖,未注入时 CheckResource 返回错误)。
func NewPGStore(db dbtx, cust CustomerLookup, checker ...ResourceChecker) *PGStore {
	var c ResourceChecker
	if len(checker) > 0 {
		c = checker[0]
	}
	return &PGStore{db: db, cust: cust, checker: c}
}

const orderCols = `id, order_no, customer_id, offer_id, address_id, stage, status, channel_id, legal_entity_id, region_path, created_at`

// Submit 下单(环节1):校验渠道/客户后建单,status=PENDING、stage=1,写环节日志。
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

	n := s.seq.Add(1)
	o := &Order{
		OrderNo:       fmt.Sprintf("ORD-%s-%06d", time.Now().Format("20060102"), n),
		CustomerID:    req.CustomerID,
		OfferID:       req.OfferID,
		AddressID:     req.AddressID,
		Stage:         1,
		Status:        "PENDING",
		ChannelID:     req.ChannelID,
		LegalEntityID: req.LegalEntityID,
		RegionPath:    req.RegionPath,
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

// Reserve 端口预占(环节3):状态机 PENDING→RESERVED。
func (s *PGStore) Reserve(ctx context.Context, orderID int64) error {
	var status string
	err := s.db.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1`, orderID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrOrderNotFound
	}
	if err != nil {
		return fmt.Errorf("order: reserve select: %w", err)
	}
	next, err := transition(status, "reserve")
	if err != nil {
		return err // ErrIllegalTransition
	}
	if _, err := s.db.Exec(ctx, `UPDATE orders SET status = $2 WHERE id = $1`, orderID, next); err != nil {
		return fmt.Errorf("order: reserve update: %w", err)
	}
	return s.appendStage(ctx, orderID, 3, "DONE")
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

// appendStage 写环节日志。
func (s *PGStore) appendStage(ctx context.Context, orderID int64, stage int8, result string) error {
	if _, err := s.db.Exec(ctx,
		`INSERT INTO order_stages(order_id, stage, result) VALUES($1,$2,$3)`, orderID, stage, result); err != nil {
		return fmt.Errorf("order: append stage: %w", err)
	}
	return nil
}
