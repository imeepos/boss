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
		tk, err := a.WorkOrder.GetDispatchTicketByNo(c.Request.Context(), ticketNo)
		if err != nil {
			respondErr(c, err)
			return
		}
		if !workerOwnedTicket(c, tk) {
			return
		}
		_, err = a.WorkOrder.CreateComplaint(c.Request.Context(), order.Complaint{
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
		tk, err := a.WorkOrder.GetDispatchTicketByNo(c.Request.Context(), ticketNo)
		if err != nil {
			respondErr(c, err)
			return
		}
		if !workerOwnedTicket(c, tk) {
			return
		}
		if req.Result == "FIXED" {
			if err := a.WorkOrder.CloseComplaint(c.Request.Context(), c.Param("ticketNo")); err != nil {
				respond(c, apitypes.CodeOK, gin.H{"reviewPassed": false, "status": "PROCESSING"})
				return
			}
			respond(c, apitypes.CodeOK, gin.H{"reviewPassed": true, "status": "CLOSED"})
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"reviewPassed": false, "status": "PROCESSING", "remark": req.Remark})
	}
}
