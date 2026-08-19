package userapi

// 报修工单动作:催单(CHG 单号 + 站内消息)与联系师傅(派单工单回填明文电话,归属校验)。

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalUrgeFault POST /faults/:ticketNo/urge:催单(CHG 单号留痕 + 站内消息回执)。
func portalUrgeFault(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		f, ok := portalOwnedFault(a, c, cid)
		if !ok {
			return
		}
		if f.Status == "CLOSED" {
			respond(c, apitypes.CodeConflict, nil)
			return
		}
		urgeNo, err := a.Portal.NextNo(c.Request.Context(), "CHG")
		if err != nil {
			respondErr(c, err)
			return
		}
		now := time.Now()
		msgID, err := a.Portal.NextNo(c.Request.Context(), "MSG")
		if err != nil {
			respondErr(c, err)
			return
		}
		_ = a.Portal.PutMessage(c.Request.Context(), cid, gin.H{
			"messageId": msgID, "category": "fault", "title": "催单已受理",
			"content": "工单 " + f.TicketNo + " 已加急(催单号 " + urgeNo + ")", "tag": "催单",
			"tagLevel": "fault", "createdAt": now, "read": false,
		})
		respond(c, apitypes.CodeOK, gin.H{"ok": true, "urgeNo": urgeNo, "urgedAt": now.Format(time.RFC3339)})
	}
}

// portalFaultContact GET /faults/:ticketNo/contact:受理师傅明文联系方式。
// 报障 complaints 无师傅字段,受理师傅=客户最近一张已派师傅的派单工单(装机/维修同人)。
func portalFaultContact(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		if _, ok := portalOwnedFault(a, c, cid); !ok {
			return
		}
		name, phone, ok := portalTechnician(a, c, cid, 0)
		if !ok {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"technicianName": name, "technicianPhone": phone})
	}
}

// portalOwnedFault 归属校验:工单号 + 客户双匹配,失败即回 404。
func portalOwnedFault(a *app.Application, c *gin.Context, cid int64) (order.Complaint, bool) {
	list, err := a.WorkOrder.ListComplaints(c.Request.Context())
	if err != nil {
		respondErr(c, err)
		return order.Complaint{}, false
	}
	for _, f := range list {
		if f.TicketNo == c.Param("ticketNo") && f.CustomerID == cid {
			return f, true
		}
	}
	respond(c, apitypes.CodeNotFound, nil)
	return order.Complaint{}, false
}

// portalTechnician 解析师傅明文联系方式:orderID>0 按订单派单,否则取客户最近已派工单;
// 命中 worker_id 后从师傅域取明文电话(worker_name 为派单快照,电话不入快照)。
func portalTechnician(a *app.Application, c *gin.Context, cid, orderID int64) (string, string, bool) {
	if a.Worker == nil {
		return "", "", false
	}
	tickets, err := a.WorkOrder.ListDispatchTickets(c.Request.Context())
	if err != nil {
		return "", "", false
	}
	orderIDs := map[int64]bool{}
	if orderID > 0 {
		orderIDs[orderID] = true
	} else if list, err := a.Order.List(c.Request.Context(), order.OrderQuery{CustomerID: cid}); err == nil {
		for _, item := range list {
			if o, err := a.Order.GetByNo(c.Request.Context(), item.OrderNo); err == nil {
				orderIDs[o.ID] = true
			}
		}
	}
	var hit *order.DispatchTicket
	for i := range tickets {
		t := &tickets[i]
		if t.WorkerID == 0 || !orderIDs[t.OrderID] {
			continue
		}
		if hit == nil || t.TicketID > hit.TicketID {
			hit = t
		}
	}
	if hit == nil {
		return "", "", false
	}
	w, err := a.Worker.GetWorker(c.Request.Context(), hit.WorkerID)
	if err != nil || w == nil || w.Phone == "" {
		return "", "", false
	}
	name := hit.WorkerName
	if name == "" {
		name = w.Name
	}
	return name, w.Phone, true
}
