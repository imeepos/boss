package adminapi

// 列表分页参数解析测试(P3-T1):默认口径/钳制/白名单 400/过滤透传。

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/domain/worker"

	"github.com/gin-gonic/gin"
)

// fakeAsset 分页桩:记录收到的 ListQuery 供断言(方法落在桩类型上,不增 ledger_test 行数)。
func (f *fakeAsset) ListAssetsPage(_ context.Context, q asset.ListQuery) (*asset.AssetPage, error) {
	f.pagedQ = q
	return &asset.AssetPage{Items: []asset.Asset{{AssetID: 7, AssetCode: "A-20260007", Status: "IN_STOCK"}}, Total: 1}, nil
}

func (f *fakeAsset) ListTagsPage(_ context.Context, q asset.ListQuery) (*asset.TagPage, error) {
	f.tagQ = q
	return &asset.TagPage{Items: []asset.Tag{{TagID: 3, TagNo: "T-3", Status: "UNBOUND"}}, Total: 1}, nil
}

func TestAssetListPaging(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fa := &fakeAsset{}
	eng := ledgerRouter(&fakeResourceSub{}, fa, &fakeWorkerSvc{w: &worker.Worker{ID: 5, Name: "张师傅", Status: 1}})

	t.Run("默认分页不带参数", func(t *testing.T) {
		w := doJSON(eng, http.MethodGet, "/api/admin/v1/assets", "")
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		if fa.pagedQ.Limit != 50 || fa.pagedQ.Offset != 0 || fa.pagedQ.Sort != "" {
			t.Fatalf("q=%+v", fa.pagedQ)
		}
		var body struct {
			Code int `json:"code"`
			Data struct {
				Items []json.RawMessage `json:"items"`
				Total int64             `json:"total"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Code != 0 || body.Data.Total != 1 || len(body.Data.Items) != 1 {
			t.Fatalf("body=%+v", body)
		}
	})

	t.Run("参数透传与limit钳制", func(t *testing.T) {
		w := doJSON(eng, http.MethodGet,
			"/api/admin/v1/assets?offset=40&limit=99999&status=IN_STOCK&type=ONU&modelId=9&q=A-50&sort=asset_code", "")
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d", w.Code)
		}
		q := fa.pagedQ
		if q.Offset != 40 || q.Limit != 200 || q.Status != "IN_STOCK" || q.Type != "ONU" || q.ModelID != 9 || q.Q != "A-50" || q.Sort != "asset_code" {
			t.Fatalf("q=%+v", q)
		}
	})

	t.Run("sort白名单外400", func(t *testing.T) {
		w := doJSON(eng, http.MethodGet, "/api/admin/v1/assets?sort=battery", "")
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var body struct {
			Code int `json:"code"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		if body.Code != 42200 {
			t.Fatalf("code=%d", body.Code)
		}
	})

	t.Run("tags过滤与sort白名单", func(t *testing.T) {
		w := doJSON(eng, http.MethodGet, "/api/admin/v1/tags?limit=10&status=UNBOUND&q=T-3&sort=tag_no", "")
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d", w.Code)
		}
		if fa.tagQ.Limit != 10 || fa.tagQ.Status != "UNBOUND" || fa.tagQ.Q != "T-3" || fa.tagQ.Sort != "tag_no" {
			t.Fatalf("tagQ=%+v", fa.tagQ)
		}
		w = doJSON(eng, http.MethodGet, "/api/admin/v1/tags?sort=asset_code", "")
		if w.Code != http.StatusBadRequest {
			t.Fatalf("tags sort=asset_code status=%d", w.Code)
		}
	})
}
