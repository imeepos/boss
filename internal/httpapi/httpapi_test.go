package httpapi_test

// 组合根冒烟测试:零值 Application + 内存 auth.Manager 即可完成三端路由装配,
// 证明 RegisterRoutes 注册期不依赖外部资源(DB/Redis 在 handler 调用时才触碰)。
// 各端 handler 行为由 httpapi/admin、httpapi/user、httpapi/worker 包内测试覆盖。

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/httpapi"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// stubUser 内嵌接口零值满足 user.Service:注册期仅取 HasPermission 方法值,
// 不调用;handler 逻辑由 admin/user/worker 各包测试覆盖。
type stubUser struct {
	user.Service
}

func TestRegisterRoutes_AssemblesAllThreePortals(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	httpapi.RegisterRoutes(r, &app.Application{User: stubUser{}}, auth.NewManager("test-secret", time.Hour))

	ts := httptest.NewServer(r)
	defer ts.Close()

	// 三端前缀各取一个公开端点,断言路由已挂载(非 404)。
	cases := []struct{ method, path string }{
		{http.MethodPost, "/api/admin/v1/auth/login"},
		{http.MethodPost, "/api/user/v1/auth/login"},
		{http.MethodPost, "/api/worker/v1/auth/login"},
	}
	for _, tc := range cases {
		resp, err := http.Post(ts.URL+tc.path, "application/json", nil)
		if err != nil {
			t.Fatalf("%s: %v", tc.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			t.Fatalf("%s not registered", tc.path)
		}
	}
}
