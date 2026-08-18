package app

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// fakeOrgAdmin 在 fakeUser 基础上落账法人写操作与权限矩阵。
type fakeOrgAdmin struct {
	*fakeUser
	created *user.LegalEntity
	updated *struct {
		id int64
		e  user.LegalEntity
	}
	matrix user.MenuPermMatrix
}

func (f *fakeOrgAdmin) CreateLegalEntity(_ context.Context, e user.LegalEntity) (int64, error) {
	f.created = &e
	return 7, nil
}
func (f *fakeOrgAdmin) UpdateLegalEntity(_ context.Context, id int64, e user.LegalEntity) error {
	f.updated = &struct {
		id int64
		e  user.LegalEntity
	}{id, e}
	return nil
}
func (f *fakeOrgAdmin) ListMenuPermMatrix(context.Context) (user.MenuPermMatrix, error) {
	return f.matrix, nil
}

func newOrgAdminRouter(f *fakeOrgAdmin, mgr *auth.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r, &Application{User: f}, mgr)
	return r
}

// TestCreateLegalEntity 契约:新建法人,code/name 必填,返回 id。
func TestCreateLegalEntity(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	f := &fakeOrgAdmin{fakeUser: &fakeUser{permOk: true}}
	r := newOrgAdminRouter(f, mgr)

	w := postBodyAuth(t, r, "/api/v1/legal-entities", `{"code":"LEG-D","name":"D 公司"}`, authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if f.created == nil || f.created.Code != "LEG-D" || f.created.Name != "D 公司" {
		t.Fatalf("created=%+v", f.created)
	}
	var body struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Data.ID != 7 {
		t.Fatalf("body=%s", w.Body.String())
	}
}

// TestCreateLegalEntityMissingName 契约:缺 name 拒绝(InvalidParam)。
func TestCreateLegalEntityMissingName(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	f := &fakeOrgAdmin{fakeUser: &fakeUser{permOk: true}}
	r := newOrgAdminRouter(f, mgr)

	w := postBodyAuth(t, r, "/api/v1/legal-entities", `{"code":"LEG-D"}`, authToken(t, mgr))
	var body struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Code == 0 {
		t.Fatalf("应拒绝缺 name: %s", w.Body.String())
	}
}

// TestUpdateLegalEntity 契约:按 id 编辑法人。
func TestUpdateLegalEntity(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	f := &fakeOrgAdmin{fakeUser: &fakeUser{permOk: true}}
	r := newOrgAdminRouter(f, mgr)

	w := putAuth(t, r, "/api/v1/legal-entities/7", `{"code":"LEG-D","name":"D 集团"}`, authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if f.updated == nil || f.updated.id != 7 || f.updated.e.Name != "D 集团" {
		t.Fatalf("updated=%+v", f.updated)
	}
}

// TestMenuPerms 契约:权限矩阵含三层模型说明 + 角色×菜单行。
func TestMenuPerms(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	f := &fakeOrgAdmin{fakeUser: &fakeUser{permOk: true}, matrix: user.MenuPermMatrix{
		RoleColumns: []user.MenuRoleCol{{RoleCode: "sysadmin", RoleName: "系统管理员"}},
		Rows:        []user.MenuPermRow{{Code: "menu:dashboard", Name: "运营总览·工作台", Roles: []string{"sysadmin"}}},
	}}
	r := newOrgAdminRouter(f, mgr)

	w := getJSON(t, r, "/api/v1/menu-perms", authToken(t, mgr))
	var body struct {
		Code int `json:"code"`
		Data struct {
			Model struct {
				Layers []string `json:"layers"`
			} `json:"model"`
			Matrix user.MenuPermMatrix `json:"matrix"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 0 || len(body.Data.Model.Layers) != 3 {
		t.Fatalf("model=%+v", body.Data.Model)
	}
	rows := body.Data.Matrix.Rows
	if len(rows) != 1 || rows[0].Code != "menu:dashboard" || len(rows[0].Roles) != 1 ||
		rows[0].Roles[0] != "sysadmin" {
		t.Fatalf("matrix=%+v", body.Data.Matrix)
	}
}

// TestDataScopes 契约(org.yaml /data-scopes):返回账号数据范围清单,menu:datascope 门禁。
func TestDataScopes(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	f := &fakeOrgAdmin{fakeUser: &fakeUser{permOk: true, accounts: []user.AccountRow{{
		ID: 1, Username: "boss", RealName: "老板", RoleName: "系统管理员", RegionScope: "",
	}}}}
	r := newOrgAdminRouter(f, mgr)

	w := getJSON(t, r, "/api/v1/data-scopes?keyword=boss", authToken(t, mgr))
	var body struct {
		Code int               `json:"code"`
		Data []user.AccountRow `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK || body.Code != 0 || len(body.Data) != 1 ||
		body.Data[0].Username != "boss" {
		t.Fatalf("status=%d code=%d rows=%+v", w.Code, body.Code, body.Data)
	}
}
