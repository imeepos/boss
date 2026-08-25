// Package radius 自研 Go RADIUS 服务:基于 layeh.com/radius(RFC 2865/2866)。
// 定位:性能服务群独立部署(cmd/aaa),替换 FreeRADIUS 终态(技术栈方案 · 风险备选3)。
package radius

import (
	"context"
	"log"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"
	"layeh.com/radius/rfc2869"

	"github.com/ymm-001/boss/internal/domain/aaa"
	aaability "github.com/ymm-001/boss/internal/domain/aaa/billing"
)

// AuthLogWriter 认证日志写口(实现为 aaa.PGStore)。
type AuthLogWriter interface {
	AppendAuthLog(ctx context.Context, l aaa.AuthLog) (int64, error)
}

// Handler 处理 RADIUS 认证与计费请求,依赖域接口注入。
type Handler struct {
	Auth aaa.Authorizer
	CDR  aaability.Emitter
	Log  AuthLogWriter // 认证日志写口;nil=不记录(降级)
}

// ServeRADIUS 实现 radius.Handler,按 Code 分流认证/计费。
func (h *Handler) ServeRADIUS(w radius.ResponseWriter, r *radius.Request) {
	switch r.Code {
	case radius.CodeAccessRequest:
		h.serveAuth(w, r)
	case radius.CodeAccountingRequest:
		h.serveAccounting(w, r)
	default:
		w.Write(nil)
	}
}

// serveAuth 处理 Access-Request:按 User-Name(即 LOID)授权,放行或拒绝;结果写认证日志。
func (h *Handler) serveAuth(w radius.ResponseWriter, r *radius.Request) {
	loid := rfc2865.UserName_GetString(r.Packet)
	if loid == "" {
		h.reject(w, r)
		h.logAuth(loid, "FAILED")
		return
	}
	dec, err := h.Auth.Decide(context.Background(), loid)
	if err != nil || !dec.Authorize {
		h.reject(w, r)
		h.logAuth(loid, "FAILED")
		return
	}
	h.accept(w, r, dec)
	h.logAuth(loid, "SUCCESS")
}

// accept 组装 Access-Accept,下发带宽模板与 Session 超时。
func (h *Handler) accept(w radius.ResponseWriter, r *radius.Request, dec aaa.Decision) {
	resp := r.Response(radius.CodeAccessAccept)
	_ = rfc2865.ServiceType_Set(resp, rfc2865.ServiceType_Value_FramedUser)
	_ = rfc2865.SessionTimeout_Set(resp, rfc2865.SessionTimeout(dec.SessionTTL))
	_ = rfc2869.FramedPool_SetString(resp, dec.Bandwidth) // 带宽模板,阶段7可替换为厂商 VSA
	w.Write(resp)
}

// reject 组装 Access-Reject。
func (h *Handler) reject(w radius.ResponseWriter, r *radius.Request) {
	resp := r.Response(radius.CodeAccessReject)
	_ = rfc2865.ReplyMessage_SetString(resp, "auth rejected")
	w.Write(resp)
}

// serveAccounting 处理 Accounting-Request:转话单投递后回 Accounting-Response。
// 投递失败仍返回成功(RADIUS 计费协议层面不可拒绝),但记录错误供运维排查。
func (h *Handler) serveAccounting(w radius.ResponseWriter, r *radius.Request) {
	if h.CDR != nil {
		if err := h.CDR.Emit(context.Background(), h.toCDR(r)); err != nil {
			log.Printf("radius acct: cdr emit: %v", err)
		}
	}
	w.Write(r.Response(radius.CodeAccountingResponse))
}

// logAuth 写认证日志;降级静默失败。
func (h *Handler) logAuth(loid, result string) {
	if h.Log == nil || loid == "" {
		return
	}
	_, _ = h.Log.AppendAuthLog(context.Background(), aaa.AuthLog{Loid: loid, Result: result})
}

// toCDR 从计费请求提取话单字段。
func (h *Handler) toCDR(r *radius.Request) aaability.CDR {
	return aaability.CDR{
		LOID:         rfc2865.UserName_GetString(r.Packet),
		Username:     rfc2865.UserName_GetString(r.Packet),
		AcctStatus:   int(rfc2866.AcctStatusType_Get(r.Packet)),
		SessionID:    rfc2866.AcctSessionID_GetString(r.Packet),
		InputOctets:  uint64(rfc2866.AcctInputOctets_Get(r.Packet)),
		OutputOctets: uint64(rfc2866.AcctOutputOctets_Get(r.Packet)),
		NASIP:        rfc2865.NASIPAddress_Get(r.Packet).String(),
	}
}
