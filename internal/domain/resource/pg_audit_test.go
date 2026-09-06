package resource

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// auditCountsFixture 检查码 → 违规行数(测试夹具)。
var auditCountsFixture = map[string]int64{
	"PORT_SPLITTER_MISSING":     2,
	"SPLITTER_UPSTREAM_MISSING": 1,
	"USED_PORT_NO_QUAD_LINK":    1,
	"RESERVED_PORT_STALE":       0,
	"RESOURCE_CODE_BAD":         2,
	"PORT_CODE_BAD":             0,
}

// expectAuditChecks 为每个(或过滤后的)检查注册 mock 查询。
func expectAuditChecks(t *testing.T, mock pgxmock.PgxPoolIface, checks []auditCheck, staleHours int) {
	t.Helper()
	for _, ck := range checks {
		eq := mock.ExpectQuery(regexp.QuoteMeta(ck.sql + " LIMIT 50"))
		if ck.staleArg {
			eq = eq.WithArgs(staleHours)
		}
		rows := pgxmock.NewRows([]string{"total", "id", "code", "detail"})
		for i := int64(1); i <= auditCountsFixture[ck.code]; i++ {
			rows.AddRow(auditCountsFixture[ck.code], i, "CODE-X", "ctx")
		}
		eq.WillReturnRows(rows)
	}
}

func TestAuditInventoryAll(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	expectAuditChecks(t, mock, auditChecks, 48)
	rep, err := NewPGStore(mock).AuditInventory(context.Background(), AuditOptions{})
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	// 分类计数:ownership=3 state=1 coding=2;全量 total=6。
	if rep.Counts[0].Category != AuditCatOwnership || rep.Counts[0].Count != 3 {
		t.Fatalf("ownership count = %+v", rep.Counts)
	}
	if rep.Counts[1].Category != AuditCatState || rep.Counts[1].Count != 1 {
		t.Fatalf("state count = %+v", rep.Counts)
	}
	if rep.Counts[2].Category != AuditCatCoding || rep.Counts[2].Count != 2 {
		t.Fatalf("coding count = %+v", rep.Counts)
	}
	if rep.Total != 6 {
		t.Fatalf("total = %d, want 6", rep.Total)
	}
	if len(rep.Items) != 6 {
		t.Fatalf("items = %d, want 6(零计数检查无明细)", len(rep.Items))
	}
	if rep.StaleHours != 48 {
		t.Fatalf("staleHours = %d, want 48", rep.StaleHours)
	}
}

func TestAuditInventoryCategoryFilter(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	var stateChecks []auditCheck
	for _, ck := range auditChecks {
		if ck.category == AuditCatState {
			stateChecks = append(stateChecks, ck)
		}
	}
	expectAuditChecks(t, mock, stateChecks, 24)
	rep, err := NewPGStore(mock).AuditInventory(context.Background(),
		AuditOptions{Category: AuditCatState, ReservedStaleHours: 24})
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if len(rep.Counts) != 1 || rep.Counts[0].Category != AuditCatState || rep.Counts[0].Count != 1 {
		t.Fatalf("filtered counts = %+v", rep.Counts)
	}
	if rep.Total != 1 {
		t.Fatalf("filtered total = %d, want 1", rep.Total)
	}
}

func TestAuditInventoryQueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery(regexp.QuoteMeta(auditChecks[0].sql + " LIMIT 50")).
		WillReturnError(errors.New("db down"))
	if _, err := NewPGStore(mock).AuditInventory(context.Background(), AuditOptions{}); err == nil {
		t.Fatal("期望单检查失败整轮报错,实际 nil")
	}
}
