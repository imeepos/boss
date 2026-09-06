package adminapi

// P2-T4 事件流查询 handler 单测:limit 夹取/action 白名单 42200/40400 映射/统一信封。

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/asset"
)

type fakeEventAsset struct {
	asset.AssetService
	tagEvents                  []asset.TagEvent
	tagErr                     error
	assetEvents                []asset.TagEvent
	assetErr                   error
	gotID, gotLimit, gotBefore int64
	gotActions                 []string
}

func (f *fakeEventAsset) ListTagEvents(_ context.Context, id, limit, beforeID int64, actions []string) ([]asset.TagEvent, error) {
	f.gotID, f.gotLimit, f.gotBefore, f.gotActions = id, limit, beforeID, actions
	return f.tagEvents, f.tagErr
}

func (f *fakeEventAsset) ListAssetEvents(_ context.Context, id, limit, beforeID int64, actions []string) ([]asset.TagEvent, error) {
	f.gotID, f.gotLimit, f.gotBefore, f.gotActions = id, limit, beforeID, actions
	return f.assetEvents, f.assetErr
}

func newEventRouter(a *app.Application) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/admin/v1")
	g.GET("/tags/:tagId/events", tagEventsHandler(a))
	g.GET("/assets/:assetId/events", assetEventsHandler(a))
	return r
}

func TestTagEventsHandler_QueryParams(t *testing.T) {
	t.Run("limit 超上限夹到 100", func(t *testing.T) {
		f := &fakeEventAsset{tagEvents: []asset.TagEvent{{ID: 9, Action: "BIND"}}}
		r := newEventRouter(&app.Application{Asset: f})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/api/admin/v1/tags/5/events?limit=999", nil))
		if w.Code != 200 || f.gotLimit != 100 || f.gotID != 5 {
			t.Fatalf("status=%d limit=%d id=%d", w.Code, f.gotLimit, f.gotID)
		}
		if !strings.Contains(w.Body.String(), "items") {
			t.Fatalf("信封缺 items: %s", w.Body.String())
		}
	})

	t.Run("缺省 limit=50 且 before_id 透传", func(t *testing.T) {
		f := &fakeEventAsset{}
		r := newEventRouter(&app.Application{Asset: f})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/api/admin/v1/tags/5/events?before_id=77", nil))
		if w.Code != 200 || f.gotLimit != 50 || f.gotBefore != 77 {
			t.Fatalf("status=%d limit=%d before=%d", w.Code, f.gotLimit, f.gotBefore)
		}
	})

	t.Run("action 多值白名单透传", func(t *testing.T) {
		f := &fakeEventAsset{}
		r := newEventRouter(&app.Application{Asset: f})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/api/admin/v1/tags/5/events?action=BIND&action=RECYCLE", nil))
		if w.Code != 200 || len(f.gotActions) != 2 {
			t.Fatalf("status=%d actions=%v", w.Code, f.gotActions)
		}
	})

	t.Run("白名单外 action → 42200", func(t *testing.T) {
		f := &fakeEventAsset{}
		r := newEventRouter(&app.Application{Asset: f})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/api/admin/v1/tags/5/events?action=SCRAP", nil))
		if !strings.Contains(w.Body.String(), "42200") {
			t.Fatalf("期望 42200: %s", w.Body.String())
		}
	})

	t.Run("标签不存在 → 40400", func(t *testing.T) {
		f := &fakeEventAsset{tagErr: asset.ErrNotFound}
		r := newEventRouter(&app.Application{Asset: f})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/api/admin/v1/tags/5/events", nil))
		if !strings.Contains(w.Body.String(), "40400") {
			t.Fatalf("期望 40400: %s", w.Body.String())
		}
	})
}

func TestAssetEventsHandler_QueryParams(t *testing.T) {
	t.Run("limit+action 组合透传", func(t *testing.T) {
		f := &fakeEventAsset{assetEvents: []asset.TagEvent{{ID: 3, Action: "UNBIND"}}}
		r := newEventRouter(&app.Application{Asset: f})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/api/admin/v1/assets/2/events?limit=10&action=UNBIND", nil))
		if w.Code != 200 || f.gotLimit != 10 || f.gotID != 2 || len(f.gotActions) != 1 || f.gotActions[0] != "UNBIND" {
			t.Fatalf("status=%d limit=%d id=%d actions=%v", w.Code, f.gotLimit, f.gotID, f.gotActions)
		}
	})

	t.Run("资产不存在 → 40400", func(t *testing.T) {
		f := &fakeEventAsset{assetErr: asset.ErrNotFound}
		r := newEventRouter(&app.Application{Asset: f})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/api/admin/v1/assets/2/events", nil))
		if !strings.Contains(w.Body.String(), "40400") {
			t.Fatalf("期望 40400: %s", w.Body.String())
		}
	})
}
