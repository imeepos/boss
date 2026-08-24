package report

// S2 补偿任务 PG 存储:compensation_tasks 表 CRUD + 状态迁移 + 审计。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// colCompTask 全字段列表(不含 audit_log,单独处理)。
const colCompTask = `id, source, biz_type, biz_id, failure_reason, priority, status,
assignee_id, assignee_name, sla_deadline, claimed_at, claimed_by,
closed_at, closed_by, close_reason, retry_count, max_retries, last_retry_at,
created_at, updated_at`

// scanCompTask 扫描一行。
func scanCompTask(row pgx.Row) (*CompTask, error) {
	var t CompTask
	var auditRaw []byte
	var assigneeName, closeReason *string
	err := row.Scan(
		&t.ID, &t.Source, &t.BizType, &t.BizID, &t.FailureReason, &t.Priority, &t.Status,
		&t.AssigneeID, &assigneeName, &t.SLADeadline, &t.ClaimedAt, &t.ClaimedBy,
		&t.ClosedAt, &t.ClosedBy, &closeReason, &t.RetryCount, &t.MaxRetries, &t.LastRetryAt,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if assigneeName != nil {
		t.AssigneeName = *assigneeName
	}
	if closeReason != nil {
		t.CloseReason = *closeReason
	}
	t.AuditLog = SanitizeAudit(auditRaw)
	return &t, nil
}

// scanCompTaskWithAudit 扫描含 audit_log 的行。
func scanCompTaskWithAudit(row pgx.Row) (*CompTask, error) {
	var t CompTask
	var auditRaw []byte
	var assigneeName, closeReason *string
	err := row.Scan(
		&t.ID, &t.Source, &t.BizType, &t.BizID, &t.FailureReason, &t.Priority, &t.Status,
		&t.AssigneeID, &assigneeName, &t.SLADeadline, &t.ClaimedAt, &t.ClaimedBy,
		&t.ClosedAt, &t.ClosedBy, &closeReason, &t.RetryCount, &t.MaxRetries, &t.LastRetryAt,
		&auditRaw, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if assigneeName != nil {
		t.AssigneeName = *assigneeName
	}
	if closeReason != nil {
		t.CloseReason = *closeReason
	}
	t.AuditLog = SanitizeAudit(auditRaw)
	return &t, nil
}

// ListCompTasks 查询任务列表(支持过滤+分页),返回(任务,总数,err)。
func (s *PGStore) ListCompTasks(ctx context.Context, f CompTaskFilter) ([]CompTask, int, error) {
	where := "WHERE 1=1"
	args := make([]any, 0, 5)
	ai := 1
	if f.Status != "" {
		where += fmt.Sprintf(" AND status=$%d", ai)
		ai++
		args = append(args, f.Status)
	}
	if f.Source != "" {
		where += fmt.Sprintf(" AND source=$%d", ai)
		ai++
		args = append(args, f.Source)
	}
	if f.Priority != "" {
		where += fmt.Sprintf(" AND priority=$%d", ai)
		ai++
		args = append(args, f.Priority)
	}
	if f.AssigneeID != nil {
		where += fmt.Sprintf(" AND assignee_id=$%d", ai)
		ai++
		args = append(args, *f.AssigneeID)
	}
	if f.BizType != "" {
		where += fmt.Sprintf(" AND biz_type=$%d", ai)
		ai++
		args = append(args, f.BizType)
	}
	// 总数
	var total int
	cntSQL := "SELECT count(*) FROM compensation_tasks " + where
	if err := s.db.QueryRow(ctx, cntSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("comp: list count: %w", err)
	}
	// 分页
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = DefaultCompLimit
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)
	sql := fmt.Sprintf("SELECT %s FROM compensation_tasks %s ORDER BY id DESC LIMIT $%d OFFSET $%d",
		colCompTask, where, ai, ai+1)
	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("comp: list: %w", err)
	}
	defer rows.Close()
	out := make([]CompTask, 0, limit)
	for rows.Next() {
		t, err := scanCompTask(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("comp: list scan: %w", err)
		}
		out = append(out, *t)
	}
	return out, total, rows.Err()
}

// GetCompTask 按ID取。
func (s *PGStore) GetCompTask(ctx context.Context, id int64) (*CompTask, error) {
	sql := fmt.Sprintf("SELECT %s, audit_log FROM compensation_tasks WHERE id=$1", colCompTask)
	t, err := scanCompTaskWithAudit(s.db.QueryRow(ctx, sql, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCompTaskNotFound
		}
		return nil, fmt.Errorf("comp: get: %w", err)
	}
	return t, nil
}

// CreateCompTask 创建任务。
func (s *PGStore) CreateCompTask(ctx context.Context, t *CompTask) error {
	auditRaw, _ := json.Marshal(t.AuditLog)
	return s.db.QueryRow(ctx, `
		INSERT INTO compensation_tasks(source, biz_type, biz_id, failure_reason, priority, status,
			assignee_id, assignee_name, sla_deadline, max_retries, audit_log)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb)
		RETURNING id, created_at, updated_at`,
		t.Source, t.BizType, t.BizID, t.FailureReason, t.Priority, t.Status,
		t.AssigneeID, t.AssigneeName, t.SLADeadline, t.MaxRetries, string(auditRaw)).
		Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

// UpdateCompTask 更新任务。
func (s *PGStore) UpdateCompTask(ctx context.Context, t *CompTask) error {
	auditRaw, _ := json.Marshal(t.AuditLog)
	_, err := s.db.Exec(ctx, `
		UPDATE compensation_tasks SET source=$2, biz_type=$3, biz_id=$4, failure_reason=$5,
			priority=$6, status=$7, assignee_id=$8, assignee_name=$9,
			sla_deadline=$10, retry_count=$11, max_retries=$12,
			audit_log=$13::jsonb, updated_at=now()
		WHERE id=$1`,
		t.ID, t.Source, t.BizType, t.BizID, t.FailureReason,
		t.Priority, t.Status, t.AssigneeID, t.AssigneeName,
		t.SLADeadline, t.RetryCount, t.MaxRetries, string(auditRaw))
	if err != nil {
		return fmt.Errorf("comp: update: %w", err)
	}
	return nil
}

// ClaimCompTask 领取任务:OPEN→CLAIMED。
func (s *PGStore) ClaimCompTask(ctx context.Context, id, actorID int64, actorName string) error {
	now := time.Now()
	entry := AuditEntry{At: now, Actor: actorName, Action: "claim", Detail: "领取任务"}
	entryRaw, _ := json.Marshal([]AuditEntry{entry})
	tag, err := s.db.Exec(ctx, `
		UPDATE compensation_tasks
		SET status='CLAIMED', claimed_at=$2, claimed_by=$3, updated_at=$2,
		    audit_log = audit_log || $4::jsonb
		WHERE id=$1 AND status='OPEN'`, id, now, actorID, string(entryRaw))
	if err != nil {
		return fmt.Errorf("comp: claim: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrIllegalStatus
	}
	return nil
}

// TransferCompTask 转派任务:变更 assignee 并留审计。
func (s *PGStore) TransferCompTask(ctx context.Context, id, fromID, toID int64, toName, actorName string) error {
	now := time.Now()
	detail := fmt.Sprintf("转派: from %d to %d", fromID, toID)
	entry := AuditEntry{At: now, Actor: actorName, Action: "transfer", Detail: detail}
	entryRaw, _ := json.Marshal([]AuditEntry{entry})
	tag, err := s.db.Exec(ctx, `
		UPDATE compensation_tasks
		SET assignee_id=$2, assignee_name=$3, updated_at=now(),
		    audit_log = audit_log || $4::jsonb
		WHERE id=$1`, id, toID, toName, string(entryRaw))
	if err != nil {
		return fmt.Errorf("comp: transfer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCompTaskNotFound
	}
	return nil
}

// CloseCompTask 关闭任务:任意非CLOSED→CLOSED。
func (s *PGStore) CloseCompTask(ctx context.Context, id, actorID int64, actorName, reason string) error {
	now := time.Now()
	entry := AuditEntry{At: now, Actor: actorName, Action: "close", Detail: reason}
	entryRaw, _ := json.Marshal([]AuditEntry{entry})
	tag, err := s.db.Exec(ctx, `
		UPDATE compensation_tasks
		SET status='CLOSED', closed_at=$2, closed_by=$3, close_reason=$4,
		    updated_at=$2, audit_log = audit_log || $5::jsonb
		WHERE id=$1 AND status <> 'CLOSED'`, id, now, actorID, reason, string(entryRaw))
	if err != nil {
		return fmt.Errorf("comp: close: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrIllegalStatus
	}
	return nil
}

// IncrementRetry 重试计数+1。
func (s *PGStore) IncrementRetry(ctx context.Context, id int64) error {
	now := time.Now()
	entry := AuditEntry{At: now, Actor: "system", Action: "retry", Detail: "重试执行"}
	entryRaw, _ := json.Marshal([]AuditEntry{entry})
	_, err := s.db.Exec(ctx, `
		UPDATE compensation_tasks
		SET retry_count=retry_count+1, last_retry_at=$2, updated_at=$2,
		    audit_log = audit_log || $3::jsonb
		WHERE id=$1`, id, now, string(entryRaw))
	if err != nil {
		return fmt.Errorf("comp: increment retry: %w", err)
	}
	return nil
}

// BatchInsertCompTasks 批量插入(用于自动汇聚)。
func (s *PGStore) BatchInsertCompTasks(ctx context.Context, tasks []CompTask) error {
	for _, t := range tasks {
		auditRaw, _ := json.Marshal(t.AuditLog)
		_, err := s.db.Exec(ctx, `
			INSERT INTO compensation_tasks(source, biz_type, biz_id, failure_reason, priority, status,
				assignee_id, assignee_name, sla_deadline, max_retries, audit_log)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb)
			ON CONFLICT DO NOTHING`,
			t.Source, t.BizType, t.BizID, t.FailureReason, t.Priority, t.Status,
			t.AssigneeID, t.AssigneeName, t.SLADeadline, t.MaxRetries, string(auditRaw))
		if err != nil {
			return fmt.Errorf("comp: batch insert: %w", err)
		}
	}
	return nil
}
