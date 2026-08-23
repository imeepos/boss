package adminapi

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/provision"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func provisionUpdateTemplateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "templateId")
		if !ok {
			return
		}
		var t provision.Template
		if !httpx.BindAndValidate(c, &t, func() error {
			return httpx.CollectErrors(httpx.RequirePositiveID(t.LegalEntityID, "legalEntityId"), httpx.RequireString(t.Code, "code", 64), httpx.RequireString(t.Name, "name", 128))
		}) {
			return
		}
		t.ID = id
		if err := a.Provision.UpdateTemplate(c.Request.Context(), t); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "provision_template", fmt.Sprint(id), map[string]any{"op": "update"})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func provisionSetTemplateStatusHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "templateId")
		if !ok {
			return
		}
		var body struct {
			Status string `json:"status" binding:"required"`
		}
		if !httpx.BindBody(c, &body) {
			return
		}
		if err := a.Provision.SetTemplateStatus(c.Request.Context(), id, body.Status); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func provisionDeleteTemplateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "templateId")
		if !ok {
			return
		}
		if err := a.Provision.DeleteTemplate(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据删除", "provision_template", fmt.Sprint(id), nil)
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
