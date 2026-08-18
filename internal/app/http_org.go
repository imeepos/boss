package app

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// legalEntityReq 法人新建/编辑请求体(code/name 必填)。
type legalEntityReq struct {
	Code string `json:"code" binding:"required"`
	Name string `json:"name" binding:"required"`
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
func registerOrgRoutes(g *gin.RouterGroup, a *Application) {
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
		a.recordAudit(c, "org.create-legal-entity", "legal_entity", req.Code, nil)
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
		a.recordAudit(c, "org.update-legal-entity", "legal_entity", c.Param("legalEntityId"), nil)
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

	g.GET("/departments", requirePerm(a.User, "menu:department"), func(c *gin.Context) {
		list, err := a.User.ListDepartments(c.Request.Context(), queryInt64(c, "legalEntityId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})

	g.GET("/posts", requirePerm(a.User, "menu:post"), func(c *gin.Context) {
		list, err := a.User.ListPosts(c.Request.Context(), queryInt64(c, "deptId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})

	g.GET("/regions", requirePerm(a.User, "menu:region"), func(c *gin.Context) {
		list, err := a.User.ListRegions(c.Request.Context(), c.Query("parentPath"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})

	g.GET("/addresses", requirePerm(a.User, "menu:address"), func(c *gin.Context) {
		list, err := a.User.ListAddresses(c.Request.Context(), queryInt64(c, "parentId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})

	g.POST("/addresses/import", requirePerm(a.User, "menu:importer"), func(c *gin.Context) {
		var req struct {
			Rows []struct {
				Path string `json:"path" binding:"required"`
				Name string `json:"name" binding:"required"`
			} `json:"rows" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		rows := make([]user.AddressRow, 0, len(req.Rows))
		for _, r := range req.Rows {
			rows = append(rows, user.AddressRow{Path: r.Path, Name: r.Name})
		}
		imported, err := a.User.ImportAddresses(c.Request.Context(), rows)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"imported": imported})
	})
}
