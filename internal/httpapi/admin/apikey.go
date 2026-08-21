package adminapi

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/apikey"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// createAPIKeyReq 创建 API key 请求体。
// subjectType ∈ {account, worker, customer};subjectRef 为对应主体表主键。
type createAPIKeyReq struct {
	SubjectType string `json:"subjectType" binding:"required"`
	SubjectRef  int64  `json:"subjectRef" binding:"required"`
	Name        string `json:"name" binding:"required"`
}

// registerAPIKeyRoutes 注册 API key 管理路由(需 sysadmin 权限)。
func registerAPIKeyRoutes(g *gin.RouterGroup, a *app.Application) {
	ak := g.Group("", requirePerm(a.User, "menu:apikey"))

	ak.GET("/api-keys", func(c *gin.Context) {
		list, err := a.APIKey.List(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	ak.POST("/api-keys", func(c *gin.Context) {
		var req createAPIKeyReq
		if !httpx.BindAndValidate(c, &req, func() error {
			if !apikey.ValidSubjectType(req.SubjectType) {
				return &httpx.ValidationError{Field: "subjectType", Message: "must be account, worker, or customer"}
			}
			return httpx.CollectErrors(
				httpx.RequirePositiveID(req.SubjectRef, "subjectRef"),
				httpx.RequireString(req.Name, "name", 64),
			)
		}) {
			return
		}
		// 主体必须真实存在,避免签出悬空密钥
		if _, err := resolveAPISubject(c.Request.Context(), a, req.SubjectType, req.SubjectRef); err != nil {
			respondErr(c, err)
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		res, err := a.APIKey.Create(c.Request.Context(), req.SubjectType, req.SubjectRef, claims.AccountID, req.Name)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"id": res.ID, "subjectType": res.SubjectType, "subjectRef": res.SubjectRef,
			"name": res.Name, "plainKey": res.PlainKey,
		})
	})

	ak.DELETE("/api-keys/:id", func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		if err := a.APIKey.Revoke(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, nil)
	})
}

// resolveAPISubject 按主体类型加载主体展示名与角色码(中间件 SubjectResolver 复用)。
// account → 账号 profile;worker → 师傅档案;customer → 客户档案。
func resolveAPISubject(ctx context.Context, a *app.Application, subjType string, ref int64) (string, error) {
	switch subjType {
	case apikey.SubjectAccount:
		p, err := a.User.GetProfile(ctx, ref)
		if err != nil {
			return "", err
		}
		return p.Username, nil
	case apikey.SubjectWorker:
		w, err := a.Worker.GetWorker(ctx, ref)
		if err != nil {
			return "", err
		}
		return w.Name, nil
	case apikey.SubjectCustomer:
		cu, err := a.Customer.Get(ctx, ref)
		if err != nil {
			return "", err
		}
		return cu.Name, nil
	}
	return "", errors.New("app: unknown api key subject type")
}
