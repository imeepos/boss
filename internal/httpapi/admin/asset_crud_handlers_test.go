package adminapi

// 资产台账 admin CRUD handler 测试(P2-W1-T1):建档校验/详情 404/
// 批次门禁 42200/删除守卫 40900 阻断项与报废硬删拒绝通道。

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// fakeAssetCRUD 桩 asset.AssetService(嵌入接口,仅实现 CRUD 四方法)。
type fakeAssetCRUD struct {
	asset.AssetService
	created   *asset.Asset
	getAssets map[int64]*asset.Asset
	updateReq struct {
		id int64
		in asset.AssetUpdate
	}
	updateErr  error
	deletedID  int64
	deleteCode string
	deleteErr  error
}

func (f *fakeAssetCRUD) CreateAsset(_ context.Context, a asset.Asset) (int64, error) {
	f.created = &a
	return 77, nil
}

func (f *fakeAssetCRUD) GetAsset(_ context.Context, id int64) (*asset.Asset, error) {
	if a, ok := f.getAssets[id]; ok {
		return a, nil
	}
	return nil, asset.ErrNotFound
}

func (f *fakeAssetCRUD) UpdateAsset(_ context.Context, id int64, in asset.AssetUpdate, _ int64) error {
	f.updateReq.id, f.updateReq.in = id, in
	return f.updateErr
}

func (f *fakeAssetCRUD) DeleteAsset(_ context.Context, id int64) (string, error) {
	f.deletedID = id
	return f.deleteCode, f.deleteErr
}

// assetCRUDRouter 构造资产域路由测试引擎(登录态 + 全权限,同 ledgerRouter 构形)。
func assetCRUDRouter(fa *fakeAssetCRUD) *gin.Engine {
	r := gin.New()
	a := &app.Application{User: &fakeUser{permOk: true}, Asset: fa}
	mgr := auth.NewManager("test-secret", time.Hour)
	g := r.Group("/api/admin/v1", middleware.Authn(mgr, auth.AudAdmin))
	registerAssetRoutes(g, a)
	return r
}

func TestAssetCRUDHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("POST /assets 缺批次→42200", func(t *testing.T) {
		eng := assetCRUDRouter(&fakeAssetCRUD{})
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/assets", `{"type":"ONU"}`)
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeInvalidParam) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeInvalidParam)
		}
	})

	t.Run("POST /assets 建档成功(状态固定 IN_STOCK)", func(t *testing.T) {
		fa := &fakeAssetCRUD{}
		eng := assetCRUDRouter(fa)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/assets", `{"batchId":2,"type":"ONU","tagId":3}`)
		var out struct {
			Code int `json:"code"`
			Data struct {
				ID int64 `json:"id"`
			} `json:"data"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeOK) || out.Data.ID != 77 {
			t.Fatalf("code=%d id=%d", out.Code, out.Data.ID)
		}
		if fa.created == nil || fa.created.BatchID != 2 || fa.created.Status != "IN_STOCK" {
			t.Fatalf("created=%+v", fa.created)
		}
	})

	t.Run("GET /assets/5 未命中→40400", func(t *testing.T) {
		eng := assetCRUDRouter(&fakeAssetCRUD{})
		w := doJSON(eng, http.MethodGet, "/api/admin/v1/assets/5", "")
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeNotFound) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeNotFound)
		}
	})

	t.Run("GET /assets/5 详情含企业/区域快照", func(t *testing.T) {
		fa := &fakeAssetCRUD{getAssets: map[int64]*asset.Asset{
			5: {AssetID: 5, AssetCode: "A-00000005-00001", Status: "IN_STOCK", LegalEntityID: 1, RegionName: "马尼拉"},
		}}
		eng := assetCRUDRouter(fa)
		w := doJSON(eng, http.MethodGet, "/api/admin/v1/assets/5", "")
		var out struct {
			Code int         `json:"code"`
			Data asset.Asset `json:"data"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeOK) || out.Data.AssetCode != "A-00000005-00001" || out.Data.RegionName != "马尼拉" {
			t.Fatalf("code=%d data=%+v", out.Code, out.Data)
		}
	})

	t.Run("PUT /assets/5 非 IN_STOCK 改批次→42200", func(t *testing.T) {
		fa := &fakeAssetCRUD{updateErr: asset.ErrBatchNotEditable, getAssets: map[int64]*asset.Asset{
			5: {AssetID: 5, Status: "DEPLOYED"},
		}}
		eng := assetCRUDRouter(fa)
		w := doJSON(eng, http.MethodPut, "/api/admin/v1/assets/5", `{"batchId":2}`)
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeInvalidParam) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeInvalidParam)
		}
	})

	t.Run("PUT /assets/5 无变化幂等成功", func(t *testing.T) {
		fa := &fakeAssetCRUD{getAssets: map[int64]*asset.Asset{
			5: {AssetID: 5, Status: "IN_STOCK"},
		}}
		eng := assetCRUDRouter(fa)
		w := doJSON(eng, http.MethodPut, "/api/admin/v1/assets/5", `{"batchId":1}`)
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeOK) {
			t.Fatalf("code=%d", out.Code)
		}
		if fa.updateReq.id != 5 || fa.updateReq.in.BatchID != 1 {
			t.Fatalf("updateReq=%+v", fa.updateReq)
		}
	})

	t.Run("DELETE /assets/5 命中引用→40900 且 message 列阻断项", func(t *testing.T) {
		fa := &fakeAssetCRUD{deleteErr: &asset.ErrAssetReferenced{Blockers: []string{"标签绑定", "换新单"}}}
		eng := assetCRUDRouter(fa)
		w := doJSON(eng, http.MethodDelete, "/api/admin/v1/assets/5", "")
		var out struct {
			Code int `json:"code"`
			Data struct {
				Reason string `json:"reason"`
			} `json:"data"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeConflict) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeConflict)
		}
		if !strings.Contains(out.Data.Reason, "标签绑定") || !strings.Contains(out.Data.Reason, "换新单") {
			t.Fatalf("reason=%q", out.Data.Reason)
		}
		if fa.deletedID != 5 {
			t.Fatalf("deletedID=%d", fa.deletedID)
		}
	})

	t.Run("DELETE /assets/5 SCRAPPED 拒硬删→40900", func(t *testing.T) {
		fa := &fakeAssetCRUD{deleteErr: fmt.Errorf("asset 5 已报废(SCRAPPED)禁止硬删,请走报废端点: %w", asset.ErrAssetScrapped)}
		eng := assetCRUDRouter(fa)
		w := doJSON(eng, http.MethodDelete, "/api/admin/v1/assets/5", "")
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeConflict) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeConflict)
		}
	})

	t.Run("DELETE /assets/5 无引用物理删成功", func(t *testing.T) {
		fa := &fakeAssetCRUD{deleteCode: "A-00000005-00001"}
		eng := assetCRUDRouter(fa)
		w := doJSON(eng, http.MethodDelete, "/api/admin/v1/assets/5", "")
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeOK) || fa.deletedID != 5 {
			t.Fatalf("code=%d deletedID=%d", out.Code, fa.deletedID)
		}
	})
}
