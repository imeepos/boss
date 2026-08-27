package adminapi

// 实名核验通知闭环(fields.md §7.6/§7.7):
//   - 提交落 PENDING → 消息中心 todo(refType=realname,refID=subjectType/subjectID,(三键)幂等);
//   - 审核终态(单主体端点与审核中心共用)→ 办结 todo + 客户站内消息(portal_messages,
//     category=system,App 端未知分类回退默认图标并展示于"全部"页签);
//   - 自动通道即时判定(PASS/FAIL)不落待办也不发回执,端上提交响应已即时给结论。
// 通知均尽力而为:Notify/Portal 为 nil(单测桩)或写入失败时不阻断主流程,与 partner_handlers 先例一致。

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/notify"
)

const realnameRefType = "realname"

// realnameRefID 幂等键后半段;同一主体的多条 PENDING 共享一个待办。
func realnameRefID(subjectType string, subjectID int64) string {
	return subjectType + "/" + strconv.FormatInt(subjectID, 10)
}

// emitRealnamePendingTodo PENDING 待审落后台待办,标题带申报实名便于扫一眼分派。
func emitRealnamePendingTodo(a *app.Application, c *gin.Context, subjectType string, subjectID int64, realName string) {
	if a.Notify == nil {
		return
	}
	_ = a.Notify.Emit(c.Request.Context(), notify.Input{
		Category: notify.CategoryTodo, Level: notify.LevelWarn,
		Title:   "实名待审核:" + realName + "(" + subjectType + ")",
		RefType: realnameRefType, RefID: realnameRefID(subjectType, subjectID),
		Link: "/base/realname-review",
	})
}

// resolveRealnameTodo 审核终态办结该主体全部未办待办。
func resolveRealnameTodo(a *app.Application, c *gin.Context, subjectType string, subjectID int64) {
	if a.Notify == nil {
		return
	}
	_ = a.Notify.Resolve(c.Request.Context(), realnameRefType, realnameRefID(subjectType, subjectID))
}

// notifyCustomerRealnameResult 人工审核终态写客户站内消息;reason 仅 FAIL 时回传驳回原因。
func notifyCustomerRealnameResult(a *app.Application, c *gin.Context, customerID int64, result, reason string) {
	if a.Portal == nil {
		return
	}
	msgID, err := a.Portal.NextNo(c.Request.Context(), "MSG")
	if err != nil {
		return
	}
	title, tagLevel, content := "实名认证已通过", "info", "您的实名认证已通过审核"
	if result != customer.RealNamePass {
		title, tagLevel = "实名认证未通过", "warn"
		if reason == "" {
			content = "您提交的实名认证未通过审核，请核对姓名、证件号与证件照片后重新提交"
		} else {
			content = "您提交的实名认证未通过审核：" + reason
		}
	}
	_ = a.Portal.PutMessage(c.Request.Context(), customerID, gin.H{
		"messageId": msgID, "category": "system", "title": title,
		"content": content, "tag": "实名", "tagLevel": tagLevel,
		"createdAt": time.Now(), "read": false,
	})
}
