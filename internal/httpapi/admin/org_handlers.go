package adminapi

// W 组织域路由 handler 实现(承接 registerOrgRoutes 的扁平路由表)。

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// orgListLegalEntitiesHandler GET /legal-entities:法人列表(W1 模板:Authn→Authz→域查询→统一 envelope)。
func orgListLegalEntitiesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.User.ListLegalEntities(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// orgCreateLegalEntityHandler POST /legal-entities:法人新建(org.yaml createLegalEntity)。
func orgCreateLegalEntityHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req legalEntityReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		id, err := a.User.CreateLegalEntity(c.Request.Context(), user.LegalEntity{
			Code: req.Code, Name: req.Name,
			TaxJurisdiction: req.TaxJurisdiction, TaxChannel: req.TaxChannel,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "org.create-legal-entity", "legal_entity", req.Code, nil)
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// orgUpdateLegalEntityHandler PUT /legal-entities/{legalEntityId}:法人编辑(org.yaml updateLegalEntity)。
func orgUpdateLegalEntityHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "legalEntityId")
		if !ok {
			return
		}
		var req legalEntityReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if err := a.User.UpdateLegalEntity(c.Request.Context(), id, user.LegalEntity{
			Code: req.Code, Name: req.Name,
			TaxJurisdiction: req.TaxJurisdiction, TaxChannel: req.TaxChannel,
		}); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "org.update-legal-entity", "legal_entity", c.Param("legalEntityId"), nil)
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// orgListAccountsHandler GET /accounts:账号列表(全量,账号量级小不分页);基础配置 · 账号与角色页。
func orgListAccountsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := a.User.ListAccounts(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, rows)
	}
}

// orgCreateAccountHandler POST /accounts:受权建号(封闭模型):上级分配角色/组织归属/数据范围。
func orgCreateAccountHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req user.AccountInput
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(req.Username, "username", 64),
				httpx.RequireString(req.Password, "password", 128),
				httpx.RequireString(req.RealName, "realName", 64),
				httpx.RequireString(req.RoleCode, "roleCode", 32),
			)
		}) {
			return
		}
		id, err := a.User.CreateAccount(c.Request.Context(), req)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "权限变更", "account", fmt.Sprint(id), map[string]any{
			"username": req.Username, "roleCode": req.RoleCode, "op": "create",
		})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// orgUpdateAccountHandler PUT /accounts/{accountId}:受权改号:角色/组织/数据范围/启停/改密(密码留空不改)。
func orgUpdateAccountHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "accountId")
		if !ok {
			return
		}
		var req user.AccountInput
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(req.Username, "username", 64),
				httpx.RequireString(req.RealName, "realName", 64),
				httpx.RequireString(req.RoleCode, "roleCode", 32),
			)
		}) {
			return
		}
		if err := a.User.UpdateAccount(c.Request.Context(), id, req); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "权限变更", "account", c.Param("accountId"), map[string]any{
			"username": req.Username, "roleCode": req.RoleCode, "op": "update",
		})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// orgListRolesHandler GET /roles:角色清单(建号表单数据源;fields.md 1.2,7 角色码)。
func orgListRolesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, err := a.User.ListRoles(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, roles)
	}
}

// orgListMenuPermsHandler GET /menu-perms:菜单权限矩阵:三层权限模型的菜单层(角色×menu:* 权限)。
func orgListMenuPermsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		m, err := a.User.ListMenuPermMatrix(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"model": gin.H{"layers": []string{
				"菜单权限 role→menu(本矩阵)",
				"功能权限 role→perm code(RBAC 判定)",
				"数据权限 account→org(数据范围)",
			}},
			"matrix": m,
		})
	}
}

// orgListDataScopesHandler GET /data-scopes:数据权限清单(账号级数据范围;org.yaml /data-scopes,kw 过滤账号/姓名/角色)。
func orgListDataScopesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := a.User.ListDataScopes(c.Request.Context(), c.Query("keyword"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, rows)
	}
}
