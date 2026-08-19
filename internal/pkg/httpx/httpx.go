package httpx

import (
	"context"
	"errors"

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

// RecordAudit 记录关键操作审计(异步、尽力而为);未装配审计 writer 时静默跳过。
func RecordAudit(a *app.Application, c *gin.Context, action, targetType, targetID string, detail map[string]any) {
	if a == nil || a.Audit == nil {
		return
	}
	_ = a.Audit.Write(c.Request.Context(), audit.Event{
		AccountID: ClaimsAccountID(c), Action: action, TargetType: targetType, TargetID: targetID,
		Detail: detail, IP: c.ClientIP(),
	})
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
