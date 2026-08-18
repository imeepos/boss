package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// fakeUser 桩 user.Service:登录/权限/组织列表可配置,其余方法返回零值。
type fakeUser struct {
	loginRes *user.LoginResult
	loginErr error
	permOk   bool
	entities []user.LegalEntity
	accounts []user.AccountRow
}

func (f *fakeUser) Login(ctx context.Context, u, p string) (*user.LoginResult, error) {
	return f.loginRes, f.loginErr
}
func (f *fakeUser) Register(context.Context, string, string, string) (*user.LoginResult, error) {
	return &user.LoginResult{AccountID: 2, Username: "newbie", RealName: "新员工", RoleCode: "ops", RoleName: "业务运营/客服人员"}, nil
}
func (f *fakeUser) HasPermission(context.Context, int64, string) (bool, error) { return f.permOk, nil }
func (f *fakeUser) HasDataScope(context.Context, int64, user.DataScope) (bool, error) {
	return false, nil
}
func (f *fakeUser) ListAddresses(context.Context, int64) ([]user.Address, error)    { return nil, nil }
func (f *fakeUser) ImportAddresses(context.Context, []user.AddressRow) (int, error) { return 0, nil }
func (f *fakeUser) ListRegions(context.Context, string) ([]user.Region, error)      { return nil, nil }
func (f *fakeUser) ListLegalEntities(context.Context) ([]user.LegalEntity, error) {
	return f.entities, nil
}
func (f *fakeUser) CreateLegalEntity(context.Context, user.LegalEntity) (int64, error) {
	return 1, nil
}
func (f *fakeUser) UpdateLegalEntity(context.Context, int64, user.LegalEntity) error {
	return nil
}
func (f *fakeUser) ListMenuPermMatrix(context.Context) (user.MenuPermMatrix, error) {
	return user.MenuPermMatrix{}, nil
}
func (f *fakeUser) ListDepartments(context.Context, int64) ([]user.Department, error) {
	return nil, nil
}
func (f *fakeUser) ListPosts(context.Context, int64) ([]user.Post, error) { return nil, nil }
func (f *fakeUser) GetDataScope(context.Context, int64) (user.DataScope, error) {
	return user.DataScope{}, nil
}
func (f *fakeUser) GetProfile(context.Context, int64) (*user.Profile, error) {
	return &user.Profile{AccountID: 1, Username: "boss", RealName: "老板", RoleName: "系统管理员"}, nil
}
func (f *fakeUser) ListAccounts(context.Context) ([]user.AccountRow, error) {
	return f.accounts, nil
}

func newTestRouter(f *fakeUser, mgr *auth.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r, &Application{User: f}, mgr)
	return r
}

func postJSON(t *testing.T, r *gin.Engine, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestLoginHandler 契约:登录成功返回 token+身份;错误码对齐 apitypes。
func TestLoginHandler(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)

	t.Run("成功", func(t *testing.T) {
		f := &fakeUser{loginRes: &user.LoginResult{
			AccountID: 1, Username: "boss", RealName: "老板", RoleCode: "sysadmin", RoleName: "系统管理员",
		}}
		r := newTestRouter(f, mgr)
		w := postJSON(t, r, "/api/v1/auth/login", `{"username":"boss","password":"secret"}`)

		if w.Code != http.StatusOK {
			t.Fatalf("status=%d", w.Code)
		}
		var body struct {
			Code int `json:"code"`
			Data struct {
				Token     string `json:"token"`
				AccountID int64  `json:"accountId"`
				RealName  string `json:"realName"`
				RoleName  string `json:"roleName"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Code != 0 || body.Data.AccountID != 1 || body.Data.RealName != "老板" || body.Data.RoleName != "系统管理员" {
			t.Fatalf("body=%+v", body)
		}
		if _, err := mgr.Verify(body.Data.Token); err != nil {
			t.Fatalf("token invalid: %v", err)
		}
	})

	t.Run("认证失败", func(t *testing.T) {
		f := &fakeUser{loginErr: user.ErrUnauthorized}
		r := newTestRouter(f, mgr)
		w := postJSON(t, r, "/api/v1/auth/login", `{"username":"boss","password":"bad"}`)

		var body struct {
			Code int32 `json:"code"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Code != int32(apitypes.CodeUnauthorized) {
			t.Fatalf("code=%d, want %d", body.Code, apitypes.CodeUnauthorized)
		}
	})

	t.Run("参数缺失", func(t *testing.T) {
		f := &fakeUser{}
		r := newTestRouter(f, mgr)
		w := postJSON(t, r, "/api/v1/auth/login", `{"username":"boss"}`)

		var body struct {
			Code int32 `json:"code"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Code != int32(apitypes.CodeInvalidParam) {
			t.Fatalf("code=%d, want %d", body.Code, apitypes.CodeInvalidParam)
		}
	})
}
