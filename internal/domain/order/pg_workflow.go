package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// advance 推进一个环节:顺序守卫(stage 必须等于上一环节)+ status 迁移(经 orderSM)+ 环节日志。
// 单事实源:所有环节推进都必须过此原语,禁止直接改 stage/status。
func (s *PGStore) advance(ctx context.Context, orderID int64, event string) error {
	step, ok := workflowByEvent[event]
	if !ok {
		return fmt.Errorf("order: unknown event %q", event)
	}
	var stage int8
	var status string
	err := s.db.QueryRow(ctx, `SELECT stage, status FROM orders WHERE id = $1`, orderID).Scan(&stage, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrOrderNotFound
	}
	if err != nil {
		return fmt.Errorf("order: advance select: %w", err)
	}
	if stage != step.stage-1 {
		return ErrIllegalTransition
	}
	nextStatus := status
	if step.statusEvent != "" {
		ns, err := transition(status, step.statusEvent)
		if err != nil {
			return err
		}
		nextStatus = ns
	}
	if _, err := s.db.Exec(ctx, `UPDATE orders SET stage = $2, status = $3 WHERE id = $1`, orderID, step.stage, nextStatus); err != nil {
		return fmt.Errorf("order: advance update: %w", err)
	}
	s.syncDispatchTicket(ctx, orderID, nextStatus)
	return s.appendStage(ctx, orderID, step.stage, "DONE")
}

// syncDispatchTicket 订单终态同步派单工单:订单 DONE/CANCELED 时工单随动,
// 避免订单已完成而工单仍停留 PENDING(师傅端出现"12/12 待领取")。
func (s *PGStore) syncDispatchTicket(ctx context.Context, orderID int64, orderStatus string) {
	mapped := ""
	switch orderStatus {
	case "DONE":
		mapped = "DONE"
	case "CANCELED":
		mapped = "CANCELED"
	}
	if mapped == "" {
		return
	}
	if _, err := s.db.Exec(ctx,
		`UPDATE dispatch_tickets SET status = $2 WHERE order_id = $1 AND status <> $2`, orderID, mapped); err != nil {
		// 同步失败不阻断主流程;工单视图另有订单终态兜底(read model)
		_ = err
	}
}

// 环节 4~12(terms.md §1)。各环节目前是「人工确认」推进;自动化在阶段7 经同一状态机升级。

// ChargeContract 环节4 合同收费(未收费不派单的硬约束由顺序守卫保证:dispatch 需 stage=7)。
func (s *PGStore) ChargeContract(ctx context.Context, orderID int64) error {
	return s.advance(ctx, orderID, "chargeContract")
}

// ApplyTag 环节5 标签预绑定:预占端口 + 落四码关联(UNLINKED,资产扫码时回填)。
// 前置:环节4 合同收费已推进(顺序守卫保证 stage=4)。
// 依赖:PortReserver.ReserveFirstAvailable(选端口) + QuadLinkPrebinder.CreateLink(落四码)。
func (s *PGStore) ApplyTag(ctx context.Context, orderID int64) error {
	// 读订单上下文。
	var addressID, customerID, legalEntityID int64
	var regionPath string
	err := s.db.QueryRow(ctx,
		`SELECT address_id, customer_id, legal_entity_id, COALESCE(region_path, '')
		 FROM orders WHERE id = $1`, orderID).Scan(&addressID, &customerID, &legalEntityID, &regionPath)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOrderNotFound
		}
		return fmt.Errorf("order: applyTag select: %w", err)
	}

	// 端口预占(环节3 未预占端口时由本环节兜底;已预占则 ReserveFirstAvailable 幂等)。
	if s.reserve == nil {
		return errors.New("order: port reserver not wired")
	}
	portID, err := s.reserve.ReserveFirstAvailable(ctx, addressID, orderID)
	if err != nil {
		return fmt.Errorf("order: applyTag reserve port: %w", err)
	}

	// 四码预绑定(UNLINKED;资产为空,扫码环节9回填)。
	if s.quad == nil {
		return errors.New("order: quad link prebinder not wired")
	}
	_, err = s.quad.CreateLink(ctx, QuadLinkBindReq{
		CustomerID: customerID, PortID: portID, AddressID: addressID,
		LegalEntityID: legalEntityID, Status: "UNLINKED",
	})
	if err != nil {
		return fmt.Errorf("order: applyTag create quad link: %w", err)
	}

	return s.advance(ctx, orderID, "applyTag")
}

// CreateUserProfile 环节6 创建认证账号。
func (s *PGStore) CreateUserProfile(ctx context.Context, orderID int64) error {
	return s.advance(ctx, orderID, "createUserProfile")
}

// PreConfigOLT 环节7 预下发配置。
func (s *PGStore) PreConfigOLT(ctx context.Context, orderID int64) error {
	return s.advance(ctx, orderID, "preConfigOLT")
}

// DispatchOrder 环节8 派单(status: RESERVED→INSTALLING)+ 生成待派工单入池。
func (s *PGStore) DispatchOrder(ctx context.Context, orderID int64) error {
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

// ScanBind 环节9 扫码绑定。
func (s *PGStore) ScanBind(ctx context.Context, orderID int64) error {
	return s.advance(ctx, orderID, "scanBind")
}

// ActivateUser 环节10 激活。
func (s *PGStore) ActivateUser(ctx context.Context, orderID int64) error {
	return s.advance(ctx, orderID, "activateUser")
}

// NotifyActivation 环节11 激活回调(status: INSTALLING→DONE)。
func (s *PGStore) NotifyActivation(ctx context.Context, orderID int64) error {
	return s.advance(ctx, orderID, "notifyActivation")
}

// UpdateMap 环节12 更新 GIS。
func (s *PGStore) UpdateMap(ctx context.Context, orderID int64) error {
	return s.advance(ctx, orderID, "updateMap")
}

// Cancel 取消订单:任一未完成状态可取消(status→CANCELLED),不动环节序号。
func (s *PGStore) Cancel(ctx context.Context, orderID int64) error {
	return s.transitionStatus(ctx, orderID, "cancel")
}

// Release 端口释放:RESERVED→PENDING(超时/取消的预占回滚),不动环节序号。
func (s *PGStore) Release(ctx context.Context, orderID int64) error {
	return s.transitionStatus(ctx, orderID, "release")
}

// transitionStatus 只做 status 迁移(不改 stage,不写环节日志)。
func (s *PGStore) transitionStatus(ctx context.Context, orderID int64, event string) error {
	var status string
	err := s.db.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1`, orderID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrOrderNotFound
	}
	if err != nil {
		return fmt.Errorf("order: status select: %w", err)
	}
	next, err := transition(status, event)
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(ctx, `UPDATE orders SET status = $2 WHERE id = $1`, orderID, next); err != nil {
		return fmt.Errorf("order: status update: %w", err)
	}
	s.syncDispatchTicket(ctx, orderID, next)
	return nil
}
