package app

// gRPC 服务端注册(债务偿还):契约 api/proto/boss/{quadlink,aaa,device,provision}/v1 已生成,
// 此处补齐实现并在 cmd/server 同进程直连(模块化单体,ADR-001)。

import (
	"errors"

	"google.golang.org/grpc"

	aaav1 "github.com/ymm-001/boss/api/proto/boss/aaa/v1"
	commonv1 "github.com/ymm-001/boss/api/proto/boss/common/v1"
	devicev1 "github.com/ymm-001/boss/api/proto/boss/device/v1"
	provisionv1 "github.com/ymm-001/boss/api/proto/boss/provision/v1"
	quadlinkv1 "github.com/ymm-001/boss/api/proto/boss/quadlink/v1"

	"github.com/ymm-001/boss/pkg/apitypes"
)

// defaultPacketLossAlarmPct 丢包率告警阈值(%)(与 cmd/collector 对齐)。
const defaultPacketLossAlarmPct = 5.0

// RegisterGRPC 在 grpc.Server 上注册全部服务间契约;每个服务仅依赖域接口,便于单测注入。
func RegisterGRPC(s *grpc.Server, a *Application) {
	quadlinkv1.RegisterQuadLinkServiceServer(s, &quadlinkGRPC{work: a.WorkOrder, quad: a.QuadLink, ord: a.Order})
	aaav1.RegisterAaaServiceServer(s, &aaaGRPC{aaaSvc: a.Aaa, auth: a.AaaAuth, cdr: a.Cdr, sessCtl: a.SessCtl})
	devicev1.RegisterDeviceIngestServiceServer(s, &deviceGRPC{
		dev: a.Device, alarm: a.Alarm, res: a.Resource, packetLossAlarmPct: defaultPacketLossAlarmPct,
	})
	provisionv1.RegisterProvisionServiceServer(s, &provisionGRPC{prov: a.Provision, ord: a.Order, aaaSvc: a.Aaa})
}

// toGRPCCode 统一错误码 → 契约枚举(数值与 pkg/apitypes 对齐)。
func toGRPCCode(c apitypes.Code) commonv1.Code {
	return commonv1.Code(int32(c))
}

// grpcCodeFor 领域错误 → 契约错误码;未命中一律内部错误。
func grpcCodeFor(err error, mapping map[error]apitypes.Code) commonv1.Code {
	for e, c := range mapping {
		if errors.Is(err, e) {
			return toGRPCCode(c)
		}
	}
	return commonv1.Code_CODE_INTERNAL
}
