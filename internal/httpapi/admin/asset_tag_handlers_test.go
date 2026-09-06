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
	created      *asset.Tag
	createErr    error
	disabledID   int64
	enabledID    int64
	modelUpdated int64
	modelActive  map[int64]bool
	batch        *asset.AssetBatch
	batchErr     error
	assigned     *asset.AssetAssignment
	assignErr    error
	returnedID   int64
	returnErr    error
	cancelledID  int64
	cancelErr    error
	modelErr     error
	disableErr   error
	tagEvents    []asset.TagEvent
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

func (f *fakeTagAdmin) UpdateModel(_ context.Context, id int64, _ asset.AssetModel) error {
	if f.modelErr != nil {
		return f.modelErr
	}
	f.modelUpdated = id
	return nil
}

func TestModelUpdateHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("PUT /asset-models/3 编辑成功+审计", func(t *testing.T) {
		au := &recAudit{}
		fa := &fakeTagAdmin{}
		eng := tagAdminRouter(fa, au)
		w := doJSON(eng, http.MethodPut, "/api/admin/v1/asset-models/3", `{"vendor":"华为","model":"X2","category":"ONU","partNumber":"PN"}`)
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeOK) || fa.modelUpdated != 3 {
			t.Fatalf("code=%d id=%d", out.Code, fa.modelUpdated)
		}
		if len(au.events) != 1 || au.events[0].TargetType != "asset_model" || au.events[0].Detail["op"] != "update" {
			t.Fatalf("events=%+v", au.events)
		}
	})

	t.Run("PUT /asset-models/3 停用型号→40900", func(t *testing.T) {
		fa := &fakeTagAdmin{modelErr: fmt.Errorf("asset: model 3 deactivated, enable first: %w", asset.ErrModelInactive)}
		eng := tagAdminRouter(fa, nil)
		w := doJSON(eng, http.MethodPut, "/api/admin/v1/asset-models/3", `{"model":"X2","category":"ONU"}`)
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeConflict) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeConflict)
		}
	})

	t.Run("PUT /asset-models/3 缺类别→42200", func(t *testing.T) {
		eng := tagAdminRouter(&fakeTagAdmin{}, nil)
		w := doJSON(eng, http.MethodPut, "/api/admin/v1/asset-models/3", `{"model":"X2"}`)
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeInvalidParam) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeInvalidParam)
		}
	})
}

func (f *fakeTagAdmin) SetModelActive(_ context.Context, id int64, active bool) error {
	if f.modelErr != nil {
		return f.modelErr
	}
	f.modelActive[id] = active
	return nil
}

func TestModelDisableEnableHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("POST /asset-models/3/disable 成功+审计", func(t *testing.T) {
		au := &recAudit{}
		fa := &fakeTagAdmin{modelActive: map[int64]bool{}}
		eng := tagAdminRouter(fa, au)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/asset-models/3/disable", "")
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeOK) || fa.modelActive[3] != false {
			t.Fatalf("code=%d modelActive=%v", out.Code, fa.modelActive)
		}
		if len(au.events) != 1 || au.events[0].Detail["op"] != "disable" {
			t.Fatalf("events=%+v", au.events)
		}
	})

	t.Run("POST /asset-models/3/enable 幂等成功", func(t *testing.T) {
		eng := tagAdminRouter(&fakeTagAdmin{modelActive: map[int64]bool{}}, nil)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/asset-models/3/enable", "")
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeOK) {
			t.Fatalf("code=%d", out.Code)
		}
	})

	t.Run("POST /asset-models/9/enable 未命中→40400", func(t *testing.T) {
		eng := tagAdminRouter(&fakeTagAdmin{modelActive: map[int64]bool{}, modelErr: asset.ErrNotFound}, nil)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/asset-models/9/enable", "")
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeNotFound) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeNotFound)
		}
	})
}

func (f *fakeTagAdmin) CreateBatch(_ context.Context, b asset.AssetBatch) (int64, error) {
	if f.batchErr != nil {
		return 0, f.batchErr
	}
	f.batch = &b
	return 88, nil
}

func TestBatchCreateHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("POST /asset-batches 缺名称→42200", func(t *testing.T) {
		eng := tagAdminRouter(&fakeTagAdmin{}, nil)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/asset-batches", `{"legalEntityId":1}`)
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeInvalidParam) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeInvalidParam)
		}
	})

	t.Run("POST /asset-batches 编码缺省自动生成 RK 风格", func(t *testing.T) {
		au := &recAudit{}
		fa := &fakeTagAdmin{}
		eng := tagAdminRouter(fa, au)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/asset-batches", `{"legalEntityId":1,"name":"9月光猫"}`)
		var out struct {
			Code int `json:"code"`
			Data struct {
				ID   int64  `json:"id"`
				Code string `json:"code"`
			} `json:"data"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeOK) || out.Data.ID != 88 {
			t.Fatalf("code=%d id=%d", out.Code, out.Data.ID)
		}
		// RK-YYYYMMDD-NNNNN:前缀+8 位日期+5 位序号(17 字符,对齐采购 nextBatchCode)。
		if len(out.Data.Code) != 17 || out.Data.Code[:3] != "RK-" || out.Data.Code[11:12] != "-" {
			t.Fatalf("generated code=%q", out.Data.Code)
		}
		if fa.batch.LegalEntityID != 1 || fa.batch.Name != "9月光猫" {
			t.Fatalf("batch=%+v", fa.batch)
		}
	})

	t.Run("POST /asset-batches 法人不存在→42200", func(t *testing.T) {
		eng := tagAdminRouter(&fakeTagAdmin{batchErr: fmt.Errorf("asset: legal entity 9: %w", asset.ErrForeignKeyViolation)}, nil)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/asset-batches", `{"legalEntityId":9,"name":"X"}`)
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeInvalidParam) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeInvalidParam)
		}
	})
}

func (f *fakeTagAdmin) CreateAssignment(_ context.Context, a asset.AssetAssignment) (int64, error) {
	if f.assignErr != nil {
		return 0, f.assignErr
	}
	f.assigned = &a
	return 21, nil
}

func TestAssignmentCreateHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("POST /asset-assignments 缺事由→42200", func(t *testing.T) {
		eng := tagAdminRouter(&fakeTagAdmin{}, nil)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/asset-assignments", `{"assetId":5,"workerId":7}`)
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeInvalidParam) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeInvalidParam)
		}
	})

	t.Run("POST /asset-assignments 领用成功+审计+开段", func(t *testing.T) {
		au := &recAudit{}
		fa := &fakeTagAdmin{}
		eng := tagAdminRouter(fa, au)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/asset-assignments", `{"assetId":5,"workerId":7,"reason":"装机备件"}`)
		var out struct {
			Code int `json:"code"`
			Data struct {
				ID            int64  `json:"id"`
				EffectiveFrom string `json:"effectiveFrom"`
			} `json:"data"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeOK) || out.Data.ID != 21 {
			t.Fatalf("code=%d id=%d", out.Code, out.Data.ID)
		}
		if out.Data.EffectiveFrom == "" {
			t.Fatalf("effectiveFrom empty")
		}
		if fa.assigned == nil || fa.assigned.AssetID != 5 || fa.assigned.Reason != "装机备件" {
			t.Fatalf("assigned=%+v", fa.assigned)
		}
		if len(au.events) != 1 || au.events[0].TargetType != "asset_assignment" || au.events[0].Detail["op"] != "assign" {
			t.Fatalf("events=%+v", au.events)
		}
	})

	t.Run("POST /asset-assignments 非库存态→40900", func(t *testing.T) {
		eng := tagAdminRouter(&fakeTagAdmin{assignErr: fmt.Errorf("asset: asset 5 status DEPLOYED: %w", asset.ErrAssetNotInStock)}, nil)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/asset-assignments", `{"assetId":5,"workerId":7,"reason":"X"}`)
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeConflict) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeConflict)
		}
	})
}

func (f *fakeTagAdmin) ReturnAssignment(_ context.Context, id int64) (*time.Time, error) {
	if f.returnErr != nil {
		return nil, f.returnErr
	}
	now := time.Now().UTC()
	f.returnedID = id
	return &now, nil
}

func TestAssignmentReturnHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("POST /asset-assignments/21/return 闭合成功+审计", func(t *testing.T) {
		au := &recAudit{}
		fa := &fakeTagAdmin{}
		eng := tagAdminRouter(fa, au)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/asset-assignments/21/return", "")
		var out struct {
			Code int `json:"code"`
			Data struct {
				ID          int64  `json:"id"`
				EffectiveTo string `json:"effectiveTo"`
			} `json:"data"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeOK) || out.Data.ID != 21 || out.Data.EffectiveTo == "" {
			t.Fatalf("code=%d out=%+v", out.Code, out.Data)
		}
		if fa.returnedID != 21 {
			t.Fatalf("returnedID=%d", fa.returnedID)
		}
		if len(au.events) != 1 || au.events[0].Action != "状态变更" || au.events[0].Detail["op"] != "return" {
			t.Fatalf("events=%+v", au.events)
		}
	})

	t.Run("POST /asset-assignments/21/return 重复归还→40900", func(t *testing.T) {
		eng := tagAdminRouter(&fakeTagAdmin{returnErr: fmt.Errorf("asset: assignment 21 closed at 2026-09-05T00:00:00Z: %w", asset.ErrAssignmentClosed)}, nil)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/asset-assignments/21/return", "")
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeConflict) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeConflict)
		}
	})
}

func (f *fakeTagAdmin) CancelReplacement(_ context.Context, id int64) (*asset.Replacement, error) {
	if f.cancelErr != nil {
		return nil, f.cancelErr
	}
	f.cancelledID = id
	return &asset.Replacement{ID: id, ReplacementNo: "RPL-1", Status: "CANCELLED"}, nil
}

func TestReplacementCancelHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("POST /replacements/15/cancel 成功+审计", func(t *testing.T) {
		au := &recAudit{}
		fa := &fakeTagAdmin{}
		eng := tagAdminRouter(fa, au)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/replacements/15/cancel", "")
		var out struct {
			Code int `json:"code"`
			Data struct {
				ID     int64  `json:"id"`
				Status string `json:"status"`
			} `json:"data"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeOK) || out.Data.ID != 15 || out.Data.Status != "CANCELLED" {
			t.Fatalf("code=%d data=%+v", out.Code, out.Data)
		}
		if len(au.events) != 1 || au.events[0].TargetType != "replacement" || au.events[0].Detail["op"] != "cancel" {
			t.Fatalf("events=%+v", au.events)
		}
	})

	t.Run("POST /replacements/15/cancel 非PENDING→40900", func(t *testing.T) {
		eng := tagAdminRouter(&fakeTagAdmin{cancelErr: fmt.Errorf("asset: replacement 15 not PENDING: %w", asset.ErrReplacementNotCancellable)}, nil)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/replacements/15/cancel", "")
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeConflict) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeConflict)
		}
	})

	t.Run("POST /replacements/99/cancel 未命中→40400", func(t *testing.T) {
		eng := tagAdminRouter(&fakeTagAdmin{cancelErr: asset.ErrNotFound}, nil)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/replacements/99/cancel", "")
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeNotFound) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeNotFound)
		}
	})
}
