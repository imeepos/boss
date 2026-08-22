package order

// 报障工单(complaints)PG 存取(从 pg_sub.go 拆出,保持单文件 ≤300 行)。

import (
	"context"
	"fmt"
)

// ListComplaints 列出全部报障工单。
func (s *PGStore) ListComplaints(ctx context.Context) ([]Complaint, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, ticket_no, customer_id, COALESCE(order_id, 0), legal_entity_id, legal_entity_name,
		       type, status,
		       COALESCE(TO_CHAR(created_at, 'YYYY-MM-DD HH24:MI'), ''),
		       COALESCE(remote_diagnosis, ''),
		       COALESCE(TO_CHAR(sla_deadline, 'YYYY-MM-DD HH24:MI'), '')
		FROM complaints ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("order: list complaints: %w", err)
	}
	defer rows.Close()
	out := make([]Complaint, 0)
	for rows.Next() {
		var c Complaint
		if err := rows.Scan(&c.ID, &c.TicketNo, &c.CustomerID, &c.OrderID,
			&c.LegalEntityID, &c.LegalEntityName, &c.Type, &c.Status,
			&c.CreatedAt, &c.RemoteDiagnosis, &c.SlaDeadline); err != nil {
			return nil, fmt.Errorf("order: scan complaint: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CreateComplaint 新建报障工单,返回自增 id。
// 校验 customer_id 和 order_id(若非零)存在性,防止孤儿投诉。
// Caller 可传 RemoteDiagnosis 和 SlaDeadline(格式 YYYY-MM-DD HH24:MI);空值走 DEFAULT。
func (s *PGStore) CreateComplaint(ctx context.Context, c Complaint) (int64, error) {
	// 关联完整性校验
	if c.CustomerID > 0 {
		ok, err := s.exists(ctx, "customers", c.CustomerID, "")
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("order: customer %d: %w", c.CustomerID, ErrForeignKeyViolation)
		}
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
	err := s.db.QueryRow(ctx, `
		INSERT INTO complaints(ticket_no, customer_id, order_id, legal_entity_id, legal_entity_name,
		                       type, status, remote_diagnosis, sla_deadline)
		VALUES($1,$2,$3,$4,$5,$6,$7,
		       $8,
		       CASE WHEN $9 = '' THEN NULL ELSE TO_TIMESTAMP($9, 'YYYY-MM-DD HH24:MI') END)
		RETURNING id`,
		c.TicketNo, c.CustomerID, idOrNil(c.OrderID), c.LegalEntityID, c.LegalEntityName,
		c.Type, c.Status, c.RemoteDiagnosis, c.SlaDeadline).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("order: create complaint: %w", err)
	}
	return id, nil
}

// CloseComplaint 投诉办结(ticketNo 寻址,status→CLOSED);未命中返回 ErrOrderNotFound。
func (s *PGStore) CloseComplaint(ctx context.Context, ticketNo string) error {
	tag, err := s.db.Exec(ctx, `UPDATE complaints SET status='CLOSED' WHERE ticket_no=$1`, ticketNo)
	if err != nil {
		return fmt.Errorf("order: close complaint: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}
