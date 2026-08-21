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

