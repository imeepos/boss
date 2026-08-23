package partner

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const commissionCols = `id, order_id, legal_entity_id, order_amount, commission_rate, commission_amount, status, settled_at, COALESCE(settled_by, 0), created_at`

func (s *PGStore) ListCommissionLedger(ctx context.Context, accountID int64, status string) ([]CommissionLedgerRow, error) {
	entityID, err := s.entityOfAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if entityID == 0 {
		return nil, ErrNotPartner
	}
	rows, err := s.db.Query(ctx, `SELECT `+commissionCols+` FROM partner_commission_ledger WHERE legal_entity_id=$1 AND ($2='' OR status=$2) ORDER BY created_at DESC, id DESC`, entityID, status)
	if err != nil {
		return nil, fmt.Errorf("partner: list commission ledger: %w", err)
	}
	defer rows.Close()
	out := make([]CommissionLedgerRow, 0)
	for rows.Next() {
		var row CommissionLedgerRow
		var settledAt, createdAt *time.Time
		if err := rows.Scan(&row.ID, &row.OrderID, &row.LegalEntityID, &row.OrderAmount, &row.CommissionRate, &row.CommissionAmount, &row.Status, &settledAt, &row.SettledBy, &createdAt); err != nil {
			return nil, err
		}
		if settledAt != nil {
			row.SettledAt = settledAt.Format(time.RFC3339)
		}
		if createdAt != nil {
			row.CreatedAt = createdAt.Format(time.RFC3339)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *PGStore) AccrueCommission(ctx context.Context, orderID, legalEntityID int64, orderAmount, rate float64) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `INSERT INTO partner_commission_ledger(order_id, legal_entity_id, order_amount, commission_rate, commission_amount) VALUES($1,$2,$3,$4,round(($3*$4)::numeric,2)) ON CONFLICT (order_id, legal_entity_id) DO UPDATE SET order_amount=EXCLUDED.order_amount, commission_rate=EXCLUDED.commission_rate, commission_amount=EXCLUDED.commission_amount WHERE partner_commission_ledger.status='ACCRUED' RETURNING id`, orderID, legalEntityID, orderAmount, rate).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("partner: accrue commission: %w", err)
	}
	return id, nil
}

func (s *PGStore) SettleCommissionLedger(ctx context.Context, ledgerID, accountID int64) error {
	tag, err := s.db.Exec(ctx, `UPDATE partner_commission_ledger l SET status='SETTLED', settled_at=now(), settled_by=$2 FROM accounts a WHERE l.id=$1 AND a.id=$2 AND a.legal_entity_id=l.legal_entity_id AND l.status='ACCRUED'`, ledgerID, accountID)
	if err != nil {
		return fmt.Errorf("partner: settle commission: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
