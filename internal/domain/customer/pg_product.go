package customer

import (
	"context"
	"fmt"
)

// ListProducts 列出产品;legalEntityID=0 返回全部,否则按公司过滤。
func (s *PGStore) ListProducts(ctx context.Context, legalEntityID int64) ([]ProductOffer, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, legal_entity_id, name, bandwidth, monthly_fee, effective_at, status
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
		if err := rows.Scan(&p.ID, &p.LegalEntityID, &p.Name, &p.Bandwidth, &p.MonthlyFee, &p.EffectiveAt, &p.Status); err != nil {
			return nil, fmt.Errorf("customer: scan product: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// CreateProduct 新建产品,返回自增 id。
func (s *PGStore) CreateProduct(ctx context.Context, p ProductOffer) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO product_offers(legal_entity_id, name, bandwidth, monthly_fee, effective_at, status)
		VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,
		p.LegalEntityID, p.Name, p.Bandwidth, p.MonthlyFee, p.EffectiveAt, p.Status).Scan(&id)
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
func (s *PGStore) CreateRegionOffer(ctx context.Context, r RegionOffer) (int64, error) {
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
