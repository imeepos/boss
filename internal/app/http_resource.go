package app

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/resource"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// genNo 生成台账单号(缺省时):前缀-日期-纳秒尾(12 位熵,防跨轮持久库唯一号撞车)。
func genNo(prefix string) string {
	now := time.Now()
	return fmt.Sprintf("%s-%s-%012d", prefix, now.Format("20060102"), now.UnixNano()%1e12)
}

// registerResourceRoutes 注册网络资源域路由(承接 api/openapi/admin/oss.yaml)。
func registerResourceRoutes(g *gin.RouterGroup, a *Application) {
	res := g.Group("", requirePerm(a.User, "menu:resource"))
	res.GET("/resources", func(c *gin.Context) {
		list, err := a.Resource.ListResources(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	res.GET("/ports", func(c *gin.Context) {
		list, err := a.Resource.ListPorts(c.Request.Context(), queryInt64(c, "resourceId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	res.GET("/ports/:portId/change-history", func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("portId"), 10, 64)
		list, err := a.ResourceSub.ListPortHistory(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	res.POST("/reserves/:reserveId/release", func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("reserveId"), 10, 64)
		if err := a.ResourceSub.ReleaseReserve(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "状态变更", "reserve", c.Param("reserveId"), nil)
		respond(c, apitypes.CodeOK, nil)
	})

	res.GET("/reserves", func(c *gin.Context) {
		list, err := a.ResourceSub.ListReserveRecords(c.Request.Context(), queryInt64(c, "portId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	tr := g.Group("", requirePerm(a.User, "menu:transfer"))
	tr.POST("/transfers", func(c *gin.Context) {
		var t resource.Transfer
		if err := c.ShouldBindJSON(&t); err != nil || t.ResourceID == 0 || t.FromRegionID == 0 || t.ToRegionID == 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if t.TransferNo == "" {
			t.TransferNo = genNo("TRF")
		}
		if t.Status == "" {
			t.Status = "PENDING"
		}
		id, err := a.ResourceSub.CreateTransfer(c.Request.Context(), t)
		if err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "数据变更", "transfer", t.TransferNo, map[string]any{"resourceId": t.ResourceID, "toRegionId": t.ToRegionID})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "transferNo": t.TransferNo})
	})
	tr.POST("/transfers/:transferNo/approve", func(c *gin.Context) {
		no := c.Param("transferNo")
		if err := a.ResourceSub.ApproveTransfer(c.Request.Context(), no); err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "状态变更", "transfer", no, map[string]any{"result": "approved"})
		respond(c, apitypes.CodeOK, nil)
	})
	tr.POST("/transfers/:transferNo/reject", func(c *gin.Context) {
		no := c.Param("transferNo")
		if err := a.ResourceSub.RejectTransfer(c.Request.Context(), no); err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "状态变更", "transfer", no, map[string]any{"result": "rejected"})
		respond(c, apitypes.CodeOK, nil)
	})

	tr.GET("/transfers", func(c *gin.Context) {
		list, err := a.ResourceSub.ListTransfers(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
	tr.GET("/expansions", func(c *gin.Context) {
		list, err := a.ResourceSub.ListExpansions(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
	tr.POST("/expansions", func(c *gin.Context) {
		var e resource.Expansion
		if err := c.ShouldBindJSON(&e); err != nil || e.LegalEntityID == 0 || e.RegionID == 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if e.ExpansionNo == "" {
			e.ExpansionNo = genNo("EXP")
		}
		if e.Status == "" {
			e.Status = "PENDING"
		}
		id, err := a.ResourceSub.CreateExpansion(c.Request.Context(), e)
		if err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "数据变更", "expansion", e.ExpansionNo, map[string]any{"regionId": e.RegionID, "expectedPorts": e.ExpectedPorts})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "expansionNo": e.ExpansionNo})
	})

	tr.GET("/expansions/qos-templates", func(c *gin.Context) {
		list, err := a.ResourceAssign.ListQosTemplates(c.Request.Context(), queryInt64(c, "legalEntityId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
}
