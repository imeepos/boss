package adminapi

// 工程结算路由(P-INFRA-1 W1,迁移 000204;契约 admin/odn.yaml)。
// 状态机 PENDING→SETTLED / PENDING|SETTLED→VOIDED(原因必填);写操作全部入审计。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerODNSettlementRoutes 注册结算路由(menu:odn 门禁;由 registerODNConstructionRoutes 调)。
func registerODNSettlementRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.POST("/odn/constructions/:id/settlements", perm, odnCreateSettlementHandler(a))
	g.GET("/odn/constructions/:id/settlements", perm, odnListSettlementsHandler(a))
	g.GET("/odn/settlements/:id", perm, odnGetSettlementHandler(a))
	g.POST("/odn/settlements/:id/settle", perm, odnSettleSettlementHandler(a))
	g.POST("/odn/settlements/:id/void", perm, odnVoidSettlementHandler(a))
}

// odnCreateSettlementHandler POST /odn/constructions/{id}/settlements:发起结算。
// 前置:项目 ACCEPTED 且已指定承包商;应付=清单金额汇总(后端计算)。
func odnCreateSettlementHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		operator := httpx.ClaimsAccountID(c)
		st, err := a.ODN.CreateSettlement(c.Request.Context(), id, operator)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.settlement.create", "construction_settlements", strconv.FormatInt(st.ID, 10),
			map[string]any{"projectId": id, "settlementNo": st.SettlementNo, "totalAmount": st.TotalAmount, "itemCount": st.ItemCount})
		respond(c, apitypes.CodeOK, st)
	}
}

// odnListSettlementsHandler GET /odn/constructions/{id}/settlements:项目结算单列表(含作废历史)。
func odnListSettlementsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		list, err := a.ODN.ListSettlements(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// odnGetSettlementHandler GET /odn/settlements/{id}:结算单详情。
func odnGetSettlementHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		st, err := a.ODN.GetSettlement(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		if st == nil {
			respond(c, apitypes.CodeNotFound, gin.H{"reason": "settlement not found"})
			return
		}
		respond(c, apitypes.CodeOK, st)
	}
}

// odnSettleSettlementHandler POST /odn/settlements/{id}/settle:确认结算 PENDING→SETTLED,
// 同事务自动生成应付记录(W6/F8);审计含应付单号。
func odnSettleSettlementHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		operator := httpx.ClaimsAccountID(c)
		ap, err := a.ODN.SettleSettlement(c.Request.Context(), id, operator)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.settlement.settle", "construction_settlements", c.Param("id"),
			map[string]any{"to": "SETTLED", "payableId": ap.ID, "payableNo": ap.PayableNo, "payableAmount": ap.PayableAmount})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "status": "SETTLED",
			"payableId": ap.ID, "payableNo": ap.PayableNo})
	}
}

// odnVoidSettlementHandler POST /odn/settlements/{id}/void:作废(原因必填);VOIDED 终态。
func odnVoidSettlementHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		operator := httpx.ClaimsAccountID(c)
		var req odnVoidReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if err := a.ODN.VoidSettlement(c.Request.Context(), id, operator, req.Reason); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.settlement.void", "construction_settlements", c.Param("id"),
			map[string]any{"to": "VOIDED", "reason": req.Reason})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "status": "VOIDED"})
	}
}

// odnVoidReq 作废请求体(作废必带原因,审计与 void_reason 列同源)。
type odnVoidReq struct {
	Reason string `json:"reason" binding:"required,max=255"`
}
