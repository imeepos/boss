package adminapi

// W 组织域部门/岗位 handlers(自 org_handlers.go 拆出,保持 300 行红线)。
// 请求结构 departmentReq/postReq 定义在 org.go。

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// orgListDepartmentsHandler GET /departments:部门列表(按 legalEntityId 过滤)。
func orgListDepartmentsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.User.ListDepartments(c.Request.Context(), queryInt64(c, "legalEntityId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// orgCreateDepartmentHandler POST /departments:部门受权维护(建号前基础数据);menu:department 门禁。
func orgCreateDepartmentHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req departmentReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(req.LegalEntityID, "legalEntityId"),
				httpx.RequireString(req.Name, "name", 64),
			)
		}) {
			return
		}
		id, err := a.User.CreateDepartment(c.Request.Context(), req.LegalEntityID, req.Name)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "department", fmt.Sprint(id), map[string]any{"name": req.Name, "op": "create"})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// orgUpdateDepartmentHandler PUT /departments/{deptId}:部门编辑。
func orgUpdateDepartmentHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "deptId")
		if !ok {
			return
		}
		var req departmentReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(req.LegalEntityID, "legalEntityId"),
				httpx.RequireString(req.Name, "name", 64),
			)
		}) {
			return
		}
		if err := a.User.UpdateDepartment(c.Request.Context(), id, req.LegalEntityID, req.Name); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "department", c.Param("deptId"), map[string]any{"name": req.Name, "op": "update"})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// orgDeleteDepartmentHandler DELETE /departments/{deptId}:部门删除;仍有岗位/账号挂靠时 40900 拒。
func orgDeleteDepartmentHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "deptId")
		if !ok {
			return
		}
		if err := a.User.DeleteDepartment(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "department", c.Param("deptId"), map[string]any{"op": "delete"})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// orgDeletePostHandler DELETE /posts/{postId}:岗位删除(连同 post_roles);仍有账号挂岗时 40900 拒。
func orgDeletePostHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "postId")
		if !ok {
			return
		}
		if err := a.User.DeletePost(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "post", c.Param("postId"), map[string]any{"op": "delete"})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// orgListPostsHandler GET /posts:岗位列表(按 deptId 过滤)。
func orgListPostsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.User.ListPosts(c.Request.Context(), queryInt64(c, "deptId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// orgCreatePostHandler POST /posts:岗位受权维护(建号前基础数据);menu:post 门禁;roles=功能角色码全量替换。
func orgCreatePostHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req postReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(req.DeptID, "deptId"),
				httpx.RequireString(req.Code, "code", 32),
				httpx.RequireString(req.Name, "name", 64),
			)
		}) {
			return
		}
		id, err := a.User.CreatePost(c.Request.Context(), req.DeptID, req.Code, req.Name, req.Roles)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "post", fmt.Sprint(id), map[string]any{"code": req.Code, "op": "create"})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// orgUpdatePostHandler PUT /posts/{postId}:岗位编辑。
func orgUpdatePostHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "postId")
		if !ok {
			return
		}
		var req postReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(req.DeptID, "deptId"),
				httpx.RequireString(req.Code, "code", 32),
				httpx.RequireString(req.Name, "name", 64),
			)
		}) {
			return
		}
		if err := a.User.UpdatePost(c.Request.Context(), id, req.DeptID, req.Code, req.Name, req.Roles); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "post", c.Param("postId"), map[string]any{"code": req.Code, "op": "update"})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
