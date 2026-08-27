package userapi

// 自助实名提交 → 后台待办提醒(fields.md §7.6):复用 admin 侧 refType=realname 约定,
// (realname, customer/{cid}, todo) 幂等;自动通道即时判定时端上已得结论,不落待办。
// 标题用脱敏姓名,避免提醒中心扩散明文证件信息。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/notify"
)

// emitPortalRealnameTodo 自助提交仍需人工核验(PENDING)时落后台待办。
func emitPortalRealnameTodo(a *app.Application, c *gin.Context, cid int64, realName string) {
	if a.Notify == nil {
		return
	}
	_ = a.Notify.Emit(c.Request.Context(), notify.Input{
		Category: notify.CategoryTodo, Level: notify.LevelWarn,
		Title:   "实名待审核:" + portalMaskName(realName) + "(customer)",
		RefType: "realname", RefID: "customer/" + strconv.FormatInt(cid, 10),
		Link: "/base/realname-review",
	})
}
