package adminapi

// 契约:GET /geo/subdivisions 增强参数(parentCode 下钻/keyword 搜索/limit 截断)
// 与 GET /geo/default-country(登录管理员即可读,无 menu 门槛)。

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/geo"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// fakeGeo 桩 geo.GeoService:列表过滤条件捕获 + 默认国家可配置,其余返回零值。
type fakeGeo struct {
	filter     geo.SubdivisionFilter
	listCalled bool
	list       []geo.Subdivision
	defaultCty geo.DefaultCountry
}

func (f *fakeGeo) ListSubdivisions(_ context.Context, fl geo.SubdivisionFilter) ([]geo.Subdivision, error) {
	f.filter, f.listCalled = fl, true
	return f.list, nil
}
func (f *fakeGeo) GetDefaultCountry(context.Context) (geo.DefaultCountry, error) {
	return f.defaultCty, nil
}
func (f *fakeGeo) ListCountries(context.Context, string) ([]geo.Country, error) {
	return nil, nil
}
func (f *fakeGeo) GetCountry(context.Context, string) (*geo.CountryDetail, error) {
	return nil, geo.ErrNotFound
}
func (f *fakeGeo) CreateCountry(context.Context, geo.Country) error         { return nil }
func (f *fakeGeo) UpdateCountry(context.Context, string, geo.Country) error { return nil }
func (f *fakeGeo) SetCountryActive(context.Context, string, bool) error     { return nil }
func (f *fakeGeo) AddCountryName(context.Context, string, geo.CountryName) error {
	return nil
}
func (f *fakeGeo) RemoveCountryName(context.Context, string, string, string) error {
	return nil
}
func (f *fakeGeo) ReplaceCountryAttrs(context.Context, string, geo.CountryAttrs) error {
	return nil
}
func (f *fakeGeo) GetSubdivision(context.Context, string) (*geo.Subdivision, error) {
	return nil, geo.ErrNotFound
}
func (f *fakeGeo) ListSubdivisionNames(context.Context, string) ([]geo.SubdivisionName, error) {
	return nil, nil
}
func (f *fakeGeo) CreateSubdivision(context.Context, geo.Subdivision) error { return nil }
func (f *fakeGeo) UpdateSubdivision(context.Context, string, geo.Subdivision) error {
	return nil
}
func (f *fakeGeo) SetSubdivisionActive(context.Context, string, bool) error { return nil }
func (f *fakeGeo) AddSubdivisionName(context.Context, string, geo.SubdivisionName) error {
	return nil
}
func (f *fakeGeo) RemoveSubdivisionName(context.Context, string, string, string) error {
	return nil
}
func (f *fakeGeo) Import(context.Context, geo.ImportData) (geo.ImportCounts, error) {
	return geo.ImportCounts{}, nil
}
func (f *fakeGeo) GetAddress(context.Context, int64) (*geo.AddressInfo, error) { return nil, nil }

func newGeoTestRouter(f *fakeUser, g *fakeGeo, mgr *auth.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{User: f, Geo: g}, mgr)
	return r
}

// decodeEnv 解 envelope 的 data 到 out。
func decodeEnv(t *testing.T, body []byte, out any) {
	t.Helper()
	if err := json.Unmarshal(body, out); err != nil {
		t.Fatalf("decode env: %v body=%s", err, body)
	}
}

func TestGeoSubdivisionListParams(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")

	t.Run("兼容口径 country+locale 无 parentCode 无 keyword", func(t *testing.T) {
		g := &fakeGeo{}
		r := newGeoTestRouter(&fakeUser{permOk: true}, g, mgr)
		w := getJSON(t, r, "/api/admin/v1/geo/subdivisions?country=PH&locale=zh-Hans", token)
		if w.Code != http.StatusOK || !g.listCalled {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		if g.filter.CountryCode != "PH" || g.filter.Locale != "zh-Hans" || g.filter.Keyword != "" {
			t.Fatalf("filter=%+v", g.filter)
		}
		if g.filter.ParentCode != nil || g.filter.Limit != 0 {
			t.Fatalf("filter=%+v", g.filter)
		}
	})

	t.Run("parentCode 三态 缺省/空值/非空", func(t *testing.T) {
		g := &fakeGeo{}
		r := newGeoTestRouter(&fakeUser{permOk: true}, g, mgr)
		if w := getJSON(t, r, "/api/admin/v1/geo/subdivisions?country=PH", token); w.Code != http.StatusOK {
			t.Fatalf("status=%d", w.Code)
		}
		if g.filter.ParentCode != nil {
			t.Fatalf("absent want nil got %+v", g.filter.ParentCode)
		}
		if w := getJSON(t, r, "/api/admin/v1/geo/subdivisions?country=PH&parentCode=", token); w.Code != http.StatusOK {
			t.Fatalf("status=%d", w.Code)
		}
		if g.filter.ParentCode == nil || *g.filter.ParentCode != "" {
			t.Fatalf("empty want present-empty got %+v", g.filter.ParentCode)
		}
		if w := getJSON(t, r, "/api/admin/v1/geo/subdivisions?country=PH&parentCode=PH-NCR", token); w.Code != http.StatusOK {
			t.Fatalf("status=%d", w.Code)
		}
		if g.filter.ParentCode == nil || *g.filter.ParentCode != "PH-NCR" {
			t.Fatalf("code want PH-NCR got %+v", g.filter.ParentCode)
		}
	})

	t.Run("keyword 须配合 country 否则参数错", func(t *testing.T) {
		g := &fakeGeo{}
		r := newGeoTestRouter(&fakeUser{permOk: true}, g, mgr)
		w := getJSON(t, r, "/api/admin/v1/geo/subdivisions?keyword=quezon", token)
		var env struct {
			Code int `json:"code"`
		}
		decodeEnv(t, w.Body.Bytes(), &env)
		if w.Code != http.StatusOK || env.Code == 0 {
			t.Fatalf("status=%d env=%+v", w.Code, env)
		}
		if g.listCalled {
			t.Fatalf("must reject before service call")
		}
	})

	t.Run("keyword+country 透传 limit 原样传域层钳制", func(t *testing.T) {
		g := &fakeGeo{}
		r := newGeoTestRouter(&fakeUser{permOk: true}, g, mgr)
		w := getJSON(t, r, "/api/admin/v1/geo/subdivisions?country=PH&keyword=quez&limit=500", token)
		if w.Code != http.StatusOK || !g.listCalled {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		if g.filter.Keyword != "quez" || g.filter.CountryCode != "PH" || g.filter.Limit != 500 {
			t.Fatalf("filter=%+v", g.filter)
		}
	})

	t.Run("limit 非法回退默认口径", func(t *testing.T) {
		g := &fakeGeo{}
		r := newGeoTestRouter(&fakeUser{permOk: true}, g, mgr)
		if w := getJSON(t, r, "/api/admin/v1/geo/subdivisions?country=PH&limit=abc", token); w.Code != http.StatusOK {
			t.Fatalf("status=%d", w.Code)
		}
		if g.filter.Limit != 0 {
			t.Fatalf("limit=abc want 0 got %d", g.filter.Limit)
		}
	})
}

func TestGeoDefaultCountry(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")

	t.Run("已配置 返回读视图", func(t *testing.T) {
		g := &fakeGeo{defaultCty: geo.DefaultCountry{CountryCode: "PH", Configured: true}}
		r := newGeoTestRouter(&fakeUser{permOk: true}, g, mgr)
		w := getJSON(t, r, "/api/admin/v1/geo/default-country", token)
		var env struct {
			Code int `json:"code"`
			Data struct {
				CountryCode string `json:"countryCode"`
				Configured  bool   `json:"configured"`
			} `json:"data"`
		}
		decodeEnv(t, w.Body.Bytes(), &env)
		if w.Code != http.StatusOK || env.Code != 0 || env.Data.CountryCode != "PH" || !env.Data.Configured {
			t.Fatalf("status=%d env=%+v", w.Code, env)
		}
	})

	t.Run("未配置 返回空值对象", func(t *testing.T) {
		g := &fakeGeo{}
		r := newGeoTestRouter(&fakeUser{permOk: true}, g, mgr)
		w := getJSON(t, r, "/api/admin/v1/geo/default-country", token)
		var env struct {
			Code int `json:"code"`
			Data struct {
				CountryCode string `json:"countryCode"`
				Configured  bool   `json:"configured"`
			} `json:"data"`
		}
		decodeEnv(t, w.Body.Bytes(), &env)
		if w.Code != http.StatusOK || env.Code != 0 || env.Data.CountryCode != "" || env.Data.Configured {
			t.Fatalf("status=%d env=%+v", w.Code, env)
		}
	})

	t.Run("无 menu 权限的登录管理员可读", func(t *testing.T) {
		g := &fakeGeo{}
		r := newGeoTestRouter(&fakeUser{permOk: false}, g, mgr)
		w := getJSON(t, r, "/api/admin/v1/geo/default-country", token)
		if w.Code != http.StatusOK {
			t.Fatalf("login-only endpoint must not require menu perm: status=%d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("geo 维护写端点仍受 menu:geo 门禁", func(t *testing.T) {
		g := &fakeGeo{}
		r := newGeoTestRouter(&fakeUser{permOk: false}, g, mgr)
		w := getJSON(t, r, "/api/admin/v1/geo/subdivisions?country=PH", token)
		if w.Code != http.StatusForbidden {
			t.Fatalf("status=%d want 403", w.Code)
		}
	})
}
