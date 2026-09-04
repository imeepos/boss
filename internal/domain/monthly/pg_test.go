package monthly

// PG 层单测(pgxmock):幂等 upsert、白名单/月份拒绝、导入行级错误、导出、汇总 KPI 分母 0。

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

// regionRows 预置活跃区域查询结果(Anilao/Atlag)。
func regionRows(mock pgxmock.PgxPoolIface) {
	rows := mock.NewRows([]string{"region"}).AddRow("Anilao").AddRow("Atlag")
	mock.ExpectQuery(`SELECT region FROM monthly_regions WHERE active`).WillReturnRows(rows)
}

// expectUpsertAnilao 预置 Anilao 行 upsert(断言 SQL 含 PK 冲突子句)。
func expectUpsertAnilao(mock pgxmock.PgxPoolIface) {
	mock.ExpectExec(`INSERT INTO monthly_user_revenue .+ ON CONFLICT \(month, region\) DO UPDATE SET opening_active=EXCLUDED`).
		WithArgs("2026-08", "Anilao", int64(518), int64(34), int64(15), int64(0), int64(568645), int64(45492), int64(51000), int64(12283), int64(3071)).
		WillReturnResult(pgconn.NewCommandTag("INSERT 0 1"))
}

// expectUpsertAtlag 预置 Atlag 行 upsert。
func expectUpsertAtlag(mock pgxmock.PgxPoolIface) {
	mock.ExpectExec(`INSERT INTO monthly_user_revenue .+ ON CONFLICT \(month, region\) DO UPDATE SET opening_active=EXCLUDED`).
		WithArgs("2026-08", "Atlag", int64(438), int64(31), int64(9), int64(0), int64(480186), int64(38415), int64(46500), int64(10372), int64(2593)).
		WillReturnResult(pgconn.NewCommandTag("INSERT 0 1"))
}

// sampleCSV 两行用户与收入样本(与模板首两行同值)。
func sampleCSV(t *testing.T) []byte {
	t.Helper()
	return fixtureCSV("2026-08,Anilao,518,34,15,0,568645,45492,51000,12283,3071", "2026-08,Atlag,438,31,9,0,480186,38415,46500,10372,2593")
}

// TestImportCSV_Idempotent 二次导入同文件:每行仍走同一条 PK 冲突 upsert,
// 库内行数不增(mock 断言 SQL 形状与逐行调用次数;两轮均 imported=2)。
func TestImportCSV_Idempotent(t *testing.T) {
	for round := 0; round < 2; round++ {
		mock, _ := pgxmock.NewPool()
		regionRows(mock)
		expectUpsertAnilao(mock)
		expectUpsertAtlag(mock)
		s := NewPGStore(mock)
		res, err := s.ImportCSV(context.Background(), TableUserRevenue, sampleCSV(t))
		if err != nil {
			t.Fatal(err)
		}
		if res.Total != 2 || res.Imported != 2 || res.Failed != 0 || len(res.Errors) != 0 {
			t.Fatalf("round %d: res=%+v", round, res)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("round %d: %v", round, err)
		}
		mock.Close()
	}
}

// TestImportCSV_RowErrors 非法月份/白名单外区域逐行拒绝,行号=文件行(表头=1)。
func TestImportCSV_RowErrors(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	regionRows(mock)
	s := NewPGStore(mock)
	data := fixtureCSV("2026-13,Anilao,1,2,3,4,5,6,7,8,9", "2026-08,Bagong Bayan,1,2,3,4,5,6,7,8,9")
	res, err := s.ImportCSV(context.Background(), TableUserRevenue, data)
	if err != nil {
		t.Fatal(err)
	}
	if res.Imported != 0 || res.Failed != 2 {
		t.Fatalf("res=%+v", res)
	}
	if res.Errors[0].Line != 2 || res.Errors[1].Line != 3 {
		t.Fatalf("lines=%+v", res.Errors)
	}
	if !bytes.Contains([]byte(res.Errors[1].Reason), []byte("白名单")) {
		t.Fatalf("reason=%q", res.Errors[1].Reason)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestUpsertWhitelistReject 单行 upsert:区域不在白名单 / 月份非法 → ErrInvalidInput。
func TestUpsertWhitelistReject(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`SELECT active FROM monthly_regions WHERE region = \$1`).WithArgs("Nowhere").WillReturnError(pgx.ErrNoRows)
	s := NewPGStore(mock)
	err := s.UpsertUserRevenue(context.Background(), UserRevenue{Month: "2026-08", Region: "Nowhere"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
	if err := s.UpsertUserRevenue(context.Background(), UserRevenue{Month: "2026-13", Region: "Anilao"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("bad month want ErrInvalidInput, got %v", err)
	}
}

// TestExportCSV 导出按 month 过滤、排序 month,region;输出含 BOM/CRLF 并可回读导入。
func TestExportCSV(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	rows := mock.NewRows([]string{"month", "region", "opening_active", "new_users", "churned_users", "adjusted_users", "broadband_revenue", "value_added_revenue", "onetime_charge", "discount_amount", "refund_reversal"}).
		AddRow("2026-08", "Anilao", int64(518), int64(34), int64(15), int64(0), int64(568645), int64(45492), int64(51000), int64(12283), int64(3071))
	mock.ExpectQuery(`FROM monthly_user_revenue WHERE \(\$1 = '' OR month = \$1\) ORDER BY month, region`).
		WithArgs("2026-08").WillReturnRows(rows)
	s := NewPGStore(mock)
	out, err := s.ExportCSV(context.Background(), TableUserRevenue, "2026-08")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out[:3], bom) {
		t.Fatal("want BOM")
	}
	if !bytes.Contains(out, []byte("\r\n")) {
		t.Fatal("want CRLF")
	}
	back, err := parseCSVTable(TableUserRevenue, out)
	if err != nil || len(back) != 1 || back[0][2] != "518" {
		t.Fatalf("roundtrip=%v err=%v", back, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestSummary_KPI 汇总率:分母 0 → nil(装机申请=0 → 及时完工率 null)。
func TestSummary_KPI(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	cols := []string{"closing", "revenue", "requests", "ontime", "deployed", "active", "invoiced", "collected"}
	mock.ExpectQuery(`SELECT COALESCE\(\(SELECT SUM\(closing_active\) FROM monthly_user_revenue`).
		WithArgs("2026-08").
		WillReturnRows(mock.NewRows(cols).AddRow(int64(1000), int64(650000), int64(0), int64(0), int64(2000), int64(1200), int64(600000), int64(570000)))
	s := NewPGStore(mock)
	sum, err := s.Summary(context.Background(), "2026-08")
	if err != nil {
		t.Fatal(err)
	}
	if sum.ClosingActiveTotal != 1000 || sum.TotalRevenueTotal != 650000 {
		t.Fatalf("sum=%+v", sum)
	}
	if sum.OntimeRate != nil {
		t.Fatal("ontime rate should be nil when requests=0")
	}
	if sum.PortUtilization == nil || *sum.PortUtilization != 0.6 {
		t.Fatalf("portUtil=%v", sum.PortUtilization)
	}
	if sum.CollectionRate == nil || *sum.CollectionRate != 0.95 {
		t.Fatalf("collection=%v", sum.CollectionRate)
	}
	if sum.Arpu == nil || *sum.Arpu != 650 {
		t.Fatalf("arpu=%v", sum.Arpu)
	}
}
