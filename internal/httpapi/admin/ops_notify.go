// 运维横切:后台脚本经 API key 上报的提醒入口(隧道/巡检/cron 等无人值守场景)。
// 不开放给前端用户界面;account-type API key 才能调用(worker/customer 主体拒),
// 仅允许受控 refType 白名单,避免任意字符串污染提醒中心。
package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/apikey"
	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// opsNotifyRefTypes 运维上报允许的 refType 白名单;refType 与 RefID 组成提醒中心幂等键,
// 白名单防脚本误传脏数据导致提示重复无法聚合。
var opsNotifyRefTypes = map[string]struct{}{
	"stripe_webhook_guard": {}, // Stripe 隧道/endpoint 自愈告警(stripe_webhook_guard.go 复用)
	"stripe_tunnel":        {}, // 隧道 URL 上报脚本(stripe-tunnel-url.sh)
}

// opsNotifyEmitHandler POST /ops/notify-emit:运维脚本上报提醒(仅 account 主体)。
// 鉴权链已包 menu:dispatch 门禁;此处再卡一次:worker/customer 主体 API key 直接拒。
func opsNotifyEmitHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.Notify == nil {
			respond(c, apitypes.CodeInternal, nil)
			return
		}
		if s := middleware.SubjectFrom(c); s != nil && s.Type != apikey.SubjectAccount {
			respond(c, apitypes.CodeUnauthorized, nil)
			return
		}
		var req opsNotifyReq
		if !httpx.BindAndValidate(c, &req, func() error {
			levelErr := httpx.RequireEnum(req.Level, "level",
				notify.LevelInfo, notify.LevelWarn, notify.LevelUrgent)
			refTypeErr := wrapRefTypeErr(validateOpsNotifyRefType(req.RefType))
			return httpx.CollectErrors(
				httpx.RequireString(req.RefType, "refType", 64),
				httpx.RequireString(req.RefID, "refID", 128),
				httpx.RequireString(req.Title, "title", 128),
				httpx.RequireString(req.Content, "content", 1024),
				levelErr,
				refTypeErr,
			)
		}) {
			return
		}
		if err := a.Notify.Emit(c.Request.Context(), notify.Input{
			Category: notify.CategoryTask,
			Level:    req.Level,
			Title:    req.Title,
			Content:  req.Content,
			Link:     req.Link,
			RefType:  req.RefType,
			RefID:    req.RefID,
		}); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

type opsNotifyReq struct {
	RefType string `json:"refType" binding:"required"`
	RefID   string `json:"refID" binding:"required"`
	Level   string `json:"level"`
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
	Link    string `json:"link"`
}

// validateOpsNotifyRefType refType 白名单校验器。
func validateOpsNotifyRefType(refType string) error {
	if _, ok := opsNotifyRefTypes[refType]; !ok {
		return &opsNotifyRefTypeError{refType: refType}
	}
	return nil
}

// wrapRefTypeErr refType 自定义错误转 ValidationError,统一进 BindAndValidate 错误信封。
func wrapRefTypeErr(err error) *httpx.ValidationError {
	if err == nil {
		return nil
	}
	return &httpx.ValidationError{Field: "refType", Message: err.Error()}
}

// opsNotifyRefTypeError refType 不在白名单时的具体错误(便于日志定位)。
type opsNotifyRefTypeError struct{ refType string }

func (e *opsNotifyRefTypeError) Error() string { return "opsNotify refType 不在白名单: " + e.refType }