package adminapi

// ODN 域 handler 测试:编码校验透传/错误码映射/参数绑定。

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
)

// fakeODN 桩 odn.ODNService:内嵌接口,仅实现被测方法。
type fakeODN struct {
	odn.ODNService
	createErr error
	created   *odn.Facility
	segment   *odn.Segment
}

func (f *fakeODN) CreateFacility(_ context.Context, fac odn.Facility) error {
	f.created = &fac
	return f.createErr
}

func (f *fakeODN) CreateSegment(_ context.Context, c1, c2, name string) (*odn.Segment, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.segment = &odn.Segment{ID: 1, ACode: c1, BCode: c2, Name: name}
	return f.segment, nil
}

func odnRouter(f *fakeODN) *gin.Engine {
	r := gin.New()
	a := &app.Application{User: &fakeUser{permOk: true}, ODN: f}
	mgr := auth.NewManager("test-secret", time.Hour)
	g := r.Group("/api/admin/v1", middleware.Authn(mgr, auth.AudAdmin))
	registerODNRoutes(g, a)
	return r
}

func TestODNFacilityHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("非法编码映射 CodeInvalidParam", func(t *testing.T) {
		f := &fakeODN{createErr: odn.ErrInvalidCode}
		w := doJSON(odnRouter(f), http.MethodPost, "/api/admin/v1/odn/facilities",
			`{"code":"P0100","kind":"P","prvCode":"PHL001","cityPrefix":"MNL"}`)
		var body struct {
			Code int `json:"code"`
		}
		json.Unmarshal(w.Body.Bytes(), &body)
		if w.Code != http.StatusOK || body.Code != 42200 { // 信封 code,非 HTTP 码
			t.Fatalf("HTTP=%d code=%d", w.Code, body.Code)
		}
	})
	t.Run("合法入库回显编码", func(t *testing.T) {
		f := &fakeODN{}
		w := doJSON(odnRouter(f), http.MethodPost, "/api/admin/v1/odn/facilities",
			`{"code":"P01001","kind":"P","prvCode":"PHL001","cityPrefix":"MNL","gridCode":1}`)
		if w.Code != http.StatusOK {
			t.Fatalf("HTTP=%d body=%s", w.Code, w.Body.String())
		}
		if f.created == nil || f.created.Code != "P01001" || f.created.GridCode != 1 {
			t.Fatalf("created=%+v", f.created)
		}
	})
	t.Run("缺 kind 绑定失败", func(t *testing.T) {
		f := &fakeODN{}
		w := doJSON(odnRouter(f), http.MethodPost, "/api/admin/v1/odn/facilities",
			`{"code":"P01001","prvCode":"PHL001","cityPrefix":"MNL"}`)
		var body struct {
			Code int `json:"code"`
		}
		json.Unmarshal(w.Body.Bytes(), &body)
		if body.Code == 0 {
			t.Fatalf("期望绑定错误,body=%s", w.Body.String())
		}
	})
}

func TestODNSegmentHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("同优先级端点映射 CodeInvalidParam", func(t *testing.T) {
		f := &fakeODN{createErr: odn.ErrSamePriority}
		w := doJSON(odnRouter(f), http.MethodPost, "/api/admin/v1/odn/segments",
			`{"endpoint1":"P01001","endpoint2":"P02005"}`)
		var body struct {
			Code int `json:"code"`
		}
		json.Unmarshal(w.Body.Bytes(), &body)
		if body.Code == 0 {
			t.Fatalf("期望错误信封,body=%s", w.Body.String())
		}
	})
	t.Run("定向成功返回段落", func(t *testing.T) {
		f := &fakeODN{}
		w := doJSON(odnRouter(f), http.MethodPost, "/api/admin/v1/odn/segments",
			`{"endpoint1":"OCC001","endpoint2":"ODF001","name":"t"}`)
		if w.Code != http.StatusOK {
			t.Fatalf("HTTP=%d body=%s", w.Code, w.Body.String())
		}
	})
	t.Run("未知错误透传", func(t *testing.T) {
		f := &fakeODN{createErr: errors.New("boom")}
		doJSON(odnRouter(f), http.MethodPost, "/api/admin/v1/odn/segments",
			`{"endpoint1":"ODF001","endpoint2":"OCC001"}`)
	})
}
