package provision

import (
	"context"
	"database/sql"
	"encoding/json"
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

// ErrForeignKeyViolation 关联实体不存在(孤儿数据防护)。
var ErrForeignKeyViolation = errors.New("provision: foreign key violation")

// exists 校验单表存在性(provision_templates 无外键约束,关联完整性由本域应用层保证)。
func (s *PGStore) exists(ctx context.Context, table string, id int64) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM `+table+` WHERE id = $1)`, id).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("provision: check %s %d: %w", table, id, err)
	}
	return ok, nil
}

// ListTemplates 列出全部下发模板(含绑定套餐数 bound_offers,admin 可见性)。
func (s *PGStore) ListTemplates(ctx context.Context) ([]Template, error) {
	rows, err := s.db.Query(ctx, `
		SELECT t.id, t.legal_entity_id, t.code, t.name, t.content, t.version, t.status, t.updated_at,
		       COALESCE((SELECT count(*) FROM offer_provision_bindings b WHERE b.template_id = t.id), 0)
		FROM provision_templates t ORDER BY t.id`)
	if err != nil {
		return nil, fmt.Errorf("provision: list templates: %w", err)
	}
	defer rows.Close()
	out := make([]Template, 0)
	for rows.Next() {
		var t Template
		var raw []byte
		if err := rows.Scan(&t.ID, &t.LegalEntityID, &t.Code, &t.Name, &raw, &t.Version, &t.Status, &t.UpdatedAt, &t.BoundOffers); err != nil {
			return nil, fmt.Errorf("provision: scan template: %w", err)
		}
		if len(raw) > 0 && string(raw) != "null" {
			if err := json.Unmarshal(raw, &t.Content); err != nil {
				return nil, fmt.Errorf("provision: decode template content: %w", err)
			}
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// CreateTemplate 新建下发模板,返回自增 id。
// 校验 legal_entity_id 存在性,防止孤儿模板。
func (s *PGStore) CreateTemplate(ctx context.Context, t Template) (int64, error) {
	// 关联完整性校验
	if t.LegalEntityID > 0 {
		ok, err := s.exists(ctx, "legal_entities", t.LegalEntityID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("provision: legal entity %d: %w", t.LegalEntityID, ErrForeignKeyViolation)
		}
	}

	var id int64
	content := t.Content
	if content == nil {
		content = map[string]any{}
	}
	raw, err := json.Marshal(content)
	if err != nil {
		return 0, fmt.Errorf("provision: encode template content: %w", err)
	}
	err = s.db.QueryRow(ctx,
		`INSERT INTO provision_templates(legal_entity_id, code, name, content, version, status) VALUES($1,$2,$3,$4::jsonb,1,$5) RETURNING id`,
		t.LegalEntityID, t.Code, t.Name, string(raw), normalizedStatus(t.Status)).Scan(&id)
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
// 幂等(环节7重推进场景,原实现注释声称幂等但撞 task_no 唯一索引必报错):
// 同 task_no 已存在时复用既有任务,FAILED 则重置 PENDING 并留 RETRY 痕。
func (s *PGStore) CreateTask(ctx context.Context, t Task) (int64, error) {
	if t.TaskNo == "" {
		t.TaskNo = fmt.Sprintf("TASK-%d", time.Now().UnixNano())
	}
	if id, reused, err := s.reuseTask(ctx, t.TaskNo); reused || err != nil {
		return id, err
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

// reuseTask 同 task_no 已有任务时的幂等处置:FAILED→PENDING 重试留痕,其余原样复用。
func (s *PGStore) reuseTask(ctx context.Context, taskNo string) (int64, bool, error) {
	var id int64
	var status string
	err := s.db.QueryRow(ctx,
		`SELECT id, status FROM provision_tasks WHERE task_no = $1`, taskNo).Scan(&id, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("provision: reuse task: %w", err)
	}
	if status == "FAILED" {
		if err := s.RetryTask(ctx, id, 0); err != nil {
			return 0, false, fmt.Errorf("provision: reuse task retry: %w", err)
		}
	}
	return id, true, nil
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

// ListLogs 列出下发日志;taskID=0 返回全部,否则按任务过滤(不含指令/应答大字段,详情页另取)。
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

// GetLogDetail 日志详情聚合:日志(含指令/应答)+ 任务 + 订单(offer 名)+ 模板。
// 单查询 JOIN 软引用;订单/模板已删时相应维度留零值,不阻断详情。
func (s *PGStore) GetLogDetail(ctx context.Context, logID int64) (*LogDetail, error) {
	var d LogDetail
	var commands, deviceResp sql.NullString
	var orderNo, orderStatus, offerName, tplCode, tplName, tplStatus sql.NullString
	var tplVersion sql.NullInt32
	var content []byte
	err := s.db.QueryRow(ctx, `
		SELECT l.id, l.task_id, l.resource_id, COALESCE(l.resource_code, ''),
		       l.template_id, COALESCE(l.template_code, ''), l.result, l.retries,
		       COALESCE(l.commands, ''), COALESCE(l.device_response, ''), l.driver, l.created_at,
		       t.id, t.task_no, t.order_id, t.stage_event, t.lo_account_id, t.template_id, t.status,
		       o.order_no, o.status, COALESCE(po.name, ''),
		       t2.code, t2.name, t2.status, t2.version, COALESCE(t2.content, '{}')
		FROM provision_logs l
		JOIN provision_tasks t ON t.id = l.task_id
		LEFT JOIN orders o ON o.id = t.order_id
		LEFT JOIN product_offers po ON po.id = o.offer_id
		LEFT JOIN provision_templates t2 ON t2.id = l.template_id
		WHERE l.id = $1`, logID).Scan(
		&d.Log.ID, &d.Log.TaskID, &d.Log.ResourceID, &d.Log.ResourceCode,
		&d.Log.TemplateID, &d.Log.TemplateCode, &d.Log.Result, &d.Log.Retries,
		&commands, &deviceResp, &d.Log.Driver, &d.Log.CreatedAt,
		&d.Task.ID, &d.Task.TaskNo, &d.Task.OrderID, &d.Task.StageEvent, &d.Task.LoAccountID, &d.Task.TemplateID, &d.Task.Status,
		&orderNo, &orderStatus, &offerName,
		&tplCode, &tplName, &tplStatus, &tplVersion, &content)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrLogNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("provision: log detail: %w", err)
	}
	d.Log.Commands, d.Log.DeviceResp = commands.String, deviceResp.String
	d.Order = LogOrderInfo{OrderNo: orderNo.String, Status: orderStatus.String, OfferName: offerName.String}
	d.Template = LogTemplateInfo{Code: tplCode.String, Name: tplName.String, Status: tplStatus.String, Version: tplVersion.Int32}
	if len(content) > 0 && string(content) != "{}" {
		_ = json.Unmarshal(content, &d.Template.Content)
	}
	return &d, nil
}

// AppendLog 追加下发日志,返回自增 id。
func (s *PGStore) AppendLog(ctx context.Context, l Log) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO provision_logs(task_id, resource_id, resource_code, template_id, template_code, result, retries, commands, device_response, driver)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`,
		l.TaskID, l.ResourceID, l.ResourceCode, l.TemplateID, l.TemplateCode, l.Result, l.Retries, l.Commands, l.DeviceResp, l.Driver).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("provision: append log: %w", err)
	}
	return id, nil
}
