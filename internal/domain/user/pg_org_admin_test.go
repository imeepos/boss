package user

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// TestPGStore_LegalEntityWrite 契约:法人可新建;编辑未命中返回 ErrNotFound。
func TestPGStore_LegalEntityWrite(t *testing.T) {
	t.Run("新建", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`INSERT INTO legal_entities`).
			WithArgs("LEG-D", "D 公司").
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(7)))
		id, err := NewPGStore(mock).CreateLegalEntity(context.Background(), LegalEntity{Code: "LEG-D", Name: "D 公司"})
		if err != nil || id != 7 {
			t.Fatalf("id=%d err=%v", id, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("编辑", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE legal_entities SET code=\$2, name=\$3 WHERE id=\$1`).
			WithArgs(int64(7), "LEG-D", "D 集团").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		if err := NewPGStore(mock).UpdateLegalEntity(context.Background(), 7,
			LegalEntity{Code: "LEG-D", Name: "D 集团"}); err != nil {
			t.Fatalf("UpdateLegalEntity: %v", err)
		}
	})
	t.Run("编辑未命中", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE legal_entities`).WithArgs(int64(99), "LEG-X", "X").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		err := NewPGStore(mock).UpdateLegalEntity(context.Background(), 99,
			LegalEntity{Code: "LEG-X", Name: "X"})
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v", err)
		}
	})
}

// TestPGStore_ListMenuPermMatrix 契约:矩阵=角色列 × menu:* 权限行(行内含角色码)。
func TestPGStore_ListMenuPermMatrix(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery(`SELECT code, name, is_builtin FROM roles ORDER BY is_builtin DESC, id`).
		WillReturnRows(mock.NewRows([]string{"code", "name", "is_builtin"}).
			AddRow("sysadmin", "系统管理员", true).
			AddRow("ops", "运营", true).
			AddRow("custom_ab", "派生角色", false))
	mock.ExpectQuery(`SELECT p.code, p.name`).
		WillReturnRows(mock.NewRows([]string{"code", "name", "roles"}).
			AddRow("menu:dashboard", "运营总览·工作台", []string{"ops", "sysadmin"}).
			AddRow("menu:company", "子公司/法人", []string{"sysadmin"}))

	m, err := NewPGStore(mock).ListMenuPermMatrix(context.Background())
	if err != nil {
		t.Fatalf("ListMenuPermMatrix: %v", err)
	}
	if len(m.RoleColumns) != 3 || m.RoleColumns[0].RoleCode != "sysadmin" {
		t.Fatalf("roles=%+v", m.RoleColumns)
	}
	if !m.RoleColumns[0].IsBuiltin || m.RoleColumns[2].IsBuiltin {
		t.Fatalf("is_builtin 标记错误: %+v", m.RoleColumns)
	}
	if len(m.Rows) != 2 {
		t.Fatalf("rows=%+v", m.Rows)
	}
	var dash *MenuPermRow
	for i := range m.Rows {
		if m.Rows[i].Code == "menu:dashboard" {
			dash = &m.Rows[i]
		}
	}
	if dash == nil || len(dash.Roles) != 2 || dash.Roles[0] != "ops" {
		t.Fatalf("dash=%+v", dash)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
