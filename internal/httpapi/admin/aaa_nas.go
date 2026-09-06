package adminapi

// AAA-A5:NAS 客户端注册表管理(per-NAS 密钥/厂商/CoA 端口/启停),权限码沿用 menu:loaccount。
// 密钥仅创建/更新时收明文,任何响应只含安全字段,绝不回显明文/密文。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// nasPayload 创建/更新请求体(Secret 仅写;更新空=不改密钥)。
type nasPayload struct {
	Name    string `json:"name"`
	NasIP   string `json:"nasIp"`
	Secret  string `json:"secret"`
	Vendor  string `json:"vendor"`
	CoAPort int    `json:"coaPort"`
	Enabled *bool  `json:"enabled"`
}

func (p nasPayload) upsert() aaa.NasUpsert {
	return aaa.NasUpsert{
		Name: p.Name, NasIP: p.NasIP, SecretPlain: p.Secret,
		Vendor: aaa.NasVendor(p.Vendor), CoAPort: p.CoAPort, Enabled: p.Enabled,
	}
}

// aaaNasService 断言 NAS 管理服务(PGStore 实现);不可用=装配缺失。
func aaaNasService(a *app.Application, c *gin.Context) (aaa.NasAdminService, bool) {
	svc, ok := a.Aaa.(aaa.NasAdminService)
	if !ok {
		respond(c, apitypes.CodeInternal, nil)
	}
	return svc, ok
}

// aaaNasPageHandler GET /aaa/nas:注册表分页(keyword 模糊,vendor/enabled 过滤)。
func aaaNasPageHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		svc, ok := aaaNasService(a, c)
		if !ok {
			return
		}
		result, err := svc.ListNasPage(c.Request.Context(), aaa.NasPage{
			Page: queryInt(c, "page", 1), PageSize: queryInt(c, "pageSize", 20),
			Keyword: c.Query("keyword"), Vendor: c.Query("vendor"), Enabled: c.Query("enabled"),
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, result)
	}
}

// aaaNasCreateHandler POST /aaa/nas:新建(密钥必填,加密落库;IP 重复拒绝)。
func aaaNasCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		svc, ok := aaaNasService(a, c)
		if !ok {
			return
		}
		var p nasPayload
		if err := c.ShouldBindJSON(&p); err != nil {
			respondBadRequest(c, "invalid payload")
			return
		}
		if p.Secret == "" {
			respondBadRequest(c, "secret required")
			return
		}
		id, err := svc.CreateNas(c.Request.Context(), p.upsert())
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "新建", "aaa_nas", p.NasIP, map[string]any{"vendor": p.Vendor})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "nasIp": p.NasIP})
	}
}

// aaaNasGetHandler GET /aaa/nas/:id:安全字段查询。
func aaaNasGetHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		svc, ok := aaaNasService(a, c)
		if !ok {
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			respondBadRequest(c, "invalid id")
			return
		}
		nas, err := svc.GetNas(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, nas)
	}
}

// aaaNasUpdateHandler PUT /aaa/nas/:id:更新(Secret 空=不改密钥)。
func aaaNasUpdateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		svc, ok := aaaNasService(a, c)
		if !ok {
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			respondBadRequest(c, "invalid id")
			return
		}
		var p nasPayload
		if err := c.ShouldBindJSON(&p); err != nil {
			respondBadRequest(c, "invalid payload")
			return
		}
		if err := svc.UpdateNas(c.Request.Context(), id, p.upsert()); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "修改", "aaa_nas", c.Param("id"), map[string]any{"op": "update"})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "action": "update"})
	}
}

// aaaNasDeleteHandler DELETE /aaa/nas/:id:删除。
func aaaNasDeleteHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		svc, ok := aaaNasService(a, c)
		if !ok {
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			respondBadRequest(c, "invalid id")
			return
		}
		if err := svc.DeleteNas(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "删除", "aaa_nas", c.Param("id"), map[string]any{"op": "delete"})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "action": "delete"})
	}
}
