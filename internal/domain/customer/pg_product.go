package customer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ListProducts 列出产品;legalEntityID=0 返回全部,否则按公司过滤。
func (s *PGStore) ListProducts(ctx context.Context, legalEntityID int64) ([]ProductOffer, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, legal_entity_id, name, bandwidth, monthly_fee, category, effective_at, status
		FROM product_offers
		WHERE ($1 = 0 OR legal_entity_id = $1)
		ORDER BY id`, legalEntityID)
	if err != nil {
		return nil, fmt.Errorf("customer: list products: %w", err)
	}
	defer rows.Close()
	out := make([]ProductOffer, 0)
	for rows.Next() {
		var p ProductOffer
		if err := rows.Scan(&p.ID, &p.LegalEntityID, &p.Name, &p.Bandwidth, &p.MonthlyFee, &p.Category, &p.EffectiveAt, &p.Status); err != nil {
			return nil, fmt.Errorf("customer: scan product: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// CreateProduct 新建产品,返回自增 id。
// 校验 legal_entity_id 存在性,防止孤儿产品。
func (s *PGStore) CreateProduct(ctx context.Context, p ProductOffer) (int64, error) {
	// 关联完整性校验
	if p.LegalEntityID > 0 {
		ok, err := s.exists(ctx, "legal_entities", p.LegalEntityID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("customer: legal entity %d: %w", p.LegalEntityID, ErrForeignKeyViolation)
		}
	}

	var id int64
	category := p.Category
	if category == "" {
		category = "broadband"
	}
	err := s.db.QueryRow(ctx, `
		INSERT INTO product_offers(legal_entity_id, name, bandwidth, monthly_fee, category, effective_at, status)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		p.LegalEntityID, p.Name, p.Bandwidth, p.MonthlyFee, category, p.EffectiveAt, p.Status).Scan(&id)
	if isPgUniqueViolation(err) {
		return 0, ErrDuplicate
	}
	if err != nil {
		return 0, fmt.Errorf("customer: create product: %w", err)
	}
	return id, nil
}

// ListRegionOffers 列出区域运营包;offerID=0 返回全部,否则按产品过滤。
func (s *PGStore) ListRegionOffers(ctx context.Context, offerID int64) ([]RegionOffer, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, offer_id, region_path, COALESCE(name, ''), monthly_fee, COALESCE(reason, '')
		FROM region_offers
		WHERE ($1 = 0 OR offer_id = $1)
		ORDER BY id`, offerID)
	if err != nil {
		return nil, fmt.Errorf("customer: list region offers: %w", err)
	}
	defer rows.Close()
	out := make([]RegionOffer, 0)
	for rows.Next() {
		var r RegionOffer
		if err := rows.Scan(&r.ID, &r.OfferID, &r.RegionPath, &r.Name, &r.MonthlyFee, &r.Reason); err != nil {
			return nil, fmt.Errorf("customer: scan region offer: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CreateRegionOffer 新建区域运营包,返回自增 id。
// 校验 offer_id 存在性,防止孤儿区域包。
func (s *PGStore) CreateRegionOffer(ctx context.Context, r RegionOffer) (int64, error) {
	// 关联完整性校验
	if r.OfferID > 0 {
		ok, err := s.exists(ctx, "product_offers", r.OfferID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("customer: offer %d: %w", r.OfferID, ErrForeignKeyViolation)
		}
	}

	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO region_offers(offer_id, region_path, name, monthly_fee, reason)
		VALUES($1,$2,$3,$4,$5) RETURNING id`,
		r.OfferID, r.RegionPath, r.Name, r.MonthlyFee, r.Reason).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("customer: create region offer: %w", err)
	}
	return id, nil
}

// ChangeProductPrice 产品调价:显式事务内 读旧月费→更新产品→追加调价台账,返回台账 id。
func (s *PGStore) ChangeProductPrice(ctx context.Context, offerID int64, newFee float64, effectiveAt time.Time, reason string, operatorAccountID int64) (int64, error) {
	tx, err := s.db.(beginner).Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("customer: begin price change: %w", err)
	}
	defer tx.Rollback(ctx)

	var oldFee float64
	if err := tx.QueryRow(ctx,
		`SELECT monthly_fee FROM product_offers WHERE id=$1 FOR UPDATE`, offerID).Scan(&oldFee); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrProductNotFound
		}
		return 0, fmt.Errorf("customer: read product fee: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE product_offers SET monthly_fee=$2, effective_at=$3, updated_at=now() WHERE id=$1`,
		offerID, newFee, effectiveAt); err != nil {
		return 0, fmt.Errorf("customer: update product fee: %w", err)
	}
	var historyID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO product_price_histories(offer_id, old_monthly_fee, new_monthly_fee, effective_at, reason, operator_account_id)
		VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,
		offerID, oldFee, newFee, effectiveAt, reason, idOrNil(operatorAccountID)).Scan(&historyID); err != nil {
		return 0, fmt.Errorf("customer: append price history: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("customer: commit price change: %w", err)
	}
	return historyID, nil
}

// UpdateProduct 编辑产品基础信息(名称/带宽/分类);公司归属与月费/状态不在此口(分别防孤儿/走台账/走状态机)。
func (s *PGStore) UpdateProduct(ctx context.Context, offerID int64, name, bandwidth, category string) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE product_offers
		SET name=$2, bandwidth=$3, category=COALESCE(NULLIF($4,''),'broadband'), updated_at=now()
		WHERE id=$1`, offerID, name, bandwidth, category)
	if err != nil {
		return fmt.Errorf("customer: update product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrProductNotFound
	}
	return nil
}

// UpdateProductStatus 上下架;发布即生效,刷新 effective_at(与调价同口径,fields.md §2.2)。
func (s *PGStore) UpdateProductStatus(ctx context.Context, offerID int64, status string) error {
	if status != "DRAFT" && status != "PUBLISHED" && status != "OFFLINE" {
		return ErrInvalidProductStatus
	}
	var tag pgconn.CommandTag
	var err error
	if status == "PUBLISHED" {
		tag, err = s.db.Exec(ctx,
			`UPDATE product_offers SET status=$2, effective_at=now(), updated_at=now() WHERE id=$1`, offerID, status)
	} else {
		tag, err = s.db.Exec(ctx,
			`UPDATE product_offers SET status=$2, updated_at=now() WHERE id=$1`, offerID, status)
	}
	if err != nil {
		return fmt.Errorf("customer: update product status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrProductNotFound
	}
	return nil
}
