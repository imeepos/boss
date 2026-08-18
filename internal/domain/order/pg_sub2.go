package order

import (
	"context"
	"fmt"
)

// ListDismantles 列出全部拆机单。
func (s *PGStore) ListDismantles(ctx context.Context) ([]Dismantle, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, dismantle_no, order_id, legal_entity_id, legal_entity_name, asset_id, port_id, status
		 FROM dismantles ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("order: list dismantles: %w", err)
	}
	defer rows.Close()
	out := make([]Dismantle, 0)
	for rows.Next() {
		var d Dismantle
		if err := rows.Scan(&d.ID, &d.DismantleNo, &d.OrderID, &d.LegalEntityID, &d.LegalEntityName, &d.AssetID, &d.PortID, &d.Status); err != nil {
			return nil, fmt.Errorf("order: scan dismantle: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// CreateDismantle 新建拆机单,返回自增 id。
func (s *PGStore) CreateDismantle(ctx context.Context, d Dismantle) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO dismantles(dismantle_no, order_id, legal_entity_id, legal_entity_name, asset_id, port_id, status)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		d.DismantleNo, d.OrderID, d.LegalEntityID, d.LegalEntityName, d.AssetID, d.PortID, d.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("order: create dismantle: %w", err)
	}
	return id, nil
}

// ListActivationCallbacks 列出全部激活回调。
func (s *PGStore) ListActivationCallbacks(ctx context.Context) ([]ActivationCallback, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, order_id, result, retries FROM activation_callbacks ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("order: list activation callbacks: %w", err)
	}
	defer rows.Close()
	out := make([]ActivationCallback, 0)
	for rows.Next() {
		var c ActivationCallback
		if err := rows.Scan(&c.ID, &c.OrderID, &c.Result, &c.Retries); err != nil {
			return nil, fmt.Errorf("order: scan activation callback: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// AppendActivationCallback 追加激活回调,返回自增 id。
func (s *PGStore) AppendActivationCallback(ctx context.Context, c ActivationCallback) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO activation_callbacks(order_id, result, retries) VALUES($1,$2,$3) RETURNING id`,
		c.OrderID, c.Result, c.Retries).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("order: append activation callback: %w", err)
	}
	return id, nil
}

// RetryActivationCallback 回调重试:retries+1;未命中返回 ErrNotFound。
func (s *PGStore) RetryActivationCallback(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx, `UPDATE activation_callbacks SET retries = retries + 1 WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("order: retry activation callback: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}

// ListDispatchTransfers 列出改派台账;ticketID=0 返回全部,否则按工单过滤。
func (s *PGStore) ListDispatchTransfers(ctx context.Context, ticketID int64) ([]DispatchTransfer, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, ticket_id, COALESCE(from_worker_id, 0), COALESCE(from_worker_name, ''),
		       COALESCE(to_worker_id, 0), COALESCE(to_worker_name, ''),
		       COALESCE(reason, ''), COALESCE(operator_account_id, 0), transferred_at
		FROM dispatch_transfers WHERE ($1 = 0 OR ticket_id = $1) ORDER BY transferred_at, id`, ticketID)
	if err != nil {
		return nil, fmt.Errorf("order: list dispatch transfers: %w", err)
	}
	defer rows.Close()
	out := make([]DispatchTransfer, 0)
	for rows.Next() {
		var t DispatchTransfer
		if err := rows.Scan(&t.ID, &t.TicketID, &t.FromWorkerID, &t.FromWorkerName, &t.ToWorkerID, &t.ToWorkerName,
			&t.Reason, &t.OperatorAccountID, &t.TransferredAt); err != nil {
			return nil, fmt.Errorf("order: scan dispatch transfer: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// AppendDispatchTransfer 追加改派记录,返回自增 id。
func (s *PGStore) AppendDispatchTransfer(ctx context.Context, t DispatchTransfer) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO dispatch_transfers(ticket_id, from_worker_id, from_worker_name, to_worker_id, to_worker_name, reason, operator_account_id, transferred_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
		t.TicketID, idOrNil(t.FromWorkerID), t.FromWorkerName, idOrNil(t.ToWorkerID), t.ToWorkerName,
		t.Reason, idOrNil(t.OperatorAccountID), t.TransferredAt).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("order: append dispatch transfer: %w", err)
	}
	return id, nil
}
