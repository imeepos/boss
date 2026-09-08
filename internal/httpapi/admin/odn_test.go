package adminapi

// ODN 域 handler 测试:编码校验透传/错误码映射/参数绑定。

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
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
	createErr   error
	created     *odn.Facility
	segment     *odn.Segment
	dictErr     error
	nextCodeErr error
	createdDev  *odn.Device
}

func (f *fakeODN) ListRegions(_ context.Context) ([]odn.RegionOption, error) {
	if f.dictErr != nil {
		return nil, f.dictErr
	}
	return []odn.RegionOption{{PrvCode: "PHL001", Name: "国家首都区"}}, nil
}

func (f *fakeODN) ListCities(_ context.Context, prvCode string) ([]odn.CityOption, error) {
	if f.dictErr != nil {
		return nil, f.dictErr
	}
	_ = prvCode
	return []odn.CityOption{{CityPrefix: "MNL", Name: "马尼拉"}}, nil
}

func (f *fakeODN) NextFacilityCode(_ context.Context, kind string, gridCode int16) (string, error) {
	if f.nextCodeErr != nil {
		return "", f.nextCodeErr
	}
	return odn.FormatFacilityCode(kind, gridCode, 1), nil
}

func (f *fakeODN) NextSiteNo(_ context.Context, _, _ string) (int16, error) {
	return 89, nil
}

// DeviceIDByCode 编码反查桩:仅 OCC001 命中 id=7,其余 ErrNotFound。
func (f *fakeODN) DeviceIDByCode(_ context.Context, _, _, code string) (int64, error) {
	if code == "OCC001" {
		return 7, nil
	}
	return 0, odn.ErrNotFound
}

func (f *fakeODN) CreateFacility(_ context.Context, fac odn.Facility) error {
	f.created = &fac
	return f.createErr
}

func (f *fakeODN) CreateSite(_ context.Context, st odn.Site) error {
	return f.createErr
}

func (f *fakeODN) CreateDevice(_ context.Context, d odn.Device) error {
	if f.createErr == nil {
		f.createdDev = &d
	}
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
	t.Run("不传 lifecycleStatus 补 PLANNED(T2 规划新建默认口径)", func(t *testing.T) {
		f := &fakeODN{}
		w := doJSON(odnRouter(f), http.MethodPost, "/api/admin/v1/odn/facilities",
			`{"code":"P01001","kind":"P","prvCode":"PHL001","cityPrefix":"MNL","gridCode":1}`)
		var body struct {
			Data struct {
				LifecycleStatus string `json:"lifecycleStatus"`
			} `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &body)
		if f.created == nil || f.created.LifecycleStatus != odn.LCPlanned {
			t.Fatalf("入库态 created=%+v", f.created)
		}
		if body.Data.LifecycleStatus != odn.LCPlanned {
			t.Fatalf("回显态 body=%s", w.Body.String())
		}
	})
	t.Run("显式 IN_SERVICE 透传(登记既有在网设施)", func(t *testing.T) {
		f := &fakeODN{}
		w := doJSON(odnRouter(f), http.MethodPost, "/api/admin/v1/odn/facilities",
			`{"code":"P01001","kind":"P","prvCode":"PHL001","cityPrefix":"MNL","gridCode":1,"lifecycleStatus":"IN_SERVICE"}`)
		if f.created == nil || f.created.LifecycleStatus != odn.LCInService {
			t.Fatalf("HTTP=%d created=%+v body=%s", w.Code, f.created, w.Body.String())
		}
	})
	t.Run("lifecycleStatus=RETIRED 绑定拒绝(退役非出生态)", func(t *testing.T) {
		f := &fakeODN{}
		w := doJSON(odnRouter(f), http.MethodPost, "/api/admin/v1/odn/facilities",
			`{"code":"P01001","kind":"P","prvCode":"PHL001","cityPrefix":"MNL","gridCode":1,"lifecycleStatus":"RETIRED"}`)
		var body struct {
			Code int `json:"code"`
		}
		json.Unmarshal(w.Body.Bytes(), &body)
		if body.Code == 0 || f.created != nil {
			t.Fatalf("期望拒绝且不入库,code=%d created=%+v", body.Code, f.created)
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

func TestODNSiteDeviceHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("POST /odn/sites 回显 NodeCode", func(t *testing.T) {
		f := &fakeODN{}
		w := doJSON(odnRouter(f), http.MethodPost,
			"/api/admin/v1/odn/sites?prvCode=PHL001&cityPrefix=MNL", `{"siteNo":1,"name":"中心局点"}`)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "MNL001") {
			t.Fatalf("HTTP=%d body=%s", w.Code, w.Body.String())
		}
	})
	t.Run("POST /odn/devices 层级错误映射", func(t *testing.T) {
		f := &fakeODN{createErr: odn.ErrBadHierarchy}
		w := doJSON(odnRouter(f), http.MethodPost, "/api/admin/v1/odn/devices",
			`{"code":"SDB001","kind":"SDB","prvCode":"PHL001","cityPrefix":"MNL","parentId":1}`)
		var body struct {
			Code int `json:"code"`
		}
		json.Unmarshal(w.Body.Bytes(), &body)
		if body.Code != 42200 {
			t.Fatalf("期望 42200,body=%s", w.Body.String())
		}
	})
	t.Run("POST /odn/devices query 传省市区(前端用法)成功", func(t *testing.T) {
		f := &fakeODN{}
		w := doJSON(odnRouter(f), http.MethodPost,
			"/api/admin/v1/odn/devices?prvCode=PHL001&cityPrefix=MNL",
			`{"code":"OLT001","kind":"OLT","siteNo":1}`)
		if w.Code != http.StatusOK {
			t.Fatalf("HTTP=%d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestODNDictHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("GET /odn/regions 返回省级字典", func(t *testing.T) {
		w := doJSON(odnRouter(&fakeODN{}), http.MethodGet, "/api/admin/v1/odn/regions", "")
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "PHL001") {
			t.Fatalf("HTTP=%d body=%s", w.Code, w.Body.String())
		}
	})
	t.Run("GET /odn/cities 缺 prvCode 拒绝", func(t *testing.T) {
		w := doJSON(odnRouter(&fakeODN{}), http.MethodGet, "/api/admin/v1/odn/cities", "")
		var body struct {
			Code int `json:"code"`
		}
		json.Unmarshal(w.Body.Bytes(), &body)
		if body.Code != 42200 {
			t.Fatalf("期望 42200,body=%s", w.Body.String())
		}
	})
	t.Run("GET /odn/cities 按 prvCode 过滤", func(t *testing.T) {
		w := doJSON(odnRouter(&fakeODN{}), http.MethodGet,
			"/api/admin/v1/odn/cities?prvCode=PHL001", "")
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "MNL") {
			t.Fatalf("HTTP=%d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestODNNextCodeHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("GET /odn/facility-next-code P+网格12", func(t *testing.T) {
		w := doJSON(odnRouter(&fakeODN{}), http.MethodGet,
			"/api/admin/v1/odn/facility-next-code?kind=P&gridCode=12", "")
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "P12001") {
			t.Fatalf("HTTP=%d body=%s", w.Code, w.Body.String())
		}
	})
	t.Run("GET /odn/facility-next-code 非法 kind 透传域错误", func(t *testing.T) {
		f := &fakeODN{nextCodeErr: odn.ErrInvalidCode}
		w := doJSON(odnRouter(f), http.MethodGet,
			"/api/admin/v1/odn/facility-next-code?kind=XX&gridCode=1", "")
		var body struct {
			Code int `json:"code"`
		}
		json.Unmarshal(w.Body.Bytes(), &body)
		if body.Code != 42200 {
			t.Fatalf("期望 42200,body=%s", w.Body.String())
		}
	})
	t.Run("GET /odn/site-next-no 缺城市拒绝", func(t *testing.T) {
		w := doJSON(odnRouter(&fakeODN{}), http.MethodGet,
			"/api/admin/v1/odn/site-next-no?prvCode=PHL001", "")
		var body struct {
			Code int `json:"code"`
		}
		json.Unmarshal(w.Body.Bytes(), &body)
		if body.Code != 42200 {
			t.Fatalf("期望 42200,body=%s", w.Body.String())
		}
	})
	t.Run("GET /odn/site-next-no 回显下一序号", func(t *testing.T) {
		w := doJSON(odnRouter(&fakeODN{}), http.MethodGet,
			"/api/admin/v1/odn/site-next-no?prvCode=PHL001&cityPrefix=MNL", "")
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "89") {
			t.Fatalf("HTTP=%d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestODNDeviceParentCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("parentCode 解析为上级 id", func(t *testing.T) {
		f := &fakeODN{}
		w := doJSON(odnRouter(f), http.MethodPost, "/api/admin/v1/odn/devices",
			`{"code":"ODB001","kind":"ODB","prvCode":"PHL001","cityPrefix":"MNL","parentCode":"OCC001"}`)
		if w.Code != http.StatusOK {
			t.Fatalf("HTTP=%d body=%s", w.Code, w.Body.String())
		}
		if f.createdDev == nil || f.createdDev.ParentID != 7 {
			t.Fatalf("createdDev=%+v", f.createdDev)
		}
	})
	t.Run("parentCode 未命中映射 40400", func(t *testing.T) {
		w := doJSON(odnRouter(&fakeODN{}), http.MethodPost, "/api/admin/v1/odn/devices",
			`{"code":"SDB001","kind":"SDB","prvCode":"PHL001","cityPrefix":"MNL","parentCode":"NOPE001"}`)
		var body struct {
			Code int `json:"code"`
		}
		json.Unmarshal(w.Body.Bytes(), &body)
		if body.Code != 40400 {
			t.Fatalf("期望 40400,body=%s", w.Body.String())
		}
	})
	t.Run("parentId 显式优先于 parentCode", func(t *testing.T) {
		f := &fakeODN{}
		w := doJSON(odnRouter(f), http.MethodPost, "/api/admin/v1/odn/devices",
			`{"code":"ODB001","kind":"ODB","prvCode":"PHL001","cityPrefix":"MNL","parentId":3,"parentCode":"OCC001"}`)
		if w.Code != http.StatusOK || f.createdDev == nil || f.createdDev.ParentID != 3 {
			t.Fatalf("HTTP=%d body=%s createdDev=%+v", w.Code, w.Body.String(), f.createdDev)
		}
	})
}
