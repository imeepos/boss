package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/apikey"
	"github.com/ymm-001/boss/internal/pkg/audit"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// ErrGeoInvalidParam 参数绑定失败哨兵,RespondErr 统一映射 CodeInvalidParam。
var ErrGeoInvalidParam = errors.New("httpapi: geo invalid param")

// Respond 统一响应 envelope:{code,msg,data};跨进程错误码对齐 pkg/apitypes(D3)。
func Respond(c *gin.Context, code apitypes.Code, data any) {
	c.JSON(200, gin.H{"code": code, "msg": code.Message(), "data": data})
}

// auditWriteParams 审计写入的失败处理参数:同步落库后单次快速重试,
// 覆盖连接抖动等瞬时故障;仍失败则以告警日志留全量载荷供人工补记。
const (
	auditWriteAttempts = 2
	auditWriteTimeout  = 5 * time.Second
	auditRetryGap      = 200 * time.Millisecond
)

// RecordAudit 记录关键操作审计(同步落库);未装配审计 writer 时输出告警留痕。
// 与请求上下文解耦(WithoutCancel):客户端中途断开不得丢业务留痕。
// 写入瞬时失败重试一次;最终失败记 [audit] WRITE FAILED 日志并附完整事件,
// 运维可据此补记——审计是业务事实的一部分,静默丢弃等于事实缺口
// (纪要 2026-08-28 待定项③:writer 未装配同样不得静默,降级放行但留告警)。
func RecordAudit(a *app.Application, c *gin.Context, action, targetType, targetID string, detail map[string]any) {
	if a == nil {
		return
	}
	if a.Audit == nil {
		// 生产装配恒非空,nil 仅测试/降级场景;告警可 grep,提示人工补记。
		log.Printf("[audit] WRITER NIL need-manual-recovery action=%s target=%s/%s ip=%s detail=%v",
			action, targetType, targetID, c.ClientIP(), detail)
		return
	}
	ev := audit.Event{
		AccountID: ClaimsAccountID(c), Action: action, TargetType: targetType, TargetID: targetID,
		Detail: detail, IP: c.ClientIP(),
	}
	var err error
	for i := 0; i < auditWriteAttempts; i++ {
		if i > 0 {
			time.Sleep(auditRetryGap)
		}
		ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), auditWriteTimeout)
		err = a.Audit.Write(ctx, ev)
		cancel()
		if err == nil {
			return
		}
	}
	logAuditFailure(ev, err)
}

// logAuditFailure 最终失败兜底:一行日志携带重建审计所需的全部字段。
func logAuditFailure(e audit.Event, err error) {
	d, merr := json.Marshal(e.Detail)
	if merr != nil {
		d = []byte(fmt.Sprintf("%v", e.Detail))
	}
	log.Printf("[audit] WRITE FAILED need-manual-recovery account=%d action=%s target=%s/%s ip=%s detail=%s err=%v",
		e.AccountID, e.Action, e.TargetType, e.TargetID, e.IP, d, err)
}

// ClaimsAccountID 取当前请求账号 id(未认证返回 0)。
func ClaimsAccountID(c *gin.Context) int64 {
	if v, ok := c.Get(middleware.CtxClaims); ok {
		if claims, ok := v.(*auth.Claims); ok {
			return claims.AccountID
		}
	}
	return 0
}

// APIKeySubjectResolver 构造中间件用的 SubjectResolver:
// account 主体注入完整 RBAC 身份,worker/customer 注入受限身份。
func APIKeySubjectResolver(a *app.Application) middleware.SubjectResolver {
	return func(ctx context.Context, subjType string, ref int64) (string, string, error) {
		switch subjType {
		case apikey.SubjectAccount:
			p, err := a.User.GetProfile(ctx, ref)
			if err != nil {
				return "", "", err
			}
			return p.Username, p.RoleCode, nil
		case apikey.SubjectWorker:
			w, err := a.Worker.GetWorker(ctx, ref)
			if err != nil {
				return "", "", err
			}
			return w.Name, "worker", nil
		case apikey.SubjectCustomer:
			cu, err := a.Customer.Get(ctx, ref)
			if err != nil {
				return "", "", err
			}
			return cu.Name, "customer", nil
		}
		return "", "", nil
	}
}
