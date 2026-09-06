package adminapi

// 资源台账稽核只读端点(P5-W3):三类计数+明细,?category= 按类过滤(menu:resource)。
// 只报不修;RESERVED 阈值走 biz_params(resource.audit.reservedStaleHours,缺省 48)。

import (
	"log"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/resource"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// ossAuditStaleParamKey 稽核 RESERVED 阈值参数键(biz_params,小时)。
const ossAuditStaleParamKey = "resource.audit.reservedStaleHours"

// ossInventoryAuditHandler GET /inventory-audit?category=ownership|state|coding
func ossInventoryAuditHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		aud, ok := a.Resource.(resource.InventoryAuditor)
		if !ok {
			respond(c, apitypes.CodeInternal, gin.H{"reason": "resource store does not support inventory audit"})
			return
		}
		cat := c.Query("category")
		switch cat {
		case "", resource.AuditCatOwnership, resource.AuditCatState, resource.AuditCatCoding:
		default:
			respond(c, apitypes.CodeInvalidParam, gin.H{"reason": "category 必须为 ownership/state/coding 之一"})
			return
		}
		opts := resource.AuditOptions{Category: cat, ReservedStaleHours: ossAuditStaleParam(c, a)}
		rep, err := aud.AuditInventory(c.Request.Context(), opts)
		if err != nil {
			log.Printf("[oss-audit] ENDPOINT QUERY FAILED category=%s err=%v", cat, err)
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"counts": rep.Counts, "items": rep.Items, "total": rep.Total,
			"staleHours": rep.StaleHours, "generatedAt": rep.GeneratedAt,
		})
	}
}

// ossAuditStaleParam 读阈值参数;缺失/非法回退默认 48(读失败留 FAILED 日志)。
func ossAuditStaleParam(c *gin.Context, a *app.Application) int {
	pg, ok := a.User.(paramGetter)
	if !ok {
		return resource.AuditDefaultStaleHours
	}
	v, err := pg.GetParam(c.Request.Context(), ossAuditStaleParamKey)
	if err != nil {
		log.Printf("[oss-audit] PARAM READ FAILED key=%s err=%v", ossAuditStaleParamKey, err)
		return resource.AuditDefaultStaleHours
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return resource.AuditDefaultStaleHours
	}
	return n
}
