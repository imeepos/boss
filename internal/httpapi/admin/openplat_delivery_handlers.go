// openplat 域 M2 具名 handler:投递 outbox 管理 + 测试事件(自助验收)。
package adminapi

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// openPlatDeliveryListHandler GET /openplat/deliveries?subscriptionId=:投递 outbox
// (subscriptionId 缺省 0=全部;倒序最近 200 条)。
func openPlatDeliveryListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		subID := queryInt64(c, "subscriptionId")
		list, err := a.OpenPlat.ListDeliveries(c.Request.Context(), subID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// openPlatDeliveryRequeueHandler POST /openplat/deliveries/{id}/requeue:
// 死信/失败行重置为待投递(attempts 清零,下一轮立即处理);已投递行拒绝。
func openPlatDeliveryRequeueHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		if err := a.OpenPlat.Requeue(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, nil)
	}
}

// openPlatTestEventHandler POST /openplat/apps/{id}/test-event:
// 向该应用全部启用订阅发一条 openplat.test 事件,集成方据此自助验收签名与连通性。
// event_id 带时间戳,同一应用可反复触发(不与业务幂等键冲突)。
func openPlatTestEventHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		eventID := fmt.Sprintf("test-%d-%s", id, time.Now().UTC().Format("20060102T150405"))
		n, err := a.OpenWebhook.Emit(c.Request.Context(), "openplat.test", eventID, gin.H{
			"type": "openplat.test", "eventId": eventID, "appId": id, "sentAt": time.Now().UTC().Format(time.RFC3339),
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"eventId": eventID, "queued": n})
	}
}
