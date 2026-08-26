package order

// 报障工单(complaints)PG 存取(从 pg_sub.go 拆出,保持单文件 ≤300 行)。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// complaintColumns SELECT 子句与 Scan 顺序的单一事实源,被 ListComplaints /
// ListComplaintsByCustomerPaged / GetComplaintByNoAndCustomer 三处共用,避免
// 列序错位。加列时只需改这里一处。
const complaintColumns = `id, ticket_no, customer_id, COALESCE(order_id, 0), legal_entity_id, legal_entity_name,
		type, status,
		COALESCE(description, ''),
		COALESCE(contact, ''),
		COALESCE(rel_order_no, ''),
		COALESCE(TO_CHAR(created_at, 'YYYY-MM-DD HH24:MI'), ''),
		COALESCE(remote_diagnosis, ''),
		COALESCE(TO_CHAR(sla_deadline, 'YYYY-MM-DD HH24:MI'), ''), closed_at, COALESCE(closed_by, 0), resolution`

// scanComplaintRow 从 rows 中按 complaintColumns 顺序填充一行。
func scanComplaintRow(rows pgx.Row, c *Complaint) error {
	return rows.Scan(
		&c.ID, &c.TicketNo, &c.CustomerID, &c.OrderID,
		&c.LegalEntityID, &c.LegalEntityName, &c.Type, &c.Status,
		&c.Description, &c.Contact, &c.RelOrderNo,
		&c.CreatedAt, &c.RemoteDiagnosis, &c.SlaDeadline, &c.ClosedAt, &c.ClosedBy, &c.Resolution,
	)
}

// ListComplaints 列出全部报障工单(管理后台/巡检用)。
func (s *PGStore) ListComplaints(ctx context.Context) ([]Complaint, error) {
	rows, err := s.db.Query(ctx, `SELECT `+complaintColumns+` FROM complaints ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("order: list complaints: %w", err)
	}
	defer rows.Close()
	out := make([]Complaint, 0)
	for rows.Next() {
		var c Complaint
		if err := scanComplaintRow(rows, &c); err != nil {
			return nil, fmt.Errorf("order: scan complaint: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListComplaintsByCustomerPaged 用户端 /complaints 用:按 customer_id 过滤 +
// LIMIT/OFFSET 分页,最新在前。hasMore 判定比 LIMIT 多取一条后丢弃。
func (s *PGStore) ListComplaintsByCustomerPaged(ctx context.Context, customerID int64, page, pageSize int) ([]Complaint, bool, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}
	rows, err := s.db.Query(ctx, `SELECT `+complaintColumns+`
		FROM complaints WHERE customer_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3`, customerID, pageSize+1, (page-1)*pageSize)
	if err != nil {
		return nil, false, fmt.Errorf("order: list complaints paged: %w", err)
	}
	defer rows.Close()
	out := make([]Complaint, 0, pageSize)
	for rows.Next() {
		var c Complaint
		if err := scanComplaintRow(rows, &c); err != nil {
			return nil, false, fmt.Errorf("order: scan complaint paged: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(out) > pageSize
	if hasMore {
		out = out[:pageSize]
	}
	return out, hasMore, nil
}

// GetComplaintByNoAndCustomer 按工单号 + 客户双重寻址,归属校验内嵌 SQL,
// 详情页天然防越权。未命中返回 ErrOrderNotFound。
func (s *PGStore) GetComplaintByNoAndCustomer(ctx context.Context, ticketNo string, customerID int64) (*Complaint, error) {
	row := s.db.QueryRow(ctx, `SELECT `+complaintColumns+`
		FROM complaints WHERE ticket_no = $1 AND customer_id = $2`, ticketNo, customerID)
	var c Complaint
	if err := scanComplaintRow(row, &c); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("order: get complaint by no: %w", err)
	}
	return &c, nil
}

// CreateComplaint 新建报障工单,返回自增 id。
// 关联完整性:customer_id NOT NULL 必须存在——未显式传入时从订单推导(工单域
// 投诉挂订单,tk.OrderID 可溯源客户);order_id 非零时校验存在性。防孤儿投诉。
// Caller 可传 RemoteDiagnosis 和 SlaDeadline(格式 YYYY-MM-DD HH24:MI);空值走 DEFAULT;
// Description/Contact/RelOrderNo 由用户端投诉与建议(迁移 000151)写入。
func (s *PGStore) CreateComplaint(ctx context.Context, c Complaint) (int64, error) {
	// 关联完整性校验:客户缺失时经订单推导归属(流程数据完善),仍无归属直接拒。
	if c.CustomerID == 0 && c.OrderID > 0 {
		_ = s.db.QueryRow(ctx,
			`SELECT customer_id FROM orders WHERE id = $1`, c.OrderID).Scan(&c.CustomerID)
	}
	if c.CustomerID <= 0 {
		return 0, fmt.Errorf("order: customer_id required: %w", ErrForeignKeyViolation)
	}
	ok, err := s.exists(ctx, "customers", c.CustomerID, "")
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("order: customer %d: %w", c.CustomerID, ErrForeignKeyViolation)
	}
	if c.OrderID > 0 {
		ok, err := s.exists(ctx, "orders", c.OrderID, "")
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("order: order %d: %w", c.OrderID, ErrForeignKeyViolation)
		}
	}

	var id int64
	err = s.db.QueryRow(ctx, `
		INSERT INTO complaints(ticket_no, customer_id, order_id, legal_entity_id, legal_entity_name,
		                       type, status, description, contact, rel_order_no,
		                       remote_diagnosis, sla_deadline)
		VALUES($1,$2,$3,$4,$5,$6,$7,
		       COALESCE(NULLIF($8, ''), ''),
		       COALESCE(NULLIF($9, ''), ''),
		       COALESCE(NULLIF($10, ''), ''),
		       $11,
		       CASE WHEN $12 = '' THEN NULL ELSE TO_TIMESTAMP($12, 'YYYY-MM-DD HH24:MI') END)
		RETURNING id`,
		c.TicketNo, c.CustomerID, idOrNil(c.OrderID), c.LegalEntityID, c.LegalEntityName,
		c.Type, c.Status, c.Description, c.Contact, c.RelOrderNo,
		c.RemoteDiagnosis, c.SlaDeadline).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("order: create complaint: %w", err)
	}
	return id, nil
}

// CloseComplaint 投诉办结(ticketNo 寻址,status→CLOSED);未命中返回 ErrOrderNotFound。
func (s *PGStore) CloseComplaint(ctx context.Context, ticketNo string) error {
	tag, err := s.db.Exec(ctx, `
		WITH current AS (
			SELECT id, status FROM complaints WHERE ticket_no=$1
		), changed AS (
			UPDATE complaints c SET status='CLOSED', closed_at=now()
			FROM current WHERE c.id=current.id
			RETURNING c.id, current.status AS from_status
		)
		INSERT INTO cs_ticket_events(ticket_id, event_type, from_status, to_status, note)
		SELECT id, 'STATUS_CHANGED', from_status, 'CLOSED', 'complaint closed' FROM changed`, ticketNo)
	if err != nil {
		return fmt.Errorf("order: close complaint: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}
