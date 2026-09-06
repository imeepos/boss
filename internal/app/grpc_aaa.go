package app

// gRPC aaa/v1 服务端:停复机即时生效、授权查询(LOID→带宽/QoS/状态)、实时话单入账。
// 语义与 cmd/aaa(PGAuthorizer + 话单双写)一致,契约服务在 cmd/server 内直连。

import (
	"context"
	"errors"

	aaav1 "github.com/ymm-001/boss/api/proto/boss/aaa/v1"
	commonv1 "github.com/ymm-001/boss/api/proto/boss/common/v1"

	"github.com/ymm-001/boss/internal/domain/aaa"
	aaability "github.com/ymm-001/boss/internal/domain/aaa/billing"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// aaaGRPC AaaService 实现(仅依赖域接口,便于单测注入)。
type aaaGRPC struct {
	aaav1.UnimplementedAaaServiceServer
	aaaSvc  aaa.AaaService
	auth    aaa.Authorizer
	cdr     aaability.Emitter
	sessCtl *aaa.SessionControlService // 可空:停机成功联动下线全部在线会话(AAA-A2)
}

// aaaErrMap AAA 域错误 → 契约错误码。
var aaaErrMap = map[error]apitypes.Code{
	aaa.ErrNotFound:          apitypes.CodeNotFound,
	aaa.ErrIllegalTransition: apitypes.CodeStateInvalid,
}

// Suspend 停机(欠费/人工):ACTIVE→SUSPENDED,即时生效;装配会话控制时联动下线全部在线会话。
func (s *aaaGRPC) Suspend(ctx context.Context, req *aaav1.SuspendRequest) (*aaav1.OpResponse, error) {
	suspend := s.aaaSvc.SuspendLoAccount
	if s.sessCtl != nil {
		suspend = s.sessCtl.SuspendWithOffline
	}
	if err := suspend(ctx, req.LoAccountId); err != nil {
		return &aaav1.OpResponse{Code: grpcCodeFor(err, aaaErrMap)}, nil
	}
	return &aaav1.OpResponse{Code: commonv1.Code_CODE_OK, Effective: true}, nil
}

// Resume 复机(缴费):SUSPENDED→ACTIVE,即时生效。
func (s *aaaGRPC) Resume(ctx context.Context, req *aaav1.ResumeRequest) (*aaav1.OpResponse, error) {
	if err := s.aaaSvc.ResumeLoAccount(ctx, req.LoAccountId); err != nil {
		return &aaav1.OpResponse{Code: grpcCodeFor(err, aaaErrMap)}, nil
	}
	return &aaav1.OpResponse{Code: commonv1.Code_CODE_OK, Effective: true}, nil
}

// GetAuthorization 授权查询:LOID → 状态/带宽/QoS(停机返回 status=SUSPENDED,注销返回 CLOSED)。
func (s *aaaGRPC) GetAuthorization(ctx context.Context, req *aaav1.GetAuthorizationRequest) (*aaav1.Authorization, error) {
	lo, err := s.aaaSvc.GetLoAccountByLoid(ctx, req.Loid)
	if err != nil {
		return &aaav1.Authorization{Code: grpcCodeFor(err, aaaErrMap)}, nil
	}
	dec, err := s.auth.Decide(ctx, req.Loid)
	// ErrSuspended/ErrClosed 是有业务的拒绝态:标记已查到的账号状态而非按错误码返回。
	if err != nil && !errors.Is(err, aaa.ErrSuspended) && !errors.Is(err, aaa.ErrClosed) {
		return &aaav1.Authorization{Code: grpcCodeFor(err, aaaErrMap)}, nil
	}
	return &aaav1.Authorization{
		Code: commonv1.Code_CODE_OK, Status: lo.Status,
		Bandwidth: dec.Bandwidth, QosTemplate: dec.QosTemplate,
	}, nil
}

// EmitCDR 实时话单入账(RADIUS Acct → 投递链路)。
// 口径(A4 核对):旁路只进话单链路逐包入 cdr_records,不参与在线会话维护——在线会话流量的
// 唯一写入口是 RADIUS 主链路(radius.Handler.maintainSession,覆盖式累计口径),不存在第二套
// 会话语义。octets 原样透传不换算;假设上报方遵循 RFC 2866 累计口径,验证方法:
// SessionMaintainer 调用方仅 radius/handler.go 且 aaaGRPC 无 Sessions 依赖(grep 可复核)。
func (s *aaaGRPC) EmitCDR(ctx context.Context, req *aaav1.CDR) (*aaav1.OpResponse, error) {
	cdr := aaability.CDR{
		LOID: req.Loid, Username: req.Loid, SessionID: req.SessionId,
		AcctStatus:  acctStatusFromEvent(req.EventType),
		InputOctets: req.InputOctets, OutputOctets: req.OutputOctets,
	}
	if req.FinishedAt > req.StartedAt {
		cdr.SessionTime = uint32(req.FinishedAt - req.StartedAt)
	}
	if err := s.cdr.Emit(ctx, cdr); err != nil {
		return &aaav1.OpResponse{Code: commonv1.Code_CODE_DOWNSTREAM_ERR}, nil
	}
	return &aaav1.OpResponse{Code: commonv1.Code_CODE_OK, Effective: true}, nil
}

// acctStatusFromEvent 事件类型 → RADIUS AcctStatusType(1 Start/2 Stop/3 Interim)。
func acctStatusFromEvent(event string) int {
	switch event {
	case "Start":
		return 1
	case "Stop":
		return 2
	case "Interim-Update":
		return 3
	default:
		return 0
	}
}
