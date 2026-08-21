package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// idOrNil 把 0 归一为 NULL(可空外键约定:0=空)。
func idOrNil(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

// ListDispatchTickets 列出全部派单工单。
func (s *PGStore) ListDispatchTickets(ctx context.Context) ([]DispatchTicket, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, ticket_no, order_id, COALESCE(worker_id, 0), COALESCE(worker_name, ''),
		       COALESCE(group_id, 0), COALESCE(group_name, ''), COALESCE(region_id, 0), COALESCE(region_name, ''),
		       legal_entity_id, legal_entity_name, status
		FROM dispatch_tickets ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("order: list dispatch tickets: %w", err)
	}
	defer rows.Close()
	out := make([]DispatchTicket, 0)
	for rows.Next() {
		var t DispatchTicket
		if err := rows.Scan(&t.TicketID, &t.TicketNo, &t.OrderID, &t.WorkerID, &t.WorkerName,
			&t.GroupID, &t.GroupName, &t.RegionID, &t.RegionName, &t.LegalEntityID, &t.LegalEntityName, &t.Status); err != nil {
			return nil, fmt.Errorf("order: scan dispatch ticket: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListTicketItems 列表读模型:派单工单联表订单/客户/地址(师傅端列表页)。
func (s *PGStore) ListTicketItems(ctx context.Context) ([]TicketItem, error) {
	rows, err := s.db.Query(ctx, `
		SELECT t.id, t.ticket_no, t.order_id, COALESCE(t.worker_id, 0), t.status,
		       COALESCE(o.status, ''), COALESCE(c.name, ''), COALESCE(a.name, ua.detail, ''), COALESCE(o.stage, 0)
		FROM dispatch_tickets t
		LEFT JOIN orders o ON t.order_id = o.id
		LEFT JOIN customers c ON o.customer_id = c.id
		LEFT JOIN addresses a ON o.address_id = a.id
		LEFT JOIN user_addresses ua ON o.address_id = ua.id
		ORDER BY t.id`)
	if err != nil {
		return nil, fmt.Errorf("order: list ticket items: %w", err)
	}
	defer rows.Close()
	out := make([]TicketItem, 0)
	for rows.Next() {
		var it TicketItem
		if err := rows.Scan(&it.TicketID, &it.TicketNo, &it.OrderID, &it.WorkerID, &it.Status,
			&it.OrderStatus, &it.CustomerName, &it.Address, &it.Stage); err != nil {
			return nil, fmt.Errorf("order: scan ticket item: %w", err)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// CreateDispatchTicket 新建派单工单,返回自增 id。
func (s *PGStore) CreateDispatchTicket(ctx context.Context, t DispatchTicket) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO dispatch_tickets(ticket_no, order_id, worker_id, worker_name, group_id, group_name, region_id, region_name, legal_entity_id, legal_entity_name, status)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		t.TicketNo, t.OrderID, idOrNil(t.WorkerID), t.WorkerName, idOrNil(t.GroupID), t.GroupName,
		idOrNil(t.RegionID), t.RegionName, t.LegalEntityID, t.LegalEntityName, t.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("order: create dispatch ticket: %w", err)
	}
	return id, nil
}

// ListComplaints 列出全部报障工单。
func (s *PGStore) ListComplaints(ctx context.Context) ([]Complaint, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, ticket_no, customer_id, COALESCE(order_id, 0), legal_entity_id, legal_entity_name, type, status
		FROM complaints ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("order: list complaints: %w", err)
	}
	defer rows.Close()
	out := make([]Complaint, 0)
	for rows.Next() {
		var c Complaint
		if err := rows.Scan(&c.ID, &c.TicketNo, &c.CustomerID, &c.OrderID, &c.LegalEntityID, &c.LegalEntityName, &c.Type, &c.Status); err != nil {
			return nil, fmt.Errorf("order: scan complaint: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CreateComplaint 新建报障工单,返回自增 id。
func (s *PGStore) CreateComplaint(ctx context.Context, c Complaint) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO complaints(ticket_no, customer_id, order_id, legal_entity_id, legal_entity_name, type, status)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		c.TicketNo, c.CustomerID, idOrNil(c.OrderID), c.LegalEntityID, c.LegalEntityName, c.Type, c.Status).Scan(&id)
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

// GetDispatchTicketByNo 按工单号寻址(扫码闭环入口:ticketNo → orderID)。
func (s *PGStore) GetDispatchTicketByNo(ctx context.Context, ticketNo string) (*DispatchTicket, error) {
	var t DispatchTicket
	err := s.db.QueryRow(ctx, `
		SELECT id, ticket_no, order_id, COALESCE(worker_id, 0), COALESCE(worker_name, ''),
		       COALESCE(group_id, 0), COALESCE(group_name, ''), COALESCE(region_id, 0), COALESCE(region_name, ''),
		       legal_entity_id, legal_entity_name, status
		FROM dispatch_tickets WHERE ticket_no = $1`, ticketNo).
		Scan(&t.TicketID, &t.TicketNo, &t.OrderID, &t.WorkerID, &t.WorkerName,
			&t.GroupID, &t.GroupName, &t.RegionID, &t.RegionName, &t.LegalEntityID, &t.LegalEntityName, &t.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("order: get dispatch ticket: %w", err)
	}
	return &t, nil
}

// AssignDispatchTicket 指派师傅:回填 worker_id/worker_name;未命中返回 ErrOrderNotFound。
func (s *PGStore) AssignDispatchTicket(ctx context.Context, ticketNo string, workerID int64, workerName string) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE dispatch_tickets SET worker_id=$2, worker_name=$3 WHERE ticket_no=$1`,
		ticketNo, workerID, workerName)
	if err != nil {
		return fmt.Errorf("order: assign dispatch ticket: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}

// AssignPendingDispatchTicket 仅在待派且未指派时抢占工单，保证先到先得。
func (s *PGStore) AssignPendingDispatchTicket(ctx context.Context, ticketNo string, workerID int64, workerName string) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE dispatch_tickets SET worker_id=$2, worker_name=$3
		WHERE ticket_no=$1 AND status='PENDING' AND COALESCE(worker_id, 0)=0`,
		ticketNo, workerID, workerName)
	if err != nil {
		return fmt.Errorf("order: claim dispatch ticket: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}

// ListScanLogs 列出扫码绑定记录;orderID=0 返回全部,否则按订单过滤。
func (s *PGStore) ListScanLogs(ctx context.Context, orderID int64) ([]ScanLog, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, order_id, worker_id, worker_name, tag_id, result FROM scan_logs WHERE ($1 = 0 OR order_id = $1) ORDER BY id`, orderID)
	if err != nil {
		return nil, fmt.Errorf("order: list scan logs: %w", err)
	}
	defer rows.Close()
	out := make([]ScanLog, 0)
	for rows.Next() {
		var l ScanLog
		if err := rows.Scan(&l.ID, &l.OrderID, &l.WorkerID, &l.WorkerName, &l.TagID, &l.Result); err != nil {
			return nil, fmt.Errorf("order: scan scan log: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// AppendScanLog 追加扫码绑定记录,返回自增 id。
func (s *PGStore) AppendScanLog(ctx context.Context, l ScanLog) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO scan_logs(order_id, worker_id, worker_name, tag_id, result)
		VALUES($1,$2,$3,$4,$5) RETURNING id`,
		l.OrderID, l.WorkerID, l.WorkerName, l.TagID, l.Result).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("order: append scan log: %w", err)
	}
	return id, nil
}
