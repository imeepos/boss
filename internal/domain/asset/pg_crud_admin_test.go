// 资产台账 admin CRUD 单测(P2-W1-T1):建档自动编码/批次校验/受限编辑换绑事件/
// 幂等/批次门禁/守卫删除阻断项与报废硬删拒绝。
package asset

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// TestGenAssetCode 编码风格:A-{批次 8 位}-{序号 5 位},16 字符恒满足 VARCHAR(32)。
func TestGenAssetCode(t *testing.T) {
	if got := genAssetCode(42, 7); got != "A-00000042-00007" {
		t.Fatalf("got %s", got)
	}
}

// 建档:编码缺省服务端生成 + 企业快照自批次回填 + 初始轨迹行(同事务)。
func TestPGStore_CreateAsset_AdminAutoCode(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`LEFT JOIN legal_entities`).WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"legal_entity_id", "name"}).AddRow(int64(1), "主品牌·企业"))
	mock.ExpectBegin()
	// 编码为自动生成,AnyArg;企业快照已自批次回填。
	mock.ExpectQuery(`INSERT INTO assets`).
		WithArgs(pgxmock.AnyArg(), int64(1), int64(1), "主品牌·企业", nil, nil, nil, "", "ONU", "IN_STOCK", nil).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(5)))
	// 初始轨迹行:建档即留痕。
	mock.ExpectExec(`INSERT INTO asset_lifecycles`).
		WithArgs(int64(5), "IN_STOCK").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	s := NewPGStore(mock)
	id, err := s.CreateAsset(context.Background(), Asset{BatchID: 1, Type: "ONU", Status: "IN_STOCK"})
	if err != nil {
		t.Fatalf("CreateAsset: %v", err)
	}
	if id != 5 {
		t.Fatalf("id=%d, want 5", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// 批次缺失 → 既有校验错误族 ErrForeignKeyViolation(42200)。
func TestPGStore_CreateAsset_MissingBatch(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(9)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))

	s := NewPGStore(mock)
	_, err = s.CreateAsset(context.Background(), Asset{BatchID: 9, Status: "IN_STOCK"})
	if !errors.Is(err, ErrForeignKeyViolation) {
		t.Fatalf("want ErrForeignKeyViolation, got %v", err)
	}
}

// 换绑事件成对:旧绑写 UNBIND、新绑写 BIND,同事务先后落库。
func TestPGStore_UpdateAsset_RebindTagEvents(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`FOR UPDATE`).WithArgs(pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"status", "type", "batch_id", "model_id", "tag_id", "le_id", "le_name"}).
			AddRow("IN_STOCK", "ONU", int64(1), int64(0), int64(9), int64(1), "主品牌·企业"))
	// 旧绑 9 解绑。
	mock.ExpectExec(`UPDATE tags SET bound_asset_id = NULL`).
		WithArgs(int64(9), int64(3)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	// UNBIND 事件(带 actor 与 changed)。
	mock.ExpectExec(`INSERT INTO tag_events`).
		WithArgs(int64(9), int64(3), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	// 新绑 7 存在性校验 + 绑定。
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(`status = 'BOUND'`).
		WithArgs(int64(7), int64(3)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	// BIND 事件(bindTagEvent 三参)。
	mock.ExpectExec(`INSERT INTO tag_events`).
		WithArgs(int64(7), int64(3), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(`UPDATE assets`).
		WithArgs(int64(3), "ONU", nil, int64(7), int64(1), int64(1), "主品牌·企业").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	s := NewPGStore(mock)
	err = s.UpdateAsset(context.Background(), 3, AssetUpdate{TagID: 7, BatchID: 1}, 42)
	if err != nil {
		t.Fatalf("UpdateAsset: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// 换绑冲突:新签已被其他资产占用 → ErrBindingConflict 整单回滚不留孤儿。
func TestPGStore_UpdateAsset_RebindConflictRollback(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`FOR UPDATE`).WithArgs(pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"status", "type", "batch_id", "model_id", "tag_id", "le_id", "le_name"}).
			AddRow("IN_STOCK", "ONU", int64(1), int64(0), int64(9), int64(1), "主品牌·企业"))
	mock.ExpectExec(`UPDATE tags SET bound_asset_id = NULL`).
		WithArgs(int64(9), int64(3)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`INSERT INTO tag_events`).
		WithArgs(int64(9), int64(3), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	// 新签 7 已绑另一资产,UPDATE 0 行 → 冲突回滚(旧绑解绑一并回滚)。
	mock.ExpectExec(`status = 'BOUND'`).
		WithArgs(int64(7), int64(3)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	mock.ExpectRollback()

	s := NewPGStore(mock)
	err = s.UpdateAsset(context.Background(), 3, AssetUpdate{TagID: 7, BatchID: 1}, 42)
	if !errors.Is(err, ErrBindingConflict) {
		t.Fatalf("want ErrBindingConflict, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// 非 IN_STOCK 改批次被拒(ErrBatchNotEditable → 42200)。
func TestPGStore_UpdateAsset_BatchNotEditable(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`FOR UPDATE`).WithArgs(pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"status", "type", "batch_id", "model_id", "tag_id", "le_id", "le_name"}).
			AddRow("DEPLOYED", "ONU", int64(1), int64(0), int64(0), int64(1), "主品牌·企业"))
	mock.ExpectRollback()

	s := NewPGStore(mock)
	err = s.UpdateAsset(context.Background(), 3, AssetUpdate{BatchID: 2}, 42)
	if !errors.Is(err, ErrBatchNotEditable) {
		t.Fatalf("want ErrBatchNotEditable, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// 改批次同步企业归属快照(IN_STOCK 态)。
func TestPGStore_UpdateAsset_BatchSyncsSnapshot(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`FOR UPDATE`).WithArgs(pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"status", "type", "batch_id", "model_id", "tag_id", "le_id", "le_name"}).
			AddRow("IN_STOCK", "ONU", int64(1), int64(0), int64(0), int64(1), "主品牌·企业"))
	mock.ExpectQuery(`FROM asset_batches b`).
		WithArgs(int64(2)).
		WillReturnRows(mock.NewRows([]string{"legal_entity_id", "name"}).AddRow(int64(2), "新公司"))
	mock.ExpectExec(`UPDATE assets`).
		WithArgs(int64(3), "ONU", nil, nil, int64(2), int64(2), "新公司").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	s := NewPGStore(mock)
	err = s.UpdateAsset(context.Background(), 3, AssetUpdate{BatchID: 2}, 42)
	if err != nil {
		t.Fatalf("UpdateAsset: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// 四键无变化:零写入幂等成功(仅锁行,无 UPDATE 无事件)。
func TestPGStore_UpdateAsset_Idempotent(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`FOR UPDATE`).WithArgs(pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"status", "type", "batch_id", "model_id", "tag_id", "le_id", "le_name"}).
			AddRow("IN_STOCK", "ONU", int64(1), int64(0), int64(0), int64(1), "主品牌·企业"))
	mock.ExpectRollback()

	s := NewPGStore(mock)
	err = s.UpdateAsset(context.Background(), 3, AssetUpdate{BatchID: 1}, 42)
	if err != nil {
		t.Fatalf("UpdateAsset: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// 删除命中引用:ErrAssetReferenced message 全量列阻断项。
func TestPGStore_DeleteAsset_Blocked(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`FOR UPDATE`).WithArgs(pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"status", "asset_code"}).AddRow("IN_STOCK", "A-00000003-00001"))
	// 守卫序:tags/assignments/replacements/stocktake_items/quad_links;命中两项。
	for _, hit := range []bool{false, true, true, false, false} {
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(3)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(hit))
	}
	mock.ExpectRollback()

	s := NewPGStore(mock)
	_, err = s.DeleteAsset(context.Background(), 3)
	var refErr *ErrAssetReferenced
	if !errors.As(err, &refErr) {
		t.Fatalf("want ErrAssetReferenced, got %v", err)
	}
	if got := refErr.Error(); !strings.Contains(got, "持有台账") || !strings.Contains(got, "换新单") {
		t.Fatalf("message missing blockers: %s", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// SCRAPPED 一律拒绝硬删,提示走报废端点。
func TestPGStore_DeleteAsset_Scrapped(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`FOR UPDATE`).WithArgs(pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"status", "asset_code"}).AddRow("SCRAPPED", "A-00000009-00001"))
	mock.ExpectRollback()

	s := NewPGStore(mock)
	_, err = s.DeleteAsset(context.Background(), 9)
	if !errors.Is(err, ErrAssetScrapped) {
		t.Fatalf("want ErrAssetScrapped, got %v", err)
	}
	if got := err.Error(); !strings.Contains(got, "报废") || !strings.Contains(got, "scrap") {
		t.Fatalf("message missing scrap hint: %s", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// 无引用 IN_STOCK:物理删除成功,返回资产编码供审计载荷。
func TestPGStore_DeleteAsset_OK(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`FOR UPDATE`).WithArgs(pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"status", "asset_code"}).AddRow("IN_STOCK", "A-00000007-00002"))
	// 守卫 SQL 必须落到真实表名。回归兜底:守卫 SQL 曾因原始串内 "+gr.table+" 拼接
	// 失效而原样带占位符上线,pgxmock 宽松正则(SELECT EXISTS)未拦住,真实 PG 42P01。
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM tags WHERE bound_asset_id = \$1\)`).
		WithArgs(int64(3)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))
	for _, g := range []struct{ table, column string }{
		{"asset_assignments", "asset_id"}, {"replacements", "asset_id"},
		{"stocktake_items", "asset_id"}, {"quad_links", "asset_id"},
	} {
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM ` + g.table + ` WHERE ` + g.column + ` = \$1\)`).
			WithArgs(int64(3)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))
	}
	// 建档即留痕(000184):删除主档前必须同事务清理自有轨迹行,
	// 否则 FK asset_lifecycles_asset_id_fkey 使硬删必败(23503,102 E2E 实证)。
	mock.ExpectExec(`DELETE FROM asset_lifecycles WHERE asset_id = \$1`).
		WithArgs(int64(3)).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectExec(`DELETE FROM assets`).
		WithArgs(int64(3)).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectCommit()

	s := NewPGStore(mock)
	code, err := s.DeleteAsset(context.Background(), 3)
	if err != nil {
		t.Fatalf("DeleteAsset: %v", err)
	}
	if code != "A-00000007-00002" {
		t.Fatalf("code=%s", code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
