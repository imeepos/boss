package asset

// P1-T3 型号字典单测:列表/建型冲突/CreateAsset 挂型号派生 Type。

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_ListModels(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`FROM asset_models ORDER BY`).
		WillReturnRows(mock.NewRows([]string{"id", "vendor", "model", "category", "part_number", "spec", "is_active", "created_at"}).
			AddRow(int64(1), "(存量未登记)", "ONU", "ONU", "", nil, true, ts).
			AddRow(int64(2), "华为", "EchoLife", "ONU", "HG8145V", nil, false, ts))

	s := NewPGStore(mock)
	got, err := s.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(got) != 2 || got[0].Model != "ONU" || got[1].PartNumber != "HG8145V" || got[1].IsActive {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateModel(t *testing.T) {
	t.Run("新建成功", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`INSERT INTO asset_models`).
			WithArgs("华为", "EchoLife", "ONU", "HG8145V", pgxmock.AnyArg()).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(9)))

		s := NewPGStore(mock)
		id, err := s.CreateModel(context.Background(), AssetModel{Vendor: "华为", Model: "EchoLife", Category: "ONU", PartNumber: "HG8145V"})
		if err != nil || id != 9 {
			t.Fatalf("id=%d err=%v", id, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("spec 缺省归一为空对象(回归:nil 直插 23502)", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`INSERT INTO asset_models`).
			WithArgs("华为", "EchoLife", "ONU", "", []byte("{}")).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(9)))

		s := NewPGStore(mock)
		id, err := s.CreateModel(context.Background(), AssetModel{Vendor: "华为", Model: "EchoLife", Category: "ONU"})
		if err != nil || id != 9 {
			t.Fatalf("id=%d err=%v", id, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("四元组重复 → ErrModelExists", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`INSERT INTO asset_models`).
			WithArgs("华为", "EchoLife", "ONU", "HG8145V", pgxmock.AnyArg()).
			WillReturnError(pgx.ErrNoRows)

		s := NewPGStore(mock)
		if _, err := s.CreateModel(context.Background(), AssetModel{Vendor: "华为", Model: "EchoLife", Category: "ONU", PartNumber: "HG8145V"}); !errors.Is(err, ErrModelExists) {
			t.Fatalf("err=%v, want ErrModelExists", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

func TestPGStore_CreateAsset_WithModel(t *testing.T) {
	t.Run("挂型号且 Type 空 → 派生 category", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT category, is_active FROM asset_models WHERE id`).
			WithArgs(int64(77)).
			WillReturnRows(mock.NewRows([]string{"category", "is_active"}).AddRow("OLT", true))
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO assets`).
			WithArgs("A-MODEL1", int64(1), int64(1), "主品牌·企业", nil, nil, nil, "", "OLT", "IN_STOCK", int64(77)).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(30)))
		mock.ExpectExec(`INSERT INTO asset_lifecycles`).
			WithArgs(int64(30), "IN_STOCK").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectCommit()

		s := NewPGStore(mock)
		id, err := s.CreateAsset(context.Background(), Asset{AssetCode: "A-MODEL1", BatchID: 1, LegalEntityID: 1, LegalEntityName: "主品牌·企业", ModelID: 77, Status: "IN_STOCK"})
		if err != nil || id != 30 {
			t.Fatalf("id=%d err=%v", id, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("型号不存在 → ErrForeignKeyViolation", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT category, is_active FROM asset_models WHERE id`).
			WithArgs(int64(99)).
			WillReturnError(pgx.ErrNoRows)

		s := NewPGStore(mock)
		if _, err := s.CreateAsset(context.Background(), Asset{AssetCode: "A-X", ModelID: 99, Type: "ONU", Status: "IN_STOCK"}); !errors.Is(err, ErrForeignKeyViolation) {
			t.Fatalf("err=%v, want ErrForeignKeyViolation", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

// UpdateModel(P2-W2-T1 D):正常编辑 / 唯一冲突 40900 / 停用拒绝 / 不存在 404。
func TestPGStore_UpdateModel(t *testing.T) {
	ctx := context.Background()
	in := AssetModel{Vendor: "华为", Model: "HN-ONU-X2", Category: "ONU", PartNumber: "PN-2", Spec: map[string]any{"ports": 2}}
	t.Run("正常编辑", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery("SELECT is_active FROM asset_models").
			WithArgs(int64(3)).
			WillReturnRows(mock.NewRows([]string{"is_active"}).AddRow(true))
		mock.ExpectExec("UPDATE asset_models SET vendor").
			WithArgs(int64(3), in.Vendor, in.Model, in.Category, in.PartNumber, []byte(`{"ports":2}`)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		s := NewPGStore(mock)
		if err := s.UpdateModel(ctx, 3, in); err != nil {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("四元组冲突40900", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery("SELECT is_active FROM asset_models").
			WithArgs(int64(3)).
			WillReturnRows(mock.NewRows([]string{"is_active"}).AddRow(true))
		mock.ExpectExec("UPDATE asset_models SET vendor").
			WithArgs(int64(3), in.Vendor, in.Model, in.Category, in.PartNumber, []byte(`{"ports":2}`)).
			WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: "uq_asset_models"})
		s := NewPGStore(mock)
		if err := s.UpdateModel(ctx, 3, in); !errors.Is(err, ErrModelExists) {
			t.Fatalf("err=%v, want ErrModelExists", err)
		}
	})
	t.Run("停用型号拒绝编辑", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery("SELECT is_active FROM asset_models").
			WithArgs(int64(3)).
			WillReturnRows(mock.NewRows([]string{"is_active"}).AddRow(false))
		s := NewPGStore(mock)
		if err := s.UpdateModel(ctx, 3, in); !errors.Is(err, ErrModelInactive) {
			t.Fatalf("err=%v, want ErrModelInactive", err)
		}
	})
	t.Run("不存在404", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery("SELECT is_active FROM asset_models").
			WithArgs(int64(3)).
			WillReturnError(pgx.ErrNoRows)
		s := NewPGStore(mock)
		if err := s.UpdateModel(ctx, 3, in); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
	})
}

// SetModelActive(P2-W2-T1 E):停用/启用/幂等/不存在。
func TestPGStore_SetModelActive(t *testing.T) {
	ctx := context.Background()
	t.Run("停用成功", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec("UPDATE asset_models SET is_active").
			WithArgs(int64(3), false).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		s := NewPGStore(mock)
		if err := s.SetModelActive(ctx, 3, false); err != nil {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("启用幂等成功", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec("UPDATE asset_models SET is_active").
			WithArgs(int64(3), true).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		s := NewPGStore(mock)
		if err := s.SetModelActive(ctx, 3, true); err != nil {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("不存在404", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec("UPDATE asset_models SET is_active").
			WithArgs(int64(3), true).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		s := NewPGStore(mock)
		if err := s.SetModelActive(ctx, 3, true); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
	})
}
