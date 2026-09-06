package order

// 下单(环节1)实现:从 pg.go 拆出(守 300 行红线);含下单资格预检(TMF POQ)。
// 关联完整性校验/直营风控/归属派生在 submitRegular;幂等键 RequestID 见 findByRequestID。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ymm-001/boss/internal/pkg/clock"
)

// Submit 下单(环节1):校验渠道/客户/产品/地址关联完整性后建单,status=PENDING、stage=1,写环节日志。
// 幂等(000116):req.RequestID 非空时,同客户同键重放返回已有订单,不重复发号建单。
func (s *PGStore) Submit(ctx context.Context, req SubmitReq) (*Order, error) {
	if req.PartnerOrder {
		return s.submitPartnerAtomic(ctx, req)
	}
	return s.submitRegular(ctx, req)
}

func (s *PGStore) submitRegular(ctx context.Context, req SubmitReq) (*Order, error) {
	if req.ChannelID == 0 {
		return nil, errors.New("order: channel_id required")
	}
	if req.RequestID != "" {
		if o, err := s.findByRequestID(ctx, req.CustomerID, req.RequestID); err != nil || o != nil {
			return o, err
		}
	}
	ok, err := s.cust.Exists(ctx, req.CustomerID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("order: customer %d not found", req.CustomerID)
	}
	// 关联完整性(orders 表无外键,这里拒掉孤儿引用):
	// 地址必须存在(否则归属推导会误走平台兜底放行);产品必须 PUBLISHED;渠道必须 ACTIVE。
	addrOK, err := s.exists(ctx, "addresses", req.AddressID, "")
	if err != nil {
		return nil, err
	}
	if !addrOK {
		return nil, fmt.Errorf("order: address %d: %w", req.AddressID, ErrAddressNotFound)
	}
	offerOK, err := s.exists(ctx, "product_offers", req.OfferID, ` AND status = 'PUBLISHED'`)
	if err != nil {
		return nil, err
	}
	if !offerOK {
		return nil, fmt.Errorf("order: offer %d: %w", req.OfferID, ErrOfferNotOrderable)
	}
	chOK, err := s.exists(ctx, "channels", req.ChannelID, ` AND status = 'ACTIVE'`)
	if err != nil {
		return nil, err
	}
	if !chOK {
		return nil, fmt.Errorf("order: channel %d: %w", req.ChannelID, ErrChannelNotActive)
	}
	// 直营风控(放量 FMS 前哨):渠道路径已有独立风控,此处只拦直营。
	if err := s.checkDirectRisk(ctx, req); err != nil {
		return nil, err
	}
	own, err := s.resolveOwnership(ctx, req.AddressID)
	if err != nil {
		return nil, err
	}
	if err := checkOwnershipConflict(req, own); err != nil {
		return nil, err
	}
	// 下单资格预检(TMF Product Offering Qualification,adopted note
	// 2026-09-01-provision-correctness-followup 裁定二):套餐须可解析下发模板,
	// 不可开通 = 不可售——把环节7 的必然失败提前到下单时拒单,防"钱付了卡环节7"。
	// tpl 未接线(内存桩/存量测试)时跳过。
	if s.tpl != nil {
		if _, err := s.tpl.FindTemplateForOffer(ctx, req.OfferID, own.LegalEntityID); err != nil {
			return nil, fmt.Errorf("order: submit qualification offer=%d: %w", req.OfferID, err)
		}
	}
	// 覆盖门控(P2,路线图 T12;odnGate 未注入=灰度关,注入后地址未 SERVED 即拒单)。
	if s.odnGate != nil {
		if err := s.odnGate.CheckOrderCoverage(ctx, req.AddressID); err != nil {
			return nil, fmt.Errorf("order: submit coverage gate address=%d: %w", req.AddressID, err)
		}
	}

	// 订单号由数据库序列发号(migrations/000031):跨进程/重启不重复。
	// 日期段按业务时区切日(会话时区 UTC,裸 now() 在马尼拉 08:00 前会算前一天)。
	var orderNo string
	if err := s.db.QueryRow(ctx,
		`SELECT 'ORD-' || to_char(now() AT TIME ZONE $1, 'YYYYMMDD') || '-' || lpad(nextval('order_no_seq')::text, 6, '0')`,
		clock.Location().String(),
	).Scan(&orderNo); err != nil {
		return nil, fmt.Errorf("order: next order_no: %w", err)
	}
	if req.BillingMode != BillingModePrepaid && req.BillingMode != BillingModePostpaid {
		req.BillingMode = BillingModePostpaid
	}
	if req.BuyMonths < 0 || req.BuyMonths > 60 {
		return nil, fmt.Errorf("order: buyMonths %d: %w", req.BuyMonths, ErrInvalidInput)
	}
	o := &Order{
		OrderNo:       orderNo,
		CustomerID:    req.CustomerID,
		OfferID:       req.OfferID,
		AddressID:     req.AddressID,
		Stage:         1,
		Status:        "PENDING",
		ChannelID:     req.ChannelID,
		LegalEntityID: own.LegalEntityID,
		PartnerEntity: req.PartnerEntity,
		RegionPath:    own.RegionPath,
		BillingMode:   req.BillingMode,
		BuyMonths:     req.BuyMonths,
	}
	err = s.db.QueryRow(ctx, `
		INSERT INTO orders(order_no, customer_id, offer_id, address_id, stage, status, channel_id, legal_entity_id, partner_entity_id, region_path, billing_mode, buy_months, request_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,0),$10,$11,$12,NULLIF($13,'')) RETURNING id`,
		o.OrderNo, o.CustomerID, o.OfferID, o.AddressID, o.Stage, o.Status, o.ChannelID, o.LegalEntityID, o.PartnerEntity, o.RegionPath, o.BillingMode, o.BuyMonths, req.RequestID).Scan(&o.ID)
	if err != nil {
		// 并发重放:唯一索引 uq_orders_customer_request 撞号 → 回读已有订单。
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && req.RequestID != "" {
			if exist, ferr := s.findByRequestID(ctx, req.CustomerID, req.RequestID); ferr == nil && exist != nil {
				return exist, nil
			}
		}
		return nil, fmt.Errorf("order: submit insert: %w", err)
	}
	// 下单即完成环节1(terms.md §1):submit 成功落 DONE + finished_at=now()(≈ created_at),
	// 与 advance 原语同口径——推进成功才写完成时间,时间轴环节1不再永久「未完成」。
	if err := s.appendStage(ctx, o.ID, 1, "DONE"); err != nil {
		return nil, err
	}
	return o, nil
}
