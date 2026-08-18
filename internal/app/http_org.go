package app

import (
	"context"
	"fmt"
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
		a.recordAudit(c, "权限变更", "account", fmt.Sprint(id), map[string]any{
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
		a.recordAudit(c, "权限变更", "account", c.Param("accountId"), map[string]any{
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
		a.recordAudit(c, "数据变更", "department", fmt.Sprint(id), map[string]any{"name": req.Name, "op": "create"})
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
		a.recordAudit(c, "数据变更", "department", c.Param("deptId"), map[string]any{"name": req.Name, "op": "update"})
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
		a.recordAudit(c, "数据变更", "post", fmt.Sprint(id), map[string]any{"code": req.Code, "op": "create"})
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
		a.recordAudit(c, "数据变更", "post", c.Param("postId"), map[string]any{"code": req.Code, "op": "update"})
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

	g.GET("/addresses", requirePerm(a.User, "menu:address"), func(c *gin.Context) {
		var (
			list []user.Address
			err  error
		)
		if c.Query("unlinked") == "1" {
			list, err = a.User.ListUnlinkedRoots(c.Request.Context())
		} else {
			list, err = a.User.ListAddresses(c.Request.Context(), queryInt64(c, "parentId"))
		}
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})

	// 挂接/改挂国家与一级行政区锚点(仅根节点;字段口径 docs/contract/fields.md 1.5.1)。
	g.PUT("/addresses/:id/geo", requirePerm(a.User, "menu:address"), func(c *gin.Context) {
		var body struct {
			CountryCode string `json:"countryCode" binding:"required,len=2"`
			AdminCode   string `json:"adminCode"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if err := validateGeoAnchor(c.Request.Context(), a, body.CountryCode, body.AdminCode); err != nil {
			respondErr(c, err)
			return
		}
		id := queryInt64(c, "id")
		if err := a.User.SetAddressGeo(c.Request.Context(), id, body.CountryCode, body.AdminCode); err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "address.set-geo", "addresses", strconv.FormatInt(id, 10),
			map[string]any{"countryCode": body.CountryCode, "adminCode": body.AdminCode})
		respond(c, apitypes.CodeOK, nil)
	})

	// 全树搜索:命中节点 + 祖先链(前端懒加载树自动展开用)。
	g.GET("/addresses/search", requirePerm(a.User, "menu:address"), func(c *gin.Context) {
		kw := c.Query("q")
		if kw == "" {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		hits, err := a.User.SearchAddresses(c.Request.Context(), kw)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, hits)
	})

	// 新增节点:parentId=0 建根(可带锚点),否则加子级;label 为 path 末段(小写字母数字)。
	g.POST("/addresses", requirePerm(a.User, "menu:address"), func(c *gin.Context) {
		var body struct {
			ParentID    int64  `json:"parentId"`
			Label       string `json:"label" binding:"required"`
			Name        string `json:"name" binding:"required"`
			CountryCode string `json:"countryCode"`
			AdminCode   string `json:"adminCode"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if body.CountryCode != "" || body.AdminCode != "" {
			if err := validateGeoAnchor(c.Request.Context(), a, body.CountryCode, body.AdminCode); err != nil {
				respondErr(c, err)
				return
			}
		}
		id, err := a.User.CreateAddress(c.Request.Context(),
			body.ParentID, body.Label, body.Name, body.CountryCode, body.AdminCode)
		if err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "address.create", "addresses", strconv.FormatInt(id, 10), map[string]any{
			"parentId": body.ParentID, "label": body.Label, "name": body.Name,
		})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})

	// 改名(path 权威不可变)。
	g.PUT("/addresses/:id", requirePerm(a.User, "menu:address"), func(c *gin.Context) {
		var body struct {
			Name string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		id := queryInt64(c, "id")
		if err := a.User.UpdateAddressName(c.Request.Context(), id, body.Name); err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "address.rename", "addresses", strconv.FormatInt(id, 10),
			map[string]any{"name": body.Name})
		respond(c, apitypes.CodeOK, nil)
	})

	// 删除叶节点(有子级/被业务引用返回 409)。
	g.DELETE("/addresses/:id", requirePerm(a.User, "menu:address"), func(c *gin.Context) {
		id := queryInt64(c, "id")
		if err := a.User.DeleteAddress(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "address.delete", "addresses", strconv.FormatInt(id, 10), nil)
		respond(c, apitypes.CodeOK, nil)
	})

	g.POST("/addresses/import", requirePerm(a.User, "menu:importer"), func(c *gin.Context) {
		var req struct {
			Rows []struct {
				Path        string `json:"path" binding:"required"`
				Name        string `json:"name" binding:"required"`
				CountryCode string `json:"countryCode"`
				AdminCode   string `json:"adminCode"`
			} `json:"rows" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		rows := make([]user.AddressRow, 0, len(req.Rows))
		for _, r := range req.Rows {
			rows = append(rows, user.AddressRow{
				Path: r.Path, Name: r.Name,
				CountryCode: r.CountryCode, AdminCode: r.AdminCode,
			})
		}
		imported, err := a.User.ImportAddresses(c.Request.Context(), rows)
		if err != nil {
			respondErr(c, err)
			return
		}
		_ = a.User.RecordImportTask(c.Request.Context(), "addresses", claimsAccountID(c), int(imported), 0, nil)
		respond(c, apitypes.CodeOK, gin.H{"imported": imported})
	})
}

// validateGeoAnchor 锚点前置校验:国家启用、区划(若给)存在/启用/同国家/一级行政区。
// 非法锚点返回 errGeoInvalidParam(422),国家/区划不存在返回 geo.ErrNotFound(404)。
func validateGeoAnchor(ctx context.Context, a *Application, countryCode, adminCode string) error {
	country, err := a.Geo.GetCountry(ctx, countryCode)
	if err != nil {
		return err
	}
	if !country.IsActive {
		return errGeoInvalidParam
	}
	if adminCode == "" {
		return nil
	}
	sub, err := a.Geo.GetSubdivision(ctx, adminCode)
	if err != nil {
		return err
	}
	if !sub.IsActive || sub.CountryCode != countryCode || sub.Level != 1 {
		return errGeoInvalidParam
	}
	return nil
}
