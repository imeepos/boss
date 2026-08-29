package order

// 派单工单寻址与改派/领取/改期(dispatch_tickets)PG 存取(从 pg_sub.go 拆出)。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// GetDispatchTicketByNo 按工单号寻址(扫码闭环入口:ticketNo → orderID)。
func (s *PGStore) GetDispatchTicketByNo(ctx context.Context, ticketNo string) (*DispatchTicket, error) {
	var t DispatchTicket
	err := s.db.QueryRow(ctx, `
		SELECT id, ticket_no, order_id, COALESCE(worker_id, 0), COALESCE(worker_name, ''),
		       COALESCE(group_id, 0), COALESCE(group_name, ''), COALESCE(region_id, 0), COALESCE(region_name, ''),
		       legal_entity_id, legal_entity_name, status,
		       arrived_at, arrive_lat, arrive_lng
		FROM dispatch_tickets WHERE ticket_no = $1`, ticketNo).
		Scan(&t.TicketID, &t.TicketNo, &t.OrderID, &t.WorkerID, &t.WorkerName,
			&t.GroupID, &t.GroupName, &t.RegionID, &t.RegionName, &t.LegalEntityID, &t.LegalEntityName, &t.Status,
			&t.ArrivedAt, &t.ArriveLat, &t.ArriveLng)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("order: get dispatch ticket: %w", err)
	}
	return &t, nil
}

// GetDispatchTicketByOrder 按订单寻址工单(order_id 唯一约束保证至多一行);
// 无工单(环节8前)返回 nil,nil——调用方按"未派单"渲染,非错误态。
func (s *PGStore) GetDispatchTicketByOrder(ctx context.Context, orderID int64) (*DispatchTicket, error) {
	var t DispatchTicket
	err := s.db.QueryRow(ctx, `
		SELECT id, ticket_no, order_id, COALESCE(worker_id, 0), COALESCE(worker_name, ''),
		       COALESCE(group_id, 0), COALESCE(group_name, ''), COALESCE(region_id, 0), COALESCE(region_name, ''),
		       legal_entity_id, legal_entity_name, status,
		       arrived_at, arrive_lat, arrive_lng
		FROM dispatch_tickets WHERE order_id = $1`, orderID).
		Scan(&t.TicketID, &t.TicketNo, &t.OrderID, &t.WorkerID, &t.WorkerName,
			&t.GroupID, &t.GroupName, &t.RegionID, &t.RegionName, &t.LegalEntityID, &t.LegalEntityName, &t.Status,
			&t.ArrivedAt, &t.ArriveLat, &t.ArriveLng)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("order: get dispatch ticket by order: %w", err)
	}
	return &t, nil
}

// AssignDispatchTicket 指派师傅:回填 worker_id/worker_name + 可选 schedule_slot/pre_bind_tag。
// opt 非空时追加写入(空串不覆盖已有值);未命中返回 ErrOrderNotFound。
func (s *PGStore) AssignDispatchTicket(ctx context.Context, ticketNo string, workerID int64, workerName string, opt ...AssignOpt) error {
	var o AssignOpt
	if len(opt) > 0 {
		o = opt[0]
	}
	if o.ScheduleSlot != "" || o.PreBindTag != "" {
		tag, err := s.db.Exec(ctx, `
			UPDATE dispatch_tickets SET worker_id=$2, worker_name=$3,
				schedule_slot = CASE WHEN $4 = '' THEN schedule_slot ELSE $4 END,
				pre_bind_tag  = CASE WHEN $5 = '' THEN pre_bind_tag  ELSE $5 END
			WHERE ticket_no=$1`,
			ticketNo, workerID, workerName, o.ScheduleSlot, o.PreBindTag)
		if err != nil {
			return fmt.Errorf("order: assign dispatch ticket: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrOrderNotFound
		}
		return nil
	}
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

// ClaimDispatchTicket 师傅领取:回填师傅并 PENDING→DOING;非待派返回 ErrOrderNotFound。
func (s *PGStore) ClaimDispatchTicket(ctx context.Context, ticketNo string, workerID int64, workerName string) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE dispatch_tickets SET worker_id=$2, worker_name=$3, status='DOING'
		WHERE ticket_no=$1 AND status='PENDING'`,
		ticketNo, workerID, workerName)
	if err != nil {
		return fmt.Errorf("order: claim dispatch ticket: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}

// AssignPendingDispatchTicket 抢单:待派且未指派时抢占,同时 PENDING→DOING(先到先得)。
func (s *PGStore) AssignPendingDispatchTicket(ctx context.Context, ticketNo string, workerID int64, workerName string, opt ...AssignOpt) error {
	var o AssignOpt
	if len(opt) > 0 {
		o = opt[0]
	}
	if o.ScheduleSlot != "" || o.PreBindTag != "" {
		tag, err := s.db.Exec(ctx, `
			UPDATE dispatch_tickets SET worker_id=$2, worker_name=$3, status='DOING',
				schedule_slot = CASE WHEN $4 = '' THEN schedule_slot ELSE $4 END,
				pre_bind_tag  = CASE WHEN $5 = '' THEN pre_bind_tag  ELSE $5 END
			WHERE ticket_no=$1 AND status='PENDING' AND COALESCE(worker_id, 0)=0`,
			ticketNo, workerID, workerName, o.ScheduleSlot, o.PreBindTag)
		if err != nil {
			return fmt.Errorf("order: assign pending ticket: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrOrderNotFound
		}
		return nil
	}
	tag, err := s.db.Exec(ctx, `
		UPDATE dispatch_tickets SET worker_id=$2, worker_name=$3, status='DOING'
		WHERE ticket_no=$1 AND status='PENDING' AND COALESCE(worker_id, 0)=0`,
		ticketNo, workerID, workerName)
	if err != nil {
		return fmt.Errorf("order: assign pending ticket: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}

// UpdateScheduleSlot 改约:更新预约时间段 + 重置报障 SLA 截止时间。
func (s *PGStore) UpdateScheduleSlot(ctx context.Context, ticketNo string, scheduleSlot string) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE dispatch_tickets SET schedule_slot=$2 WHERE ticket_no=$1`, ticketNo, scheduleSlot)
	if err != nil {
		return fmt.Errorf("order: update schedule slot: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}

// DispatchOrder 环节8 派单(status: RESERVED→INSTALLING)+ 生成待派工单入池。
// 幂等自愈:订单已到环节8 时不再推进状态机,仅确保工单存在(createTicketOnDispatch
// ON CONFLICT DO NOTHING)。闭环了「派单已推进但工单落库失败 → Automation 幂等续推
// 跳过 → 环节8 无工单」的孤儿类(audit 2026-08-25:330/331/332/333/350)。
func (s *PGStore) DispatchOrder(ctx context.Context, orderID int64) error {
	var stage int8
	err := s.db.QueryRow(ctx, `SELECT stage FROM orders WHERE id = $1`, orderID).Scan(&stage)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrOrderNotFound
	}
	if err != nil {
		return fmt.Errorf("order: dispatch select stage: %w", err)
	}
	if stage >= 8 {
		return s.createTicketOnDispatch(ctx, orderID)
	}
	if err := s.advance(ctx, orderID, "dispatchOrder"); err != nil {
		return err
	}
	return s.createTicketOnDispatch(ctx, orderID)
}

// createTicketOnDispatch 派单时落工单(worker 空=PENDING 入池);order_id 唯一约束 +
// ON CONFLICT DO NOTHING 保证 Automation 失败重试幂等。ticket_no 由 order_no 派生
// (ORD-→DT-)保证唯一可追溯;区域/班组快照留空,由指派时回填。
func (s *PGStore) createTicketOnDispatch(ctx context.Context, orderID int64) error {
	if _, err := s.db.Exec(ctx, `
		INSERT INTO dispatch_tickets(ticket_no, order_id, legal_entity_id, legal_entity_name, status)
		SELECT 'DT-' || substr(o.order_no, 5), o.id, o.legal_entity_id, COALESCE(le.name, ''), 'PENDING'
		FROM orders o
		LEFT JOIN legal_entities le ON le.id = o.legal_entity_id
		WHERE o.id = $1
		ON CONFLICT (order_id) DO NOTHING`,
		orderID,
	); err != nil {
		return fmt.Errorf("order: dispatch create ticket: %w", err)
	}
	return nil
}
