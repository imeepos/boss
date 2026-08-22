package workerapi

// W 师傅端门户工单投诉/修复上报动作。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// workerComplaintReq 现场投诉登记。
type workerComplaintReq struct {
	Category string `json:"category" binding:"required"`
	Content  string `json:"content" binding:"required"`
}

// workerComplaintHandler 投诉登记:转客服域(complaints,status=OPEN)。
func workerComplaintHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		ticketNo := c.Param("ticketNo")
		if ticketNo == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "ticketNo is required"})
			return
		}
		var req workerComplaintReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		tk, ok := workerOwnedTicketByNo(c, a, ticketNo)
		if !ok {
			return
		}
		_, err := a.WorkOrder.CreateComplaint(c.Request.Context(), order.Complaint{
			TicketNo: tk.TicketNo, OrderID: tk.OrderID,
			LegalEntityID: tk.LegalEntityID, LegalEntityName: tk.LegalEntityName,
			Type: req.Category, Status: "OPEN",
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// workerOwnedTicketByNo 工单动作共用:取单 + 归属校验;失败已回写响应。
func workerOwnedTicketByNo(c *gin.Context, a *app.Application, ticketNo string) (*order.DispatchTicket, bool) {
	tk, err := a.WorkOrder.GetDispatchTicketByNo(c.Request.Context(), ticketNo)
	if err != nil {
		respondErr(c, err)
		return nil, false
	}
	if !workerOwnedTicket(c, tk) {
		return nil, false
	}
	return tk, true
}

// workerRepairReportReq 修复结果上报。
type workerRepairReportReq struct {
	Result string `json:"result"`
	Remark string `json:"remark"`
}

// workerRepairReportHandler 修复上报:FIXED → 报障 CLOSED,UNFIXED → PROCESSING。
func workerRepairReportHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		ticketNo := c.Param("ticketNo")
		if ticketNo == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "ticketNo is required"})
			return
		}
		var req workerRepairReportReq
		if !httpx.BindAndValidate(c, &req, func() error {
			if req.Result != "FIXED" && req.Result != "UNFIXED" {
				return &httpx.ValidationError{Field: "result", Message: "must be FIXED or UNFIXED"}
			}
			return nil
		}) {
			return
		}
		tk, ok := workerOwnedTicketByNo(c, a, ticketNo)
		if !ok {
			return
		}
		respondRepairResult(c, a, tk.TicketNo, req)
	}
}

// respondRepairResult 修复上报结果:FIXED → 报障 CLOSED(关闭失败回落 PROCESSING),
// UNFIXED → PROCESSING。
func respondRepairResult(c *gin.Context, a *app.Application, ticketNo string, req workerRepairReportReq) {
	if req.Result == "FIXED" {
		if err := a.WorkOrder.CloseComplaint(c.Request.Context(), ticketNo); err != nil {
			respond(c, apitypes.CodeOK, gin.H{"reviewPassed": false, "status": "PROCESSING"})
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"reviewPassed": true, "status": "CLOSED"})
		return
	}
	respond(c, apitypes.CodeOK, gin.H{"reviewPassed": false, "status": "PROCESSING", "remark": req.Remark})
}
