package odn

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// settlementCols 结算单查询列(000204)。
const settlementCols = "id, settlement_no, project_id, project_no, COALESCE(contractor_id,0), " +
	"contractor_name, total_amount, item_count, status, void_reason, " +
	"COALESCE(created_by,0), COALESCE(settled_by,0), COALESCE(voided_by,0), " +
	"to_char(created_at,'YYYY-MM-DD HH24:MI:SS'), " +
	"COALESCE(to_char(settled_at,'YYYY-MM-DD HH24:MI:SS'),''), " +
	"COALESCE(to_char(voided_at,'YYYY-MM-DD HH24:MI:SS'),'')"

// scanSettlement 扫描结算单行。
func scanSettlement(row pgx.Row, s *Settlement) error {
	return row.Scan(&s.ID, &s.SettlementNo, &s.ProjectID, &s.ProjectNo, &s.ContractorID,
		&s.ContractorName, &s.TotalAmount, &s.ItemCount, &s.Status, &s.VoidReason,
		&s.CreatedBy, &s.SettledBy, &s.VoidedBy, &s.CreatedAt, &s.SettledAt, &s.VoidedAt)
}

// nextSettlementNo 生成结算单号 ST-YYYYMMDD-NNNNN(口径同采购单号,时间戳后 5 位兜底)。
func nextSettlementNo() string {
	return fmt.Sprintf("ST-%s-%05d", time.Now().Format("20060102"), time.Now().UnixNano()%100000)
}

// CreateSettlement 发起结算:项目须 ACCEPTED 且已指定承包商;同事务汇总清单金额形成应付。
// 同项目同时最多一张有效结算单(部分唯一索引兜底并发)。
func (s *PGStore) CreateSettlement(ctx context.Context, projectID, createdBy int64) (*Settlement, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Printf("[odn-settlement] TX BEGIN FAILED proj=%d: %v", projectID, err)
		return nil, fmt.Errorf("odn: settlement begin: %w", err)
	}
	defer tx.Rollback(ctx)

	var status, projNo, contractorName string
	var contractorID *int64
	err = tx.QueryRow(ctx, `SELECT status, proj_no, contractor_id, COALESCE(contractor_name,'')
		FROM construction_projects WHERE id=$1 FOR UPDATE`, projectID).
		Scan(&status, &projNo, &contractorID, &contractorName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		log.Printf("[odn-settlement] PROJECT READ FAILED proj=%d: %v", projectID, err)
		return nil, fmt.Errorf("odn: settlement project read: %w", err)
	}
	if status != CAccepted {
		return nil, fmt.Errorf("odn: project status=%s: %w", status, ErrInvalidProjStatus)
	}
	if contractorID == nil {
		return nil, fmt.Errorf("odn: project %d: %w", projectID, ErrNoContractor)
	}

	var total float64
	var items int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(amount),0), count(*)
		FROM construction_items WHERE project_id=$1`, projectID).Scan(&total, &items); err != nil {
		log.Printf("[odn-settlement] AMOUNT SUM FAILED proj=%d: %v", projectID, err)
		return nil, fmt.Errorf("odn: settlement sum: %w", err)
	}

	no := nextSettlementNo()
	var id int64
	var createdAt string
	err = tx.QueryRow(ctx, `INSERT INTO construction_settlements
		(settlement_no, project_id, project_no, contractor_id, contractor_name,
		 total_amount, item_count, created_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, to_char(created_at,'YYYY-MM-DD HH24:MI:SS')`,
		no, projectID, projNo, *contractorID, contractorName, total, items, createdBy).
		Scan(&id, &createdAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, fmt.Errorf("odn: project %d 已有有效结算单: %w", projectID, ErrSettlementState)
		}
		log.Printf("[odn-settlement] CREATE FAILED proj=%d no=%s: %v", projectID, no, err)
		return nil, fmt.Errorf("odn: settlement create: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		log.Printf("[odn-settlement] COMMIT FAILED proj=%d no=%s: %v", projectID, no, err)
		return nil, fmt.Errorf("odn: settlement commit: %w", err)
	}
	return &Settlement{ID: id, SettlementNo: no, ProjectID: projectID, ProjectNo: projNo,
		ContractorID: *contractorID, ContractorName: contractorName, TotalAmount: total,
		ItemCount: items, Status: SPending, CreatedBy: createdBy, CreatedAt: createdAt}, nil
}

// ListSettlements 项目结算单列表(近单优先;作废单含历史)。
func (s *PGStore) ListSettlements(ctx context.Context, projectID int64) ([]Settlement, error) {
	rows, err := s.db.Query(ctx, `SELECT `+settlementCols+
		` FROM construction_settlements WHERE project_id=$1 ORDER BY id DESC`, projectID)
	if err != nil {
		log.Printf("[odn-settlement] LIST FAILED proj=%d: %v", projectID, err)
		return nil, fmt.Errorf("odn: list settlements: %w", err)
	}
	defer rows.Close()
	out := []Settlement{}
	for rows.Next() {
		var st Settlement
		if err := scanSettlement(rows, &st); err != nil {
			return nil, fmt.Errorf("odn: scan settlement: %w", err)
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

// GetSettlement 结算单详情。
func (s *PGStore) GetSettlement(ctx context.Context, id int64) (*Settlement, error) {
	var st Settlement
	err := scanSettlement(s.db.QueryRow(ctx, `SELECT `+settlementCols+
		` FROM construction_settlements WHERE id=$1`, id), &st)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("odn: get settlement: %w", err)
	}
	return &st, nil
}

// SettleSettlement 确认结算 PENDING→SETTLED(CAS,记结算人/时间)。
func (s *PGStore) SettleSettlement(ctx context.Context, id, accountID int64) error {
	tag, err := s.db.Exec(ctx, `UPDATE construction_settlements
		SET status='SETTLED', settled_by=$2, settled_at=now(), updated_at=now()
		WHERE id=$1 AND status='PENDING'`, id, accountID)
	if err != nil {
		log.Printf("[odn-settlement] SETTLE FAILED id=%d: %v", id, err)
		return fmt.Errorf("odn: settle: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return s.settlementConflict(ctx, id)
	}
	return nil
}

// VoidSettlement 作废 PENDING/SETTLED→VOIDED(原因必填,记作废人/时间);VOIDED 终态。
func (s *PGStore) VoidSettlement(ctx context.Context, id, accountID int64, reason string) error {
	if reason == "" {
		return fmt.Errorf("odn: void reason required: %w", ErrInvalidInput)
	}
	tag, err := s.db.Exec(ctx, `UPDATE construction_settlements
		SET status='VOIDED', voided_by=$2, voided_at=now(), void_reason=$3, updated_at=now()
		WHERE id=$1 AND status IN ('PENDING','SETTLED')`, id, accountID, reason)
	if err != nil {
		log.Printf("[odn-settlement] VOID FAILED id=%d: %v", id, err)
		return fmt.Errorf("odn: void: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return s.settlementConflict(ctx, id)
	}
	return nil
}

// settlementConflict 0 行更新时区分不存在与状态漂移。
func (s *PGStore) settlementConflict(ctx context.Context, id int64) error {
	if _, err := s.GetSettlement(ctx, id); err != nil {
		return err
	}
	var st string
	if err := s.db.QueryRow(ctx, `SELECT status FROM construction_settlements WHERE id=$1`, id).Scan(&st); err != nil {
		return ErrNotFound
	}
	return fmt.Errorf("odn: settlement %d status=%s: %w", id, st, ErrSettlementState)
}
