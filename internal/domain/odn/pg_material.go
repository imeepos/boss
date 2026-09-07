package odn

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
)

// issueCols 出库单查询列(000215;asset_ids 聚合子查询)。
const issueCols = "i.id, i.issue_no, i.project_id, i.project_no, i.status, COALESCE(i.remark,''), " +
	"COALESCE(i.created_by,0), COALESCE(i.issued_by,0), COALESCE(i.cancelled_by,0), " +
	"to_char(i.created_at,'YYYY-MM-DD HH24:MI:SS'), " +
	"COALESCE(to_char(i.issued_at,'YYYY-MM-DD HH24:MI:SS'),''), " +
	"COALESCE(to_char(i.cancelled_at,'YYYY-MM-DD HH24:MI:SS'),''), " +
	"COALESCE((SELECT array_agg(it.asset_id ORDER BY it.asset_id) FROM odn_material_issue_items it " +
	"WHERE it.issue_id = i.id), '{}'::bigint[])"

func scanIssue(row pgx.Row, m *MaterialIssue) error {
	return row.Scan(&m.ID, &m.IssueNo, &m.ProjectID, &m.ProjectNo, &m.Status, &m.Remark,
		&m.CreatedBy, &m.IssuedBy, &m.CancelledBy, &m.CreatedAt, &m.IssuedAt, &m.CancelledAt, &m.AssetIDs)
}

func nextIssueNo() string {
	return fmt.Sprintf("MI-%s-%05d", time.Now().Format("20060102"), time.Now().UnixNano()%100000)
}

// CreateIssue 建出库单(OPEN):项目须存在;资产须 IN_STOCK 且不重复;不动资产状态。
func (s *PGStore) CreateIssue(ctx context.Context, projectID int64, assetIDs []int64, remark string, accountID int64) (*MaterialIssue, error) {
	if projectID <= 0 || len(assetIDs) == 0 || len(assetIDs) > 200 {
		return nil, ErrIssueInput
	}
	seen := map[int64]bool{}
	for _, id := range assetIDs {
		if id <= 0 || seen[id] {
			return nil, fmt.Errorf("odn: duplicate/invalid asset %d: %w", id, ErrIssueInput)
		}
		seen[id] = true
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Printf("[odn-issue] TX BEGIN FAILED proj=%d: %v", projectID, err)
		return nil, fmt.Errorf("odn: issue begin: %w", err)
	}
	defer tx.Rollback(ctx)
	var projNo string
	err = tx.QueryRow(ctx, `SELECT proj_no FROM construction_projects WHERE id=$1`, projectID).Scan(&projNo)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("odn: project %d not found: %w", projectID, ErrNotFound)
	}
	if err != nil {
		log.Printf("[odn-issue] PROJECT READ FAILED proj=%d: %v", projectID, err)
		return nil, fmt.Errorf("odn: issue project read: %w", err)
	}
	for _, id := range assetIDs {
		var status string
		err := tx.QueryRow(ctx, `SELECT status FROM assets WHERE id=$1 FOR UPDATE`, id).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("odn: asset %d not found: %w", id, ErrIssueAsset)
		}
		if err != nil {
			log.Printf("[odn-issue] ASSET READ FAILED asset=%d: %v", id, err)
			return nil, fmt.Errorf("odn: issue asset read: %w", err)
		}
		if status != "IN_STOCK" {
			return nil, fmt.Errorf("odn: asset %d status=%s: %w", id, status, ErrIssueAsset)
		}
	}
	no := nextIssueNo()
	m := &MaterialIssue{IssueNo: no, ProjectID: projectID, ProjectNo: projNo,
		Status: IssueOpen, Remark: remark, AssetIDs: assetIDs, CreatedBy: accountID}
	err = tx.QueryRow(ctx, `INSERT INTO odn_material_issues
		(issue_no, project_id, project_no, remark, created_by)
		VALUES ($1,$2,$3,$4,$5) RETURNING id, to_char(created_at,'YYYY-MM-DD HH24:MI:SS')`,
		no, projectID, projNo, remark, accountID).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		log.Printf("[odn-issue] INSERT FAILED no=%s: %v", no, err)
		return nil, fmt.Errorf("odn: issue insert: %w", err)
	}
	for _, id := range assetIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO odn_material_issue_items(issue_id, asset_id) VALUES ($1,$2)`, m.ID, id); err != nil {
			log.Printf("[odn-issue] ITEM INSERT FAILED no=%s asset=%d: %v", no, id, err)
			return nil, fmt.Errorf("odn: issue item insert: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		log.Printf("[odn-issue] COMMIT FAILED no=%s: %v", no, err)
		return nil, fmt.Errorf("odn: issue commit: %w", err)
	}
	return m, nil
}

// ListIssues 出库单列表(projectId 0=全部;status 空=全部;近单优先)。
func (s *PGStore) ListIssues(ctx context.Context, projectID int64, status string, limit int) ([]MaterialIssue, error) {
	sql := `SELECT ` + issueCols + ` FROM odn_material_issues i WHERE 1=1`
	args := []any{}
	if projectID > 0 {
		sql += ` AND i.project_id=` + sqlPlaceholder(len(args)+1)
		args = append(args, projectID)
	}
	if status == IssueOpen || status == IssueConfirmed || status == IssueCancelled {
		sql += ` AND i.status=` + sqlPlaceholder(len(args)+1)
		args = append(args, status)
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	sql += ` ORDER BY i.id DESC LIMIT ` + sqlPlaceholder(len(args)+1)
	args = append(args, limit)
	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		log.Printf("[odn-issue] LIST FAILED proj=%d: %v", projectID, err)
		return nil, fmt.Errorf("odn: list issues: %w", err)
	}
	defer rows.Close()
	out := []MaterialIssue{}
	for rows.Next() {
		var m MaterialIssue
		if err := scanIssue(rows, &m); err != nil {
			return nil, fmt.Errorf("odn: scan issue: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// GetIssue 出库单详情。
func (s *PGStore) GetIssue(ctx context.Context, id int64) (*MaterialIssue, error) {
	var m MaterialIssue
	err := scanIssue(s.db.QueryRow(ctx, `SELECT `+issueCols+
		` FROM odn_material_issues i WHERE i.id=$1`, id), &m)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		log.Printf("[odn-issue] GET FAILED id=%d: %v", id, err)
		return nil, fmt.Errorf("odn: get issue: %w", err)
	}
	return &m, nil
}

// ConfirmIssue 出库确认 OPEN→CONFIRMED:全部明细资产 IN_STOCK→IN_TRANSIT+轨迹(同事务)。
// 任一资产状态漂移即整体回滚(台账一致性优先)。
func (s *PGStore) ConfirmIssue(ctx context.Context, id, accountID int64) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Printf("[odn-issue] TX BEGIN FAILED confirm id=%d: %v", id, err)
		return fmt.Errorf("odn: issue confirm begin: %w", err)
	}
	defer tx.Rollback(ctx)
	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM odn_material_issues WHERE id=$1 FOR UPDATE`, id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("odn: issue %d not found: %w", id, ErrNotFound)
	}
	if err != nil {
		log.Printf("[odn-issue] READ FAILED confirm id=%d: %v", id, err)
		return fmt.Errorf("odn: issue confirm read: %w", err)
	}
	if err := ValidateIssueTransition(status, IssueConfirmed); err != nil {
		return fmt.Errorf("odn: issue %d status=%s: %w", id, status, err)
	}
	ids, err := issueAssetIDs(ctx, tx, id)
	if err != nil {
		return err
	}
	for _, assetID := range ids {
		if err := flipAssetStatus(ctx, tx, assetID, "IN_STOCK", "IN_TRANSIT"); err != nil {
			return err
		}
	}
	tag, err := tx.Exec(ctx, `UPDATE odn_material_issues SET status=$1, issued_by=$2, issued_at=now(),
		updated_at=now() WHERE id=$3 AND status=$4`, IssueConfirmed, accountID, id, IssueOpen)
	if err != nil || tag.RowsAffected() == 0 {
		log.Printf("[odn-issue] CONFIRM FAILED id=%d: %v", id, err)
		return fmt.Errorf("odn: issue confirm update: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		log.Printf("[odn-issue] CONFIRM COMMIT FAILED id=%d: %v", id, err)
		return fmt.Errorf("odn: issue confirm commit: %w", err)
	}
	return nil
}

// CancelIssue 取消 OPEN=作废不碰资产;取消 CONFIRMED=退库(在途资产回 IN_STOCK+轨迹)。
func (s *PGStore) CancelIssue(ctx context.Context, id, accountID int64) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Printf("[odn-issue] TX BEGIN FAILED cancel id=%d: %v", id, err)
		return fmt.Errorf("odn: issue cancel begin: %w", err)
	}
	defer tx.Rollback(ctx)
	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM odn_material_issues WHERE id=$1 FOR UPDATE`, id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("odn: issue %d not found: %w", id, ErrNotFound)
	}
	if err != nil {
		log.Printf("[odn-issue] READ FAILED cancel id=%d: %v", id, err)
		return fmt.Errorf("odn: issue cancel read: %w", err)
	}
	if err := ValidateIssueTransition(status, IssueCancelled); err != nil {
		return fmt.Errorf("odn: issue %d status=%s: %w", id, status, err)
	}
	if status == IssueConfirmed {
		ids, err := issueAssetIDs(ctx, tx, id)
		if err != nil {
			return err
		}
		for _, assetID := range ids {
			if err := flipAssetStatus(ctx, tx, assetID, "IN_TRANSIT", "IN_STOCK"); err != nil {
				return err
			}
		}
	}
	tag, err := tx.Exec(ctx, `UPDATE odn_material_issues SET status=$1, cancelled_by=$2, cancelled_at=now(),
		updated_at=now() WHERE id=$3 AND status=$4`, IssueCancelled, accountID, id, status)
	if err != nil || tag.RowsAffected() == 0 {
		log.Printf("[odn-issue] CANCEL FAILED id=%d: %v", id, err)
		return fmt.Errorf("odn: issue cancel update: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		log.Printf("[odn-issue] CANCEL COMMIT FAILED id=%d: %v", id, err)
		return fmt.Errorf("odn: issue cancel commit: %w", err)
	}
	return nil
}

// issueAssetIDs 出库单明细资产 id 列表(锁定头行后读取)。
func issueAssetIDs(ctx context.Context, tx pgx.Tx, issueID int64) ([]int64, error) {
	rows, err := tx.Query(ctx,
		`SELECT asset_id FROM odn_material_issue_items WHERE issue_id=$1 ORDER BY asset_id`, issueID)
	if err != nil {
		log.Printf("[odn-issue] ITEMS READ FAILED id=%d: %v", issueID, err)
		return nil, fmt.Errorf("odn: issue items read: %w", err)
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("odn: scan issue item: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// flipAssetStatus 资产状态原子翻转+轨迹行(0 行受影响=状态漂移,40900 整体回滚)。
func flipAssetStatus(ctx context.Context, tx pgx.Tx, assetID int64, from, to string) error {
	tag, err := tx.Exec(ctx,
		`UPDATE assets SET status=$1, updated_at=now() WHERE id=$2 AND status=$3`, to, assetID, from)
	if err != nil {
		log.Printf("[odn-issue] ASSET FLIP FAILED asset=%d %s->%s: %v", assetID, from, to, err)
		return fmt.Errorf("odn: issue asset flip: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("odn: asset %d not %s: %w", assetID, from, ErrIssueAsset)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO asset_lifecycles(asset_id, status, changed_at) VALUES ($1,$2,now())`, assetID, to); err != nil {
		log.Printf("[odn-issue] LIFECYCLE FAILED asset=%d to=%s: %v", assetID, to, err)
		return fmt.Errorf("odn: issue lifecycle: %w", err)
	}
	return nil
}
