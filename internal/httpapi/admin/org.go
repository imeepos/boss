package adminapi

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// legalEntityReq 法人新建/编辑请求体(code/name 必填)。
type legalEntityReq struct {
	Code string `json:"code" binding:"required"`
	Name string `json:"name" binding:"required"`
}

// departmentReq 部门新建/编辑请求体(挂靠子公司,名称必填)。
type departmentReq struct {
	LegalEntityID int64  `json:"legalEntityId" binding:"required"`
	Name          string `json:"name" binding:"required"`
}

// postReq 岗位新建/编辑请求体(部门内 code 唯一;roles 为功能角色码,可空)。
type postReq struct {
	DeptID int64    `json:"deptId" binding:"required"`
	Code   string   `json:"code" binding:"required"`
	Name   string   `json:"name" binding:"required"`
	Roles  []string `json:"roles"`
}

// requirePerm 返回 RBAC 中间件:账号需持有 permCode 才可访问。
// 列表端点按「菜单可见性」权限码门禁(menu:*),与 000003 种子对齐。
func requirePerm(svc user.Service, permCode string) gin.HandlerFunc {
	return middleware.Authz(svc.HasPermission, permCode)
}

// queryInt64 解析整型 query 参数;缺省/非法返回 0(域层按 0=全部 处理)。
func queryInt64(c *gin.Context, name string) int64 {
	v, _ := strconv.ParseInt(c.Query(name), 10, 64)
	return v
}

// registerOrgRoutes 注册组织域只读路由(W1 模板:Authn→Authz→域查询→统一 envelope)。
func registerOrgRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/legal-entities", requirePerm(a.User, "menu:company"), func(c *gin.Context) {
		list, err := a.User.ListLegalEntities(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})

	// 法人写操作(org.yaml createLegalEntity/updateLegalEntity)。
	g.POST("/legal-entities", requirePerm(a.User, "menu:company"), func(c *gin.Context) {
		var req legalEntityReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		id, err := a.User.CreateLegalEntity(c.Request.Context(), user.LegalEntity{Code: req.Code, Name: req.Name})
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "org.create-legal-entity", "legal_entity", req.Code, nil)
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})

	g.PUT("/legal-entities/:legalEntityId", requirePerm(a.User, "menu:company"), func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("legalEntityId"), 10, 64)
		var req legalEntityReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if err := a.User.UpdateLegalEntity(c.Request.Context(), id, user.LegalEntity{Code: req.Code, Name: req.Name}); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "org.update-legal-entity", "legal_entity", c.Param("legalEntityId"), nil)
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	// 账号列表:基础配置 · 账号与角色页(全量,账号量级小不分页)。
	g.GET("/accounts", requirePerm(a.User, "menu:account"), func(c *gin.Context) {
		rows, err := a.User.ListAccounts(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, rows)
	})

	// 受权建号(封闭模型):上级分配角色/组织归属/数据范围。
	g.POST("/accounts", requirePerm(a.User, "menu:account"), func(c *gin.Context) {
		var req user.AccountInput
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
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
	})

	// 受权改号:角色/组织/数据范围/启停/改密(密码留空不改)。
	g.PUT("/accounts/:accountId", requirePerm(a.User, "menu:account"), func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("accountId"), 10, 64)
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		var req user.AccountInput
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
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
	})

	// 角色清单:建号表单数据源(fields.md 1.2,7 角色码)。
	g.GET("/roles", requirePerm(a.User, "menu:account"), func(c *gin.Context) {
		roles, err := a.User.ListRoles(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, roles)
	})

	// 菜单权限矩阵:三层权限模型的菜单层(角色×menu:* 权限)。
	g.GET("/menu-perms", requirePerm(a.User, "menu:menuperm"), func(c *gin.Context) {
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
	})

	// 数据权限清单:账号级数据范围(org.yaml /data-scopes,kw 过滤账号/姓名/角色)。
	g.GET("/data-scopes", requirePerm(a.User, "menu:datascope"), func(c *gin.Context) {
		rows, err := a.User.ListDataScopes(c.Request.Context(), c.Query("keyword"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, rows)
	})

	g.GET("/departments", requirePerm(a.User, "menu:department"), func(c *gin.Context) {
		list, err := a.User.ListDepartments(c.Request.Context(), queryInt64(c, "legalEntityId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})

	// 部门受权维护(建号前基础数据);menu:department 门禁。
	g.POST("/departments", requirePerm(a.User, "menu:department"), func(c *gin.Context) {
		var req departmentReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		id, err := a.User.CreateDepartment(c.Request.Context(), req.LegalEntityID, req.Name)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "department", fmt.Sprint(id), map[string]any{"name": req.Name, "op": "create"})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})

	g.PUT("/departments/:deptId", requirePerm(a.User, "menu:department"), func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("deptId"), 10, 64)
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		var req departmentReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if err := a.User.UpdateDepartment(c.Request.Context(), id, req.LegalEntityID, req.Name); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "department", c.Param("deptId"), map[string]any{"name": req.Name, "op": "update"})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	g.GET("/posts", requirePerm(a.User, "menu:post"), func(c *gin.Context) {
		list, err := a.User.ListPosts(c.Request.Context(), queryInt64(c, "deptId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})

	// 岗位受权维护(建号前基础数据);menu:post 门禁;roles=功能角色码全量替换。
	g.POST("/posts", requirePerm(a.User, "menu:post"), func(c *gin.Context) {
		var req postReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		id, err := a.User.CreatePost(c.Request.Context(), req.DeptID, req.Code, req.Name, req.Roles)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "post", fmt.Sprint(id), map[string]any{"code": req.Code, "op": "create"})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})

	g.PUT("/posts/:postId", requirePerm(a.User, "menu:post"), func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("postId"), 10, 64)
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		var req postReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if err := a.User.UpdatePost(c.Request.Context(), id, req.DeptID, req.Code, req.Name, req.Roles); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "post", c.Param("postId"), map[string]any{"code": req.Code, "op": "update"})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	g.GET("/regions", requirePerm(a.User, "menu:region"), func(c *gin.Context) {
		list, err := a.User.ListRegions(c.Request.Context(), c.Query("parentPath"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})

}
