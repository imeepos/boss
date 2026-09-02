package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// List 订单列表读模型:联表取客户/产品/地址名,按 keyword/status/客户过滤 + 分页。
func (s *PGStore) List(ctx context.Context, q OrderQuery) ([]OrderListItem, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := s.db.Query(ctx, `
		SELECT o.id, o.order_no, COALESCE(c.name,''), COALESCE(p.name,''),
		       COALESCE(a.name, ''), o.address_id, o.stage, o.status, o.created_at
		FROM orders o
		LEFT JOIN customers c ON o.customer_id = c.id
		LEFT JOIN product_offers p ON o.offer_id = p.id
		LEFT JOIN addresses a ON o.address_id = a.id
		WHERE ($1 = '' OR o.order_no ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR o.status = ANY(string_to_array($2, ',')))
		  AND ($3::bigint = 0 OR o.customer_id = $3)
		  AND ($4::bigint = 0 OR o.legal_entity_id = $4)
		  AND ($5 = '' OR o.region_path = $5 OR o.region_path LIKE $5 || '.%')
		ORDER BY o.id DESC
		LIMIT $6 OFFSET $7`,
		q.Keyword, q.Status, q.CustomerID, q.LegalEntityID, q.RegionScope, limit, q.Offset)
	if err != nil {
		return nil, fmt.Errorf("order: list: %w", err)
	}
	defer rows.Close()
	out := make([]OrderListItem, 0)
	for rows.Next() {
		var it OrderListItem
		if err := rows.Scan(&it.ID, &it.OrderNo, &it.Customer, &it.Product, &it.Address,
			&it.AddressID, &it.Stage, &it.Status, &it.CreatedAt); err != nil {
			return nil, fmt.Errorf("order: scan list item: %w", err)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// GetByNo 按订单号查订单(REST 以 orderNo 寻址);未命中返回 ErrOrderNotFound。
func (s *PGStore) GetByNo(ctx context.Context, orderNo string) (*Order, error) {
	var o Order
	err := s.db.QueryRow(ctx, `SELECT `+orderCols+` FROM orders WHERE order_no = $1`, orderNo).
		Scan(&o.ID, &o.OrderNo, &o.CustomerID, &o.OfferID, &o.AddressID, &o.Stage, &o.Status,
			&o.ChannelID, &o.LegalEntityID, &o.RegionPath, &o.BillingMode, &o.BuyMonths, &o.GiftMonths, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("order: get by no: %w", err)
	}
	return &o, nil
}

// findByRequestID 幂等键回读(000116);不存在返回 (nil,nil) 由调用方继续建单。
func (s *PGStore) findByRequestID(ctx context.Context, customerID int64, requestID string) (*Order, error) {
	var o Order
	err := s.db.QueryRow(ctx,
		`SELECT `+orderCols+` FROM orders WHERE customer_id = $1 AND request_id = $2`,
		customerID, requestID,
	).Scan(&o.ID, &o.OrderNo, &o.CustomerID, &o.OfferID, &o.AddressID, &o.Stage, &o.Status,
		&o.ChannelID, &o.LegalEntityID, &o.RegionPath, &o.BillingMode, &o.BuyMonths, &o.GiftMonths, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("order: find by request_id: %w", err)
	}
	return &o, nil
}
