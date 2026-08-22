package workerapi

// W 师傅端扫码绑定域 handler 实现(scan.go 仅留路由表 + 通用辅助)。

import (
	"mime/multipart"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/attachment"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/quadlink"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// workerScanBindReq 扫码绑定请求体。
type workerScanBindReq struct {
	EPC     string `json:"epc" binding:"required"`
	Offline bool   `json:"offline"`
}

// workerScanBindHandler 扫码绑定(环节9):四码核对,MATCH 才推进订单环节9。
func workerScanBindHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		ticketNo := c.Param("ticketNo")
		if ticketNo == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "ticketNo is required"})
			return
		}
		var req workerScanBindReq
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
		result, err := a.QuadLink.VerifyScan(c.Request.Context(), workerScanBindReqOf(c, tk, req))
		if err != nil {
			httpx.RespondScanErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, workerScanBindPayload(c, a, result, tk))
	}
}

// workerScanBindReqOf 拼装 quadlink.ScanReq(师傅身份 + EPC + OfflineCalc)。
func workerScanBindReqOf(c *gin.Context, tk *order.DispatchTicket, req workerScanBindReq) quadlink.ScanReq {
	workerID, workerName := portalWorker(c)
	return quadlink.ScanReq{
		OrderID: tk.OrderID, WorkerID: workerID, WorkerName: workerName,
		ScannedEPC: req.EPC, OfflineCalc: req.Offline,
	}
}

// workerScanBindPayload 扫码结果视图(MATCH 时落 Order.ScanBind 并切 quad 状态)。
func workerScanBindPayload(c *gin.Context, a *app.Application, result string, tk *order.DispatchTicket) gin.H {
	quad := gin.H{"status": "UNLINKED"}
	if result == "MATCH" {
		if err := a.Order.ScanBind(c.Request.Context(), tk.OrderID); err != nil {
			respondErr(c, err)
			return gin.H{}
		}
		quad["status"] = "LINKED"
	}
	return gin.H{
		"matched": result == "MATCH", "result": result,
		"message": "", "quad": quad,
	}
}

// workerPhotoUploadHandler 取证上传:照片进入 MinIO,元数据登记后返回附件信息。
func workerPhotoUploadHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		tk, _, err := ticketOrder(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		if !workerOwnedTicket(c, tk) {
			return
		}
		fh, f, ok := workerPhotoOpen(c)
		if !ok {
			return
		}
		defer f.Close()
		at, err := a.Attachment.Upload(c.Request.Context(), workerPhotoAttachment(fh, tk.WorkerID), f, fh.Size)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"photoId": at.ID, "fileName": at.FileName, "linked": true, "objectKey": at.ObjectKey})
	}
}

// workerPhotoOpen 解析 multipart 表单 file 字段(失败已 respond → false)。
func workerPhotoOpen(c *gin.Context) (*multipart.FileHeader, multipart.File, bool) {
	fh, err := c.FormFile("file")
	if err != nil {
		respond(c, apitypes.CodeInvalidParam, nil)
		return nil, nil, false
	}
	f, err := fh.Open()
	if err != nil {
		respond(c, apitypes.CodeInvalidParam, nil)
		return nil, nil, false
	}
	return fh, f, true
}

// workerPhotoAttachment 取证附件元数据(师傅为上传者)。
func workerPhotoAttachment(fh *multipart.FileHeader, workerID int64) *attachment.Attachment {
	return &attachment.Attachment{
		FileName: fh.Filename, ContentType: fh.Header.Get("Content-Type"),
		UploaderType: attachment.UploaderWorker, UploaderID: workerID,
	}
}