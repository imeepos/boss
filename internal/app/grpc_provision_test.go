package app

// gRPC provision/v1 契约单测:任务入队(模板编码→模板/订单→LO账号)、失败重试、状态查询。

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/ymm-001/boss/api/proto/boss/common/v1"
	provisionv1 "github.com/ymm-001/boss/api/proto/boss/provision/v1"

	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/provision"
)

type stubProv struct {
	provision.ProvisionService
	templates []provision.Template
	created   provision.Task
	byNo      *provision.Task
	byNoErr   error
	retryErr  error
}

func (s *stubProv) ListTemplates(context.Context) ([]provision.Template, error) {
	return s.templates, nil
}
func (s *stubProv) CreateTask(_ context.Context, t provision.Task) (int64, error) {
	s.created = t
	return 42, nil
}
func (s *stubProv) GetTaskByNo(context.Context, string) (*provision.Task, error) {
	return s.byNo, s.byNoErr
}
func (s *stubProv) RetryTask(context.Context, int64, int16) error { return s.retryErr }

type stubOrderTrack struct {
	order.OrderService
	o   *order.Order
	err error
}

func (s *stubOrderTrack) Track(_ context.Context, _ int64) (*order.Order, []order.StageLog, error) {
	return s.o, nil, s.err
}

type stubAaaByCustomer struct {
	aaa.AaaService
	lo  *aaa.LoAccount
	err error
}

func (s *stubAaaByCustomer) GetLoAccountByCustomer(context.Context, int64) (*aaa.LoAccount, error) {
	return s.lo, s.err
}

func provisionGRPCWith(p *stubProv, o *stubOrderTrack, a *stubAaaByCustomer) *provisionGRPC {
	return &provisionGRPC{prov: p, ord: o, aaaSvc: a}
}

func TestProvisionGRPC_EnqueueTask(t *testing.T) {
	ctx := context.Background()

	t.Run("入队成功", func(t *testing.T) {
		p := &stubProv{templates: []provision.Template{{ID: 1, Code: "TPL-FTTH"}}}
		o := &stubOrderTrack{o: &order.Order{CustomerID: 7}}
		a := &stubAaaByCustomer{lo: &aaa.LoAccount{ID: 88}}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			provisionv1.RegisterProvisionServiceServer(s, provisionGRPCWith(p, o, a))
		})
		resp, err := provisionv1.NewProvisionServiceClient(conn).EnqueueTask(ctx, &provisionv1.Task{
			OrderId: 9, TaskNo: "TASK-1", StageEvent: "preConfigOLT", TemplateCode: "TPL-FTTH",
		})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Code != commonv1.Code_CODE_OK || resp.TaskNo != "TASK-1" || resp.Status != "PENDING" {
			t.Fatalf("resp=%+v", resp)
		}
		want := provision.Task{TaskNo: "TASK-1", OrderID: 9, StageEvent: "preConfigOLT", LoAccountID: 88, TemplateID: 1, Status: "PENDING"}
		if p.created != want {
			t.Fatalf("created=%+v, want %+v", p.created, want)
		}
	})

	t.Run("未知模板 42200", func(t *testing.T) {
		p := &stubProv{templates: []provision.Template{{ID: 1, Code: "TPL-FTTH"}}}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			provisionv1.RegisterProvisionServiceServer(s, provisionGRPCWith(p, &stubOrderTrack{}, &stubAaaByCustomer{}))
		})
		resp, err := provisionv1.NewProvisionServiceClient(conn).EnqueueTask(ctx, &provisionv1.Task{TemplateCode: "TPL-NOPE"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Code != commonv1.Code_CODE_INVALID_PARAM {
			t.Fatalf("code=%v, want INVALID_PARAM", resp.Code)
		}
	})

	t.Run("订单不存在 404", func(t *testing.T) {
		p := &stubProv{templates: []provision.Template{{ID: 1, Code: "TPL-FTTH"}}}
		o := &stubOrderTrack{err: order.ErrOrderNotFound}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			provisionv1.RegisterProvisionServiceServer(s, provisionGRPCWith(p, o, &stubAaaByCustomer{}))
		})
		resp, err := provisionv1.NewProvisionServiceClient(conn).EnqueueTask(ctx, &provisionv1.Task{OrderId: 9, TemplateCode: "TPL-FTTH"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Code != commonv1.Code_CODE_NOT_FOUND {
			t.Fatalf("code=%v, want NOT_FOUND", resp.Code)
		}
	})
}

func TestProvisionGRPC_RetryTask(t *testing.T) {
	ctx := context.Background()

	t.Run("失败重试回 PENDING", func(t *testing.T) {
		p := &stubProv{byNo: &provision.Task{ID: 5, TaskNo: "TASK-1", Status: "FAILED"}}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			provisionv1.RegisterProvisionServiceServer(s, provisionGRPCWith(p, &stubOrderTrack{}, &stubAaaByCustomer{}))
		})
		resp, err := provisionv1.NewProvisionServiceClient(conn).RetryTask(ctx, &provisionv1.RetryTaskRequest{TaskNo: "TASK-1"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Code != commonv1.Code_CODE_OK || resp.Status != "PENDING" {
			t.Fatalf("resp=%+v", resp)
		}
	})

	t.Run("任务不存在 404", func(t *testing.T) {
		p := &stubProv{byNoErr: provision.ErrTaskNotFound}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			provisionv1.RegisterProvisionServiceServer(s, provisionGRPCWith(p, &stubOrderTrack{}, &stubAaaByCustomer{}))
		})
		resp, err := provisionv1.NewProvisionServiceClient(conn).RetryTask(ctx, &provisionv1.RetryTaskRequest{TaskNo: "TASK-NOPE"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Code != commonv1.Code_CODE_NOT_FOUND {
			t.Fatalf("code=%v, want NOT_FOUND", resp.Code)
		}
	})

	t.Run("非失败态拒 40910", func(t *testing.T) {
		p := &stubProv{byNo: &provision.Task{ID: 5, TaskNo: "TASK-1", Status: "DONE"}, retryErr: provision.ErrIllegalTransition}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			provisionv1.RegisterProvisionServiceServer(s, provisionGRPCWith(p, &stubOrderTrack{}, &stubAaaByCustomer{}))
		})
		resp, err := provisionv1.NewProvisionServiceClient(conn).RetryTask(ctx, &provisionv1.RetryTaskRequest{TaskNo: "TASK-1"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Code != commonv1.Code_CODE_STATE_INVALID {
			t.Fatalf("code=%v, want STATE_INVALID", resp.Code)
		}
	})
}

func TestProvisionGRPC_GetTask(t *testing.T) {
	ctx := context.Background()

	t.Run("命中返回模板编码", func(t *testing.T) {
		p := &stubProv{
			templates: []provision.Template{{ID: 1, Code: "TPL-FTTH"}},
			byNo:      &provision.Task{ID: 5, TaskNo: "TASK-1", OrderID: 9, StageEvent: "activateUser", TemplateID: 1, Status: "DONE"},
		}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			provisionv1.RegisterProvisionServiceServer(s, provisionGRPCWith(p, &stubOrderTrack{}, &stubAaaByCustomer{}))
		})
		resp, err := provisionv1.NewProvisionServiceClient(conn).GetTask(ctx, &provisionv1.GetTaskRequest{TaskNo: "TASK-1"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.TaskNo != "TASK-1" || resp.TemplateCode != "TPL-FTTH" || resp.StageEvent != "activateUser" {
			t.Fatalf("resp=%+v", resp)
		}
	})

	t.Run("未命中 NotFound", func(t *testing.T) {
		p := &stubProv{byNoErr: provision.ErrTaskNotFound}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			provisionv1.RegisterProvisionServiceServer(s, provisionGRPCWith(p, &stubOrderTrack{}, &stubAaaByCustomer{}))
		})
		_, err := provisionv1.NewProvisionServiceClient(conn).GetTask(ctx, &provisionv1.GetTaskRequest{TaskNo: "TASK-NOPE"})
		if status.Code(err) != codes.NotFound {
			t.Fatalf("err=%v, want NotFound", err)
		}
	})
}
