package user

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// TestPGStore_CreateCustomRole 契约:新建派生角色(名称+权限码全集,事务内绑定)。
func TestPGStore_CreateCustomRole(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO roles`).
		WithArgs(pgxmock.AnyArg(), "运维班长").
		WillReturnRows(mock.NewRows([]string{"id", "code"}).AddRow(int64(9), "custom_ab12"))
	mock.ExpectExec(`DELETE FROM role_permissions`).WithArgs(int64(9)).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))
	mock.ExpectExec(`INSERT INTO role_permissions`).
		WithArgs(int64(9), "menu:dashboard").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	r, err := NewPGStore(mock).CreateCustomRole(context.Background(), "运维班长", []string{"menu:dashboard", "menu:dashboard", " "})
	if err != nil {
		t.Fatalf("CreateCustomRole: %v", err)
	}
	if r.ID != 9 || r.Code != "custom_ab12" || len(r.PermissionCodes) != 1 {
		t.Fatalf("role=%+v", r)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestPGStore_CreateCustomRole_InvalidName 契约:名称空白/超长返回 ErrInvalidInput,不触库。
func TestPGStore_CreateCustomRole_InvalidName(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	if _, err := NewPGStore(mock).CreateCustomRole(context.Background(), "  ", nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("err=%v", err)
	}
}

// TestPGStore_UpdateCustomRole 契约:内置拒改(ErrRoleProtected);派生角色改名+权限集替换。
func TestPGStore_UpdateCustomRole(t *testing.T) {
	t.Run("内置拒改", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT is_builtin FROM roles`).
			WithArgs(int64(1)).WillReturnRows(mock.NewRows([]string{"is_builtin"}).AddRow(true))
		err := NewPGStore(mock).UpdateCustomRole(context.Background(), 1, "改名", nil)
		if !errors.Is(err, ErrRoleProtected) {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("派生角色编辑", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT is_builtin FROM roles`).
			WithArgs(int64(9)).WillReturnRows(mock.NewRows([]string{"is_builtin"}).AddRow(false))
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE roles SET name`).WithArgs(int64(9), "运维组长").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`DELETE FROM role_permissions`).WithArgs(int64(9)).
			WillReturnResult(pgxmock.NewResult("DELETE", 2))
		mock.ExpectExec(`INSERT INTO role_permissions`).
			WithArgs(int64(9), "menu:dashboard").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectCommit()
		if err := NewPGStore(mock).UpdateCustomRole(context.Background(), 9, "运维组长", []string{"menu:dashboard"}); err != nil {
			t.Fatalf("UpdateCustomRole: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

// TestPGStore_DeleteCustomRole 契约:被账号/岗位引用拒(ErrConflict);空闲派生角色删净绑定。
func TestPGStore_DeleteCustomRole(t *testing.T) {
	t.Run("引用中拒删", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT is_builtin FROM roles`).
			WithArgs(int64(9)).WillReturnRows(mock.NewRows([]string{"is_builtin"}).AddRow(false))
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(9)).WillReturnRows(mock.NewRows([]string{"in_use"}).AddRow(true))
		err := NewPGStore(mock).DeleteCustomRole(context.Background(), 9)
		if !errors.Is(err, ErrConflict) {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("空闲删除", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT is_builtin FROM roles`).
			WithArgs(int64(9)).WillReturnRows(mock.NewRows([]string{"is_builtin"}).AddRow(false))
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(9)).WillReturnRows(mock.NewRows([]string{"in_use"}).AddRow(false))
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM role_permissions`).WithArgs(int64(9)).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))
		mock.ExpectExec(`DELETE FROM roles`).WithArgs(int64(9)).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))
		mock.ExpectCommit()
		if err := NewPGStore(mock).DeleteCustomRole(context.Background(), 9); err != nil {
			t.Fatalf("DeleteCustomRole: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

// TestPGStore_ListRoleDetails 契约:详情全集含内置标记与权限码聚合。
func TestPGStore_ListRoleDetails(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`SELECT r.id, r.code, r.name, r.is_builtin`).
		WillReturnRows(mock.NewRows([]string{"id", "code", "name", "is_builtin", "perms"}).
			AddRow(int64(1), "sysadmin", "系统管理员", true, []string{"menu:dashboard", "menu:menuperm"}).
			AddRow(int64(9), "custom_ab", "运维班长", false, []string{"menu:dashboard"}))
	list, err := NewPGStore(mock).ListRoleDetails(context.Background())
	if err != nil {
		t.Fatalf("ListRoleDetails: %v", err)
	}
	if len(list) != 2 || !list[0].IsBuiltin || list[1].IsBuiltin {
		t.Fatalf("list=%+v", list)
	}
	if len(list[0].PermissionCodes) != 2 {
		t.Fatalf("perms=%+v", list[0].PermissionCodes)
	}
}
