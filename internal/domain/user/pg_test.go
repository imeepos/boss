package user

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"golang.org/x/crypto/bcrypt"
)

// TestPGStore_ListLegalEntities 契约:返回全部子公司,按 id 升序。
func TestPGStore_ListLegalEntities(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, code, name, is_platform FROM legal_entities ORDER BY id`).
		WillReturnRows(mock.NewRows([]string{"id", "code", "name", "is_platform"}).
			AddRow(int64(1), "LEG-A", "主品牌·企业", false).
			AddRow(int64(2), "LEG-B", "家庭宽带", false).
			AddRow(int64(3), "LEG-C", "批发品牌", false))

	s := NewPGStore(mock)
	got, err := s.ListLegalEntities(context.Background())
	if err != nil {
		t.Fatalf("ListLegalEntities: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len=%d, want 3", len(got))
	}
	if got[0].Code != "LEG-A" || got[1].Code != "LEG-B" || got[2].Code != "LEG-C" {
		t.Fatalf("codes=%+v, want LEG-A/B/C", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_ListRegions 契约:返回全部经营区域,派生父路径并带覆盖主体。
func TestPGStore_ListRegions(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT r.id, r.path, r.level, r.name`).
		WillReturnRows(mock.NewRows([]string{"id", "path", "level", "name", "legal_entity_id", "legal_entity_name"}).
			AddRow(int64(1), "root", int8(1), "集团", int64(9), "平台总公司").
			AddRow(int64(2), "root.luzon", int8(2), "吕宋大区", int64(0), ""))

	s := NewPGStore(mock)
	got, err := s.ListRegions(context.Background(), "")
	if err != nil {
		t.Fatalf("ListRegions: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	if got[0].Parent != "" {
		t.Fatalf("root.Parent=%q, want empty", got[0].Parent)
	}
	if got[1].Parent != "root" {
		t.Fatalf("root.luzon.Parent=%q, want root", got[1].Parent)
	}
	if got[0].LegalEntityID != 9 || got[0].LegalEntityName != "平台总公司" {
		t.Fatalf("root coverage=%+v, want 9/平台总公司", got[0])
	}
	if got[1].LegalEntityID != 0 || got[1].LegalEntityName != "" {
		t.Fatalf("root.luzon coverage=%+v, want uncovered", got[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_AssignRegionCoverage 契约:区域挂/摘覆盖主体;0=摘除(NULLIF);未命中 ErrNotFound。
func TestPGStore_AssignRegionCoverage(t *testing.T) {
	t.Run("挂覆盖", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE regions SET legal_entity_id`).
			WithArgs(int64(2), int64(5)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		if err := NewPGStore(mock).AssignRegionCoverage(context.Background(), 2, 5); err != nil {
			t.Fatalf("AssignRegionCoverage: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("摘覆盖", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE regions SET legal_entity_id`).
			WithArgs(int64(2), int64(0)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		if err := NewPGStore(mock).AssignRegionCoverage(context.Background(), 2, 0); err != nil {
			t.Fatalf("AssignRegionCoverage(0): %v", err)
		}
	})
	t.Run("区域不存在", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE regions SET legal_entity_id`).
			WithArgs(int64(99), int64(5)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		if err := NewPGStore(mock).AssignRegionCoverage(context.Background(), 99, 5); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
	})
}

// TestPGStore_ListDepartments 契约:返回部门并冗余子公司名。
func TestPGStore_ListDepartments(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT d.id, d.legal_entity_id, COALESCE\(le.name, ''\), d.name`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"id", "legal_entity_id", "legal_entity", "name"}).
			AddRow(int64(1), int64(1), "主品牌·企业", "装维调度部").
			AddRow(int64(2), int64(1), "主品牌·企业", "客服部"))

	s := NewPGStore(mock)
	got, err := s.ListDepartments(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListDepartments: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	if got[0].LegalEntity != "主品牌·企业" || got[0].Name != "装维调度部" {
		t.Fatalf("got[0]=%+v", got[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_ListPosts 契约:返回岗位并聚合角色码(逗号串拆分)。
func TestPGStore_ListPosts(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT p.id, p.code, p.name`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"id", "code", "name", "dept_id", "dept_name", "roles"}).
			AddRow(int64(1), "dispatcher", "装维调度员", int64(1), "装维调度部", "technician").
			AddRow(int64(2), "agent", "客服坐席", int64(2), "客服部", "ops,analyst"))

	s := NewPGStore(mock)
	got, err := s.ListPosts(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListPosts: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	if len(got[0].Roles) != 1 || got[0].Roles[0] != "technician" {
		t.Fatalf("got[0].Roles=%v, want [technician]", got[0].Roles)
	}
	if len(got[1].Roles) != 2 || got[1].Roles[1] != "analyst" {
		t.Fatalf("got[1].Roles=%v, want [ops analyst]", got[1].Roles)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_GetDataScope 契约:返回账号数据范围;未命中返回 ErrNotFound。
func TestPGStore_GetDataScope(t *testing.T) {
	t.Run("命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT id, COALESCE\(legal_entity_id, 0\)`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows([]string{"id", "legal_entity_id", "dept_id", "post_id", "region_scope"}).
				AddRow(int64(1), int64(1), int64(2), int64(3), "root.luzon"))

		s := NewPGStore(mock)
		ds, err := s.GetDataScope(context.Background(), 1)
		if err != nil {
			t.Fatalf("GetDataScope: %v", err)
		}
		if ds.LegalEntityID != 1 || ds.DeptID != 2 || ds.PostID != 3 || ds.RegionScope != "root.luzon" {
			t.Fatalf("ds=%+v", ds)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
	t.Run("未命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT id, COALESCE\(legal_entity_id, 0\)`).
			WithArgs(int64(99)).
			WillReturnError(pgx.ErrNoRows)

		s := NewPGStore(mock)
		_, err = s.GetDataScope(context.Background(), 99)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

// TestPGStore_ListAddresses 契约:返回地址层级。
func TestPGStore_ListAddresses(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT a\.id, COALESCE\(a\.parent_id, 0\), a\.level, a\.name`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"id", "parent_id", "level", "name", "country_code", "admin_code", "has_children"}).
			AddRow(int64(2), int64(1), int8(2), "朝阳区", "CN", "CN-BJ", true).
			AddRow(int64(3), int64(1), int8(2), "海淀区", "CN", "CN-BJ", false))

	s := NewPGStore(mock)
	got, err := s.ListAddresses(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListAddresses: %v", err)
	}
	if len(got) != 2 || got[0].Name != "朝阳区" || got[1].ParentID != 1 {
		t.Fatalf("got=%+v", got)
	}
	if got[0].CountryCode != "CN" || got[0].AdminCode != "CN-BJ" {
		t.Fatalf("geo anchor not inherited: %+v", got[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_HasPermission 契约:账号角色绑定权限码即返回 true。
func TestPGStore_HasPermission(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1), "menu:order").
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))

	s := NewPGStore(mock)
	got, err := s.HasPermission(context.Background(), 1, "menu:order")
	if err != nil {
		t.Fatalf("HasPermission: %v", err)
	}
	if !got {
		t.Fatal("want true")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_HasDataScope 契约:组织/区域越权判定。
func TestPGStore_HasDataScope(t *testing.T) {
	scopeRow := func(le, dept, post int64, region string) *pgxmock.Rows {
		return pgxmock.NewRows([]string{"id", "legal_entity_id", "dept_id", "post_id", "region_scope"}).
			AddRow(int64(1), le, dept, post, region)
	}

	tests := []struct {
		name  string
		scope []any // legal_entity_id, dept_id, post_id, region_scope
		owner DataScope
		want  bool
	}{
		{"全集团放行", []any{int64(0), int64(0), int64(0), ""}, DataScope{LegalEntityID: 2, DeptID: 5, PostID: 7, RegionScope: "root.luzon"}, true},
		{"子公司不匹配", []any{int64(1), int64(0), int64(0), ""}, DataScope{LegalEntityID: 2}, false},
		{"区域子树内", []any{int64(0), int64(0), int64(0), "root.luzon"}, DataScope{RegionScope: "root.luzon.ncr"}, true},
		{"区域子树外", []any{int64(0), int64(0), int64(0), "root.luzon"}, DataScope{RegionScope: "root.visayas"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatal(err)
			}
			defer mock.Close()

			mock.ExpectQuery(`FROM accounts WHERE id`).
				WithArgs(int64(1)).
				WillReturnRows(scopeRow(tt.scope[0].(int64), tt.scope[1].(int64), tt.scope[2].(int64), tt.scope[3].(string)))

			s := NewPGStore(mock)
			got, err := s.HasDataScope(context.Background(), 1, tt.owner)
			if err != nil {
				t.Fatalf("HasDataScope: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got=%v, want=%v", got, tt.want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet expectations: %v", err)
			}
		})
	}
}

// TestPGStore_Login 契约:口令正确返回认证身份;错误/停用/不存在统一 ErrUnauthorized。
func TestPGStore_Login(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("口令正确", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT a.id, a.real_name, r.code, r.name, a.password_hash, a.status`).
			WithArgs("boss").
			WillReturnRows(mock.NewRows([]string{"id", "real_name", "code", "name", "password_hash", "status"}).
				AddRow(int64(1), "老板", "sysadmin", "系统管理员", string(hash), int16(1)))

		s := NewPGStore(mock)
		res, err := s.Login(context.Background(), "boss", "secret")
		if err != nil {
			t.Fatalf("Login: %v", err)
		}
		if res == nil || res.AccountID != 1 || res.Username != "boss" || res.RealName != "老板" || res.RoleCode != "sysadmin" || res.RoleName != "系统管理员" {
			t.Fatalf("res=%+v", res)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
	t.Run("口令错误", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT a.id, a.real_name, r.code, r.name, a.password_hash, a.status`).
			WithArgs("boss").
			WillReturnRows(mock.NewRows([]string{"id", "real_name", "code", "name", "password_hash", "status"}).
				AddRow(int64(1), "老板", "sysadmin", "系统管理员", string(hash), int16(1)))

		s := NewPGStore(mock)
		_, err = s.Login(context.Background(), "boss", "wrong")
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("err=%v, want ErrUnauthorized", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
	t.Run("停用", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT a.id, a.real_name, r.code, r.name, a.password_hash, a.status`).
			WithArgs("boss").
			WillReturnRows(mock.NewRows([]string{"id", "real_name", "code", "name", "password_hash", "status"}).
				AddRow(int64(1), "老板", "sysadmin", "系统管理员", string(hash), int16(0)))

		s := NewPGStore(mock)
		_, err = s.Login(context.Background(), "boss", "secret")
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("err=%v, want ErrUnauthorized", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

// TestPGStore_ImportAddresses 契约:批量导入,level/parent_id 由 path 派生。
func TestPGStore_ImportAddresses(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// 顶层节点 bj(无父):直接 insert(带国家锚点)
	mock.ExpectExec(`INSERT INTO addresses`).
		WithArgs("bj", int8(1), "北京市", nil, "CN", "CN-BJ").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	// 子节点 bj.chaoyang:查父 + insert(非根行锚点强制忽略为空串)
	mock.ExpectQuery(`SELECT id FROM addresses WHERE path`).
		WithArgs("bj").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))
	mock.ExpectExec(`INSERT INTO addresses`).
		WithArgs("bj.chaoyang", int8(2), "朝阳区", int64(1), "", "").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	s := NewPGStore(mock)
	imported, err := s.ImportAddresses(context.Background(), []AddressRow{
		{Path: "bj", Name: "北京市", CountryCode: "CN", AdminCode: "CN-BJ"},
		{Path: "bj.chaoyang", Name: "朝阳区", CountryCode: "CN", AdminCode: "CN-BJ"},
	})
	if err != nil {
		t.Fatalf("ImportAddresses: %v", err)
	}
	if imported != 2 {
		t.Fatalf("imported=%d, want 2", imported)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_GetProfile 契约:联表返回用户信息(角色名/公司名);未命中 ErrNotFound。
func TestPGStore_GetProfile(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT a.id, a.username, a.real_name, COALESCE\(a.phone, ''\), r.code, r.name, COALESCE\(le.name, ''\), COALESCE\(a.region_scope`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"id", "username", "real_name", "phone", "code", "name", "legal_entity_name", "region_scope"}).
			AddRow(int64(1), "boss", "老板", "13800000000", "sysadmin", "系统管理员", "主品牌·企业", "root.luzon"))

	s := NewPGStore(mock)
	p, err := s.GetProfile(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if p.RealName != "老板" || p.Phone != "13800000000" || p.RoleName != "系统管理员" || p.LegalEntityName != "主品牌·企业" {
		t.Fatalf("p=%+v", p)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
