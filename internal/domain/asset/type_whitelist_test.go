// 类型写入白名单单测(P4-T2):合法值放行/方言拒绝/建档与编辑路径的
// 白名单闸门(含型号派生后的最终 type)。错误断言一律 errors.Is/As(%w 包装)。
package asset

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestValidateType(t *testing.T) {
	for _, ok := range TypeWhitelist {
		if err := ValidateType(ok); err != nil {
			t.Fatalf("ValidateType(%q) = %v, want nil", ok, err)
		}
	}
	for _, dialect := range []string{"", "光猫", "MI-ONU", "SMOKE", "onu", "ROUTER-X"} {
		err := ValidateType(dialect)
		var te *ErrTypeNotAllowed
		if !errors.As(err, &te) {
			t.Fatalf("ValidateType(%q) = %v, want ErrTypeNotAllowed", dialect, err)
		}
		if te.Type != dialect || len(te.Allowed) != len(TypeWhitelist) {
			t.Fatalf("payload mismatch: %+v", te)
		}
	}
}

// 建档:白名单外 type → ErrTypeNotAllowed;校验在批次快照/型号派生之后、
// 建事务之前,事务零 SQL(先校验后写库)。
func TestPGStore_CreateAsset_TypeWhitelist(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`LEFT JOIN legal_entities`).WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"legal_entity_id", "name"}).AddRow(int64(1), "主品牌·企业"))

	s := NewPGStore(mock)
	_, err = s.CreateAsset(context.Background(), Asset{BatchID: 1, Type: "光猫", Status: "IN_STOCK"})
	var te *ErrTypeNotAllowed
	if !errors.As(err, &te) {
		t.Fatalf("want ErrTypeNotAllowed, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// 建档:合法 type 正常入库(闸门不误伤)。
func TestPGStore_CreateAsset_TypeWhitelisted(t *testing.T) {
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
	// 企业快照自批次回填(LE=1/主品牌·企业),与建档主路径同口径。
	mock.ExpectQuery(`INSERT INTO assets`).
		WithArgs("A-1", int64(1), int64(1), "主品牌·企业", nil, nil, nil, "", "ONU", "IN_STOCK", nil, nil, nil, nil).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(9)))
	mock.ExpectExec(`INSERT INTO asset_lifecycles`).
		WithArgs(int64(9), "IN_STOCK").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	s := NewPGStore(mock)
	if _, err := s.CreateAsset(context.Background(), Asset{AssetCode: "A-1", BatchID: 1, Type: "ONU", Status: "IN_STOCK"}); err != nil {
		t.Fatalf("CreateAsset: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// 编辑:白名单外显式 type → ErrTypeNotAllowed,不进入批次/标签换绑分支。
func TestPGStore_UpdateAsset_TypeWhitelist(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`FOR UPDATE`).WithArgs(pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"status", "type", "batch_id", "model_id", "tag_id", "le_id", "le_name", "sn", "mac", "loid"}).
			AddRow("IN_STOCK", "ONU", int64(1), int64(0), int64(0), int64(1), "主品牌·企业", "", "", ""))

	s := NewPGStore(mock)
	err = s.UpdateAsset(context.Background(), 3, AssetUpdate{Type: "SMOKE", BatchID: 1}, 42)
	var te *ErrTypeNotAllowed
	if !errors.As(err, &te) {
		t.Fatalf("want ErrTypeNotAllowed, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// 编辑:保持现值(newType==atype)不触发白名单,存量值不被编辑通道误伤。
func TestPGStore_UpdateAsset_TypeKeepSkipsWhitelist(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// 现值 ONU,in.Type 空 → 保持;四键无变化 → 零写入幂等成功。
	mock.ExpectBegin()
	mock.ExpectQuery(`FOR UPDATE`).WithArgs(pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"status", "type", "batch_id", "model_id", "tag_id", "le_id", "le_name", "sn", "mac", "loid"}).
			AddRow("IN_STOCK", "ONU", int64(1), int64(0), int64(0), int64(1), "主品牌·企业", "", "", ""))

	s := NewPGStore(mock)
	if err := s.UpdateAsset(context.Background(), 3, AssetUpdate{BatchID: 1}, 42); err != nil {
		t.Fatalf("UpdateAsset keep: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
