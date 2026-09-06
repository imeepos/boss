// 资产↔标签双绑回填 + 冲突检测单测(internal/domain/asset/pg_write.go):
// - CreateAsset/CreateTag 双绑回填成功 + 冲突返 ErrBindingConflict
// - 同 asset+tag 重复提交 = 幂等
// - DB 唯一约束(23505)被 classify*InsertErr 转 ErrBindingConflict
package asset

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_CreateAsset_BackfillTagBinding(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO assets`).
		WithArgs("A-20260003", int64(1), int64(1), "主品牌·企业", int64(9), nil, nil, "", "ONU", "IN_STOCK", nil, nil, nil, nil).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(3)))
	// 入账轨迹首行:建档即留痕
	mock.ExpectExec(`INSERT INTO asset_lifecycles`).
		WithArgs(int64(3), "IN_STOCK").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	// DISABLED 绑定闸门(P2-W2-T1 B):绑定前查标签状态。
	mock.ExpectQuery(`SELECT status FROM tags`).
		WithArgs(int64(9)).
		WillReturnRows(mock.NewRows([]string{"status"}).AddRow("UNBOUND"))
	// 回填:未绑定标签 → bound_asset_id + BOUND。
	mock.ExpectExec(`UPDATE tags SET bound_asset_id`).
		WithArgs(int64(9), int64(3)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	// 绑定事件流(P1-T2):BIND 随主事务落库。
	mock.ExpectCommit()
	// 绑定事件:提交后尽力而为(P2-T2 热修)
	mock.ExpectExec(`INSERT INTO tag_events`).
		WithArgs(int64(9), int64(3), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	s := NewPGStore(mock)
	id, err := s.CreateAsset(context.Background(), Asset{
		AssetCode: "A-20260003", BatchID: 1, LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		TagID: 9, Type: "ONU", Status: "IN_STOCK",
	})
	if err != nil {
		t.Fatalf("CreateAsset: %v", err)
	}
	if id != 3 {
		t.Fatalf("id=%d, want 3", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// CreateAsset 回填时检测到 tag.bound_asset_id 已被另一资产占用 → ErrBindingConflict。
func TestPGStore_CreateAsset_TagAlreadyBound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO assets`).
		WithArgs("A-20260003", int64(1), int64(1), "主品牌·企业", int64(9), nil, nil, "", "ONU", "IN_STOCK", nil, nil, nil, nil).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(3)))
	// 入账轨迹首行:建档即留痕
	mock.ExpectExec(`INSERT INTO asset_lifecycles`).
		WithArgs(int64(3), "IN_STOCK").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	// DISABLED 绑定闸门(P2-W2-T1 B):绑定前查标签状态。
	mock.ExpectQuery(`SELECT status FROM tags`).
		WithArgs(int64(9)).
		WillReturnRows(mock.NewRows([]string{"status"}).AddRow("UNBOUND"))
	// 标签已绑另一资产,UPDATE 影响 0 行 → ErrBindingConflict(整单回滚,不留孤儿资产)。
	mock.ExpectExec(`UPDATE tags SET bound_asset_id`).
		WithArgs(int64(9), int64(3)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	mock.ExpectRollback()

	s := NewPGStore(mock)
	_, err = s.CreateAsset(context.Background(), Asset{
		AssetCode: "A-20260003", BatchID: 1, LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		TagID: 9, Type: "ONU", Status: "IN_STOCK",
	})
	if !errors.Is(err, ErrBindingConflict) {
		t.Fatalf("want ErrBindingConflict, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// CreateTag 预绑定资产时,资产不存在 → ErrForeignKeyViolation。
func TestPGStore_CreateTag_AssetNotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// 法人存在性校验(P2-W2-T1 建标签端点)。
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(99)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))

	s := NewPGStore(mock)
	_, err = s.CreateTag(context.Background(), Tag{
		LegalEntityID: 1, TagNo: "TAG-0009", EpcCode: "3000000000000000000000E9", Band: "UHF",
		BoundAssetID: 99, Status: "BOUND", Battery: "95%",
	})
	if !errors.Is(err, ErrForeignKeyViolation) {
		t.Fatalf("want ErrForeignKeyViolation, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// CreateTag 预绑定时同步回填 assets.tag_id(原 d397e40 修复缺漏)。
func TestPGStore_CreateTag_BackfillAssetBinding(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// 法人存在性校验(P2-W2-T1 建标签端点)。
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	// 资产存在性预检
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(5)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	// INSERT tag,bound_asset_id=5 → NULL 转换(事务化:CreateTag 全程包 tx)
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO tags`).
		WithArgs(int64(1), "TAG-0005", "3000000000000000000000E5", "UHF", int64(5), "BOUND", "95%").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(10)))
	// 反向回填 assets.tag_id
	mock.ExpectExec(`UPDATE assets SET tag_id`).
		WithArgs(int64(5), int64(10)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	// 绑定事件流(P1-T2)。
	mock.ExpectCommit()
	// 绑定事件:提交后尽力而为(P2-T2 热修)
	mock.ExpectExec(`INSERT INTO tag_events`).
		WithArgs(int64(10), int64(5), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	s := NewPGStore(mock)
	id, err := s.CreateTag(context.Background(), Tag{
		LegalEntityID: 1, TagNo: "TAG-0005", EpcCode: "3000000000000000000000E5", Band: "UHF",
		BoundAssetID: 5, Status: "BOUND", Battery: "95%",
	})
	if err != nil {
		t.Fatalf("CreateTag: %v", err)
	}
	if id != 10 {
		t.Fatalf("id=%d, want 10", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// CreateTag 回填时检测到 asset.tag_id 已被另一标签占用 → ErrBindingConflict。
func TestPGStore_CreateTag_AssetAlreadyBound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// 法人存在性校验(P2-W2-T1 建标签端点)。
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(5)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO tags`).
		WithArgs(int64(1), "TAG-0006", "3000000000000000000000E6", "UHF", int64(5), "BOUND", "95%").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(11)))
	// 资产.tag_id 已被另一标签占用 → UPDATE 0 行 → ErrBindingConflict(整单回滚)。
	mock.ExpectExec(`UPDATE assets SET tag_id`).
		WithArgs(int64(5), int64(11)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	mock.ExpectRollback()

	s := NewPGStore(mock)
	_, err = s.CreateTag(context.Background(), Tag{
		LegalEntityID: 1, TagNo: "TAG-0006", EpcCode: "3000000000000000000000E6", Band: "UHF",
		BoundAssetID: 5, Status: "BOUND", Battery: "95%",
	})
	if !errors.Is(err, ErrBindingConflict) {
		t.Fatalf("want ErrBindingConflict, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// CreateTag INSERT 触发 DB uq_tags_bound_asset_notnull 23505 → ErrBindingConflict。
func TestPGStore_CreateTag_DBUniqueViolation_AssetBound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// 法人存在性校验(P2-W2-T1 建标签端点)。
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(5)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO tags`).
		WithArgs(int64(1), "TAG-DUP", "3000000000000000000000DD", "UHF", int64(5), "BOUND", "95%").
		WillReturnError(&pgconn.PgError{
			Code:           "23505",
			ConstraintName: "uq_tags_bound_asset_notnull",
			Message:        "duplicate key value violates unique constraint",
		})
	mock.ExpectRollback()

	s := NewPGStore(mock)
	_, err = s.CreateTag(context.Background(), Tag{
		LegalEntityID: 1, TagNo: "TAG-DUP", EpcCode: "3000000000000000000000DD", Band: "UHF",
		BoundAssetID: 5, Status: "BOUND", Battery: "95%",
	})
	if !errors.Is(err, ErrBindingConflict) {
		t.Fatalf("want ErrBindingConflict, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// CreateAsset INSERT 触发 DB uq_assets_tag_notnull 23505 → ErrBindingConflict。
func TestPGStore_CreateAsset_DBUniqueViolation_TagBound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO assets`).
		WithArgs("A-DUP", int64(1), int64(1), "主品牌·企业", int64(9), nil, nil, "", "ONU", "IN_STOCK", nil, nil, nil, nil).
		WillReturnError(&pgconn.PgError{
			Code:           "23505",
			ConstraintName: "uq_assets_tag_notnull",
			Message:        "duplicate key value violates unique constraint",
		})
	mock.ExpectRollback()

	s := NewPGStore(mock)
	_, err = s.CreateAsset(context.Background(), Asset{
		AssetCode: "A-DUP", BatchID: 1, LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		TagID: 9, Type: "ONU", Status: "IN_STOCK",
	})
	if !errors.Is(err, ErrBindingConflict) {
		t.Fatalf("want ErrBindingConflict, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// CreateTag INSERT 触发非双绑的 23505(tag_no 重复)→ ErrCodeDuplicate(40900,
// P2-W2-T1 建标签端点口径:编号唯一)。EPG 键 tags_epc_code_key 同通道(pg_tag_admin_test)。
func TestPGStore_CreateTag_DBUniqueViolation_TagNoDup(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// 法人存在性校验(P2-W2-T1 建标签端点)。
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO tags`).
		WithArgs(int64(1), "DUP-NO", "3000000000000000000000DE", "UHF", nil, "UNBOUND", "95%").
		WillReturnError(&pgconn.PgError{
			Code:           "23505",
			ConstraintName: "tags_tag_no_key",
			Message:        "duplicate key value violates unique constraint",
		})
	mock.ExpectRollback()

	s := NewPGStore(mock)
	_, err = s.CreateTag(context.Background(), Tag{
		LegalEntityID: 1, TagNo: "DUP-NO", EpcCode: "3000000000000000000000DE", Band: "UHF",
		Status: "UNBOUND", Battery: "95%",
	})
	if !errors.Is(err, ErrCodeDuplicate) {
		t.Fatalf("tag_no 重复应映射 ErrCodeDuplicate, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
