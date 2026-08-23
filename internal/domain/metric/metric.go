package metric

import (
	"context"
	"time"
)

// Status 指标目录状态。
const (
	StatusDraft     = "DRAFT"
	StatusActive    = "ACTIVE"
	StatusDeprecated = "DEPRECATED"
	StatusArchived  = "ARCHIVED"
)

// RefreshCadence 刷新频率。
const (
	CadenceRealtime = "REALTIME"
	CadenceHourly   = "HOURLY"
	CadenceDaily    = "DAILY"
	CadenceWeekly   = "WEEKLY"
	CadenceMonthly  = "MONTHLY"
	CadenceOnDemand = "ON_DEMAND"
)

// Severity 质量规则严重级别。
const (
	SeverityInfo     = "INFO"
	SeverityWarn     = "WARN"
	SeverityCritical = "CRITICAL"
)

// CatalogEntry 指标目录单条记录。
type CatalogEntry struct {
	ID             int64     `json:"id"`
	Key            string    `json:"key"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Formula        string    `json:"formula"`
	Unit           string    `json:"unit"`
	Dimensions     []string  `json:"dimensions"`
	RefreshCadence string    `json:"refreshCadence"`
	Owner          string    `json:"owner"`
	Domain         string    `json:"domain"`
	Status         string    `json:"status"`
	Version        int       `json:"version"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// QualityRule 数据质量规则。
type QualityRule struct {
	ID            int64     `json:"id"`
	RuleKey       string    `json:"ruleKey"`
	Name          string    `json:"name"`
	Scope         string    `json:"scope"`
	CheckExpr      string    `json:"checkExpr"`
	ThresholdExpr string    `json:"thresholdExpr"`
	Severity      string    `json:"severity"`
	Owner         string    `json:"owner"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// QualityViolation 单次扫描出的质量违规明细，供补偿任务中心登记。
type QualityViolation struct {
	RuleKey  string  `json:"ruleKey"`
	Severity string  `json:"severity"`
	Scope    string  `json:"scope"`
	Detail   string  `json:"detail"`
	Observed float64 `json:"observed"`
}

// Service 指标目录与质量规则领域入口。
type Service interface {
	// Catalog CRUD
	ListCatalog(ctx context.Context, status string) ([]CatalogEntry, error)
	GetCatalogByKey(ctx context.Context, key string) (*CatalogEntry, error)
	UpsertCatalog(ctx context.Context, e CatalogEntry) (*CatalogEntry, error)
	DeprecateCatalog(ctx context.Context, key string) (*CatalogEntry, error)

	// Quality Rules CRUD
	ListQualityRules(ctx context.Context, enabledOnly bool) ([]QualityRule, error)
	UpsertQualityRule(ctx context.Context, r QualityRule) (*QualityRule, error)
	DisableQualityRule(ctx context.Context, ruleKey string) error

	// Quality Scan：扫描启用的规则，返回 QualityViolation。
	// 后续由 call-site 把 violations 投递到 compensation_tasks 中心。
	ScanQuality(ctx context.Context, scope string) ([]QualityViolation, error)
}