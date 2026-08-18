package billing

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ListReconciliations 对账批次列表,按批次号倒序(最新在前)。
func (s *PGStore) ListReconciliations(ctx context.Context) ([]ReconBatch, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, batch_no, channel, channel_amount, system_amount, status, created_at, settled_at
		FROM reconciliation_batches ORDER BY batch_no DESC`)
	if err != nil {
		return nil, fmt.Errorf("billing: list reconciliations: %w", err)
	}
	defer rows.Close()
	out := make([]ReconBatch, 0)
	for rows.Next() {
		var b ReconBatch
		if err := rows.Scan(&b.ID, &b.BatchNo, &b.Channel, &b.ChannelAmount,
			&b.SystemAmount, &b.Status, &b.CreatedAt, &b.SettledAt); err != nil {
			return nil, fmt.Errorf("billing: scan reconciliation: %w", err)
		}
		b.Diff = b.ChannelAmount - b.SystemAmount
		out = append(out, b)
	}
	return out, rows.Err()
}

// AppendReconciliation 追加对账批次,返回自增 id。
func (s *PGStore) AppendReconciliation(ctx context.Context, b ReconBatch) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO reconciliation_batches(batch_no, channel, channel_amount, system_amount, status)
		VALUES($1,$2,$3,$4,$5) RETURNING id`,
		b.BatchNo, b.Channel, b.ChannelAmount, b.SystemAmount, b.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("billing: append reconciliation: %w", err)
	}
	return id, nil
}

// SettleReconciliation 差异挂起批次平账:DIFF_PENDING→SETTLED,记录平账时间。
func (s *PGStore) SettleReconciliation(ctx context.Context, batchNo string) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE reconciliation_batches SET status='SETTLED', settled_at=now()
		WHERE batch_no=$1 AND status='DIFF_PENDING'`, batchNo)
	if err != nil {
		return fmt.Errorf("billing: settle reconciliation: %w", err)
	}
	switch tag.RowsAffected() {
	case 0:
		if _, err := s.GetReconciliation(ctx, batchNo); err != nil {
			return err
		}
		return ErrIllegalReconTransition
	default:
		return nil
	}
}

// GetReconciliation 按批次号查对账批次;未命中返回 ErrNotFound。
func (s *PGStore) GetReconciliation(ctx context.Context, batchNo string) (*ReconBatch, error) {
	var b ReconBatch
	err := s.db.QueryRow(ctx, `
		SELECT id, batch_no, channel, channel_amount, system_amount, status, created_at, settled_at
		FROM reconciliation_batches WHERE batch_no=$1`, batchNo).
		Scan(&b.ID, &b.BatchNo, &b.Channel, &b.ChannelAmount, &b.SystemAmount, &b.Status, &b.CreatedAt, &b.SettledAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("billing: get reconciliation: %w", err)
	}
	return &b, nil
}
