package asset

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_ListBatches(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, legal_entity_id, code, name FROM asset_batches ORDER BY id`).
		WillReturnRows(mock.NewRows([]string{"id", "legal_entity_id", "code", "name"}).
			AddRow(int64(1), int64(1), "RK-202607-01", "7月光猫批次").
			AddRow(int64(2), int64(1), "RK-202608-01", "8月光猫批次"))

	s := NewPGStore(mock)
	got, err := s.ListBatches(context.Background())
	if err != nil {
		t.Fatalf("ListBatches: %v", err)
	}
	if len(got) != 2 || got[0].Code != "RK-202607-01" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateBatch(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// FK validation: legal entity exists
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`INSERT INTO asset_batches`).
		WithArgs(int64(1), "RK-202609-01", "9月光猫批次").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(3)))

	s := NewPGStore(mock)
	id, err := s.CreateBatch(context.Background(), AssetBatch{LegalEntityID: 1, Code: "RK-202609-01", Name: "9月光猫批次"})
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}
	if id != 3 {
		t.Fatalf("id=%d, want 3", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListTags(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, legal_entity_id, tag_no, epc_code, band, COALESCE\(bound_asset_id, 0\), status, battery`).
		WillReturnRows(mock.NewRows([]string{"id", "legal_entity_id", "tag_no", "epc_code", "band", "bound_asset_id", "status", "battery"}).
			AddRow(int64(1), int64(1), "TAG-0001", "EPC-0001", "UHF", int64(0), "UNBOUND", "86%").
			AddRow(int64(2), int64(1), "TAG-0002", "EPC-0002", "UHF", int64(10), "BOUND", "90%"))

	s := NewPGStore(mock)
	got, err := s.ListTags(context.Background())
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(got) != 2 || got[0].BoundAssetID != 0 || got[1].Status != "BOUND" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateTag(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// bound_asset_id=0 → nil
	mock.ExpectQuery(`INSERT INTO tags`).
		WithArgs(int64(1), "TAG-0003", "EPC-0003", "UHF", nil, "UNBOUND", "95%").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(3)))

	s := NewPGStore(mock)
	id, err := s.CreateTag(context.Background(), Tag{
		LegalEntityID: 1, TagNo: "TAG-0003", EpcCode: "EPC-0003", Band: "UHF", Status: "UNBOUND", Battery: "95%",
	})
	if err != nil {
		t.Fatalf("CreateTag: %v", err)
	}
	if id != 3 {
		t.Fatalf("id=%d, want 3", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListAssets(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "asset_code", "batch_id", "legal_entity_id", "legal_entity_name", "tag_id", "address_id", "region_id", "region_name", "type", "status"}
	mock.ExpectQuery(`SELECT id, asset_code, batch_id, legal_entity_id, legal_entity_name`).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), "A-20260001", int64(1), int64(1), "主品牌·企业", int64(0), int64(0), int64(0), "", "光猫", "IN_STOCK"))

	s := NewPGStore(mock)
	got, err := s.ListAssets(context.Background())
	if err != nil {
		t.Fatalf("ListAssets: %v", err)
	}
	if len(got) != 1 || got[0].AssetCode != "A-20260001" || got[0].Status != "IN_STOCK" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateAsset(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// tag/address/region = 0 → nil
	// FK validation: batch exists
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	// FK validation: legal entity exists
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`INSERT INTO assets`).
		WithArgs("A-20260002", int64(1), int64(1), "主品牌·企业", nil, nil, nil, "", "ONU", "IN_STOCK").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.CreateAsset(context.Background(), Asset{
		AssetCode: "A-20260002", BatchID: 1, LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		Type: "ONU", Status: "IN_STOCK",
	})
	if err != nil {
		t.Fatalf("CreateAsset: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// CreateAsset 带 tag_id 时回填 tags 双向绑定(ISSUE.md:置备资产不回填 tag 导致扫码 40920)。
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
	mock.ExpectQuery(`INSERT INTO assets`).
		WithArgs("A-20260003", int64(1), int64(1), "主品牌·企业", int64(9), nil, nil, "", "ONU", "IN_STOCK").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(3)))
	// 回填:未绑定标签 → bound_asset_id + BOUND。
	mock.ExpectExec(`UPDATE tags SET bound_asset_id`).
		WithArgs(int64(9), int64(3)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

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
	mock.ExpectQuery(`INSERT INTO assets`).
		WithArgs("A-20260003", int64(1), int64(1), "主品牌·企业", int64(9), nil, nil, "", "ONU", "IN_STOCK").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(3)))
	// 标签已绑另一资产,UPDATE 影响 0 行 → ErrBindingConflict。
	mock.ExpectExec(`UPDATE tags SET bound_asset_id`).
		WithArgs(int64(9), int64(3)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

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

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(99)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))

	s := NewPGStore(mock)
	_, err = s.CreateTag(context.Background(), Tag{
		LegalEntityID: 1, TagNo: "TAG-0009", EpcCode: "EPC-0009", Band: "UHF",
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

	// 资产存在性预检
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(5)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	// INSERT tag,bound_asset_id=5 → NULL 转换
	mock.ExpectQuery(`INSERT INTO tags`).
		WithArgs(int64(1), "TAG-0005", "EPC-0005", "UHF", int64(5), "BOUND", "95%").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(10)))
	// 反向回填 assets.tag_id
	mock.ExpectExec(`UPDATE assets SET tag_id`).
		WithArgs(int64(5), int64(10)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(mock)
	id, err := s.CreateTag(context.Background(), Tag{
		LegalEntityID: 1, TagNo: "TAG-0005", EpcCode: "EPC-0005", Band: "UHF",
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

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(5)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`INSERT INTO tags`).
		WithArgs(int64(1), "TAG-0006", "EPC-0006", "UHF", int64(5), "BOUND", "95%").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(11)))
	// 资产.tag_id 已被另一标签占用 → UPDATE 0 行 → ErrBindingConflict。
	mock.ExpectExec(`UPDATE assets SET tag_id`).
		WithArgs(int64(5), int64(11)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	s := NewPGStore(mock)
	_, err = s.CreateTag(context.Background(), Tag{
		LegalEntityID: 1, TagNo: "TAG-0006", EpcCode: "EPC-0006", Band: "UHF",
		BoundAssetID: 5, Status: "BOUND", Battery: "95%",
	})
	if !errors.Is(err, ErrBindingConflict) {
		t.Fatalf("want ErrBindingConflict, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// CreateAsset 幂等:重复提交同一 asset+tag,PG 行为 = 值已相等仍 UPDATE 1 行
// (实测 PG 16:UPDATE 命中条件且新值=旧值仍报 1,见 102 真表测试)。代码不依赖
// RowsAffected 区分"幂等"vs"已变更",只要没冲突都算成功。
func TestPGStore_CreateAsset_ResubmitIdempotent(t *testing.T) {
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
	mock.ExpectQuery(`INSERT INTO assets`).
		WithArgs("A-20260003", int64(1), int64(1), "主品牌·企业", int64(9), nil, nil, "", "ONU", "IN_STOCK").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(3)))
	// PG 16 行为:条件命中且值已相等仍返 1 行(并非 0 行)。
	mock.ExpectExec(`UPDATE tags SET bound_asset_id`).
		WithArgs(int64(9), int64(3)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(mock)
	id, err := s.CreateAsset(context.Background(), Asset{
		AssetCode: "A-20260003", BatchID: 1, LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		TagID: 9, Type: "ONU", Status: "IN_STOCK",
	})
	if err != nil {
		t.Fatalf("CreateAsset resubmit should be idempotent: %v", err)
	}
	if id != 3 {
		t.Fatalf("id=%d, want 3", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// CreateTag 幂等:同 asset+tag 重复预绑定,PG 行为 = UPDATE 1 行。
func TestPGStore_CreateTag_ResubmitIdempotent(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(5)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`INSERT INTO tags`).
		WithArgs(int64(1), "TAG-0007", "EPC-0007", "UHF", int64(5), "BOUND", "95%").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(12)))
	mock.ExpectExec(`UPDATE assets SET tag_id`).
		WithArgs(int64(5), int64(12)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(mock)
	id, err := s.CreateTag(context.Background(), Tag{
		LegalEntityID: 1, TagNo: "TAG-0007", EpcCode: "EPC-0007", Band: "UHF",
		BoundAssetID: 5, Status: "BOUND", Battery: "95%",
	})
	if err != nil {
		t.Fatalf("CreateTag resubmit should be idempotent: %v", err)
	}
	if id != 12 {
		t.Fatalf("id=%d, want 12", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_GetAsset(t *testing.T) {
	t.Run("命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		cols := []string{"id", "asset_code", "batch_id", "legal_entity_id", "legal_entity_name", "tag_id", "address_id", "region_id", "region_name", "type", "status"}
		mock.ExpectQuery(`SELECT id, asset_code, batch_id, legal_entity_id, legal_entity_name`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows(cols).
				AddRow(int64(1), "A-20260001", int64(1), int64(1), "主品牌·企业", int64(2), int64(100), int64(11), "马尼拉市", "光猫", "DEPLOYED"))

		s := NewPGStore(mock)
		a, err := s.GetAsset(context.Background(), 1)
		if err != nil {
			t.Fatalf("GetAsset: %v", err)
		}
		if a.AssetCode != "A-20260001" || a.TagID != 2 || a.Status != "DEPLOYED" {
			t.Fatalf("a=%+v", a)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("未命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT id, asset_code`).
			WithArgs(int64(99)).
			WillReturnError(pgx.ErrNoRows)

		s := NewPGStore(mock)
		_, err = s.GetAsset(context.Background(), 99)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

var ts = time.Date(2025, 8, 17, 10, 0, 0, 0, time.UTC)

func TestPGStore_ListLifecycles(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "asset_id", "status", "address_id", "address_name", "worker_id", "worker_name", "changed_at"}
	mock.ExpectQuery(`SELECT id, asset_id, status, COALESCE\(address_id, 0\)`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(1), "IN_STOCK", int64(0), "", int64(0), "", ts).
			AddRow(int64(2), int64(1), "DEPLOYED", int64(100), "望京X", int64(1024), "张师傅", ts))

	s := NewPGStore(mock)
	got, err := s.ListLifecycles(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListLifecycles: %v", err)
	}
	if len(got) != 2 || got[1].Status != "DEPLOYED" || got[1].WorkerName != "张师傅" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_AppendLifecycle(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO asset_lifecycles`).
		WithArgs(int64(1), "DEPLOYED", int64(100), "望京X", int64(1024), "张师傅", ts).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(3)))

	s := NewPGStore(mock)
	id, err := s.AppendLifecycle(context.Background(), AssetLifecycle{
		AssetID: 1, Status: "DEPLOYED", AddressID: 100, AddressName: "望京X", WorkerID: 1024, WorkerName: "张师傅", ChangedAt: ts,
	})
	if err != nil {
		t.Fatalf("AppendLifecycle: %v", err)
	}
	if id != 3 {
		t.Fatalf("id=%d, want 3", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListReplacements(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "replacement_no", "asset_id", "legal_entity_id", "legal_entity_name", "reason", "priority", "status"}
	mock.ExpectQuery(`SELECT id, replacement_no, asset_id, legal_entity_id, legal_entity_name, reason, priority, status FROM replacements`).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), "RPL-20260817-001", int64(5), int64(1), "主品牌·企业", "光猫故障", "HIGH", "PENDING"))

	s := NewPGStore(mock)
	got, err := s.ListReplacements(context.Background())
	if err != nil {
		t.Fatalf("ListReplacements: %v", err)
	}
	if len(got) != 1 || got[0].ReplacementNo != "RPL-20260817-001" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateReplacement(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO replacements`).
		WithArgs("RPL-20260817-002", int64(6), int64(1), "主品牌·企业", "光猫故障", "MEDIUM", "PENDING").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.CreateReplacement(context.Background(), Replacement{
		ReplacementNo: "RPL-20260817-002", AssetID: 6, LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		Reason: "光猫故障", Priority: "MEDIUM", Status: "PENDING",
	})
	if err != nil {
		t.Fatalf("CreateReplacement: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListStocktakes(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "legal_entity_id", "scope", "progress", "diff_count", "status"}
	mock.ExpectQuery(`SELECT id, legal_entity_id, scope, progress, diff_count, status FROM stocktakes`).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(1), "root.luzon", int16(80), int32(3), "DOING"))

	s := NewPGStore(mock)
	got, err := s.ListStocktakes(context.Background())
	if err != nil {
		t.Fatalf("ListStocktakes: %v", err)
	}
	if len(got) != 1 || got[0].DiffCount != 3 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateStocktake(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO stocktakes`).
		WithArgs(int64(1), "root.luzon.ncr", int16(0), int32(0), "DOING").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.CreateStocktake(context.Background(), Stocktake{
		LegalEntityID: 1, Scope: "root.luzon.ncr", Progress: 0, DiffCount: 0, Status: "DOING",
	})
	if err != nil {
		t.Fatalf("CreateStocktake: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListAssignments(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "asset_id", "worker_id", "worker_name", "address_id", "address_name", "reason", "operator_account_id", "effective_from", "effective_to"}
	mock.ExpectQuery(`SELECT id, asset_id, COALESCE\(worker_id, 0\)`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(1), int64(1024), "张师傅", int64(100), "望京X", "领用", int64(9), ts, ts).
			AddRow(int64(2), int64(1), int64(1024), "张师傅", int64(100), "望京X", "归还", int64(9), ts, nil))

	s := NewPGStore(mock)
	got, err := s.ListAssignments(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListAssignments: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	if got[0].EffectiveTo == nil {
		t.Fatal("got[0].EffectiveTo nil, want non-nil")
	}
	if got[1].EffectiveTo != nil {
		t.Fatal("got[1].EffectiveTo non-nil, want nil(至今)")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_AssignAsset(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	var nilTime *time.Time
	mock.ExpectQuery(`INSERT INTO asset_assignments`).
		WithArgs(int64(1), int64(1024), "张师傅", int64(100), "望京X", "领用", int64(9), ts, nilTime).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(3)))

	s := NewPGStore(mock)
	id, err := s.AssignAsset(context.Background(), AssetAssignment{
		AssetID: 1, WorkerID: 1024, WorkerName: "张师傅", AddressID: 100, AddressName: "望京X",
		Reason: "领用", OperatorAccountID: 9, EffectiveFrom: ts, EffectiveTo: nil,
	})
	if err != nil {
		t.Fatalf("AssignAsset: %v", err)
	}
	if id != 3 {
		t.Fatalf("id=%d, want 3", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
