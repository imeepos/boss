package provision

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// dbtx 是 PGStore 依赖的最小数据库接口;*pgxpool.Pool 天然满足,单测用 pgxmock 注入。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore 是 ProvisionService 接口的 PostgreSQL 实现(阶段7)。
type PGStore struct {
	db dbtx
}

// NewPGStore 构造 PGStore;db 传 *pgxpool.Pool 或测试 mock。
func NewPGStore(db dbtx) *PGStore {
	return &PGStore{db: db}
}

// ListTemplates 列出全部下发模板。
func (s *PGStore) ListTemplates(ctx context.Context) ([]Template, error) {
	rows, err := s.db.Query(ctx, `SELECT id, legal_entity_id, code, name FROM provision_templates ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("provision: list templates: %w", err)
	}
	defer rows.Close()
	out := make([]Template, 0)
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.ID, &t.LegalEntityID, &t.Code, &t.Name); err != nil {
			return nil, fmt.Errorf("provision: scan template: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// CreateTemplate 新建下发模板,返回自增 id。
func (s *PGStore) CreateTemplate(ctx context.Context, t Template) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO provision_templates(legal_entity_id, code, name) VALUES($1,$2,$3) RETURNING id`,
		t.LegalEntityID, t.Code, t.Name).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("provision: create template: %w", err)
	}
	return id, nil
}

// ListTasks 列出全部下发任务。
func (s *PGStore) ListTasks(ctx context.Context) ([]Task, error) {
	rows, err := s.db.Query(ctx, `SELECT id, task_no, order_id, stage_event, lo_account_id, template_id, status FROM provision_tasks ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("provision: list tasks: %w", err)
	}
	defer rows.Close()
	out := make([]Task, 0)
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.TaskNo, &t.OrderID, &t.StageEvent, &t.LoAccountID, &t.TemplateID, &t.Status); err != nil {
			return nil, fmt.Errorf("provision: scan task: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// CreateTask 新建下发任务,返回自增 id;task_no 为空时自动生成。
func (s *PGStore) CreateTask(ctx context.Context, t Task) (int64, error) {
	if t.TaskNo == "" {
		t.TaskNo = fmt.Sprintf("TASK-%d", time.Now().UnixNano())
	}
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO provision_tasks(task_no, order_id, stage_event, lo_account_id, template_id, status)
		 VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,
		t.TaskNo, t.OrderID, t.StageEvent, t.LoAccountID, t.TemplateID, t.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("provision: create task: %w", err)
	}
	return id, nil
}

// GetTaskByNo 按外部 task_no 寻址;未命中返回 ErrTaskNotFound。
func (s *PGStore) GetTaskByNo(ctx context.Context, taskNo string) (*Task, error) {
	var t Task
	err := s.db.QueryRow(ctx, `SELECT id, task_no, order_id, stage_event, lo_account_id, template_id, status FROM provision_tasks WHERE task_no = $1`, taskNo).
		Scan(&t.ID, &t.TaskNo, &t.OrderID, &t.StageEvent, &t.LoAccountID, &t.TemplateID, &t.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("provision: get task by no: %w", err)
	}
	return &t, nil
}

// ListLogs 列出下发日志;taskID=0 返回全部,否则按任务过滤。
func (s *PGStore) ListLogs(ctx context.Context, taskID int64) ([]Log, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, task_id, resource_id, COALESCE(resource_code, ''), template_id, COALESCE(template_code, ''), result, retries, created_at
		FROM provision_logs WHERE ($1 = 0 OR task_id = $1) ORDER BY id`, taskID)
	if err != nil {
		return nil, fmt.Errorf("provision: list logs: %w", err)
	}
	defer rows.Close()
	out := make([]Log, 0)
	for rows.Next() {
		var l Log
		if err := rows.Scan(&l.ID, &l.TaskID, &l.ResourceID, &l.ResourceCode, &l.TemplateID, &l.TemplateCode, &l.Result, &l.Retries, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("provision: scan log: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// AppendLog 追加下发日志,返回自增 id。
func (s *PGStore) AppendLog(ctx context.Context, l Log) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO provision_logs(task_id, resource_id, resource_code, template_id, template_code, result, retries)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		l.TaskID, l.ResourceID, l.ResourceCode, l.TemplateID, l.TemplateCode, l.Result, l.Retries).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("provision: append log: %w", err)
	}
	return id, nil
}
