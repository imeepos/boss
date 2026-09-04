package workerapi

// W 师傅端扫码异常上报(2026-09-04 任务A-g):审计留痕 + 站内消息回执。
// 此前走 workerAuditOK 纯桩,提交后师傅端无任何可见反馈。

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// workerScanAbnormalHandler 扫码异常上报:审计 + WARN 站内消息(尽力而为,
// 发送失败不阻断上报主流程,但必须留 ALERT 可 grep——禁止静默吞错)。
func workerScanAbnormalHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		ticketNo := c.Param("ticketNo")
		httpx.RecordAudit(a, c, "状态变更", "worker_ticket", ticketNo,
			map[string]any{"action": "scan-abnormal"})
		msgSent := false
		if a.WorkerLedger != nil {
			_, err := a.WorkerLedger.SendMessage(c.Request.Context(), worker.Message{
				WorkerID: workerID, Level: "WARN",
				Title:   "扫码异常上报已收到",
				Content: fmt.Sprintf("工单 %s 的扫码异常已上报,后台将安排复核处理,请勿重复提交。", ticketNo),
				SentAt:  time.Now(),
			})
			if err != nil {
				log.Printf("[worker-scan-abnormal] MESSAGE FAILED ticket=%s worker=%d err=%v",
					ticketNo, workerID, err)
			} else {
				msgSent = true
			}
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true, "messageSent": msgSent})
	}
}
