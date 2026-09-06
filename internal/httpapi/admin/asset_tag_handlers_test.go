package adminapi

// 标签域管理端 handler 测试(P2-W2-T1):建标签必填/唯一冲突 40900/审计写入断言;
// 禁用/启用/事件流见同文件后续用例。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/pkg/audit"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// fakeTagAdmin 桩 asset.AssetService(嵌入接口,仅实现标签管理方法)。
type fakeTagAdmin struct {
	asset.AssetService
	created    *asset.Tag
	createErr  error
	disabledID int64
	enabledID  int64
	disableErr error
	tagEvents  []asset.TagEvent
}

func (f *fakeTagAdmin) CreateTag(_ context.Context, t asset.Tag) (int64, error) {
	if f.createErr != nil {
		return 0, f.createErr
	}
	f.created = &t
	return 66, nil
}

// recAudit 审计写入桩:记录全部事件供断言(审计写入断言红线)。
type recAudit struct{ events []audit.Event }

func (r *recAudit) Write(_ context.Context, e audit.Event) error {
	r.events = append(r.events, e)
	return nil
}
func (r *recAudit) List(context.Context, audit.Query) ([]audit.Entry, error) { return nil, nil }

// tagAdminRouter 构造标签域路由测试引擎(登录态 + 全权限 + 审计桩)。
func tagAdminRouter(fa *fakeTagAdmin, au *recAudit) *gin.Engine {
	r := gin.New()
	a := &app.Application{User: &fakeUser{permOk: true}, Asset: fa}
	if au != nil {
		a.Audit = au
	}
	mgr := auth.NewManager("test-secret", time.Hour)
	g := r.Group("/api/admin/v1", middleware.Authn(mgr, auth.AudAdmin))
	registerAssetRoutes(g, a)
	return r
}

func TestTagCreateHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("POST /tags 缺频段→42200", func(t *testing.T) {
		eng := tagAdminRouter(&fakeTagAdmin{}, nil)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/tags", `{"legalEntityId":1,"tagNo":"T-1","epcCode":"E1"}`)
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeInvalidParam) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeInvalidParam)
		}
	})

	t.Run("POST /tags 缺法人→42200", func(t *testing.T) {
		eng := tagAdminRouter(&fakeTagAdmin{}, nil)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/tags", `{"tagNo":"T-1","epcCode":"E1","band":"UHF"}`)
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeInvalidParam) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeInvalidParam)
		}
	})

	t.Run("POST /tags 成功:状态缺省 UNBOUND+审计写入", func(t *testing.T) {
		au := &recAudit{}
		fa := &fakeTagAdmin{}
		eng := tagAdminRouter(fa, au)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/tags", `{"legalEntityId":1,"tagNo":"T-9","epcCode":"E-9","band":"UHF","battery":"90"}`)
		var out struct {
			Code int `json:"code"`
			Data struct {
				ID int64 `json:"id"`
			} `json:"data"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeOK) || out.Data.ID != 66 {
			t.Fatalf("code=%d id=%d", out.Code, out.Data.ID)
		}
		if fa.created == nil || fa.created.Status != "UNBOUND" || fa.created.TagNo != "T-9" {
			t.Fatalf("created=%+v", fa.created)
		}
		// 审计写入断言:数据变更/tag/66,载荷含编号与 EPC。
		if len(au.events) != 1 {
			t.Fatalf("audit events=%d, want 1", len(au.events))
		}
		ev := au.events[0]
		if ev.Action != "数据变更" || ev.TargetType != "tag" || ev.TargetID != "66" {
			t.Fatalf("event=%+v", ev)
		}
		if ev.Detail["tagNo"] != "T-9" || ev.Detail["epcCode"] != "E-9" {
			t.Fatalf("detail=%+v", ev.Detail)
		}
	})

	t.Run("POST /tags 编号重复→40900", func(t *testing.T) {
		eng := tagAdminRouter(&fakeTagAdmin{createErr: fmt.Errorf("asset: tagNo T-1: %w", asset.ErrCodeDuplicate)}, nil)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/tags", `{"legalEntityId":1,"tagNo":"T-1","epcCode":"E1","band":"UHF"}`)
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeConflict) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeConflict)
		}
	})

	t.Run("POST /tags 服务层未知错误→50000", func(t *testing.T) {
		eng := tagAdminRouter(&fakeTagAdmin{createErr: errors.New("db down")}, nil)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/tags", `{"legalEntityId":1,"tagNo":"T-2","epcCode":"E2","band":"UHF"}`)
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeInternal) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeInternal)
		}
	})
}

func (f *fakeTagAdmin) DisableTag(_ context.Context, id int64, _ string) error {
	if f.disableErr != nil {
		return f.disableErr
	}
	f.disabledID = id
	return nil
}

func (f *fakeTagAdmin) EnableTag(_ context.Context, id int64) error {
	f.enabledID = id
	return nil
}

func TestTagDisableEnableHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("POST /tags/5/disable 成功+审计", func(t *testing.T) {
		au := &recAudit{}
		fa := &fakeTagAdmin{}
		eng := tagAdminRouter(fa, au)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/tags/5/disable", `{"reason":"损耗"}`)
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeOK) || fa.disabledID != 5 {
			t.Fatalf("code=%d disabledID=%d", out.Code, fa.disabledID)
		}
		if len(au.events) != 1 || au.events[0].Action != "状态变更" || au.events[0].TargetType != "tag" {
			t.Fatalf("events=%+v", au.events)
		}
		if au.events[0].Detail["op"] != "disable" {
			t.Fatalf("detail=%+v", au.events[0].Detail)
		}
	})

	t.Run("POST /tags/5/disable 绑定中→40900", func(t *testing.T) {
		eng := tagAdminRouter(&fakeTagAdmin{disableErr: fmt.Errorf("asset: tag 5 bound to asset 7, unbind first: %w", asset.ErrBindingConflict)}, nil)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/tags/5/disable", `{}`)
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeConflict) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeConflict)
		}
	})

	t.Run("POST /tags/5/enable 幂等成功", func(t *testing.T) {
		eng := tagAdminRouter(&fakeTagAdmin{}, nil)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/tags/5/enable", "")
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeOK) {
			t.Fatalf("code=%d", out.Code)
		}
	})
}
