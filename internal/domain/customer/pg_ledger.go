package customer

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

// idOrNil 把 0 归一为 NULL(可空约定:0=空)。
func idOrNil(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

// ListCustomerHistories 列出客户归属台账;customerID=0 返回全部。
func (s *PGStore) ListCustomerHistories(ctx context.Context, customerID int64) ([]CustomerHistory, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, customer_id, legal_entity_id, legal_entity_name, address_id, address_name,
		       region_id, region_name, COALESCE(reason, ''), COALESCE(operator_account_id, 0), effective_from, effective_to
		FROM customer_histories WHERE ($1 = 0 OR customer_id = $1) ORDER BY effective_from, id`, customerID)
	if err != nil {
		return nil, fmt.Errorf("customer: list histories: %w", err)
	}
	defer rows.Close()
	out := make([]CustomerHistory, 0)
	for rows.Next() {
		var h CustomerHistory
		var effTo pgtype.Timestamptz
		if err := rows.Scan(&h.ID, &h.CustomerID, &h.LegalEntityID, &h.LegalEntityName, &h.AddressID, &h.AddressName,
			&h.RegionID, &h.RegionName, &h.Reason, &h.OperatorAccountID, &h.EffectiveFrom, &effTo); err != nil {
			return nil, fmt.Errorf("customer: scan history: %w", err)
		}
		if effTo.Valid {
			t := effTo.Time
			h.EffectiveTo = &t
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// AppendCustomerHistory 追加客户归属台账,返回自增 id。
func (s *PGStore) AppendCustomerHistory(ctx context.Context, h CustomerHistory) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO customer_histories(customer_id, legal_entity_id, legal_entity_name, address_id, address_name, region_id, region_name, reason, operator_account_id, effective_from, effective_to)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		h.CustomerID, h.LegalEntityID, h.LegalEntityName, h.AddressID, h.AddressName, h.RegionID, h.RegionName,
		h.Reason, idOrNil(h.OperatorAccountID), h.EffectiveFrom, h.EffectiveTo).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("customer: append history: %w", err)
	}
	return id, nil
}

// ListProductPriceHistories 列出产品调价台账;offerID=0 返回全部。
func (s *PGStore) ListProductPriceHistories(ctx context.Context, offerID int64) ([]ProductPriceHistory, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, offer_id, old_monthly_fee, new_monthly_fee, effective_at, COALESCE(reason, ''), COALESCE(operator_account_id, 0)
		FROM product_price_histories WHERE ($1 = 0 OR offer_id = $1) ORDER BY effective_at, id`, offerID)
	if err != nil {
		return nil, fmt.Errorf("customer: list product price: %w", err)
	}
	defer rows.Close()
	out := make([]ProductPriceHistory, 0)
	for rows.Next() {
		var h ProductPriceHistory
		if err := rows.Scan(&h.ID, &h.OfferID, &h.OldMonthlyFee, &h.NewMonthlyFee, &h.EffectiveAt, &h.Reason, &h.OperatorAccountID); err != nil {
			return nil, fmt.Errorf("customer: scan product price: %w", err)
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// AppendProductPriceHistory 追加产品调价台账,返回自增 id。
func (s *PGStore) AppendProductPriceHistory(ctx context.Context, h ProductPriceHistory) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO product_price_histories(offer_id, old_monthly_fee, new_monthly_fee, effective_at, reason, operator_account_id)
		VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,
		h.OfferID, h.OldMonthlyFee, h.NewMonthlyFee, h.EffectiveAt, h.Reason, idOrNil(h.OperatorAccountID)).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("customer: append product price: %w", err)
	}
	return id, nil
}

// ListRegionPriceHistories 列出区域调价台账;regionOfferID=0 返回全部。
func (s *PGStore) ListRegionPriceHistories(ctx context.Context, regionOfferID int64) ([]RegionPriceHistory, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, region_offer_id, old_monthly_fee, new_monthly_fee, effective_at, COALESCE(reason, ''), COALESCE(operator_account_id, 0)
		FROM region_price_histories WHERE ($1 = 0 OR region_offer_id = $1) ORDER BY effective_at, id`, regionOfferID)
	if err != nil {
		return nil, fmt.Errorf("customer: list region price: %w", err)
	}
	defer rows.Close()
	out := make([]RegionPriceHistory, 0)
	for rows.Next() {
		var h RegionPriceHistory
		if err := rows.Scan(&h.ID, &h.RegionOfferID, &h.OldMonthlyFee, &h.NewMonthlyFee, &h.EffectiveAt, &h.Reason, &h.OperatorAccountID); err != nil {
			return nil, fmt.Errorf("customer: scan region price: %w", err)
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// AppendRegionPriceHistory 追加区域调价台账,返回自增 id。
func (s *PGStore) AppendRegionPriceHistory(ctx context.Context, h RegionPriceHistory) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO region_price_histories(region_offer_id, old_monthly_fee, new_monthly_fee, effective_at, reason, operator_account_id)
		VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,
		h.RegionOfferID, h.OldMonthlyFee, h.NewMonthlyFee, h.EffectiveAt, h.Reason, idOrNil(h.OperatorAccountID)).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("customer: append region price: %w", err)
	}
	return id, nil
}
