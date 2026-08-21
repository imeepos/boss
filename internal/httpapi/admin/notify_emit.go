// notify 接入小工具:各来源域终态 → admin 提醒 Emit/Resolve(docs/plan/admin-notify-center.md §4)。
// Notify 未装配(nil)时静默跳过,handler 不感知。
package adminapi

import (
	"context"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/notify"
)

// 来源域标识与跳转路由(与前端 menu.def path 对齐)。
const (
	refImporter  = "importer"
	refWorkerReg = "worker_reg"
	refProvision = "provision"
	refReport    = "report"
	refBilling   = "billing"
	refComplaint = "complaint"

	linkImporter  = "/base/importer"
	linkWorkerReg = "/boss/worker-reg"
	linkProvlog   = "/provision/provlog"
	linkReport    = "/intel/report"
	linkBilling   = "/billing/billing"
	linkComplaint = "/boss/complaint"
)

// emitTask 发后台任务结果通知(failed 决定 WARN/INFO);best-effort。
func emitTask(ctx context.Context, a *app.Application, refType, refID, title, link string, failed bool) {
	if a.Notify == nil {
		return
	}
	level := notify.LevelInfo
	if failed {
		level = notify.LevelWarn
	}
	_ = a.Notify.Emit(ctx, notify.Input{
		Category: notify.CategoryTask, Level: level,
		Title: title, RefType: refType, RefID: refID, Link: link,
	})
}

// resolveTodo 办结来源域待办;best-effort。
func resolveTodo(ctx context.Context, a *app.Application, refType, refID string) {
	if a.Notify == nil {
		return
	}
	_ = a.Notify.Resolve(ctx, refType, refID)
}
