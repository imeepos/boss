package app

// gRPC quadlink/v1 契约单测:扫码绑定(不一致拒 40920)/拆机必扫码/任一码反查/对账。

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/ymm-001/boss/api/proto/boss/common/v1"
	quadlinkv1 "github.com/ymm-001/boss/api/proto/boss/quadlink/v1"

	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/quadlink"
)

type stubQuadlink struct {
	quadlink.QuadLinkService
	byAsset, byCustomer, byPort, byAddress *quadlink.QuadLink
	byErr                                  error
	verifyRes                              string
	verifyErr                              error
	unbindErr                              error
	rep                                    *quadlink.ReconcileReport
	reconcileErr                           error
}

func (s *stubQuadlink) GetByAsset(_ context.Context, _ int64) (*quadlink.QuadLink, error) {
	return s.byAsset, s.byErr
}
func (s *stubQuadlink) GetByCustomer(_ context.Context, _ int64) (*quadlink.QuadLink, error) {
	return s.byCustomer, s.byErr
}
func (s *stubQuadlink) GetByPort(_ context.Context, _ int64) (*quadlink.QuadLink, error) {
	return s.byPort, s.byErr
}
func (s *stubQuadlink) GetByAddress(_ context.Context, _ int64) (*quadlink.QuadLink, error) {
	return s.byAddress, s.byErr
}
func (s *stubQuadlink) VerifyScan(_ context.Context, _ quadlink.ScanReq) (string, error) {
	return s.verifyRes, s.verifyErr
}
func (s *stubQuadlink) UnbindRequireScan(_ context.Context, _ int64, _ string) error {
	return s.unbindErr
}
func (s *stubQuadlink) Reconcile(_ context.Context) (*quadlink.ReconcileReport, error) {
	return s.rep, s.reconcileErr
}

type stubWorkOrderGRPC struct {
	order.WorkOrderService
	ticket *order.DispatchTicket
	err    error
}

func (s *stubWorkOrderGRPC) GetDispatchTicketByNo(_ context.Context, _ string) (*order.DispatchTicket, error) {
	return s.ticket, s.err
}

type stubOrderGRPC struct {
	order.OrderService
	scanBound int64
	err       error
}

func (s *stubOrderGRPC) ScanBind(_ context.Context, id int64) error {
	s.scanBound = id
	return s.err
}

func quadlinkGRPCWith(q *stubQuadlink, wk *stubWorkOrderGRPC, o *stubOrderGRPC) *quadlinkGRPC {
	return &quadlinkGRPC{work: wk, quad: q, ord: o}
}

func TestQuadlinkGRPC_ScanBind(t *testing.T) {
	ctx := context.Background()
	wk := &stubWorkOrderGRPC{ticket: &order.DispatchTicket{TicketNo: "TIC-1", OrderID: 5, WorkerID: 9, WorkerName: "师傅甲"}}

	t.Run("MATCH 推进环节9", func(t *testing.T) {
		q := &stubQuadlink{verifyRes: "MATCH"}
		o := &stubOrderGRPC{}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			quadlinkv1.RegisterQuadLinkServiceServer(s, quadlinkGRPCWith(q, wk, o))
		})
		client := quadlinkv1.NewQuadLinkServiceClient(conn)
		resp, err := client.ScanBind(ctx, &quadlinkv1.ScanBindRequest{TicketNo: "TIC-1", Epc: "EPC-1", WorkerId: 9, WorkerName: "师傅甲"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Code != commonv1.Code_CODE_OK || resp.Result != "MATCH" || !resp.StageAdvanced {
			t.Fatalf("resp=%+v", resp)
		}
		if o.scanBound != 5 {
			t.Fatalf("scanBound=%d, want 5", o.scanBound)
		}
	})

	t.Run("不一致拒 40920", func(t *testing.T) {
		q := &stubQuadlink{verifyErr: quadlink.ErrScanMismatch}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			quadlinkv1.RegisterQuadLinkServiceServer(s, quadlinkGRPCWith(q, wk, &stubOrderGRPC{}))
		})
		resp, err := quadlinkv1.NewQuadLinkServiceClient(conn).ScanBind(ctx, &quadlinkv1.ScanBindRequest{TicketNo: "TIC-1", Epc: "EPC-X"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Code != commonv1.Code_CODE_SCAN_MISMATCH {
			t.Fatalf("code=%v, want SCAN_MISMATCH", resp.Code)
		}
	})

	t.Run("工单不存在 404", func(t *testing.T) {
		wk2 := &stubWorkOrderGRPC{err: order.ErrOrderNotFound}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			quadlinkv1.RegisterQuadLinkServiceServer(s, quadlinkGRPCWith(&stubQuadlink{}, wk2, &stubOrderGRPC{}))
		})
		resp, err := quadlinkv1.NewQuadLinkServiceClient(conn).ScanBind(ctx, &quadlinkv1.ScanBindRequest{TicketNo: "TIC-NOPE"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Code != commonv1.Code_CODE_NOT_FOUND {
			t.Fatalf("code=%v, want NOT_FOUND", resp.Code)
		}
	})
}

func TestQuadlinkGRPC_DismantleScan(t *testing.T) {
	ctx := context.Background()
	wk := &stubWorkOrderGRPC{ticket: &order.DispatchTicket{TicketNo: "TIC-1", OrderID: 5}}

	t.Run("成功解绑", func(t *testing.T) {
		conn := newBufConnServer(t, func(s *grpc.Server) {
			quadlinkv1.RegisterQuadLinkServiceServer(s, quadlinkGRPCWith(&stubQuadlink{}, wk, &stubOrderGRPC{}))
		})
		resp, err := quadlinkv1.NewQuadLinkServiceClient(conn).DismantleScan(ctx, &quadlinkv1.DismantleScanRequest{TicketNo: "TIC-1", Epc: "EPC-1"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Code != commonv1.Code_CODE_OK || !resp.Unlinked {
			t.Fatalf("resp=%+v", resp)
		}
	})

	t.Run("不扫码拦 42200", func(t *testing.T) {
		q := &stubQuadlink{unbindErr: quadlink.ErrScanRequired}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			quadlinkv1.RegisterQuadLinkServiceServer(s, quadlinkGRPCWith(q, wk, &stubOrderGRPC{}))
		})
		resp, err := quadlinkv1.NewQuadLinkServiceClient(conn).DismantleScan(ctx, &quadlinkv1.DismantleScanRequest{TicketNo: "TIC-1"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Code != commonv1.Code_CODE_INVALID_PARAM {
			t.Fatalf("code=%v, want INVALID_PARAM", resp.Code)
		}
	})
}

func TestQuadlinkGRPC_Query(t *testing.T) {
	ctx := context.Background()
	link := &quadlink.QuadLink{ID: 1, AssetID: 11, CustomerID: 22, PortID: 33, AddressID: 44, Status: "LINKED"}

	t.Run("按资产反查", func(t *testing.T) {
		q := &stubQuadlink{byAsset: link}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			quadlinkv1.RegisterQuadLinkServiceServer(s, quadlinkGRPCWith(q, &stubWorkOrderGRPC{}, &stubOrderGRPC{}))
		})
		resp, err := quadlinkv1.NewQuadLinkServiceClient(conn).Query(ctx, &quadlinkv1.QueryRequest{Key: &quadlinkv1.QueryRequest_AssetId{AssetId: 11}})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Status != "LINKED" || resp.PortId != 33 {
			t.Fatalf("resp=%+v", resp)
		}
	})

	t.Run("未命中 NotFound", func(t *testing.T) {
		q := &stubQuadlink{byErr: quadlink.ErrNotFound}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			quadlinkv1.RegisterQuadLinkServiceServer(s, quadlinkGRPCWith(q, &stubWorkOrderGRPC{}, &stubOrderGRPC{}))
		})
		_, err := quadlinkv1.NewQuadLinkServiceClient(conn).Query(ctx, &quadlinkv1.QueryRequest{Key: &quadlinkv1.QueryRequest_PortId{PortId: 33}})
		if status.Code(err) != codes.NotFound {
			t.Fatalf("err=%v, want NotFound", err)
		}
	})

	t.Run("缺 key InvalidArgument", func(t *testing.T) {
		conn := newBufConnServer(t, func(s *grpc.Server) {
			quadlinkv1.RegisterQuadLinkServiceServer(s, quadlinkGRPCWith(&stubQuadlink{}, &stubWorkOrderGRPC{}, &stubOrderGRPC{}))
		})
		_, err := quadlinkv1.NewQuadLinkServiceClient(conn).Query(ctx, &quadlinkv1.QueryRequest{})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("err=%v, want InvalidArgument", err)
		}
	})
}

func TestQuadlinkGRPC_Reconcile(t *testing.T) {
	ctx := context.Background()
	rep := &quadlink.ReconcileReport{Total: 4, Linked: 2, Conflict: 1, Unlinked: 1}
	conn := newBufConnServer(t, func(s *grpc.Server) {
		quadlinkv1.RegisterQuadLinkServiceServer(s, quadlinkGRPCWith(&stubQuadlink{rep: rep}, &stubWorkOrderGRPC{}, &stubOrderGRPC{}))
	})
	got, err := quadlinkv1.NewQuadLinkServiceClient(conn).Reconcile(ctx, &quadlinkv1.ReconcileRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 4 || got.Conflict != 1 {
		t.Fatalf("got=%+v", got)
	}
}
