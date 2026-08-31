package user

// 企业员工后台录入(000172)域测试:角色白名单/封闭归属/工号校验/启停边界。

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_CreateEntityStaff(t *testing.T) {
	t.Run("角色白名单外拒绝", func(t *testing.T) {
		s := NewPGStore(mustMock(t))
		_, err := s.CreateEntityStaff(context.Background(), 1, AccountInput{
			Username: "emp01", Password: "secret", RealName: "张三", RoleCode: "ops",
		})
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("err=%v, want ErrInvalidInput", err)
		}
	})
	t.Run("封闭归属+工号落库", func(t *testing.T) {
		mock := mustMock(t)
		mock.ExpectQuery(`INSERT INTO accounts`).
			WithArgs("emp01", pgxmock.AnyArg(), "张三", "", "E001",
				int64(7), int64(0), int64(0), "", "partner_staff").
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(9)))
		s := NewPGStore(mock)
		// 入参故意带别的 legalEntityId=99,域层必须以路径参数 7 为准(WithArgs 断言)
		other := int64(99)
		id, err := s.CreateEntityStaff(context.Background(), 7, AccountInput{
			Username: "emp01", Password: "secret", RealName: "张三",
			StaffNo: "E001", RoleCode: "partner_staff", LegalEntityID: &other,
		})
		if err != nil {
			t.Fatalf("CreateEntityStaff: %v", err)
		}
		if id != 9 {
			t.Fatalf("id=%d, want 9", id)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("工号冲突", func(t *testing.T) {
		mock := mustMock(t)
		mock.ExpectQuery(`INSERT INTO accounts`).
			WithArgs("emp01", pgxmock.AnyArg(), "张三", "", "E001",
				int64(1), int64(0), int64(0), "", "partner_staff").
			WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: "uq_accounts_staff_no"})
		s := NewPGStore(mock)
		_, err := s.CreateEntityStaff(context.Background(), 1, AccountInput{
			Username: "emp01", Password: "secret", RealName: "张三",
			StaffNo: "E001", RoleCode: "partner_staff",
		})
		if !errors.Is(err, ErrStaffNoTaken) {
			t.Fatalf("err=%v, want ErrStaffNoTaken", err)
		}
	})
	t.Run("非法工号字符集", func(t *testing.T) {
		s := NewPGStore(mustMock(t))
		_, err := s.CreateEntityStaff(context.Background(), 1, AccountInput{
			Username: "emp01", Password: "secret", RealName: "张三",
			StaffNo: "工号 A", RoleCode: "partner_staff",
		})
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("err=%v, want ErrInvalidInput", err)
		}
	})
}

func TestPGStore_SetEntityStaffStatus(t *testing.T) {
	t.Run("状态值非法", func(t *testing.T) {
		s := NewPGStore(mustMock(t))
		if err := s.SetEntityStaffStatus(context.Background(), 1, 2, 5); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("err=%v, want ErrInvalidInput", err)
		}
	})
	t.Run("企业边界未命中", func(t *testing.T) {
		mock := mustMock(t)
		mock.ExpectExec(`UPDATE accounts a SET status`).
			WithArgs(int64(1), int64(99), int16(0)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		s := NewPGStore(mock)
		if err := s.SetEntityStaffStatus(context.Background(), 1, 99, 0); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
	})
	t.Run("停用命中", func(t *testing.T) {
		mock := mustMock(t)
		mock.ExpectExec(`UPDATE accounts a SET status`).
			WithArgs(int64(1), int64(2), int16(0)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		s := NewPGStore(mock)
		if err := s.SetEntityStaffStatus(context.Background(), 1, 2, 0); err != nil {
			t.Fatalf("SetEntityStaffStatus: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

func TestPGStore_SetEntityStaffPassword(t *testing.T) {
	t.Run("密码过短", func(t *testing.T) {
		s := NewPGStore(mustMock(t))
		if err := s.SetEntityStaffPassword(context.Background(), 1, 2, "123"); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("err=%v, want ErrInvalidInput", err)
		}
	})
	t.Run("重置命中", func(t *testing.T) {
		mock := mustMock(t)
		mock.ExpectExec(`UPDATE accounts a SET password_hash`).
			WithArgs(int64(1), int64(2), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		s := NewPGStore(mock)
		if err := s.SetEntityStaffPassword(context.Background(), 1, 2, "newpass"); err != nil {
			t.Fatalf("SetEntityStaffPassword: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

func TestPGStore_ListEntityStaff(t *testing.T) {
	t.Run("entityID 非法", func(t *testing.T) {
		s := NewPGStore(mustMock(t))
		if _, err := s.ListEntityStaff(context.Background(), 0); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("err=%v, want ErrInvalidInput", err)
		}
	})
	t.Run("仅 partner 角色", func(t *testing.T) {
		mock := mustMock(t)
		mock.ExpectQuery(`FROM accounts a`).
			WithArgs(int64(3)).
			WillReturnRows(mock.NewRows([]string{
				"id", "username", "real_name", "phone", "staff_no",
				"code", "name", "legal_entity_id", "legal_entity",
				"dept_id", "dept_name", "post_id", "post_name", "region_scope", "status",
			}).AddRow(int64(11), "pt_e001", "张三", "13800000000", "E001",
				"partner_admin", "入驻企业管理员", int64(3), "测试企业",
				int64(0), "", int64(0), "", "", int16(1)))
		rows, err := NewPGStore(mock).ListEntityStaff(context.Background(), 3)
		if err != nil {
			t.Fatalf("ListEntityStaff: %v", err)
		}
		if len(rows) != 1 || rows[0].StaffNo != "E001" || rows[0].RoleCode != "partner_admin" {
			t.Fatalf("rows=%+v", rows)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

func mustMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mock.Close)
	return mock
}

var _ = pgx.ErrNoRows
