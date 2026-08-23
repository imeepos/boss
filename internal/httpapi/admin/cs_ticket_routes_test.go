package adminapi

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ymm-001/boss/internal/app"
)

func TestCSClosureRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	registerCSClosureRoutes(r.Group("/api/admin/v1"), &app.Application{User: &fakeUser{permOk: true}})
	if len(r.Routes()) != 10 {
		t.Fatalf("routes=%d, want 10", len(r.Routes()))
	}
}
