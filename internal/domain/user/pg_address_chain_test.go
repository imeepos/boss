package user

// 开单内联建址域层测试:全链新建/部分复用/label 后缀/兜底推导/显式回填/入参校验。

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

func chainInput() InlineAddressInput {
	return InlineAddressInput{CustomerID: 9, Levels: []InlineAddressLevel{
		{Level: 1, Name: "马尼拉市"}, {Level: 2, Name: "奎松区"}, {Level: 3, Name: "幸福街道"},
		{Level: 4, Name: "阳光小区"}, {Level: 5, Name: "3号楼"},
	}}
}

func emptyRows(cols ...string) *pgxmock.Rows { return pgxmock.NewRows(cols) }

func expectExistsTrue(t *testing.T, mock pgxmock.PgxPoolIface) {
	t.Helper()
	mock.ExpectQuery(`EXISTS\(SELECT 1 FROM customers`).WithArgs(int64(9)).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
}

func TestAddrLabelFromName(t *testing.T) {
	cases := map[string]string{
		"Building 3": "building_3",
		"Bldg A":     "bldg_a", // 空格转下划线
	}
	got := addrLabelFromName("3号楼")
	if len(got) != 10 || got[:2] != "n_" {
		t.Fatalf("中文应转稳定 hash: %q", got)
	}
	if addrLabelFromName("3号楼") != got {
		t.Fatalf("hash 必须稳定")
	}
	delete(cases, "3号楼")
	for name, want := range cases {
		if got := addrLabelFromName(name); got != want {
			t.Fatalf("%s → %q, want %q", name, got, want)
		}
	}
	long := make([]byte, 60)
	for i := range long {
		long[i] = 'a'
	}
	if got := addrLabelFromName(string(long)); len(got) != 48 {
		t.Fatalf("超长应截断: %d", len(got))
	}
}

// TestInlineChainFullNew 全链新建:五级 miss 即建;1-3 级 needs_review;回执定位楼栋。
func TestInlineChainFullNew(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectBegin()
	expectExistsTrue(t, mock)
	mock.ExpectQuery(`SELECT id, path::text FROM addresses`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(emptyRows("id", "path"))
	mock.ExpectQuery(`INSERT INTO addresses`).
		WithArgs(pgxmock.AnyArg(), int8(1), "马尼拉市", int64(0), pgxmock.AnyArg(), true, inlineSource).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(101)))
	for i, parent := range []string{"manila", "manila.queson", "manila.queson.xingfu", "manila.queson.xingfu.yangguang"} {
		_ = i
		mock.ExpectQuery(`SELECT id, path::text FROM addresses`).
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(emptyRows("id", "path"))
		mock.ExpectQuery(`SELECT region_id FROM addresses`).
			WithArgs(pgxmock.AnyArg()).WillReturnRows(emptyRows("region_id"))
		mock.ExpectQuery(`INSERT INTO addresses`).
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
				pgxmock.AnyArg(), pgxmock.AnyArg(), inlineSource).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(102)))
		_ = parent
	}
	mock.ExpectQuery(`LEFT JOIN LATERAL`).WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"legal_entity_id", "path", "region_id"}).
			AddRow(int64(2), "root.luzon", int64(5)))
	mock.ExpectCommit()

	res, err := NewPGStore(mock).CreateInlineAddressChain(context.Background(), chainInput())
	if err != nil {
		t.Fatalf("chain: %v", err)
	}
	if res.AddressID == 0 || res.Fallback || res.LegalEntityID != 2 || res.RegionPath != "root.luzon" {
		t.Fatalf("回执失真: %+v", res)
	}
	if len(res.NeedsReview) != 3 || res.NeedsReview[0].Level != 1 || res.NeedsReview[2].Level != 3 {
		t.Fatalf("1-3 级应全部 needs_review: %+v", res.NeedsReview)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL 序列: %v", err)
	}
}

// TestInlineChainReusePartial 部分命中:1-2 级复用既有节点,仅补建 3-5 级。
func TestInlineChainReusePartial(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectBegin()
	expectExistsTrue(t, mock)
	mock.ExpectQuery(`SELECT id, path::text FROM addresses`).
		WithArgs("马尼拉市", int64(0)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "path"}).AddRow(int64(10), "manila"))
	mock.ExpectQuery(`SELECT id, path::text FROM addresses`).
		WithArgs("奎松区", int64(10)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "path"}).AddRow(int64(11), "manila.queson"))
	for i := 0; i < 3; i++ {
		mock.ExpectQuery(`SELECT id, path::text FROM addresses`).
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(emptyRows("id", "path"))
		mock.ExpectQuery(`SELECT region_id FROM addresses`).
			WithArgs(pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"region_id"}).AddRow(int64(5)))
		mock.ExpectQuery(`INSERT INTO addresses`).
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
				pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(12)))
	}
	mock.ExpectQuery(`LEFT JOIN LATERAL`).WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"legal_entity_id", "path", "region_id"}).
			AddRow(int64(2), "root.luzon", int64(5)))
	mock.ExpectCommit()

	res, err := NewPGStore(mock).CreateInlineAddressChain(context.Background(), chainInput())
	if err != nil {
		t.Fatalf("chain: %v", err)
	}
	if len(res.NeedsReview) != 1 || res.NeedsReview[0].Level != 3 || res.NeedsReview[0].ID != 12 {
		t.Fatalf("仅新建的 3 级入复核: %+v", res.NeedsReview)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL 序列: %v", err)
	}
}

// TestInlineChainLabelSuffix 同父同 label 23505 → 稳定后缀重试,不失败。
func TestInlineChainLabelSuffix(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO addresses`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(&pgconn.PgError{Code: "23505"})
	mock.ExpectQuery(`INSERT INTO addresses`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(101)))
	mock.ExpectRollback()

	tx, err := mock.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	id, path, err := insertChainNode(context.Background(), tx, "马尼拉市", 1, 0, "")
	if err != nil || id != 101 || !strings.HasSuffix(path, "_2") {
		t.Fatalf("suffix retry: id=%d path=%q err=%v", id, path, err)
	}
}

// TestInlineChainOwnershipFallback 覆盖全空 → 兜底平台总公司,fallback=true。
func TestInlineChainOwnershipFallback(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(`LEFT JOIN LATERAL`).WithArgs(int64(505)).
		WillReturnRows(pgxmock.NewRows([]string{"legal_entity_id", "path", "region_id"}).
			AddRow(nil, nil, nil))
	mock.ExpectQuery(`is_platform`).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(1)))
	mock.ExpectRollback()

	tx, _ := mock.Begin(context.Background())
	entityID, regionPath, regionID, fallback, err := resolveChainOwnership(context.Background(), tx, 505)
	if err != nil || !fallback || entityID != 1 || regionPath != "root" || regionID != nil {
		t.Fatalf("fallback: entity=%d path=%q rid=%v fb=%v err=%v", entityID, regionPath, regionID, fallback, err)
	}
}

// TestInlineChainBackfill 显式回填:UPDATE customers 单事务内执行,RowsAffected 透传。
func TestInlineChainBackfill(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE customers`).
		WithArgs(int64(9), int64(505), int64(2), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectRollback()

	tx, _ := mock.Begin(context.Background())
	rid := int64(7)
	if ok, err := backfillCustomerAddress(context.Background(), tx,
		InlineAddressInput{CustomerID: 9, BackfillCustomer: true}, 505, 2, &rid); err != nil || !ok {
		t.Fatalf("backfill: ok=%v err=%v", ok, err)
	}
	// 未勾选回填:不发 SQL。
	if ok, err := backfillCustomerAddress(context.Background(), tx,
		InlineAddressInput{CustomerID: 9}, 505, 2, nil); ok || err != nil {
		t.Fatalf("未勾选不应回填: ok=%v err=%v", ok, err)
	}
}

// TestInlineChainValidation 入参校验:五级不全/客户缺失即拒。
func TestInlineChainValidation(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	bad := chainInput()
	bad.Levels = bad.Levels[:4]
	if _, err := NewPGStore(mock).CreateInlineAddressChain(context.Background(), bad); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("四级应拒: %v", err)
	}
	empty := chainInput()
	empty.Levels[4].Name = " "
	if _, err := NewPGStore(mock).CreateInlineAddressChain(context.Background(), empty); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("空楼栋名应拒: %v", err)
	}
	mock.ExpectBegin()
	mock.ExpectQuery(`EXISTS\(SELECT 1 FROM customers`).WithArgs(int64(9)).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectRollback()
	if _, err := NewPGStore(mock).CreateInlineAddressChain(context.Background(), chainInput()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("客户缺失应拒: %v", err)
	}
	noBackfillCust := chainInput()
	noBackfillCust.CustomerID = 0
	noBackfillCust.BackfillCustomer = true
	if _, err := NewPGStore(mock).CreateInlineAddressChain(context.Background(), noBackfillCust); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("无客户禁止回填应拒: %v", err)
	}
}

// TestInlineChainNoCustomer 未建档开户场景(customerId=0):跳过客户存在性检查,
// 全链照常补建,不产生任何回填 SQL(2026-09-01 /customers/address)。
func TestInlineChainNoCustomer(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectBegin()
	// 无 EXISTS(customers) 期望:出现即说明误查,pgxmock 会报未预期查询。
	mock.ExpectQuery(`SELECT id, path::text FROM addresses`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(emptyRows("id", "path"))
	mock.ExpectQuery(`INSERT INTO addresses`).
		WithArgs(pgxmock.AnyArg(), int8(1), "马尼拉市", int64(0), pgxmock.AnyArg(), true, inlineSource).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(101)))
	for i := 0; i < 4; i++ {
		mock.ExpectQuery(`SELECT id, path::text FROM addresses`).
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(emptyRows("id", "path"))
		mock.ExpectQuery(`SELECT region_id FROM addresses`).
			WithArgs(pgxmock.AnyArg()).WillReturnRows(emptyRows("region_id"))
		mock.ExpectQuery(`INSERT INTO addresses`).
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
				pgxmock.AnyArg(), pgxmock.AnyArg(), inlineSource).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(102)))
	}
	mock.ExpectQuery(`LEFT JOIN LATERAL`).WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"legal_entity_id", "path", "region_id"}).
			AddRow(int64(2), "root.luzon", int64(5)))
	mock.ExpectCommit()

	in := chainInput()
	in.CustomerID = 0
	res, err := NewPGStore(mock).CreateInlineAddressChain(context.Background(), in)
	if err != nil {
		t.Fatalf("chain: %v", err)
	}
	if res.AddressID == 0 || res.Backfilled {
		t.Fatalf("回执失真: %+v", res)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL 序列: %v", err)
	}
}
