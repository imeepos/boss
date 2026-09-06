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
	Auth         aaa.Authorizer
	CDR          aaability.Emitter
	Log          AuthLogWriter         // 认证日志写口;nil=不记录(降级)
	Sessions     aaa.SessionMaintainer // 计账链路会话维护;nil=跳过
	Gate         aaa.SessionGate       // 并发会话闸口;nil 或 SessionLimit<=0 不限制
	SessionLimit int                   // 同一 LOID 在线占用上限(全局)
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

// serveAuth 处理 Access-Request:授权决策 + 并发闸口,结果写认证日志(失败带原因)。
func (h *Handler) serveAuth(w radius.ResponseWriter, r *radius.Request) {
	loid := rfc2865.UserName_GetString(r.Packet)
	if loid == "" {
		h.reject(w, r)
		h.logAuth(loid, "FAILED", "")
		return
	}
	dec, err := h.Auth.Decide(context.Background(), loid)
	if err != nil || !dec.Authorize {
		h.reject(w, r)
		h.logAuth(loid, "FAILED", "")
		return
	}
	if reason := h.blockReason(context.Background(), loid); reason != "" {
		h.reject(w, r)
		h.logAuth(loid, "FAILED", reason)
		return
	}
	h.accept(w, r, dec)
	h.logAuth(loid, "SUCCESS", "")
}

// blockReason 并发会话超限返回标注原因;闸口自身故障放行并留痕
// (DB 不可用时 Decide 已先失败,此处不放大故障面)。
func (h *Handler) blockReason(ctx context.Context, loid string) string {
	if h.Gate == nil || h.SessionLimit <= 0 {
		return ""
	}
	allow, current, err := h.Gate.AllowNewSession(ctx, loid, h.SessionLimit)
	if err != nil {
		log.Printf("[aaa] session gate FAILED loid=%s: %v", loid, err)
		return ""
	}
	if allow {
		return ""
	}
	log.Printf("[aaa] concurrent limit REJECT loid=%s online=%d limit=%d", loid, current, h.SessionLimit)
	return aaa.AuthFailReasonConcurrent
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

// serveAccounting 处理 Accounting-Request:话单投递 + 会话维护后回 Accounting-Response。
// 任一下游失败仍返回成功(RADIUS 计费协议层面不可拒绝),但留日志供运维排查。
func (h *Handler) serveAccounting(w radius.ResponseWriter, r *radius.Request) {
	cdr := h.toCDR(r)
	if h.CDR != nil {
		if err := h.CDR.Emit(context.Background(), cdr); err != nil {
			log.Printf("radius acct: cdr emit: %v", err)
		}
	}
	h.maintainSession(context.Background(), cdr)
	w.Write(r.Response(radius.CodeAccountingResponse))
}

// maintainSession 按 Acct-Status-Type 维护在线会话:Start 幂等建/Interim 累加/Stop 关闭。
func (h *Handler) maintainSession(ctx context.Context, cdr aaability.CDR) {
	if h.Sessions == nil {
		return
	}
	var err error
	switch cdr.AcctStatus {
	case aaa.AcctStatusStart:
		_, err = h.Sessions.StartSession(ctx, aaa.SessionRecord{
			Loid: cdr.LOID, SessionID: cdr.SessionID, NasIP: cdr.NASIP,
			InputOctets: int64(cdr.InputOctets), OutputOctets: int64(cdr.OutputOctets),
		})
	case aaa.AcctStatusInterim:
		err = h.Sessions.TouchSessionTraffic(ctx, cdr.LOID, cdr.SessionID, int64(cdr.InputOctets), int64(cdr.OutputOctets))
	case aaa.AcctStatusStop:
		_, err = h.Sessions.StopSession(ctx, cdr.LOID, cdr.SessionID, aaa.CloseReasonAcctStop)
	}
	if err != nil {
		log.Printf("[aaa] session maintain FAILED loid=%s session=%s status=%d: %v",
			cdr.LOID, cdr.SessionID, cdr.AcctStatus, err)
	}
}

// logAuth 写认证日志;reason 标注失败原因(并发超限等),写失败留 [aaa] 痕。
func (h *Handler) logAuth(loid, result, reason string) {
	if h.Log == nil || loid == "" {
		return
	}
	l := aaa.AuthLog{Loid: loid, Result: result, Reason: reason}
	if _, err := h.Log.AppendAuthLog(context.Background(), l); err != nil {
		log.Printf("[aaa] auth log write FAILED loid=%s result=%s: %v", loid, result, err)
	}
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