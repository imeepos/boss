package metric

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// dbtx 是 PGStore 依赖的最小数据库接口;*pgxpool.Pool 天然满足,单测用 pgxmock 注入。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore PostgreSQL 实现;通过 dbtx 注入。
type PGStore struct {
	db dbtx
}

// NewPGStore 构造 PG 存储。
func NewPGStore(db dbtx) *PGStore {
	return &PGStore{db: db}
}

const pgCatalogSelect = `SELECT id, key, name, description, formula, unit, dimensions, refresh_cadence, owner, domain, status, version, created_at, updated_at FROM metric_catalog`

const pgRuleSelect = `SELECT id, rule_key, name, scope, check_expr, threshold_expr, severity, owner, enabled, created_at, updated_at FROM metric_quality_rules`

func (s *PGStore) ListCatalog(ctx context.Context, status string) ([]CatalogEntry, error) {
	if status == "" {
		rows, err := s.db.Query(ctx, pgCatalogSelect+" ORDER BY key ASC")
		if err != nil {
			return nil, fmt.Errorf("metric: list catalog: %w", err)
		}
		return scanCatalog(rows)
	}
	rows, err := s.db.Query(ctx, pgCatalogSelect+" WHERE status=$1 ORDER BY key ASC", status)
	if err != nil {
		return nil, fmt.Errorf("metric: list catalog by status: %w", err)
	}
	return scanCatalog(rows)
}

func (s *PGStore) GetCatalogByKey(ctx context.Context, key string) (*CatalogEntry, error) {
	rows, err := s.db.Query(ctx, pgCatalogSelect+" WHERE key=$1", key)
	if err != nil {
		return nil, fmt.Errorf("metric: get catalog by key: %w", err)
	}
	items, err := scanCatalog(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return &items[0], nil
}

func (s *PGStore) UpsertCatalog(ctx context.Context, e CatalogEntry) (*CatalogEntry, error) {
	if e.Key == "" || e.Name == "" {
		return nil, errors.New("metric: key and name are required")
	}
	if e.Status == "" {
		e.Status = StatusDraft
	}
	if e.RefreshCadence == "" {
		e.RefreshCadence = CadenceDaily
	}
	dims, err := json.Marshal(e.Dimensions)
	if err != nil {
		return nil, fmt.Errorf("metric: marshal dimensions: %w", err)
	}
	var existingID int64
	var existingVersion int
	if e.Status == StatusActive {
		_ = s.db.QueryRow(ctx, "SELECT id, version FROM metric_catalog WHERE key=$1", e.Key).Scan(&existingID, &existingVersion)
	}
	if existingID > 0 {
		_, err := s.db.Exec(ctx, `UPDATE metric_catalog SET name=$2,description=$3,formula=$4,unit=$5,dimensions=$6,refresh_cadence=$7,owner=$8,domain=$9,status=$10,version=version+1,updated_at=now() WHERE key=$1`,
			e.Key, e.Name, e.Description, e.Formula, e.Unit, dims, e.RefreshCadence, e.Owner, e.Domain, e.Status)
		if err != nil {
			return nil, fmt.Errorf("metric: upsert catalog update: %w", err)
		}
	} else {
		_, err := s.db.Exec(ctx, `INSERT INTO metric_catalog (key,name,description,formula,unit,dimensions,refresh_cadence,owner,domain,status,version) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,1)`,
			e.Key, e.Name, e.Description, e.Formula, e.Unit, dims, e.RefreshCadence, e.Owner, e.Domain, e.Status)
		if err != nil {
			return nil, fmt.Errorf("metric: upsert catalog insert: %w", err)
		}
	}
	return s.GetCatalogByKey(ctx, e.Key)
}

func (s *PGStore) DeprecateCatalog(ctx context.Context, key string) (*CatalogEntry, error) {
	tag, err := s.db.Exec(ctx, `UPDATE metric_catalog SET status=$2,updated_at=now() WHERE key=$1`, key, StatusDeprecated)
	if err != nil {
		return nil, fmt.Errorf("metric: deprecate: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.GetCatalogByKey(ctx, key)
}

func (s *PGStore) ListQualityRules(ctx context.Context, enabledOnly bool) ([]QualityRule, error) {
	if enabledOnly {
		rows, err := s.db.Query(ctx, pgRuleSelect+" WHERE enabled=TRUE ORDER BY rule_key ASC")
		if err != nil {
			return nil, fmt.Errorf("metric: list rules enabled: %w", err)
		}
		return scanRules(rows)
	}
	rows, err := s.db.Query(ctx, pgRuleSelect+" ORDER BY rule_key ASC")
	if err != nil {
		return nil, fmt.Errorf("metric: list rules: %w", err)
	}
	return scanRules(rows)
}

func (s *PGStore) UpsertQualityRule(ctx context.Context, r QualityRule) (*QualityRule, error) {
	if r.RuleKey == "" || r.Name == "" || r.Scope == "" {
		return nil, errors.New("metric: rule_key, name, scope required")
	}
	if r.Severity == "" {
		r.Severity = SeverityWarn
	}
	var existingID int64
	_ = s.db.QueryRow(ctx, "SELECT id FROM metric_quality_rules WHERE rule_key=$1", r.RuleKey).Scan(&existingID)
	if existingID > 0 {
		_, err := s.db.Exec(ctx, `UPDATE metric_quality_rules SET name=$2,scope=$3,check_expr=$4,threshold_expr=$5,severity=$6,owner=$7,enabled=$8,updated_at=now() WHERE rule_key=$1`,
			r.RuleKey, r.Name, r.Scope, r.CheckExpr, r.ThresholdExpr, r.Severity, r.Owner, r.Enabled)
		if err != nil {
			return nil, fmt.Errorf("metric: upsert rule update: %w", err)
		}
	} else {
		_, err := s.db.Exec(ctx, `INSERT INTO metric_quality_rules (rule_key,name,scope,check_expr,threshold_expr,severity,owner,enabled) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			r.RuleKey, r.Name, r.Scope, r.CheckExpr, r.ThresholdExpr, r.Severity, r.Owner, r.Enabled)
		if err != nil {
			return nil, fmt.Errorf("metric: upsert rule insert: %w", err)
		}
	}
	rows, err := s.db.Query(ctx, pgRuleSelect+" WHERE rule_key=$1", r.RuleKey)
	if err != nil {
		return nil, fmt.Errorf("metric: reload rule: %w", err)
	}
	rules, err := scanRules(rows)
	if err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		return nil, ErrNotFound
	}
	return &rules[0], nil
}

func (s *PGStore) DisableQualityRule(ctx context.Context, ruleKey string) error {
	tag, err := s.db.Exec(ctx, `UPDATE metric_quality_rules SET enabled=FALSE,updated_at=now() WHERE rule_key=$1`, ruleKey)
	if err != nil {
		return fmt.Errorf("metric: disable rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ScanQuality 占位实现：返回内建指标的最小自检，不直接跑 CHECK_EXPR 中的 SQL。
// CHECK_EXPR 通过规则定义沉淀，运行时由 ops 工具或独立 worker 解释执行，
// 本域不引入动态 SQL 执行面以避免越权。
func (s *PGStore) ScanQuality(ctx context.Context, scope string) ([]QualityViolation, error) {
	rules, err := s.ListQualityRules(ctx, true)
	if err != nil {
		return nil, err
	}
	out := make([]QualityViolation, 0, len(rules))
	for _, r := range rules {
		if scope != "" && r.Scope != scope {
			continue
		}
		// 自检占位：始终返回一条 INFO（enabled=true），由下游 worker 解析 check_expr 后再投递真实违规。
		out = append(out, QualityViolation{RuleKey: r.RuleKey, Severity: SeverityInfo, Scope: r.Scope, Detail: "scan_pending", Observed: 0})
	}
	return out, nil
}

// ErrNotFound 记录不存在。
var ErrNotFound = errors.New("metric: not found")

func scanCatalog(rows pgx.Rows) ([]CatalogEntry, error) {
	defer rows.Close()
	out := make([]CatalogEntry, 0)
	for rows.Next() {
		var e CatalogEntry
		var dimsJSON []byte
		if err := rows.Scan(&e.ID, &e.Key, &e.Name, &e.Description, &e.Formula, &e.Unit, &dimsJSON, &e.RefreshCadence, &e.Owner, &e.Domain, &e.Status, &e.Version, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("metric: scan catalog: %w", err)
		}
		if len(dimsJSON) > 0 {
			if err := json.Unmarshal(dimsJSON, &e.Dimensions); err != nil {
				return nil, fmt.Errorf("metric: unmarshal dimensions: %w", err)
			}
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("metric: iter catalog: %w", err)
	}
	return out, nil
}

func scanRules(rows pgx.Rows) ([]QualityRule, error) {
	defer rows.Close()
	out := make([]QualityRule, 0)
	for rows.Next() {
		var r QualityRule
		if err := rows.Scan(&r.ID, &r.RuleKey, &r.Name, &r.Scope, &r.CheckExpr, &r.ThresholdExpr, &r.Severity, &r.Owner, &r.Enabled, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("metric: scan rule: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("metric: iter rule: %w", err)
	}
	return out, nil
}