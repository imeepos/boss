package adminapi

// 质量记录 handler 测试(P0-C):测试追加/整改开单/状态转移错误映射/验收门禁错误映射。

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

// fakeQualityODN 桩:内嵌 ODNService,实现质量六方法。
type fakeQualityODN struct {
	odn.ODNService
	testErr   error
	defectErr error
	transErr  error
	openedID  int64
	tests     []odn.QualityTest
	defects   []odn.QualityDefect
}

func (f *fakeQualityODN) RecordTest(_ context.Context, t odn.QualityTest) (int64, error) {
	if f.testErr != nil {
		return 0, f.testErr
	}
	return 9, nil
}

func (f *fakeQualityODN) ListTests(_ context.Context, _ int64, _ int) ([]odn.QualityTest, error) {
	return f.tests, nil
}

func (f *fakeQualityODN) OpenDefect(_ context.Context, _ odn.QualityDefect, _ int64) (int64, error) {
	if f.defectErr != nil {
		return 0, f.defectErr
	}
	f.openedID = 5
	return f.openedID, nil
}

func (f *fakeQualityODN) ListDefects(_ context.Context, _ int64, _ string, _ int) ([]odn.QualityDefect, error) {
	return f.defects, nil
}

func (f *fakeQualityODN) TransitionDefect(_ context.Context, _ int64, _ string, _ int64) error {
	return f.transErr
}

func qualityRouter(f *fakeQualityODN) *gin.Engine {
	r := gin.New()
	a := &app.Application{User: &fakeUser{permOk: true}, ODN: f}
	mgr := auth.NewManager("test-secret", time.Hour)
	g := r.Group("/api/admin/v1", middleware.Authn(mgr, auth.AudAdmin))
	registerODNRoutes(g, a)
	return r
}

func TestODNQualityHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("测试记录追加", func(t *testing.T) {
		f := &fakeQualityODN{}
		w := doJSON(qualityRouter(f), http.MethodPost, "/api/admin/v1/odn/constructions/1/tests",
			"{\"resourceType\":\"FACILITY\",\"resourceRef\":\"P01001\",\"testKind\":\"OTDR\",\"result\":\"PASS\",\"attenuationDb\":0.32}")
		var body struct {
			Code int `json:"code"`
			Data struct {
				ID int64 `json:"id"`
			} `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &body)
		if w.Code != http.StatusOK || body.Code != 0 || body.Data.ID != 9 {
			t.Fatalf("HTTP=%d code=%d body=%s", w.Code, body.Code, w.Body.String())
		}
	})

	t.Run("开整改单", func(t *testing.T) {
		f := &fakeQualityODN{}
		w := doJSON(qualityRouter(f), http.MethodPost, "/api/admin/v1/odn/constructions/1/defects",
			"{\"facilityCode\":\"P01001\",\"severity\":\"MAJOR\",\"description\":\"接续损耗超标\"}")
		var body struct {
			Data struct {
				ID int64 `json:"id"`
			} `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &body)
		if body.Data.ID != 5 {
			t.Fatalf("body=%s", w.Body.String())
		}
	})

	t.Run("验收门禁错误映射 40900", func(t *testing.T) {
		f2 := &fakeQualityODN{transErr: odn.ErrOpenDefects}
		w2 := doJSON(qualityRouter(f2), http.MethodPost, "/api/admin/v1/odn/defects/3/rectify", "")
		var body2 struct {
			Code int `json:"code"`
		}
		json.Unmarshal(w2.Body.Bytes(), &body2)
		if body2.Code != 40900 {
			t.Fatalf("code=%d body=%s", body2.Code, w2.Body.String())
		}
	})

	t.Run("整改终态再动映射 40900", func(t *testing.T) {
		f := &fakeQualityODN{transErr: odn.ErrDefectState}
		w := doJSON(qualityRouter(f), http.MethodPost, "/api/admin/v1/odn/defects/3/verify", "")
		var body struct {
			Code int `json:"code"`
		}
		json.Unmarshal(w.Body.Bytes(), &body)
		if body.Code != 40900 {
			t.Fatalf("code=%d body=%s", body.Code, w.Body.String())
		}
	})
}
