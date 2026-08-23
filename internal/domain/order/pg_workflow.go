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
	case "INSTALLING": // 回退 reopening:订单离开终态时工单随动 DOING。
		mapped = "DOING"
	case "CANCELLED": // 订单状态机枚举双 L,工单表枚举单 LCANCELED:取消订单此前漏同步。
		mapped = "CANCELED"
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

// ApplyTag 环节5 标签预绑定:读已预占端口(环节3) + 落四码关联(UNLINKED) + 写分光器端口。
// 前置:环节3 reserve 已预占端口(ports.order_id 已挂);未预占时兜底调 ReserveFirstAvailable。
// 依赖:PortReserver(兜底) + QuadLinkPrebinder(落四码)。
func (s *PGStore) ApplyTag(ctx context.Context, orderID int64) error {
	// 读订单上下文。
	var addressID, customerID, legalEntityID int64
	err := s.db.QueryRow(ctx,
		`SELECT address_id, customer_id, legal_entity_id
		 FROM orders WHERE id = $1`, orderID).Scan(&addressID, &customerID, &legalEntityID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOrderNotFound
		}
		return fmt.Errorf("order: applyTag select: %w", err)
	}

	// 端口:环节3 已预占则读 ports.order_id;未预占时兜底 ReserveFirstAvailable。
	var portID int64
	err = s.db.QueryRow(ctx,
		`SELECT id FROM ports WHERE order_id = $1 AND status = 'RESERVED'`, orderID).Scan(&portID)
	if errors.Is(err, pgx.ErrNoRows) {
		if s.reserve == nil {
			return errors.New("order: port reserver not wired")
		}
		portID, err = s.reserve.ReserveFirstAvailable(ctx, addressID, orderID)
		if err != nil {
			return fmt.Errorf("order: applyTag reserve port: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("order: applyTag find port: %w", err)
	}

	// 写入 splitter_port:端口码 + 所属设备码。
	var portCode, resCode string
	_ = s.db.QueryRow(ctx,
		`SELECT p.port_code, COALESCE(r.code, '')
		 FROM ports p LEFT JOIN resources r ON p.resource_id = r.id
		 WHERE p.id = $1`, portID).Scan(&portCode, &resCode)
	if portCode != "" {
		sp := portCode
		if resCode != "" {
			sp = resCode + " · " + portCode
		}
		_, _ = s.db.Exec(ctx,
			`UPDATE dispatch_tickets SET splitter_port = $2 WHERE order_id = $1`, orderID, sp)
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

// UpdateMap 环节12 更新 GIS;订单终态 DONE 后端口转在用(terms.md §4:IDLE→RESERVED→USED)。
// 渠道订单到达终态时自动计提佣金(尽力而为,不影响环节推进)。
func (s *PGStore) UpdateMap(ctx context.Context, orderID int64) error {
	if err := s.advance(ctx, orderID, "updateMap"); err != nil {
		return err
	}
	if _, err := s.db.Exec(ctx,
		`UPDATE ports SET status = 'USED' WHERE order_id = $1 AND status = 'RESERVED'`, orderID,
	); err != nil {
		return fmt.Errorf("order: mark port used: %w", err)
	}
	s.accruePartnerCommission(ctx, orderID)
	return nil
}

// accruePartnerCommission 渠道订单终态自动计提佣金;尽力而为,失败只记日志不阻断。
func (s *PGStore) accruePartnerCommission(ctx context.Context, orderID int64) {
	if s.commission == nil {
		return
	}
	var entityID int64
	var amount float64
	err := s.db.QueryRow(ctx, `
SELECT o.legal_entity_id, po.monthly_fee * GREATEST(o.buy_months, 1)
FROM orders o JOIN channels ch ON ch.id=o.channel_id
JOIN product_offers po ON po.id=o.offer_id
WHERE o.id=$1 AND ch.code='AGENT'`, orderID).Scan(&entityID, &amount)
	if err != nil {
		return
	}
	if entityID == 0 {
		return
	}
	_, _ = s.commission.AccrueCommission(ctx, orderID, entityID, amount, partnerDefaultRate)
}

// Cancel 取消订单:任一未完成状态可取消(status→CANCELLED),不动环节序号;并回收预占端口。
func (s *PGStore) Cancel(ctx context.Context, orderID int64) error {
	if err := s.transitionStatus(ctx, orderID, "cancel"); err != nil {
		return err
	}
	// 取消必回收本订单预占端口(RESERVED→IDLE):否则端口死占,地址端口逐步耗尽。
	// 无预占端口(未到环节3/5)视为正常不报错;与 resource.ReleasePortByOrder 同谓词语义。
	if _, err := s.db.Exec(ctx,
		`UPDATE ports SET status = 'IDLE', order_id = NULL WHERE order_id = $1 AND status = 'RESERVED'`, orderID,
	); err != nil {
		return fmt.Errorf("order: cancel release ports: %w", err)
	}
	return nil
}

// Release 端口释放:RESERVED→PENDING(超时/取消的预占回滚),不动环节序号。
func (s *PGStore) Release(ctx context.Context, orderID int64) error {
	return s.transitionStatus(ctx, orderID, "release")
}

// RollbackStage 回退至上一完成环节(worker 端 rollback,留痕走审计):
// 删除最新环节日志 + stage 前移一位 + status 与回退后环节对齐(级联逆向:
// DONE←undone←INSTALLING←undispatch←RESERVED←release←PENDING)+ 工单随动。
// 重新推进时 advance 会重写环节日志,故删除而非标废(result 枚举无 ROLLED_BACK)。
func (s *PGStore) RollbackStage(ctx context.Context, orderID int64) error {
	var stage int8
	var status string
	err := s.db.QueryRow(ctx, `SELECT stage, status FROM orders WHERE id = $1`, orderID).Scan(&stage, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrOrderNotFound
	}
	if err != nil {
		return fmt.Errorf("order: rollback select: %w", err)
	}
	if stage < 2 {
		return ErrIllegalTransition
	}
	nextStatus, err := alignStatusToStage(status, stage-1)
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(ctx,
		`DELETE FROM order_stages WHERE order_id = $1 AND stage = $2`, orderID, stage); err != nil {
		return fmt.Errorf("order: rollback delete log: %w", err)
	}
	if _, err := s.db.Exec(ctx,
		`UPDATE orders SET stage = $2, status = $3 WHERE id = $1`, orderID, stage-1, nextStatus); err != nil {
		return fmt.Errorf("order: rollback update: %w", err)
	}
	s.syncDispatchTicket(ctx, orderID, nextStatus)
	return nil
}

// alignStatusToStage status 逆向对齐到目标环节:status 由环节3(reserve)/8(install)/
// 11(done)产生,回退到产生环节之前则沿逆向事件级联迁移。
func alignStatusToStage(status string, target int8) (string, error) {
	next := status
	for _, r := range []struct {
		from string
		rev  string
		born int8 // 产生该 status 的环节
	}{
		{"DONE", "undone", 11},
		{"INSTALLING", "undispatch", 8},
		{"RESERVED", "release", 3},
	} {
		if next == r.from && target < r.born {
			ns, err := transition(next, r.rev)
			if err != nil {
				return "", err
			}
			next = ns
		}
	}
	return next, nil
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
