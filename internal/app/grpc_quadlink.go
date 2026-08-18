package app

// gRPC quadlink/v1 服务端:扫码绑定(不一致拒)/拆机必扫码/任一码反查/对账。
// 语义与 HTTP 扫码闭环一致(见 http_scan.go),错误码对齐 common/v1。

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/ymm-001/boss/api/proto/boss/common/v1"
	quadlinkv1 "github.com/ymm-001/boss/api/proto/boss/quadlink/v1"

	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/quadlink"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// quadlinkGRPC QuadLinkService 实现(仅依赖域接口,便于单测注入)。
type quadlinkGRPC struct {
	quadlinkv1.UnimplementedQuadLinkServiceServer
	work order.WorkOrderService
	quad quadlink.QuadLinkService
	ord  order.OrderService
}

// quadErrMap 扫码域错误 → 契约错误码(与 respondScanErr 对齐)。
var quadErrMap = map[error]apitypes.Code{
	quadlink.ErrScanMismatch: apitypes.CodeScanMismatch,
	quadlink.ErrNotPrebound:  apitypes.CodeStateInvalid,
	quadlink.ErrScanRequired: apitypes.CodeInvalidParam,
	quadlink.ErrNotFound:     apitypes.CodeNotFound,
	order.ErrOrderNotFound:   apitypes.CodeNotFound,
}

// ScanBind 扫码绑定(环节9):ticket_no → orderID;MATCH 且推进成功返回 stage_advanced。
func (s *quadlinkGRPC) ScanBind(ctx context.Context, req *quadlinkv1.ScanBindRequest) (*quadlinkv1.ScanBindResponse, error) {
	tk, err := s.work.GetDispatchTicketByNo(ctx, req.TicketNo)
	if err != nil {
		return &quadlinkv1.ScanBindResponse{Code: grpcCodeFor(err, quadErrMap)}, nil
	}
	workerID, workerName := req.WorkerId, req.WorkerName
	if workerID == 0 { // 未带师傅:回退工单档案(与 HTTP 一致)
		workerID, workerName = tk.WorkerID, tk.WorkerName
	}
	result, err := s.quad.VerifyScan(ctx, quadlink.ScanReq{
		OrderID: tk.OrderID, WorkerID: workerID, WorkerName: workerName,
		ScannedEPC: req.Epc, OfflineCalc: req.Offline,
	})
	resp := &quadlinkv1.ScanBindResponse{Result: result}
	if err != nil {
		resp.Code = grpcCodeFor(err, quadErrMap)
		return resp, nil
	}
	resp.Code = commonv1.Code_CODE_OK
	if result == "MATCH" {
		if err := s.ord.ScanBind(ctx, tk.OrderID); err != nil {
			resp.Code = grpcCodeFor(err, quadErrMap)
			return resp, nil
		}
		resp.StageAdvanced = true
	}
	return resp, nil
}

// DismantleScan 拆机扫码解绑(强制):不扫码/不一致均拒;成功置 UNLINKED。
func (s *quadlinkGRPC) DismantleScan(ctx context.Context, req *quadlinkv1.DismantleScanRequest) (*quadlinkv1.DismantleScanResponse, error) {
	tk, err := s.work.GetDispatchTicketByNo(ctx, req.TicketNo)
	if err != nil {
		return &quadlinkv1.DismantleScanResponse{Code: grpcCodeFor(err, quadErrMap)}, nil
	}
	if err := s.quad.UnbindRequireScan(ctx, tk.OrderID, req.Epc); err != nil {
		return &quadlinkv1.DismantleScanResponse{Code: grpcCodeFor(err, quadErrMap)}, nil
	}
	return &quadlinkv1.DismantleScanResponse{Code: commonv1.Code_CODE_OK, Unlinked: true}, nil
}

// Query 任一码反查四码;无 code 字段,失败走 gRPC status。
func (s *quadlinkGRPC) Query(ctx context.Context, req *quadlinkv1.QueryRequest) (*quadlinkv1.QuadLink, error) {
	var (
		link *quadlink.QuadLink
		err  error
	)
	switch k := req.Key.(type) {
	case *quadlinkv1.QueryRequest_AssetId:
		link, err = s.quad.GetByAsset(ctx, k.AssetId)
	case *quadlinkv1.QueryRequest_CustomerId:
		link, err = s.quad.GetByCustomer(ctx, k.CustomerId)
	case *quadlinkv1.QueryRequest_PortId:
		link, err = s.quad.GetByPort(ctx, k.PortId)
	case *quadlinkv1.QueryRequest_AddressId:
		link, err = s.quad.GetByAddress(ctx, k.AddressId)
	default:
		return nil, status.Error(codes.InvalidArgument, "quadlink: missing key")
	}
	if err != nil {
		if quadErrMap[err] == apitypes.CodeNotFound {
			return nil, status.Error(codes.NotFound, "quadlink: not found")
		}
		return nil, status.Error(codes.Internal, "quadlink: query failed")
	}
	return &quadlinkv1.QuadLink{
		Id: link.ID, AssetId: link.AssetID, CustomerId: link.CustomerID,
		PortId: link.PortID, AddressId: link.AddressID,
		LegalEntityId: link.LegalEntityID, LegalEntityName: link.LegalEntityName,
		Status: link.Status,
	}, nil
}

// Reconcile 四码对账任务;无 code 字段,失败走 gRPC status。
func (s *quadlinkGRPC) Reconcile(ctx context.Context, _ *quadlinkv1.ReconcileRequest) (*quadlinkv1.ReconcileReport, error) {
	rep, err := s.quad.Reconcile(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "quadlink: reconcile failed")
	}
	return &quadlinkv1.ReconcileReport{
		Total: int32(rep.Total), Linked: int32(rep.Linked), Conflict: int32(rep.Conflict), Unlinked: int32(rep.Unlinked),
	}, nil
}
