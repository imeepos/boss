package adminapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func deleteJSONAuth(r *gin.Engine, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestOrgRoleHandlers 契约:角色改删受 menu:menuperm 门禁;新建返回角色详情;内置拒改 40300;引用中拒删 40900。
func TestOrgRoleHandlers(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")
	res := &user.RoleDetail{ID: 9, Code: "custom_ab", Name: "运维班长",
		PermissionCodes: []string{"menu:dashboard"}}

	t.Run("新建有权限", func(t *testing.T) {
		f := &fakeUser{permOk: true, roleRes: res}
		r := newTestRouter(f, mgr)
		w := postJSONAuth(t, r, "/api/admin/v1/roles",
			`{"name":"运维班长","permissionCodes":["menu:dashboard"]}`, token)
		if envCode(t, w) != apitypes.CodeOK {
			t.Fatalf("body=%s", w.Body.String())
		}
		var body struct {
			Data *struct {
				ID   int64  `json:"id"`
				Code string `json:"code"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Data == nil || body.Data.ID != 9 || body.Data.Code != "custom_ab" {
			t.Fatalf("body=%s", w.Body.String())
		}
	})

	t.Run("无权限拒", func(t *testing.T) {
		f := &fakeUser{permOk: false}
		r := newTestRouter(f, mgr)
		w := postJSONAuth(t, r, "/api/admin/v1/roles", `{"name":"x"}`, token)
		if w.Code != http.StatusForbidden {
			t.Fatalf("status=%d", w.Code)
		}
	})

	t.Run("内置拒改映射 40300", func(t *testing.T) {
		f := &fakeUser{permOk: true, roleErr: user.ErrRoleProtected}
		r := newTestRouter(f, mgr)
		w := putJSONAuth(t, r, "/api/admin/v1/roles/1", `{"name":"x"}`, token)
		if envCode(t, w) != apitypes.CodeForbidden {
			t.Fatalf("body=%s", w.Body.String())
		}
	})

	t.Run("引用中拒删映射 40900", func(t *testing.T) {
		f := &fakeUser{permOk: true, roleErr: user.ErrConflict}
		r := newTestRouter(f, mgr)
		w := deleteJSONAuth(r, "/api/admin/v1/roles/9", token)
		if envCode(t, w) != apitypes.CodeConflict {
			t.Fatalf("body=%s", w.Body.String())
		}
	})
}
