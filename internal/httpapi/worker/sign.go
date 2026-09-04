package workerapi

// W 师傅端签收(环节12 收口):闸门=扫码绑定(9)+激活上报(10)+激活成功(11),
// 全部满足才真实推进订单状态机(updateMap→12/DONE,端口转在用)。
// 假成功守卫:推进后 stage≠12 视为 0 行生效,显性 5xx + [fake-success] ALERT 留痕
// (回归史:sign 曾是纯审计桩,扫码/激活未成功也返回 {ok:true} 而状态机不动)。

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// workerSignHandler 签收:前置闸门 + 真实状态机推进(2026-09-04 任务A-c)。
func workerSignHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		tk, ord, err := ticketOrder(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		if !workerOwnedTicket(c, tk) {
			return
		}
		switch {
		case ord.Stage < 9:
			respond(c, apitypes.CodeStateInvalid, gin.H{"reason": "扫码绑定未完成,不可签收"})
			return
		case ord.Stage < 10:
			respond(c, apitypes.CodeStateInvalid, gin.H{"reason": "激活上报未提交,不可签收"})
			return
		case ord.Stage < 11:
			respond(c, apitypes.CodeStateInvalid, gin.H{"reason": "激活未成功,不可签收"})
			return
		}
		if ord.Stage < 12 {
			if err := a.Order.UpdateMap(c.Request.Context(), tk.OrderID); err != nil {
				respondErr(c, err)
				return
			}
			after, _, err := a.Order.Track(c.Request.Context(), tk.OrderID)
			if err != nil {
				respondErr(c, err)
				return
			}
			if after.Stage != 12 {
				log.Printf("[fake-success] ALERT sign no-effect ticket=%s order=%d stage=%d",
					tk.TicketNo, tk.OrderID, after.Stage)
				respond(c, apitypes.CodeInternal, nil)
				return
			}
		}
		httpx.RecordAudit(a, c, "状态变更", "worker_ticket", tk.TicketNo,
			map[string]any{"action": "sign", "fromStage": ord.Stage})
		respond(c, apitypes.CodeOK, gin.H{"ok": true, "stage": 12})
	}
}
