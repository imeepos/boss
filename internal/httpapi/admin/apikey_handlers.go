// apikey 域具名 handler(承接 registerAPIKeyRoutes 扁平路由表)。
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

// apikeyListHandler GET /api-keys:列出全部 API key。
func apikeyListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.APIKey.List(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// apikeyCreateHandler POST /api-keys:签发 API key(主体必真实存在,避免签出悬空密钥)。
func apikeyCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createAPIKeyReq
		if !httpx.BindAndValidate(c, &req, func() error { return validateCreateAPIKeyReq(req) }) {
			return
		}
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
	}
}

// apikeyRevokeHandler DELETE /api-keys/{id}:吊销 API key。
func apikeyRevokeHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		if err := a.APIKey.Revoke(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, nil)
	}
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

// validateCreateAPIKeyReq 创建密钥请求校验:主体类型枚举 + 主体 id + 名称。
func validateCreateAPIKeyReq(req createAPIKeyReq) error {
	if !apikey.ValidSubjectType(req.SubjectType) {
		return &httpx.ValidationError{Field: "subjectType", Message: "must be account, worker, or customer"}
	}
	return httpx.CollectErrors(
		httpx.RequirePositiveID(req.SubjectRef, "subjectRef"),
		httpx.RequireString(req.Name, "name", 64),
	)
}
