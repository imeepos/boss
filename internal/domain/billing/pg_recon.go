package billing

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ymm-001/boss/internal/pkg/clock"
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

// RecordChannelStatement 录入渠道侧流水并逐行比对:
// 重写该批次 items(MATCH 行也落),重算双边总额与状态(不等=DIFF_PENDING,语义不变)。
func (s *PGStore) RecordChannelStatement(ctx context.Context, batchID int64, rows []ChannelStatementRow) error {
	b, err := s.getReconBatchByID(ctx, batchID)
	if err != nil {
		return err
	}
	if b.Status == "SETTLED" {
		return ErrIllegalReconTransition
	}
	pays, err := s.loadDailyPayments(ctx, b.CreatedAt)
	if err != nil {
		return err
	}
	if err := s.replaceItems(ctx, batchID, CompareStatement(rows, pays)); err != nil {
		return err
	}
	channelAmt, systemAmt := sumChannelRows(rows), sumPaymentRefs(pays)
	status := "SETTLED"
	if cents(channelAmt) != cents(systemAmt) {
		status = "DIFF_PENDING"
	}
	if _, err := s.db.Exec(ctx, `
		UPDATE reconciliation_batches SET channel_amount=$2, system_amount=$3, status=$4 WHERE id=$1`,
		batchID, channelAmt, systemAmt, status); err != nil {
		return fmt.Errorf("billing: record channel statement: %w", err)
	}
	return nil
}

// getReconBatchByID 按批次 id 查对账批次;未命中返回 ErrNotFound。
func (s *PGStore) getReconBatchByID(ctx context.Context, batchID int64) (*ReconBatch, error) {
	var b ReconBatch
	err := s.db.QueryRow(ctx, `
		SELECT id, batch_no, channel, channel_amount, system_amount, status, created_at, settled_at
		FROM reconciliation_batches WHERE id=$1`, batchID).
		Scan(&b.ID, &b.BatchNo, &b.Channel, &b.ChannelAmount, &b.SystemAmount, &b.Status, &b.CreatedAt, &b.SettledAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("billing: get recon batch by id: %w", err)
	}
	return &b, nil
}

// loadDailyPayments 系统侧比对范围:批次创建同日(批次号 PC-YYYYMMDD-NN 按日出批)的 SUCCESS 缴费。
// 日界由 clock.DayBounds 按业务时区在 Go 侧切好传入——date_trunc 依赖会话时区(UTC),
// 与批次号的业务日口径不一致会在马尼拉 00:00-08:00 错切一天。
func (s *PGStore) loadDailyPayments(ctx context.Context, day time.Time) ([]PaymentRef, error) {
	start, end := clock.DayBounds(day)
	rows, err := s.db.Query(ctx, `
		SELECT id, pay_no, amount FROM payments
		WHERE status='SUCCESS'
		  AND created_at >= $1
		  AND created_at < $2
		ORDER BY id`, start, end)
	if err != nil {
		return nil, fmt.Errorf("billing: load daily payments: %w", err)
	}
	defer rows.Close()
	out := make([]PaymentRef, 0)
	for rows.Next() {
		var p PaymentRef
		if err := rows.Scan(&p.ID, &p.PayNo, &p.Amount); err != nil {
			return nil, fmt.Errorf("billing: scan payment ref: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// replaceItems 幂等重写批次明细:先清旧 items 再逐行插入。
func (s *PGStore) replaceItems(ctx context.Context, batchID int64, items []ReconItem) error {
	if _, err := s.db.Exec(ctx,
		`DELETE FROM reconciliation_items WHERE batch_id=$1`, batchID); err != nil {
		return fmt.Errorf("billing: clear recon items: %w", err)
	}
	for _, it := range items {
		if _, err := s.db.Exec(ctx, `
			INSERT INTO reconciliation_items(batch_id, payment_id, channel_ref, amount, diff_kind, note)
			VALUES($1,$2,$3,$4,$5,$6)`,
			batchID, it.PaymentID, it.ChannelRef, it.Amount, it.DiffKind, it.Note); err != nil {
			return fmt.Errorf("billing: insert recon item: %w", err)
		}
	}
	return nil
}

// ListReconciliationItems 批次行级明细,按 id 升序(差异定位用)。
func (s *PGStore) ListReconciliationItems(ctx context.Context, batchID int64) ([]ReconItem, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, batch_id, payment_id, channel_ref, amount, diff_kind, note
		FROM reconciliation_items WHERE batch_id=$1 ORDER BY id`, batchID)
	if err != nil {
		return nil, fmt.Errorf("billing: list recon items: %w", err)
	}
	defer rows.Close()
	out := make([]ReconItem, 0)
	for rows.Next() {
		var it ReconItem
		var payID sql.NullInt64
		if err := rows.Scan(&it.ID, &it.BatchID, &payID, &it.ChannelRef,
			&it.Amount, &it.DiffKind, &it.Note); err != nil {
			return nil, fmt.Errorf("billing: scan recon item: %w", err)
		}
		if payID.Valid {
			it.PaymentID = &payID.Int64
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// sumChannelRows 渠道侧流水金额合计。
func sumChannelRows(rows []ChannelStatementRow) float64 {
	var sum float64
	for _, r := range rows {
		sum += r.Amount
	}
	return sum
}

// sumPaymentRefs 系统侧缴费金额合计。
func sumPaymentRefs(pays []PaymentRef) float64 {
	var sum float64
	for _, p := range pays {
		sum += p.Amount
	}
	return sum
}
