package order

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const partnerDailyCap int64 = 100

func (s *PGStore) submitPartnerAtomic(ctx context.Context, req SubmitReq) (*Order, error) {
	tdb, ok := s.db.(transactionalDB)
	if !ok {
		return nil, fmt.Errorf("order: partner submit requires transaction support")
	}
	tx, err := tdb.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("order: begin partner submit: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	if err = lockPartnerTenant(ctx, tx, req.LegalEntityID); err != nil {
		return nil, err
	}
	if err = checkPartnerOwnership(ctx, tx, req); err != nil {
		return nil, err
	}
	if err = checkPartnerLimits(ctx, tx, req); err != nil {
		return nil, err
	}
	// C 案: 渠道法人只做佣金归属(PartnerEntity),不参与地址归属冲突校验。
	// 清零后 submitRegular 的 checkOwnershipConflict 跳过,订单法人由地址推导。
	req.LegalEntityID = 0
	store := *s
	store.db = tx
	result, err := store.submitRegular(ctx, req)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("order: commit partner submit: %w", err)
	}
	return result, nil
}

func lockPartnerTenant(ctx context.Context, tx pgx.Tx, entityID int64) error {
	var id int64
	if err := tx.QueryRow(ctx, `SELECT id FROM legal_entities WHERE id=$1 FOR UPDATE`, entityID).Scan(&id); err != nil {
		return fmt.Errorf("order: partner tenant: %w", err)
	}
	return nil
}

// checkPartnerOwnership 渠道下单归属校验(C 案):客户/产品属渠道法人 **或** 平台法人(1)
// 均放行;渠道代售平台产品时订单法人由地址推导,佣金归 partner_entity_id(见 pg_workflow.go)。
func checkPartnerOwnership(ctx context.Context, tx pgx.Tx, req SubmitReq) error {
	var ok bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM customers WHERE id=$1 AND (legal_entity_id=$2 OR legal_entity_id=1))`,
		req.CustomerID, req.LegalEntityID).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("order: partner customer ownership: %w", ErrInvalidInput)
	}
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM product_offers WHERE id=$1 AND (legal_entity_id=$2 OR legal_entity_id=1) AND status='PUBLISHED')`,
		req.OfferID, req.LegalEntityID).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("order: partner offer ownership: %w", ErrOfferNotOrderable)
	}
	return nil
}

func checkPartnerLimits(ctx context.Context, tx pgx.Tx, req SubmitReq) error {
	var daily, recent int64
	err := tx.QueryRow(ctx, `
SELECT (SELECT count(*) FROM orders WHERE legal_entity_id=$1 AND created_at >= date_trunc('day', now() AT TIME ZONE $3)),
       (SELECT count(*) FROM orders WHERE legal_entity_id=$1 AND customer_id=$2 AND created_at >= now()-interval '24 hours')`, req.LegalEntityID, req.CustomerID, "Asia/Manila").Scan(&daily, &recent)
	if err != nil {
		return fmt.Errorf("order: partner fraud check: %w", err)
	}
	if daily >= partnerDailyCap {
		return ErrPartnerDailyCap
	}
	if recent > 0 {
		return ErrPartnerCustomerCooldown
	}
	return nil
}
