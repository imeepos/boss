package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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
// 校验 order_id 存在性,防止孤儿拆机单。
func (s *PGStore) CreateDismantle(ctx context.Context, d Dismantle) (int64, error) {
	// 关联完整性校验
	if d.OrderID > 0 {
		ok, err := s.exists(ctx, "orders", d.OrderID, "")
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("order: order %d: %w", d.OrderID, ErrForeignKeyViolation)
		}
	}

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
// 幂等(000154):同订单唯一行(uq_activation_callbacks_order),重复确认/重试
// 只更新 result/retries/tried_at,不新增行——保证环节11 可反复重放不产生重复回执。
// tried_at 随每次尝试刷新(000182,任务A-d:lastTry 可见)。
func (s *PGStore) AppendActivationCallback(ctx context.Context, c ActivationCallback) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO activation_callbacks(order_id, result, retries) VALUES($1,$2,$3)
		ON CONFLICT (order_id) DO UPDATE SET
			result = EXCLUDED.result, retries = EXCLUDED.retries, tried_at = now()
		RETURNING id`,
		c.OrderID, c.Result, c.Retries).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("order: append activation callback: %w", err)
	}
	return id, nil
}

// LatestActivationCallback 取订单最近一次激活尝试(000182,任务A-d:lastTry)。
// 无记录返回 (nil, nil),调用方以空串口径呈现。
func (s *PGStore) LatestActivationCallback(ctx context.Context, orderID int64) (*ActivationCallback, error) {
	var c ActivationCallback
	var tried pgtype.Timestamptz
	err := s.db.QueryRow(ctx, `
		SELECT id, order_id, result, retries, tried_at
		FROM activation_callbacks WHERE order_id = $1
		ORDER BY id DESC LIMIT 1`, orderID).
		Scan(&c.ID, &c.OrderID, &c.Result, &c.Retries, &tried)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("order: latest activation callback: %w", err)
	}
	if tried.Valid {
		t := tried.Time
		c.TriedAt = &t
	}
	return &c, nil
}

// RetryActivationCallback 回调重试:重放环节11 确认(而非仅计数)。
//   - 回调行不存在:ErrOrderNotFound。
//   - 订单已 DONE(环节11 曾成功):回调已是成功历史,仅计数(幂等确认)。
//   - 订单未 DONE(前次确认失败留 FAILED):重跑 NotifyActivation,
//     凭证已补则落 SUCCESS 并推进订单;仍缺则保持 FAILED 可再试(补偿台账可见)。
func (s *PGStore) RetryActivationCallback(ctx context.Context, id int64) error {
	var cb ActivationCallback
	err := s.db.QueryRow(ctx,
		`SELECT id, order_id, result, retries FROM activation_callbacks WHERE id = $1`, id).
		Scan(&cb.ID, &cb.OrderID, &cb.Result, &cb.Retries)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrOrderNotFound
	}
	if err != nil {
		return fmt.Errorf("order: retry activation callback select: %w", err)
	}
	// 订单已 DONE:激活曾成功,回调行恢复 SUCCESS(幂等重放)并计重试次数。
	var status string
	_ = s.db.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1`, cb.OrderID).Scan(&status)
	if status == "DONE" {
		return s.restoreActivationSuccess(ctx, id)
	}
	// 重放确认:重新执行环节11(幂等 upsert 落账),失败重试计数。
	if err := s.NotifyActivation(ctx, cb.OrderID); err != nil {
		_ = s.bumpActivationRetry(ctx, id)
		return err
	}
	return nil
}

func (s *PGStore) bumpActivationRetry(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx, `UPDATE activation_callbacks SET retries = retries + 1 WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("order: retry activation callback: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}

// restoreActivationSuccess 订单已 DONE 时回调行恢复 SUCCESS(幂等重放),retries+1。
func (s *PGStore) restoreActivationSuccess(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE activation_callbacks SET result = 'SUCCESS', retries = retries + 1 WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("order: restore activation callback: %w", err)
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
