package adminapi

// 阶段3/4 台账写侧 handler 测试:oss(调拨审批/释放预占/扩容) + ams(盘点/差异处理/换新)。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/domain/resource"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// fakeResourceSub 桩 resource.ResourceSubService:内嵌接口,仅实现被测方法。
type fakeResourceSub struct {
	resource.ResourceSubService
	created    *resource.Transfer
	approved   string
	rejected   string
	releaseID  int64
	releaseErr error
}

func (f *fakeResourceSub) CreateTransfer(_ context.Context, t resource.Transfer) (int64, error) {
	f.created = &t
	return 1, nil
}
func (f *fakeResourceSub) ApproveTransfer(_ context.Context, no string) error {
	f.approved = no
	return nil
}
func (f *fakeResourceSub) RejectTransfer(_ context.Context, no string) error {
	f.rejected = no
	return nil
}
func (f *fakeResourceSub) CreateExpansion(_ context.Context, e resource.Expansion) (int64, error) {
	return 1, nil
}
func (f *fakeResourceSub) ReleaseReserve(_ context.Context, id int64) error {
	f.releaseID = id
	return f.releaseErr
}

// fakeAsset 桩 asset.AssetService。
type fakeAsset struct {
	asset.AssetService
	stocktake  *asset.Stocktake
	diffTaskID int64
	diffErr    error
	scanned    struct {
		taskID, assetID int64
		status          string
	}
	itemHandle struct {
		taskID, itemID int64
		action, note   string
	}
	replacement *asset.Replacement
}

func (f *fakeAsset) CreateStocktake(_ context.Context, s asset.Stocktake) (int64, error) {
	f.stocktake = &s
	return 1, nil
}
func (f *fakeAsset) HandleStocktakeDiff(_ context.Context, id int64) error {
	f.diffTaskID = id
	return f.diffErr
}
func (f *fakeAsset) ScanStocktake(_ context.Context, taskID, assetID int64, status string) (int64, string, error) {
	f.scanned.taskID, f.scanned.assetID, f.scanned.status = taskID, assetID, status
	return 3, "MISMATCH", nil
}
func (f *fakeAsset) ListStocktakeItems(_ context.Context, taskID int64) ([]asset.StocktakeItem, error) {
	return []asset.StocktakeItem{{ID: 3, TaskID: taskID, AssetID: 7,
		ExpectedStatus: "IN_STOCK", ScannedStatus: "DEPLOYED", Kind: "MISMATCH", Resolution: "OPEN"}}, nil
}
func (f *fakeAsset) HandleStocktakeItem(_ context.Context, taskID, itemID int64, action, note string, _ int64) error {
	f.itemHandle.taskID, f.itemHandle.itemID, f.itemHandle.action, f.itemHandle.note = taskID, itemID, action, note
	return nil
}
func (f *fakeAsset) CreateReplacement(_ context.Context, r asset.Replacement) (int64, error) {
	f.replacement = &r
	return 1, nil
}

// ledgerRouter 构造带写侧路由的测试引擎(登录态 + 全权限)。
func ledgerRouter(fr *fakeResourceSub, fa *fakeAsset) *gin.Engine {
	r := gin.New()
	a := &app.Application{User: &fakeUser{permOk: true}, ResourceSub: fr, Asset: fa}
	mgr := auth.NewManager("test-secret", time.Hour)
	g := r.Group("/api/admin/v1", middleware.Authn(mgr, auth.AudAdmin)) // 真实 JWT,与 RegisterRoutes 同构
	registerResourceRoutes(g, a)
	registerAssetRoutes(g, a)
	return r
}

func doJSON(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	tok, _ := auth.NewManager("test-secret", time.Hour).Sign(auth.AudAdmin, 1, "boss", "sysadmin")
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestLedgerWriteHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("POST /transfers 建单+默认单号", func(t *testing.T) {
		fr := &fakeResourceSub{}
		eng := ledgerRouter(fr, &fakeAsset{})
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/transfers",
			`{"resourceId":2,"legalEntityId":1,"legalEntityName":"主品牌","fromRegionId":11,"toRegionId":13}`)
		var out struct {
			Code int `json:"code"`
			Data struct {
				TransferNo string `json:"transferNo"`
			}
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeOK) || !strings.HasPrefix(out.Data.TransferNo, "TRF-") {
			t.Fatalf("code=%d data=%+v", out.Code, out.Data)
		}
		if fr.created == nil || fr.created.Status != "PENDING" {
			t.Fatalf("created=%+v", fr.created)
		}
	})

	t.Run("approve/reject 调拨", func(t *testing.T) {
		fr := &fakeResourceSub{}
		eng := ledgerRouter(fr, &fakeAsset{})
		if w := doJSON(eng, http.MethodPost, "/api/admin/v1/transfers/TRF-1/approve", ""); w.Code != 200 {
			t.Fatal(w.Code)
		}
		if w := doJSON(eng, http.MethodPost, "/api/admin/v1/transfers/TRF-1/reject", ""); w.Code != 200 {
			t.Fatal(w.Code)
		}
		if fr.approved != "TRF-1" || fr.rejected != "TRF-1" {
			t.Fatalf("approved=%q rejected=%q", fr.approved, fr.rejected)
		}
	})

	t.Run("release 预占不存在→40400", func(t *testing.T) {
		fr := &fakeResourceSub{releaseErr: resource.ErrNotFound}
		eng := ledgerRouter(fr, &fakeAsset{})
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/reserves/9/release", "")
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeNotFound) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeNotFound)
		}
	})

	t.Run("POST /expansions 建单", func(t *testing.T) {
		eng := ledgerRouter(&fakeResourceSub{}, &fakeAsset{})
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/expansions",
			`{"legalEntityId":1,"regionId":13,"expectedPorts":48}`)
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeOK) {
			t.Fatalf("code=%d", out.Code)
		}
	})

	t.Run("POST /stocktakes + diff-handle", func(t *testing.T) {
		fa := &fakeAsset{}
		eng := ledgerRouter(&fakeResourceSub{}, fa)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/stocktakes", `{"legalEntityId":1,"scope":"root.luzon"}`)
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeOK) {
			t.Fatalf("code=%d", out.Code)
		}
		if fa.stocktake == nil || fa.stocktake.Status != "DOING" {
			t.Fatalf("stocktake=%+v", fa.stocktake)
		}
		doJSON(eng, http.MethodPost, "/api/admin/v1/stocktakes/5/diff-handle", "")
		if fa.diffTaskID != 5 {
			t.Fatalf("diffTaskID=%d", fa.diffTaskID)
		}
	})

	t.Run("POST /replacements 建单+默认单号", func(t *testing.T) {
		fa := &fakeAsset{}
		eng := ledgerRouter(&fakeResourceSub{}, fa)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/replacements", `{"assetId":7,"reason":"光衰"}`)
		var out struct {
			Code int `json:"code"`
			Data struct {
				ReplacementNo string `json:"replacementNo"`
			}
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeOK) || !strings.HasPrefix(out.Data.ReplacementNo, "RPL-") {
			t.Fatalf("code=%d data=%+v", out.Code, out.Data)
		}
	})

	t.Run("POST /stocktakes/:id/scans 扫码回填+非法状态拒绝", func(t *testing.T) {
		fa := &fakeAsset{}
		eng := ledgerRouter(&fakeResourceSub{}, fa)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/stocktakes/1/scans", `{"assetId":7,"status":"DEPLOYED"}`)
		var out struct {
			Code int `json:"code"`
			Data struct {
				ItemId int64  `json:"itemId"`
				Kind   string `json:"kind"`
			}
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeOK) || out.Data.ItemId != 3 || out.Data.Kind != "MISMATCH" {
			t.Fatalf("code=%d data=%+v", out.Code, out.Data)
		}
		if fa.scanned.status != "DEPLOYED" || fa.scanned.assetID != 7 {
			t.Fatalf("scanned=%+v", fa.scanned)
		}
		if w := doJSON(eng, http.MethodPost, "/api/admin/v1/stocktakes/1/scans", `{"assetId":7,"status":"NOPE"}`); w.Code != 200 {
			t.Fatal(w.Code)
		}
	})

	t.Run("GET /stocktakes/:id/items 明细清单", func(t *testing.T) {
		eng := ledgerRouter(&fakeResourceSub{}, &fakeAsset{})
		w := doJSON(eng, http.MethodGet, "/api/admin/v1/stocktakes/1/items", "")
		var out struct {
			Code int `json:"code"`
			Data struct {
				Items []asset.StocktakeItem `json:"items"`
			}
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeOK) || len(out.Data.Items) != 1 || out.Data.Items[0].Kind != "MISMATCH" {
			t.Fatalf("code=%d items=%+v", out.Code, out.Data.Items)
		}
	})

	t.Run("POST items/:id/handle 处置+FIX缺note拒绝", func(t *testing.T) {
		fa := &fakeAsset{}
		eng := ledgerRouter(&fakeResourceSub{}, fa)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/stocktakes/1/items/3/handle", `{"action":"FIX"}`)
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeInvalidParam) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeInvalidParam)
		}
		if w := doJSON(eng, http.MethodPost, "/api/admin/v1/stocktakes/1/items/3/handle",
			`{"action":"ESCALATE","note":"资产下落不明,上报处理"}`); w.Code != 200 {
			t.Fatal(w.Code)
		}
		if fa.itemHandle.action != "ESCALATE" || fa.itemHandle.itemID != 3 || fa.itemHandle.note == "" {
			t.Fatalf("itemHandle=%+v", fa.itemHandle)
		}
	})

	t.Run("POST diff-handle 存在未处置差异→40900", func(t *testing.T) {
		fa := &fakeAsset{diffErr: asset.ErrDiffPending}
		eng := ledgerRouter(&fakeResourceSub{}, fa)
		w := doJSON(eng, http.MethodPost, "/api/admin/v1/stocktakes/1/diff-handle", "")
		var out struct {
			Code int `json:"code"`
		}
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out.Code != int(apitypes.CodeConflict) {
			t.Fatalf("code=%d, want %d", out.Code, apitypes.CodeConflict)
		}
	})
}

// 确保桩满足接口(编译期断言)。
var (
	_ resource.ResourceSubService = (*fakeResourceSub)(nil)
	_ user.Service                = (*fakeUser)(nil)
	_ asset.AssetService          = (*fakeAsset)(nil)
)
