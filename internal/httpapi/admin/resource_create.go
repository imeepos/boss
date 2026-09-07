package adminapi

// T2 存量开户导入配套建号接口(设计 docs/design/2026-09-07-legacy-kaihu-import.md §4-T2):
// POST /resources 与 POST /ports。错误语义:必填缺失/白名单外 42200(respondBadRequest),
// 自然键冲突(resources.code/ports.port_code/ports.quad_code)经 domain ErrDuplicate 转 40900。
// 扩展位:T1 迁移(000202)加列后,在 payload struct 加字段透传 domain 即可,不影响既有调用方。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/resource"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// resourcePayload POST /resources 请求体;status 空=ONLINE(DB 枚举 ONLINE/OFFLINE/FAULT)。
type resourcePayload struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	ParentID      int64  `json:"parentId"`
	LegalEntityID int64  `json:"legalEntityId"`
	AddressID     int64  `json:"addressId"`
	Status        string `json:"status"`
}

// resourceCreateHandler POST /resources:新建 OLT/SPLITTER 资源行(menu:resource)。
func resourceCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var p resourcePayload
		if err := c.ShouldBindJSON(&p); err != nil {
			respondBadRequest(c, "invalid payload")
			return
		}
		if p.Code == "" || p.AddressID <= 0 || p.LegalEntityID <= 0 {
			respondBadRequest(c, "code/addressId/legalEntityId required")
			return
		}
		if p.Type != "OLT" && p.Type != "SPLITTER" {
			respondBadRequest(c, "type must be OLT or SPLITTER")
			return
		}
		status := p.Status
		if status == "" {
			status = "ONLINE"
		}
		if status != "ONLINE" && status != "OFFLINE" && status != "FAULT" {
			respondBadRequest(c, "status must be ONLINE/OFFLINE/FAULT")
			return
		}
		id, err := a.Resource.CreateResource(c.Request.Context(), resource.Resource{
			Code: p.Code, Name: p.Name, Type: p.Type, ParentID: p.ParentID,
			LegalEntityID: p.LegalEntityID, AddressID: p.AddressID, Status: status,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "新建", "resource", p.Code, map[string]any{"type": p.Type, "addressId": p.AddressID})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "code": p.Code})
	}
}

// portPayload POST /ports 请求体;legalEntityId 空=取归属资源的企业快照,status 空=IDLE。
type portPayload struct {
	PortCode        string `json:"portCode"`
	QuadCode        string `json:"quadCode"`
	ResourceID      int64  `json:"resourceId"`
	LegalEntityID   int64  `json:"legalEntityId"`
	LegalEntityName string `json:"legalEntityName"`
	AddressID       int64  `json:"addressId"`
	RegionID        int64  `json:"regionId"`
	RegionName      string `json:"regionName"`
	Status          string `json:"status"`
}

// portCreateHandler POST /ports:在指定 resource 下新建端口行(menu:resource)。
// 新建仅允许 IDLE(默认)/DISABLED;RESERVED/USED 是业务流转态,不经建档直设。
func portCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var p portPayload
		if err := c.ShouldBindJSON(&p); err != nil {
			respondBadRequest(c, "invalid payload")
			return
		}
		if p.PortCode == "" || p.QuadCode == "" || p.ResourceID <= 0 || p.AddressID <= 0 || p.RegionID <= 0 || p.RegionName == "" {
			respondBadRequest(c, "portCode/quadCode/resourceId/addressId/regionId/regionName required")
			return
		}
		res, err := a.Resource.GetResource(c.Request.Context(), p.ResourceID)
		if err != nil {
			respondErr(c, err)
			return
		}
		legalID := p.LegalEntityID
		if legalID <= 0 {
			legalID = res.LegalEntityID
		}
		status := p.Status
		if status == "" {
			status = "IDLE"
		}
		if status != "IDLE" && status != "DISABLED" {
			respondBadRequest(c, "status must be IDLE or DISABLED")
			return
		}
		id, err := a.Resource.CreatePort(c.Request.Context(), resource.Port{
			PortCode: p.PortCode, QuadCode: p.QuadCode, ResourceID: p.ResourceID,
			LegalEntityID: legalID, LegalEntityName: p.LegalEntityName,
			AddressID: p.AddressID, RegionID: p.RegionID, RegionName: p.RegionName, Status: status,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "新建", "port", p.PortCode, map[string]any{"resourceId": p.ResourceID, "quadCode": p.QuadCode})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "portCode": p.PortCode})
	}
}
