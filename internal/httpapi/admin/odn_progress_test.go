package adminapi

// 施工进度上报 handler 测试(P0-B):幂等回显/范围错误映射/聚合响应。

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
)

// fakeProgressODN 桩:内嵌 ODNService,仅实现进度三方法。
type fakeProgressODN struct {
	odn.ODNService
	recErr  error
	lastRec *odn.ProgressEntry
	created bool
	entries []odn.ProgressEntry
	items   []odn.ItemProgress
}

func (f *fakeProgressODN) RecordProgress(_ context.Context, p odn.ProgressEntry) (int64, bool, error) {
	if f.recErr != nil {
		return 0, false, f.recErr
	}
	f.lastRec = &p
	return 1, f.created, nil
}

func (f *fakeProgressODN) ListProgress(_ context.Context, _ int64, _ int) ([]odn.ProgressEntry, error) {
	return f.entries, nil
}

func (f *fakeProgressODN) ItemProgressSummary(_ context.Context, _ int64) ([]odn.ItemProgress, error) {
	return f.items, nil
}

func progressRouter(f *fakeProgressODN) *gin.Engine {
	r := gin.New()
	a := &app.Application{User: &fakeUser{permOk: true}, ODN: f}
	mgr := auth.NewManager("test-secret", time.Hour)
	g := r.Group("/api/admin/v1", middleware.Authn(mgr, auth.AudAdmin))
	registerODNRoutes(g, a)
	return r
}

func TestODNProgressHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("首次上报 created=true", func(t *testing.T) {
		f := &fakeProgressODN{created: true}
		w := doJSON(progressRouter(f), http.MethodPost, "/api/admin/v1/odn/constructions/1/progress",
			"{\"facilityCode\":\"P01001\",\"doneQty\":120,\"clientMsgId\":\"msg-0001-abc\"}")
		var body struct {
			Code int `json:"code"`
			Data struct {
				Created bool `json:"created"`
			} `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &body)
		if w.Code != http.StatusOK || body.Code != 0 || !body.Data.Created {
			t.Fatalf("HTTP=%d code=%d body=%s", w.Code, body.Code, w.Body.String())
		}
		if f.lastRec == nil || f.lastRec.FacilityCode != "P01001" || f.lastRec.DoneQty != 120 {
			t.Fatalf("lastRec=%+v", f.lastRec)
		}
	})

	t.Run("重复消息 created=false 不重复计量", func(t *testing.T) {
		f := &fakeProgressODN{created: false}
		w := doJSON(progressRouter(f), http.MethodPost, "/api/admin/v1/odn/constructions/1/progress",
			"{\"facilityCode\":\"P01001\",\"doneQty\":120,\"clientMsgId\":\"msg-0001-abc\"}")
		var body struct {
			Data struct {
				Created bool `json:"created"`
			} `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &body)
		if body.Data.Created {
			t.Fatalf("重复消息应 created=false,body=%s", w.Body.String())
		}
	})

	t.Run("设施不在范围映射 40900", func(t *testing.T) {
		f := &fakeProgressODN{recErr: odn.ErrFacilityNotInScope}
		w := doJSON(progressRouter(f), http.MethodPost, "/api/admin/v1/odn/constructions/1/progress",
			"{\"facilityCode\":\"P01001\",\"doneQty\":1,\"clientMsgId\":\"msg-0002-abc\"}")
		var body struct {
			Code int `json:"code"`
		}
		json.Unmarshal(w.Body.Bytes(), &body)
		if body.Code != 40900 {
			t.Fatalf("code=%d body=%s", body.Code, w.Body.String())
		}
	})

	t.Run("GET 返回列表与聚合", func(t *testing.T) {
		f := &fakeProgressODN{entries: []odn.ProgressEntry{{ID: 7, FacilityCode: "P01001", DoneQty: 50, ClientMsgID: "msg-0003-abc"}},
			items: []odn.ItemProgress{{FacilityCode: "P01001", PlannedQty: 100, DoneQty: 50, Entries: 1}}}
		w := doJSON(progressRouter(f), http.MethodGet, "/api/admin/v1/odn/constructions/1/progress", "")
		var body struct {
			Data struct {
				Entries []odn.ProgressEntry `json:"entries"`
				Items   []odn.ItemProgress  `json:"items"`
			} `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &body)
		if len(body.Data.Entries) != 1 || len(body.Data.Items) != 1 || body.Data.Items[0].DoneQty != 50 {
			t.Fatalf("body=%s", w.Body.String())
		}
	})
}
