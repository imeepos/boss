package order

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

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

// CreateUserProfile 环节6 创建认证账号:幂等创建 LO 账号,已存在则直接推进。
func (s *PGStore) CreateUserProfile(ctx context.Context, orderID int64) error {
	if s.prof == nil {
		return errors.New("order: user profile creator not wired")
	}
	var customerID, offerID, legalEntityID int64
	var legalEntityName, regionPath, billingMode string
	err := s.db.QueryRow(ctx,
		`SELECT customer_id, offer_id, legal_entity_id, COALESCE(le.name, ''), COALESCE(region_path, ''), COALESCE(billing_mode, 'POSTPAID')
		 FROM orders o
		 LEFT JOIN legal_entities le ON le.id = o.legal_entity_id
		 WHERE o.id = $1`, orderID).Scan(&customerID, &offerID, &legalEntityID,
		&legalEntityName, &regionPath, &billingMode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOrderNotFound
		}
		return fmt.Errorf("order: createUserProfile select: %w", err)
	}
	// LOID 由 customer_code 派生(如 customer_code 为空则 fallback customer_id)
	var customerCode string
	_ = s.db.QueryRow(ctx, `SELECT COALESCE(customer_code, '') FROM customers WHERE id = $1`, customerID).Scan(&customerCode)
	loid := "LOID-" + customerCode
	if customerCode == "" {
		loid = fmt.Sprintf("LOID-C%d", customerID)
	}
	// 幂等:已存在则把生效套餐对齐到本订单(改套餐场景,TMF change order 语义:
	// 服务配置必须随变更单刷新,否则环节7 按旧套餐下发模板、RADIUS 限速也停在旧档),
	// 再推进;新建分支见下方 CreateLoAccount(offer 取自订单)。
	existing, err := s.prof.GetLoAccountByCustomer(ctx, customerID)
	if err == nil && existing != nil {
		if existing.OfferID != offerID {
			aligned, err := s.prof.AlignLoAccount(ctx, customerID, offerID, billingMode)
			if err != nil {
				return fmt.Errorf("order: createUserProfile align lo: %w", err)
			}
			if aligned {
				log.Printf("[order] LO OFFER REALIGN: customer=%d offer %d -> %d (billing=%s)",
					customerID, existing.OfferID, offerID, billingMode)
			}
		}
		return s.advance(ctx, orderID, "createUserProfile")
	}
	// 查询地址对应的 region_id
	var regionID int64
	var regionName string
	_ = s.db.QueryRow(ctx,
		`SELECT COALESCE(a.region_id, 0), COALESCE(r.name, '')
		 FROM addresses a LEFT JOIN regions r ON r.id = a.region_id
		 WHERE a.id = (SELECT address_id FROM orders WHERE id = $1)`, orderID).Scan(&regionID, &regionName)
	if _, err := s.prof.CreateLoAccount(ctx, LoidReq{
		Loid: loid, CustomerID: customerID,
		LegalEntityID: legalEntityID, LegalEntityName: legalEntityName,
		RegionID: regionID, RegionName: regionName, RegionPath: regionPath,
		OfferID: offerID, Status: "ACTIVE", BillingMode: billingMode,
	}); err != nil {
		return fmt.Errorf("order: createUserProfile create lo: %w", err)
	}
	return s.advance(ctx, orderID, "createUserProfile")
}

// ScanBind 环节9 扫码绑定。
func (s *PGStore) ScanBind(ctx context.Context, orderID int64) error {
	return s.advance(ctx, orderID, "scanBind")
}

// ActivateUser 环节10 激活。
func (s *PGStore) ActivateUser(ctx context.Context, orderID int64) error {
	return s.advance(ctx, orderID, "activateUser")
}

// NotifyActivation 环节11 激活回调:本地确认入网凭证(LO 账号)后推进并落回调行。
// 落账闭环(000154):SUCCESS/FAILED 均写 activation_callbacks(幂等 upsert),
// 失败(凭证缺失)不推进订单——留 INSTALLING 可重试,补偿台账扫 FAILED 可见。
// 确认语义:LO 账号存在即视为凭证已发(环节6 建档),杜绝伪造成功。
func (s *PGStore) NotifyActivation(ctx context.Context, orderID int64) error {
	if err := s.confirmLoAccount(ctx, orderID); err != nil {
		_, _ = s.AppendActivationCallback(ctx, ActivationCallback{OrderID: orderID, Result: "FAILED"})
		return fmt.Errorf("order: notifyActivation confirm: %w", err)
	}
	if err := s.advance(ctx, orderID, "notifyActivation"); err != nil {
		return err
	}
	_, err := s.AppendActivationCallback(ctx, ActivationCallback{OrderID: orderID, Result: "SUCCESS"})
	return err
}

// confirmLoAccount 激活确认:订单客户须已有 LO 账号(入网凭证环节6 建档)。
func (s *PGStore) confirmLoAccount(ctx context.Context, orderID int64) error {
	if s.prof == nil {
		return nil // 未接线(测试桩)视为可确认
	}
	var customerID int64
	err := s.db.QueryRow(ctx, `SELECT customer_id FROM orders WHERE id = $1`, orderID).Scan(&customerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOrderNotFound
		}
		return fmt.Errorf("order: notifyActivation select: %w", err)
	}
	lo, err := s.prof.GetLoAccountByCustomer(ctx, customerID)
	if err != nil || lo == nil {
		return fmt.Errorf("lo account missing for customer %d", customerID)
	}
	return nil
}

// UpdateMap 环节12 更新 GIS;订单终态 DONE 后端口转在用(terms.md §4:IDLE→RESERVED→USED)。
// 渠道订单到达终态时自动计提佣金(尽力而为,不影响环节推进)。
func (s *PGStore) UpdateMap(ctx context.Context, orderID int64) error {
	if err := s.advance(ctx, orderID, "updateMap"); err != nil {
		// 环节已完成但端口落库失败时允许重试，避免 DONE 订单永久残留 RESERVED。
		if !errors.Is(err, ErrIllegalTransition) || !s.isDoneAtMapStage(ctx, orderID) {
			return err
		}
	}
	if _, err := s.db.Exec(ctx,
		`UPDATE ports SET status = 'USED' WHERE order_id = $1 AND status = 'RESERVED'`, orderID,
	); err != nil {
		return fmt.Errorf("order: mark port used: %w", err)
	}
	s.accruePartnerCommission(ctx, orderID)
	return nil
}

func (s *PGStore) isDoneAtMapStage(ctx context.Context, orderID int64) bool {
	var stage int8
	var status string
	if err := s.db.QueryRow(ctx, `SELECT stage, status FROM orders WHERE id = $1`, orderID).Scan(&stage, &status); err != nil {
		return false
	}
	return stage == 12 && status == "DONE"
}

// accruePartnerCommission 渠道订单终态自动计提佣金;尽力而为,失败只记日志不阻断。
func (s *PGStore) accruePartnerCommission(ctx context.Context, orderID int64) {
	if s.commission == nil {
		return
	}
	var entityID int64
	var amount float64
	err := s.db.QueryRow(ctx, `
SELECT COALESCE(o.partner_entity_id, o.legal_entity_id), po.monthly_fee * GREATEST(o.buy_months, 1)
FROM orders o JOIN channels ch ON ch.id=o.channel_id
JOIN product_offers po ON po.id=o.offer_id
WHERE o.id=$1 AND ch.code='AGENT'`, orderID).Scan(&entityID, &amount)
	if err != nil {
		return
	}
	if entityID == 0 {
		return
	}
	_, _ = s.commission.AccrueCommission(ctx, orderID, entityID, amount, partnerRate(ctx, s.params))
}
