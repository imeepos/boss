package asset

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestListQuery_Clamp(t *testing.T) {
	cases := []struct {
		name       string
		in         ListQuery
		wantLimit  int
		wantOffset int
	}{
		{"zero value -> default", ListQuery{}, ListDefaultLimit, 0},
		{"negative limit -> default", ListQuery{Limit: -5, Offset: -1}, ListDefaultLimit, 0},
		{"over max -> clamped no error", ListQuery{Limit: 99999}, ListMaxLimit, 0},
		{"in range kept", ListQuery{Limit: 20, Offset: 40}, 20, 40},
	}
	for _, c := range cases {
		got := c.in.clamp()
		if got.Limit != c.wantLimit || got.Offset != c.wantOffset {
			t.Fatalf("%s: got=(%d,%d) want=(%d,%d)", c.name, got.Limit, got.Offset, c.wantLimit, c.wantOffset)
		}
	}
}

func TestListQuery_SortCol(t *testing.T) {
	if col, err := (ListQuery{}).sortCol(AssetSortColumns); err != nil || col != "created_at" {
		t.Fatalf("empty sort: col=%s err=%v", col, err)
	}
	if col, err := (ListQuery{Sort: "asset_code"}).sortCol(AssetSortColumns); err != nil || col != "asset_code" {
		t.Fatalf("asset_code: col=%s err=%v", col, err)
	}
	if col, err := (ListQuery{Sort: "tag_no"}).sortCol(TagSortColumns); err != nil || col != "tag_no" {
		t.Fatalf("tag_no: col=%s err=%v", col, err)
	}
	// 白名单外一律拒绝(assets 白名单里没有 tag_no,反向亦然)。
	if _, err := (ListQuery{Sort: "tag_no; DROP TABLE x"}).sortCol(AssetSortColumns); !errors.Is(err, ErrInvalidSort) {
		t.Fatalf("want ErrInvalidSort, got %v", err)
	}
	if _, err := (ListQuery{Sort: "battery"}).sortCol(TagSortColumns); !errors.Is(err, ErrInvalidSort) {
		t.Fatalf("want ErrInvalidSort, got %v", err)
	}
}

func TestLikePrefix(t *testing.T) {
	if got, want := likePrefix("A-50%_1\\x"), "A-50\\%\\_1\\\\x%"; got != want {
		t.Fatalf("got=%s want=%s", got, want)
	}
}

func assetRows(mock pgxmock.PgxPoolIface, code string) *pgxmock.Rows {
	return mock.NewRows([]string{"id", "asset_code", "batch_id", "legal_entity_id", "legal_entity_name", "tag_id", "address_id", "region_id", "region_name", "type", "status", "model_id"}).
		AddRow(int64(7), code, int64(1), int64(1), "主体", int64(0), int64(0), int64(0), "", "ONU", "IN_STOCK", int64(0))
}

// 默认查询:无 WHERE,created_at DESC+id DESC tie-breaker,LIMIT/OFFSET 参数化。
func TestPGStore_ListAssetsPage_Default(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery(`FROM assets\s+ORDER BY created_at DESC, id DESC\s+LIMIT \$1 OFFSET \$2`).
		WithArgs(int64(50), int64(0)).
		WillReturnRows(assetRows(mock, "A-20260007"))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM assets$`).
		WithArgs().
		WillReturnRows(mock.NewRows([]string{"count"}).AddRow(int64(1)))

	page, err := NewPGStore(mock).ListAssetsPage(context.Background(), ListQuery{})
	if err != nil {
		t.Fatalf("ListAssetsPage: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].AssetCode != "A-20260007" {
		t.Fatalf("page=%+v", page)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// 过滤组合:status/type/modelId/q 同 WHERE,COUNT 复用同参;limit 超限钳 200。
func TestPGStore_ListAssetsPage_Filtered(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery(`FROM assets WHERE status = \$1 AND type = \$2 AND COALESCE\(model_id, 0\) = \$3 AND asset_code LIKE \$4 ESCAPE \x27\\\x27`).
		WithArgs("IN_STOCK", "ONU", int64(9), "A-50\\%\\_%", int64(200), int64(0)).
		WillReturnRows(assetRows(mock, "A-50%_9"))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM assets WHERE`).
		WithArgs("IN_STOCK", "ONU", int64(9), "A-50\\%\\_%").
		WillReturnRows(mock.NewRows([]string{"count"}).AddRow(int64(1)))

	q := ListQuery{Limit: 99999, Status: "IN_STOCK", Type: "ONU", ModelID: 9, Q: "A-50%_", Sort: "asset_code"}
	page, err := NewPGStore(mock).ListAssetsPage(context.Background(), q)
	if err != nil {
		t.Fatalf("ListAssetsPage: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("page=%+v", page)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// 标签分页:q 按 tag_no 前缀;非法 sort 在 domain 兜底拒绝。
func TestPGStore_ListTagsPage(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery(`FROM tags WHERE status = \$1 AND tag_no LIKE \$2 ESCAPE \x27\\\x27`).
		WithArgs("UNBOUND", "acc\\_pg\\_%", int64(20), int64(0)).
		WillReturnRows(mock.NewRows([]string{"id", "legal_entity_id", "tag_no", "epc_code", "band", "bound_asset_id", "status", "battery"}).
			AddRow(int64(3), int64(1), "acc_pg_3", "EPC3", "UHF", int64(0), "UNBOUND", "100%"))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM tags WHERE`).
		WithArgs("UNBOUND", "acc\\_pg\\_%").
		WillReturnRows(mock.NewRows([]string{"count"}).AddRow(int64(1)))

	page, err := NewPGStore(mock).ListTagsPage(context.Background(), ListQuery{Limit: 20, Status: "UNBOUND", Q: "acc_pg_"})
	if err != nil {
		t.Fatalf("ListTagsPage: %v", err)
	}
	if page.Total != 1 || page.Items[0].TagNo != "acc_pg_3" {
		t.Fatalf("page=%+v", page)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}

	if _, err := NewPGStore(mock).ListTagsPage(context.Background(), ListQuery{Sort: "epc_code"}); !errors.Is(err, ErrInvalidSort) {
		t.Fatalf("want ErrInvalidSort, got %v", err)
	}
}
