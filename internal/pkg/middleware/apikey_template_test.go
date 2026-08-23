package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ymm-001/boss/internal/domain/apikey"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

func TestAuthzRestrictedPartnerTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	checks := func(context.Context, int64, string) (bool, error) { return true, nil }
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(CtxClaims, &auth.Claims{AccountID: 9})
		c.Set(CtxSubject, &Subject{Type: apikey.SubjectAccount, Ref: 9, TemplateCode: "partner-orders-read"})
		c.Next()
	})
	r.GET("/orders", Authz(checks, "menu:partner-orders"), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.GET("/admin", Authz(checks, "menu:account"), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	for _, tc := range []struct {
		path string
		want int
	}{{"/orders", http.StatusNoContent}, {"/admin", http.StatusForbidden}} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if w.Code != tc.want {
			t.Errorf("%s status=%d want=%d", tc.path, w.Code, tc.want)
		}
	}
}
