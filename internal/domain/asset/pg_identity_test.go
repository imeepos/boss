// 身份列写路径单测(P3-T2):INSERT 参数携带归一后身份值、唯一冲突 23505 分类、
// 格式非法在触库前短路。
package asset

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

// CreateAsset 携带 SN/MAC/LOID:归一后入 INSERT 参数,空串转 NULL。
func TestPGStore_CreateAsset_WithIdentity(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO assets`).
		WithArgs("A-ID-1", int64(1), int64(1), "主品牌·企业", nil, nil, nil, "", "ONU", "IN_STOCK", nil,
			"SN-001", "AA-BB-CC-DD-EE-01", "LOID-001").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(40)))
	mock.ExpectExec(`INSERT INTO asset_lifecycles`).
		WithArgs(int64(40), "IN_STOCK").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	s := NewPGStore(mock)
	// sn/loid 带首尾空格 → 归一;mac 横杠格式合法原样入库。
	_, err = s.CreateAsset(context.Background(), Asset{
		AssetCode: "A-ID-1", BatchID: 1, LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		Type: "ONU", Status: "IN_STOCK",
		SN: "  SN-001  ", MAC: "AA-BB-CC-DD-EE-01", LOID: "  LOID-001  ",
	})
	if err != nil {
		t.Fatalf("CreateAsset: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// MAC 非法在触库前短路(无任何 DB 期望,pgxmock 自动校验零调用)。
func TestPGStore_CreateAsset_InvalidMACShortCircuit(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	s := NewPGStore(mock)
	_, err := s.CreateAsset(context.Background(), Asset{
		BatchID: 0, Type: "ONU", Status: "IN_STOCK", MAC: "AA:BB:CC",
	})
	if !errors.Is(err, ErrInvalidMAC) {
		t.Fatalf("want ErrInvalidMAC, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected DB calls: %v", err)
	}
}

// EPC 非法在触库前短路(CreateTag 是 epc_code 唯一写入口)。
func TestPGStore_CreateTag_InvalidEPCShortCircuit(t *testing.T) {
	for _, epc := range []string{"SHORT", "31AAAAAAAAAAAAAAAAAAAAAAAA"} {
		mock, _ := pgxmock.NewPool()
		s := NewPGStore(mock)
		_, err := s.CreateTag(context.Background(), Tag{
			LegalEntityID: 1, TagNo: "T-X", EpcCode: epc, Band: "UHF",
		})
		if !errors.Is(err, ErrInvalidEPC) {
			t.Fatalf("epc %q: want ErrInvalidEPC, got %v", epc, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("epc %q: unexpected DB calls: %v", epc, err)
		}
		mock.Close()
	}
}

// 身份列 23505 分类:uq_assets_sn/mac/loid → ErrAssetIdentityDuplicate + message 含字段名。
func TestClassifyAssetInsertErr_IdentityDuplicate(t *testing.T) {
	for _, constraint := range []string{"uq_assets_sn", "uq_assets_mac", "uq_assets_loid"} {
		orig := &pgconn.PgError{Code: "23505", ConstraintName: constraint}
		err := classifyAssetInsertErr(context.Background(), orig, Asset{AssetCode: "A-X"})
		var dup *ErrAssetIdentityDuplicate
		if !errors.As(err, &dup) {
			t.Fatalf("%s: want ErrAssetIdentityDuplicate, got %v", constraint, err)
		}
		field := strings.TrimPrefix(constraint, "uq_assets_")
		if dup.Field != field {
			t.Fatalf("%s: field = %q, want %q", constraint, dup.Field, field)
		}
		if !strings.Contains(err.Error(), field) {
			t.Fatalf("%s: message missing field name: %s", constraint, err.Error())
		}
	}
}
