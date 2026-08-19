package workerapi

// W 师傅端门户登录:手机号 + 验证码/密码(worker/auth.yaml)。
// 验证码落 portal_sms_codes(与用户端共用表,scene=login);密码模式待 worker 域
// 暴露 LoginByStaffNo 后接入(password_hash 不经 app 层暴露)。

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// workerSmsCodeReq 发送验证码请求体。
type workerSmsCodeReq struct {
	Phone string `json:"phone" binding:"required"`
}

// workerSmsCodeHandler 发送登录验证码:师傅手机号须在职,验证码落库 5 分钟有效。
func workerSmsCodeHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req workerSmsCodeReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if _, err := findWorkerByPhone(c.Request.Context(), a, req.Phone); err != nil {
			respond(c, apitypes.CodeUnauthorized, nil)
			return
		}
		if err := a.Portal.IssueSms(c.Request.Context(), req.Phone, "login"); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"sent": true})
	}
}

// workerLoginReq 登录请求体(验证码/工号密码二选一,phone 为登录名)。
type workerLoginReq struct {
	Phone    string `json:"phone" binding:"required"`
	Mode     string `json:"mode" binding:"required"`
	SmsCode  string `json:"smsCode"`
	Password string `json:"password"`
}

// workerLoginHandler 师傅登录:phone 定位师傅(唯一索引待 DB 商议),签发 workerJWT。
func workerLoginHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req workerLoginReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		w, err := findWorkerByPhone(c.Request.Context(), a, req.Phone)
		if err != nil {
			respond(c, apitypes.CodeUnauthorized, nil)
			return
		}
		if msg := checkWorkerLogin(a, c, w, &req); msg != "" {
			respond(c, apitypes.CodeUnauthorized, gin.H{"reason": msg})
			return
		}
		token, err := signWorkerToken(w.ID, w.Name)
		if err != nil {
			respond(c, apitypes.CodeInternal, nil)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"token": token, "workerId": w.ID})
	}
}

// findWorkerByPhone 按手机号找在职师傅;未命中/离职返回错误。
func findWorkerByPhone(ctx context.Context, a *app.Application, phone string) (*worker.Worker, error) {
	list, err := a.Worker.ListWorkers(ctx, 0)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if strings.TrimSpace(list[i].Phone) == strings.TrimSpace(phone) && list[i].Status == 1 {
			return &list[i], nil
		}
	}
	return nil, worker.ErrNotFound
}

// checkWorkerLogin 校验登录方式;返回空串=通过。
func checkWorkerLogin(a *app.Application, c *gin.Context, w *worker.Worker, req *workerLoginReq) string {
	switch req.Mode {
	case "sms":
		ok, err := a.Portal.ConsumeSms(c.Request.Context(), w.Phone, "login", strings.TrimSpace(req.SmsCode))
		if err != nil || !ok {
			return "invalid sms code"
		}
		return ""
	case "password":
		// worker 域未暴露密码校验服务;待域补 LoginByStaffNo 后接入。
		return "password mode not supported yet"
	default:
		return "unknown mode"
	}
}
