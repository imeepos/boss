package metric

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func TestMemoryStoreCatalogLifecycle(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	if _, err := s.GetCatalogByKey(ctx, "portUtilization"); err == nil {
		t.Fatalf("expected ErrNotFound, got nil")
	}
	entry, err := s.UpsertCatalog(ctx, CatalogEntry{Key: "portUtilization", Name: "端口利用率", Owner: "sysadmin", Domain: "OSS", Status: StatusActive})
	if err != nil {
		t.Fatal(err)
	}
	if entry.Version != 1 {
		t.Fatalf("version=%d, want 1", entry.Version)
	}
	entry2, err := s.UpsertCatalog(ctx, CatalogEntry{Key: "portUtilization", Name: "端口利用率", Status: StatusActive})
	if err != nil {
		t.Fatal(err)
	}
	if entry2.Version != 2 {
		t.Fatalf("version=%d, want 2 after upsert", entry2.Version)
	}
	if _, err := s.DeprecateCatalog(ctx, "portUtilization"); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetCatalogByKey(ctx, "portUtilization")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusDeprecated {
		t.Fatalf("status=%s, want DEPRECATED", got.Status)
	}
}

func TestMemoryStoreListCatalog(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	_, _ = s.UpsertCatalog(ctx, CatalogEntry{Key: "a", Name: "A", Status: StatusActive})
	_, _ = s.UpsertCatalog(ctx, CatalogEntry{Key: "b", Name: "B", Status: StatusDraft})
	_, _ = s.UpsertCatalog(ctx, CatalogEntry{Key: "c", Name: "C", Status: StatusActive})
	active, err := s.ListCatalog(ctx, StatusActive)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 2 {
		t.Fatalf("active=%d, want 2", len(active))
	}
}

func TestMemoryStoreQualityRules(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	_, err := s.UpsertQualityRule(ctx, QualityRule{RuleKey: "completeness_orders", Name: "订单完整性", Scope: "orders", CheckExpr: "COUNT(*) > 0", Severity: SeverityCritical, Owner: "ops", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	rules, err := s.ListQualityRules(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || !rules[0].Enabled {
		t.Fatalf("rules=%+v", rules)
	}
	if err := s.DisableQualityRule(ctx, "completeness_orders"); err != nil {
		t.Fatal(err)
	}
	rules, _ = s.ListQualityRules(ctx, true)
	if len(rules) != 0 {
		t.Fatalf("rules=%+v after disable", rules)
	}
}

func TestMemoryStoreScanQuality(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	_, _ = s.UpsertQualityRule(ctx, QualityRule{RuleKey: "r1", Name: "R1", Scope: "orders", CheckExpr: "x>0", Severity: SeverityCritical, Enabled: true})
	_, _ = s.UpsertQualityRule(ctx, QualityRule{RuleKey: "r2", Name: "R2", Scope: "payments", CheckExpr: "x>0", Severity: SeverityWarn, Enabled: true})
	violations, err := s.ScanQuality(ctx, "orders")
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 || violations[0].RuleKey != "r1" {
		t.Fatalf("violations=%+v", violations)
	}
}

func TestPGStoreCatalogUpsert(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := NewPGStore(pool)
	ctx := context.Background()

	now := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	dimsJSON := []byte(`["region"]`)
	pool.ExpectQuery(`SELECT id, version FROM metric_catalog WHERE key=\$1`).
		WithArgs("k").
		WillReturnRows(pool.NewRows([]string{"id", "version"}).AddRow(int64(0), int(0)))
	pool.ExpectExec(`INSERT INTO metric_catalog`).
		WithArgs("k", "K", "desc", "f", "u", dimsJSON, CadenceDaily, "owner", "DOM", StatusActive).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	pool.ExpectQuery(`SELECT id, key, name, description, formula, unit, dimensions, refresh_cadence, owner, domain, status, version, created_at, updated_at FROM metric_catalog WHERE key=\$1`).
		WithArgs("k").
		WillReturnRows(pool.NewRows([]string{"id", "key", "name", "description", "formula", "unit", "dimensions", "refresh_cadence", "owner", "domain", "status", "version", "created_at", "updated_at"}).
			AddRow(int64(1), "k", "K", "desc", "f", "u", dimsJSON, CadenceDaily, "owner", "DOM", StatusActive, 1, now, now))

	if _, err := store.UpsertCatalog(ctx, CatalogEntry{Key: "k", Name: "K", Description: "desc", Formula: "f", Unit: "u", Dimensions: []string{"region"}, RefreshCadence: CadenceDaily, Owner: "owner", Domain: "DOM", Status: StatusActive}); err != nil {
		t.Fatal(err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPGStoreDeprecateCatalogNotFound(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := NewPGStore(pool)
	ctx := context.Background()
	pool.ExpectExec(`UPDATE metric_catalog SET status=\$2`).
		WithArgs("missing", StatusDeprecated).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	if _, err := store.DeprecateCatalog(ctx, "missing"); err != ErrNotFound {
		t.Fatalf("err=%v, want ErrNotFound", err)
	}
}

func TestPGStoreUpsertCatalogRequiresKeyName(t *testing.T) {
	store := NewPGStore(nil)
	if _, err := store.UpsertCatalog(context.Background(), CatalogEntry{}); err == nil {
		t.Fatal("expected error on empty key/name")
	}
}

func TestPGStoreUpsertRuleRequiresFields(t *testing.T) {
	store := NewPGStore(nil)
	if _, err := store.UpsertQualityRule(context.Background(), QualityRule{}); err == nil {
		t.Fatal("expected error on missing fields")
	}
}

func TestPGStoreDisableRuleNotFound(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := NewPGStore(pool)
	ctx := context.Background()
	pool.ExpectExec(`UPDATE metric_quality_rules SET enabled=FALSE`).
		WithArgs("missing").
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	if err := store.DisableQualityRule(ctx, "missing"); err != ErrNotFound {
		t.Fatalf("err=%v, want ErrNotFound", err)
	}
}

func TestPGStoreUpsertCatalogUpdatePath(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := NewPGStore(pool)
	ctx := context.Background()
	pool.ExpectQuery(`SELECT id, version FROM metric_catalog WHERE key=\$1`).
		WithArgs("k").
		WillReturnRows(pool.NewRows([]string{"id", "version"}).AddRow(int64(5), int(2)))
	pool.ExpectExec(`UPDATE metric_catalog SET name=\$2`).
		WithArgs("k", "K2", "", "", "", []byte("null"), CadenceDaily, "", "", StatusActive).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	pool.ExpectQuery(`SELECT id, key, name, description, formula, unit, dimensions, refresh_cadence, owner, domain, status, version, created_at, updated_at FROM metric_catalog WHERE key=\$1`).
		WithArgs("k").
		WillReturnRows(pool.NewRows([]string{"id", "key", "name", "description", "formula", "unit", "dimensions", "refresh_cadence", "owner", "domain", "status", "version", "created_at", "updated_at"}).
			AddRow(int64(5), "k", "K2", "", "", "", []byte("[]"), CadenceDaily, "", "", StatusActive, 3, time.Now(), time.Now()))
	if _, err := store.UpsertCatalog(ctx, CatalogEntry{Key: "k", Name: "K2", Status: StatusActive}); err != nil {
		t.Fatal(err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPGStoreUpsertRuleUpdatePath(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := NewPGStore(pool)
	ctx := context.Background()
	pool.ExpectQuery(`SELECT id FROM metric_quality_rules WHERE rule_key=\$1`).
		WithArgs("r").
		WillReturnRows(pool.NewRows([]string{"id"}).AddRow(int64(9)))
	pool.ExpectExec(`UPDATE metric_quality_rules SET name=\$2`).
		WithArgs("r", "R", "s", "c", "t", SeverityWarn, "", false).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	pool.ExpectQuery(`SELECT id, rule_key, name, scope, check_expr, threshold_expr, severity, owner, enabled, created_at, updated_at FROM metric_quality_rules WHERE rule_key=\$1`).
		WithArgs("r").
		WillReturnRows(pool.NewRows([]string{"id", "rule_key", "name", "scope", "check_expr", "threshold_expr", "severity", "owner", "enabled", "created_at", "updated_at"}).
			AddRow(int64(9), "r", "R", "s", "c", "t", SeverityWarn, "", true, time.Now(), time.Now()))
	if _, err := store.UpsertQualityRule(ctx, QualityRule{RuleKey: "r", Name: "R", Scope: "s", CheckExpr: "c", ThresholdExpr: "t"}); err != nil {
		t.Fatal(err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
