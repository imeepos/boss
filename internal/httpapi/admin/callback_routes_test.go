package adminapi

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/cs"
)

type callbackRoutesFake struct{}

func (callbackRoutesFake) ListCallbacks(context.Context) ([]cs.Callback, error)       { return nil, nil }
func (callbackRoutesFake) CreateCallback(context.Context, cs.Callback) (int64, error) { return 1, nil }
func (callbackRoutesFake) UpdateCallback(context.Context, int64, cs.Callback) error   { return nil }
func (callbackRoutesFake) DeleteCallback(context.Context, int64) error                { return nil }

func TestCallbackRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	registerCallbackRoutes(r.Group("/api/admin/v1"), &app.Application{User: &fakeUser{permOk: true}, Callbacks: callbackRoutesFake{}})
	if len(r.Routes()) != 4 {
		t.Fatalf("routes=%d, want 4", len(r.Routes()))
	}
}
